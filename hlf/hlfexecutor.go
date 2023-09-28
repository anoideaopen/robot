package hlf

import (
	"context"
	"time"

	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/common/filter"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	chctx "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/pkg/errors"
)

// ExecuteOptions for Execute method
type ExecuteOptions struct {
	// ExecuteTimeout is a timeout for Execute method
	ExecuteTimeout time.Duration
	// WaitCommitAttempts is a number of attempts to wait for commit event
	WaitCommitAttempts uint
	// WaitCommitAttemptTimeout is a timeout for each attempt to wait for commit event
	WaitCommitAttemptTimeout time.Duration
}

type hlfExecutor struct {
	chClient *channel.Client
	chCtx    chctx.Channel
	execOpts ExecuteOptions
}

func (he *hlfExecutor) execute(ctx context.Context, request channel.Request) (channel.Response, error) {
	var options []channel.RequestOption

	executeTimeout := he.chCtx.EndpointConfig().Timeout(fab.Execute)
	if he.execOpts.ExecuteTimeout > 0 {
		executeTimeout = he.execOpts.ExecuteTimeout
	}
	options = append(options, channel.WithTimeout(fab.Execute, executeTimeout))

	// use all channel peers from connection profile in the peers selection algorithm
	options = append(options,
		channel.WithTargetFilter(
			filter.NewEndpointFilter(
				he.chCtx, filter.EndorsingPeer)))

	h := invoke.NewSelectAndEndorseHandler(
		invoke.NewEndorsementValidationHandler(
			invoke.NewSignatureValidationHandler(
				&commitTxHandler{
					ctx:      ctx,
					execOpts: he.execOpts,
				}),
		),
	)

	return he.chClient.InvokeHandler(h, request, options...)
}

// CommitTxHandler for committing transactions
type commitTxHandler struct {
	ctx      context.Context
	execOpts ExecuteOptions
}

// Handle handles commit tx
func (cth *commitTxHandler) Handle(reqCtx *invoke.RequestContext, clientCtx *invoke.ClientContext) {
	// register tx event
	reg, statusNotifier, err := clientCtx.
		EventService.RegisterTxStatusEvent(
		string(reqCtx.Response.TransactionID))
	if err != nil {
		reqCtx.Error = errors.Wrap(errors.WithStack(err), "error registering for TxStatus event")
		return
	}
	defer clientCtx.EventService.Unregister(reg)

	tx, err := clientCtx.Transactor.CreateTransaction(
		fab.TransactionRequest{
			Proposal:          reqCtx.Response.Proposal,
			ProposalResponses: reqCtx.Response.Responses,
		})
	if err != nil {
		reqCtx.Error = errors.Wrap(errors.WithStack(err), "createTransaction failed")
		return
	}

	for attemptNum := uint(0); attemptNum < cth.execOpts.WaitCommitAttempts; attemptNum++ {
		if _, err := clientCtx.Transactor.SendTransaction(tx); err != nil {
			reqCtx.Error = errors.Wrap(errors.WithStack(err), "sendTransaction failed")
			return
		}

		var waitCommitAttemptTimeout <-chan time.Time
		if cth.execOpts.WaitCommitAttemptTimeout > 0 {
			waitCommitAttemptTimeout = time.After(cth.execOpts.WaitCommitAttemptTimeout)
		}

		select {
		case <-waitCommitAttemptTimeout:
			continue
		case txStatus := <-statusNotifier:
			reqCtx.Response.BlockNumber = txStatus.BlockNumber
			reqCtx.Response.TxValidationCode = txStatus.TxValidationCode

			if txStatus.TxValidationCode != pb.TxValidationCode_VALID {
				reqCtx.Error = errors.WithStack(
					status.New(status.EventServerStatus, int32(txStatus.TxValidationCode),
						"received invalid transaction", nil))
			}
			return
		case <-cth.ctx.Done():
			reqCtx.Error = errors.WithStack(
				status.New(status.ClientStatus, status.Unknown.ToInt32(),
					"Execute didn't receive block event (context done)", nil))
			return
		case <-reqCtx.Ctx.Done():
			reqCtx.Error = errors.WithStack(
				status.New(status.ClientStatus, status.Timeout.ToInt32(),
					"Execute didn't receive block event", nil))
			return
		}
	}
}
