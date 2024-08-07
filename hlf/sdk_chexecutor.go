package hlf

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/anoideaopen/common-component/errorshlp"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/glog"
	"github.com/anoideaopen/robot/dto/executordto"
	"github.com/anoideaopen/robot/helpers/nerrors"
	"github.com/anoideaopen/robot/logger"
	"github.com/anoideaopen/robot/metrics"
	"github.com/avast/retry-go/v4"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
)

const (
	retryExecuteAttempts = 3
	retryExecuteMaxDelay = 2 * time.Second
	retryExecuteDelay    = 500 * time.Millisecond
)

type Executor interface {
	Execute(ctx context.Context, req channel.Request) (channel.Response, error)
}

type ChExecutor struct {
	// args
	Log     glog.Logger
	Metrics metrics.Metrics
	ChName  string

	// init
	closeSdkComps        func()
	Executor             Executor
	RetryExecuteAttempts uint
	RetryExecuteMaxDelay time.Duration
	RetryExecuteDelay    time.Duration
}

// CreateChExecutor creates New ChExecutor
func CreateChExecutor(
	ctx context.Context,
	chName string,
	connectionProfile string,
	userName string,
	orgName string,
	execOpts ExecuteOptions,
) (*ChExecutor, error) {
	log := glog.FromContext(ctx).
		With(logger.Labels{
			Component: logger.ComponentExecutor,
			ChName:    chName,
		}.Fields()...)

	m := metrics.FromContext(ctx)
	m = m.CreateChild(
		metrics.Labels().RobotChannel.Create(chName),
	)

	chExec := &ChExecutor{
		Log:                  log,
		Metrics:              m,
		ChName:               chName,
		RetryExecuteAttempts: retryExecuteAttempts,
		RetryExecuteMaxDelay: retryExecuteMaxDelay,
		RetryExecuteDelay:    retryExecuteDelay,
	}
	if err := chExec.init(ctx, connectionProfile, orgName, userName, execOpts); err != nil {
		chExec.Close()
		return nil, errorshlp.WrapWithDetails(err, nerrors.ErrTypeHlf, nerrors.ComponentExecutor)
	}
	return chExec, nil
}

func (che *ChExecutor) init(
	ctx context.Context,
	connectionProfile string,
	org string,
	user string,
	execOpts ExecuteOptions,
) error {
	configBackends, err := config.FromFile(connectionProfile)()
	if err != nil {
		return err
	}

	var sdkComps *sdkComponents
	sdkComps, err = createSdkComponents(ctx, che.ChName, org, user, configBackends)
	if err != nil {
		return err
	}

	che.closeSdkComps = func() {}

	chClient, err := channel.New(sdkComps.chProvider)
	if err != nil {
		return err
	}

	chCtx, err := sdkComps.chProvider()
	if err != nil {
		return err
	}

	che.Executor = &hlfExecutor{
		chClient: chClient,
		chCtx:    chCtx,
		execOpts: execOpts,
	}
	return nil
}

func (che *ChExecutor) Execute(ctx context.Context, b *executordto.Batch, _ uint64) (uint64, error) {
	execHlp := NewExecWithSplitHlp(che.Log, che.executeBatch,
		func(_ *executordto.Batch, num int) {
			che.Metrics.TotalOrderingReqSizeExceeded().Inc(
				metrics.Labels().IsFirstAttempt.Create(strconv.FormatBool(num > 0)),
			)
		})
	return execHlp.Execute(ctx, b)
}

func (che *ChExecutor) executeBatch(ctx context.Context, b *executordto.Batch) (uint64, error) {
	batch := &pb.Batch{
		TxIDs:          b.Txs,
		Swaps:          b.Swaps,
		Keys:           b.Keys,
		MultiSwapsKeys: b.MultiKeys,
		MultiSwaps:     b.MultiSwaps,
	}
	che.Metrics.BatchItemsCount().Observe(
		float64(
			len(batch.GetTxIDs()) +
				len(batch.GetSwaps()) +
				len(batch.GetMultiSwaps()) +
				len(batch.GetKeys()) +
				len(batch.GetMultiSwapsKeys())))

	LogBatchContent(che.Log, b)

	now := time.Now()
	resp, err := che.executeWithRetry(ctx, batch)

	che.Metrics.BatchExecuteInvokeTime().Observe(time.Since(now).Seconds())
	che.Metrics.TotalBatchExecuted().Inc(
		metrics.Labels().IsErr.Create(strconv.FormatBool(err != nil)))

	addTotalExecutedTx := func(count int, txType string) {
		che.Metrics.TotalExecutedTx().Add(
			float64(count),
			metrics.Labels().TxType.Create(txType))
	}

	if err != nil {
		return 0, errorshlp.WrapWithDetails(err,
			nerrors.ErrTypeHlf, nerrors.ComponentExecutor)
	}

	logBatchResponse(che.Log, b, resp)

	addTotalExecutedTx(len(batch.GetTxIDs()), metrics.TxTypeTx)
	addTotalExecutedTx(len(batch.GetKeys()), metrics.TxTypeSwapKey)
	addTotalExecutedTx(len(batch.GetMultiSwapsKeys()), metrics.TxTypeMultiSwapKey)
	addTotalExecutedTx(len(batch.GetSwaps()), metrics.TxTypeSwap)
	addTotalExecutedTx(len(batch.GetMultiSwaps()), metrics.TxTypeMultiSwap)

	che.Metrics.HeightLedgerBlocks().Set(float64(resp.BlockNumber + 1))
	return resp.BlockNumber, nil
}

func (che *ChExecutor) CalcBatchSize(b *executordto.Batch) (uint, error) {
	batch := &pb.Batch{
		TxIDs:          b.Txs,
		Swaps:          b.Swaps,
		Keys:           b.Keys,
		MultiSwapsKeys: b.MultiKeys,
		MultiSwaps:     b.MultiSwaps,
	}

	return uint(proto.Size(batch)), nil
}

func (che *ChExecutor) Close() {
	if che.closeSdkComps != nil {
		che.closeSdkComps()
	}
}

func (che *ChExecutor) executeWithRetry(ctx context.Context, batch *pb.Batch) (channel.Response, error) {
	var resp channel.Response

	batchBytes, err := proto.Marshal(batch)
	if err != nil {
		return resp, errorshlp.WrapWithDetails(
			err,
			nerrors.ErrTypeParsing, nerrors.ComponentExecutor)
	}

	che.Metrics.BatchSize().Observe(float64(len(batchBytes)))
	che.Metrics.TotalBatchSize().Add(float64(len(batchBytes)))

	err = retry.Do(func() error {
		r, err := che.Executor.Execute(ctx,
			channel.Request{
				ChaincodeID: che.ChName,
				Fcn:         "batchExecute",
				Args:        [][]byte{batchBytes},
			})
		if err != nil {
			che.Metrics.TotalBatchExecuted().Inc(
				metrics.Labels().IsErr.Create("true"))

			if IsEndorsementMismatchErr(err) {
				che.Log.Warningf("endorsement mismatch, err: %s", err)
			}
			return err
		}

		resp = r
		return nil
	},
		retry.LastErrorOnly(true),
		retry.Attempts(che.RetryExecuteAttempts),
		retry.Delay(che.RetryExecuteDelay),
		retry.MaxDelay(che.RetryExecuteMaxDelay),
		retry.RetryIf(isExecuteErrorRecoverable),
		retry.Context(ctx),
		retry.OnRetry(func(n uint, err error) {
			che.Log.Warningf("retrying execute, attempt: %d, err: %s, batch: %s", n, err, batch)
		}),
	)
	if err != nil {
		return resp, errorshlp.WrapWithDetails(err,
			nerrors.ErrTypeHlf, nerrors.ComponentExecutor)
	}

	return resp, nil
}

func isExecuteErrorRecoverable(e error) bool {
	return IsEndorsementMismatchErr(e)
}

func LogBatchContent(log glog.Logger, b *executordto.Batch) {
	sb := strings.Builder{}
	_, _ = sb.WriteString("batch content:\n")

	_, _ = sb.WriteString(fmt.Sprintf("txs (%v):\n", len(b.Txs)))
	for _, tx := range b.Txs {
		_, _ = sb.WriteString(hex.EncodeToString(tx))
		_, _ = sb.WriteString("\n")
	}

	_, _ = sb.WriteString(fmt.Sprintf("swaps (%v):\n", len(b.Swaps)))
	for _, swap := range b.Swaps {
		_, _ = sb.WriteString(hex.EncodeToString(swap.GetId()))
		_, _ = sb.WriteString("\n")
	}

	_, _ = sb.WriteString(fmt.Sprintf("mswaps (%v):\n", len(b.MultiSwaps)))
	for _, mswap := range b.MultiSwaps {
		_, _ = sb.WriteString(hex.EncodeToString(mswap.GetId()))
		_, _ = sb.WriteString("\n")
	}

	_, _ = sb.WriteString(fmt.Sprintf("swaps-keys (%v):\n", len(b.Keys)))
	for _, k := range b.Keys {
		_, _ = sb.WriteString(hex.EncodeToString(k.GetId()))
		_, _ = sb.WriteString("\n")
	}

	_, _ = sb.WriteString(fmt.Sprintf("mswaps-keys (%v):\n", len(b.MultiKeys)))
	for _, k := range b.MultiKeys {
		_, _ = sb.WriteString(hex.EncodeToString(k.GetId()))
		_, _ = sb.WriteString("\n")
	}

	log.Debug(sb.String())
}

func logBatchResponse(log glog.Logger, b *executordto.Batch, resp channel.Response) {
	log.Infof("batch was executed in hlf, hlfTxID: %s, bn: %v, txs: %v, swaps: %v, mwaps: %v, keys: %v, mkeys: %v",
		resp.TransactionID, resp.BlockNumber,
		len(b.Txs), len(b.Swaps), len(b.MultiSwaps), len(b.Keys), len(b.MultiKeys),
	)

	const (
		txID, keyID, method, keyErrCode = "txID", "keyID", "method", "errCode"
	)

	var batchResp pb.BatchResponse
	if err := proto.Unmarshal(resp.Payload, &batchResp); err != nil {
		log.Errorf("failed to unmarshal batch response, err: %s", err)
		return
	}
	// preimage errs
	for _, txresponse := range batchResp.GetTxResponses() {
		if txresponse.GetError() != nil {
			log.Debugf("tx was executed with error, txID: %s, method: %s, writes: %v, err: %s",
				hex.EncodeToString(txresponse.GetId()), txresponse.GetMethod(), txresponse.GetWrites(), txresponse.GetError().GetError())

			log.With(
				glog.Field{K: txID, V: hex.EncodeToString(txresponse.GetId())},
				glog.Field{K: method, V: txresponse.GetMethod()},
				glog.Field{K: keyErrCode, V: txresponse.GetError().GetCode()},
			).Warning(txresponse.GetError().GetError())
		} else {
			log.Debugf("tx was executed successfully, txID: %s, method: %s, writes: %v",
				hex.EncodeToString(txresponse.GetId()), txresponse.GetMethod(), txresponse.GetWrites())
		}
	}

	// swap key errs
	for _, swapkeyresp := range batchResp.GetSwapKeyResponses() {
		if swapkeyresp.GetError() != nil {
			log.With(
				glog.Field{K: keyID, V: hex.EncodeToString(swapkeyresp.GetId())},
				glog.Field{K: keyErrCode, V: swapkeyresp.GetError().GetCode()},
			).Error(swapkeyresp.GetError().GetError())
		}
	}

	// swaps errs
	for _, swapresp := range batchResp.GetSwapResponses() {
		if swapresp.GetError() != nil {
			log.With(
				glog.Field{K: keyID, V: hex.EncodeToString(swapresp.GetId())},
				glog.Field{K: keyErrCode, V: swapresp.GetError().GetCode()},
			).Error(swapresp.GetError().GetError())
		}
	}
}
