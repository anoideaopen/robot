package hlf

import (
	"context"
	"fmt"
	"time"

	pb "github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/common/filter"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	chctx "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
)

type ExecuteOptions struct {
	ExecuteTimeout time.Duration
}

type hlfExecutor struct {
	chClient *channel.Client
	chCtx    chctx.Channel
	execOpts ExecuteOptions
}

func (he *hlfExecutor) Execute(ctx context.Context, request channel.Request) (channel.Response, error) {
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
		reqCtx.Error = fmt.Errorf("error registering for TxStatus event: %w", err)
		return
	}
	defer clientCtx.EventService.Unregister(reg)

	tx, err := clientCtx.Transactor.CreateTransaction(
		fab.TransactionRequest{
			Proposal:          reqCtx.Response.Proposal,
			ProposalResponses: reqCtx.Response.Responses,
		})
	if err != nil {
		reqCtx.Error = fmt.Errorf("createTransaction failed: %w", err)
		return
	}

	if _, err = clientCtx.Transactor.SendTransaction(tx); err != nil {
		reqCtx.Error = fmt.Errorf("sendTransaction failed: %w", err)
		return
	}

	select {
	case txStatus := <-statusNotifier:
		reqCtx.Response.BlockNumber = txStatus.BlockNumber
		reqCtx.Response.TxValidationCode = txStatus.TxValidationCode

		if txStatus.TxValidationCode != pb.TxValidationCode_VALID {
			reqCtx.Error = status.New(status.EventServerStatus, int32(txStatus.TxValidationCode),
				"received invalid transaction", nil)
		}
		return
	case <-cth.ctx.Done():
		reqCtx.Error = status.New(status.ClientStatus, status.Unknown.ToInt32(),
			"Execute didn't receive block event (context done)", nil)
		return
	case <-reqCtx.Ctx.Done():
		reqCtx.Error = status.New(status.ClientStatus, status.Timeout.ToInt32(),
			"Execute didn't receive block event", nil)
		return
	}
}
