package hlf

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/anoideaopen/cartridge/manager"
	"github.com/anoideaopen/common-component/errorshlp"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/robot/dto/executordto"
	"github.com/anoideaopen/robot/helpers/nerrors"
	"github.com/anoideaopen/robot/logger"
	"github.com/anoideaopen/robot/metrics"
	"github.com/avast/retry-go/v4"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/newity/glog"
	"github.com/pkg/errors"
)

const (
	retryExecuteAttempts = 3
	retryExecuteMaxDelay = 2 * time.Second
	retryExecuteDelay    = 500 * time.Millisecond
)

type executor interface {
	execute(ctx context.Context, req channel.Request) (channel.Response, error)
}

type chExecutor struct {
	// args
	log    glog.Logger
	m      metrics.Metrics
	chName string

	// init
	closeSdkComps        func()
	executor             executor
	retryExecuteAttempts uint
	retryExecuteMaxDelay time.Duration
	retryExecuteDelay    time.Duration
}

func createChExecutor(
	ctx context.Context,
	chName,
	connectionProfile string,
	userName, orgName string,
	execOpts ExecuteOptions,
	cryptoManager manager.Manager,
) (*chExecutor, error) {
	log := glog.FromContext(ctx).
		With(logger.Labels{
			Component: logger.ComponentExecutor,
			ChName:    chName,
		}.Fields()...)

	m := metrics.FromContext(ctx)
	m = m.CreateChild(
		metrics.Labels().RobotChannel.Create(chName),
	)

	chExec := &chExecutor{
		log:                  log,
		m:                    m,
		chName:               chName,
		retryExecuteAttempts: retryExecuteAttempts,
		retryExecuteMaxDelay: retryExecuteMaxDelay,
		retryExecuteDelay:    retryExecuteDelay,
	}
	if err := chExec.init(ctx,
		connectionProfile, orgName, userName,
		execOpts,
		cryptoManager); err != nil {
		chExec.Close()
		return nil, errorshlp.WrapWithDetails(err, nerrors.ErrTypeHlf, nerrors.ComponentExecutor)
	}
	return chExec, nil
}

func (che *chExecutor) init(ctx context.Context,
	connectionProfile, org, user string,
	execOpts ExecuteOptions,
	cryptoManager manager.Manager,
) error {
	configBackends, err := config.FromFile(connectionProfile)()
	if err != nil {
		return errors.WithStack(err)
	}

	var sdkComps *sdkComponents
	if cryptoManager != nil {
		sdkComps, err = createSdkComponentsWithCryptoMng(ctx, che.chName, org, configBackends,
			cryptoManager)
	} else {
		sdkComps, err = createSdkComponentsWithoutCryptoMng(ctx, che.chName, org, user,
			configBackends)
	}
	if err != nil {
		return err
	}

	che.closeSdkComps = func() {}

	chClient, err := channel.New(sdkComps.chProvider)
	if err != nil {
		return errors.WithStack(err)
	}

	chCtx, err := sdkComps.chProvider()
	if err != nil {
		return errors.WithStack(err)
	}

	che.executor = &hlfExecutor{
		chClient: chClient,
		chCtx:    chCtx,
		execOpts: execOpts,
	}
	return nil
}

func (che *chExecutor) Execute(ctx context.Context, b *executordto.Batch, _ uint64) (uint64, error) {
	execHlp := newExecWithSplitHlp(che.log, che.executeBatch,
		func(_ *executordto.Batch, num int) {
			che.m.TotalOrderingReqSizeExceeded().Inc(
				metrics.Labels().IsFirstAttempt.Create(fmt.Sprintf("%v", num > 0)),
			)
		})
	return execHlp.execute(ctx, b)
}

func (che *chExecutor) executeBatch(ctx context.Context, b *executordto.Batch) (uint64, error) {
	batch := &pb.Batch{
		TxIDs:          b.Txs,
		Swaps:          b.Swaps,
		Keys:           b.Keys,
		MultiSwapsKeys: b.MultiKeys,
		MultiSwaps:     b.MultiSwaps,
	}
	che.m.BatchItemsCount().Observe(
		float64(
			len(batch.TxIDs) +
				len(batch.Swaps) +
				len(batch.MultiSwaps) +
				len(batch.Keys) +
				len(batch.MultiSwapsKeys)))

	logBatchContent(che.log, b)

	now := time.Now()
	resp, err := che.executeWithRetry(ctx, batch)

	che.m.BatchExecuteInvokeTime().Observe(time.Since(now).Seconds())
	che.m.TotalBatchExecuted().Inc(
		metrics.Labels().IsErr.Create(fmt.Sprintf("%v", err != nil)))

	addTotalExecutedTx := func(count int, txType string) {
		che.m.TotalExecutedTx().Add(
			float64(count),
			metrics.Labels().TxType.Create(txType))
	}

	if err != nil {
		return 0, errorshlp.WrapWithDetails(err,
			nerrors.ErrTypeHlf, nerrors.ComponentExecutor)
	}

	logBatchResponse(che.log, b, resp)

	addTotalExecutedTx(len(batch.TxIDs), metrics.TxTypeTx)
	addTotalExecutedTx(len(batch.Keys), metrics.TxTypeSwapKey)
	addTotalExecutedTx(len(batch.MultiSwapsKeys), metrics.TxTypeMultiSwapKey)
	addTotalExecutedTx(len(batch.Swaps), metrics.TxTypeSwap)
	addTotalExecutedTx(len(batch.MultiSwaps), metrics.TxTypeMultiSwap)

	che.m.HeightLedgerBlocks().Set(float64(resp.BlockNumber + 1))
	return resp.BlockNumber, nil
}

func (che *chExecutor) CalcBatchSize(b *executordto.Batch) (uint, error) {
	batch := &pb.Batch{
		TxIDs:          b.Txs,
		Swaps:          b.Swaps,
		Keys:           b.Keys,
		MultiSwapsKeys: b.MultiKeys,
		MultiSwaps:     b.MultiSwaps,
	}

	return uint(proto.Size(batch)), nil
}

func (che *chExecutor) Close() {
	if che.closeSdkComps != nil {
		che.closeSdkComps()
	}
}

func (che *chExecutor) executeWithRetry(ctx context.Context, batch *pb.Batch) (channel.Response, error) {
	var resp channel.Response

	batchBytes, err := proto.Marshal(batch)
	if err != nil {
		return resp, errorshlp.WrapWithDetails(
			errors.WithStack(err),
			nerrors.ErrTypeParsing, nerrors.ComponentExecutor)
	}

	che.m.BatchSize().Observe(float64(len(batchBytes)))
	che.m.TotalBatchSize().Add(float64(len(batchBytes)))

	err = retry.Do(func() error {
		r, err := che.executor.execute(ctx,
			channel.Request{
				ChaincodeID: che.chName,
				Fcn:         "batchExecute",
				Args:        [][]byte{batchBytes},
			})
		if err != nil {
			che.m.TotalBatchExecuted().Inc(
				metrics.Labels().IsErr.Create(fmt.Sprintf("%v", true)))

			if IsEndorsementMismatchErr(err) {
				che.log.Warningf("endorsement mismatch, err: %s", err)
			}
			return err
		}

		resp = r
		return nil
	},
		retry.LastErrorOnly(true),
		retry.Attempts(che.retryExecuteAttempts),
		retry.Delay(che.retryExecuteDelay),
		retry.MaxDelay(che.retryExecuteMaxDelay),
		retry.RetryIf(isExecuteErrorRecoverable),
		retry.Context(ctx),
		retry.OnRetry(func(n uint, err error) {
			che.log.Warningf("retrying execute, attempt: %d, err: %s, batch: %s", n, err, batch)
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

func logBatchContent(log glog.Logger, b *executordto.Batch) {
	sb := strings.Builder{}
	_, _ = sb.WriteString("batch content:\n")

	_, _ = sb.WriteString(fmt.Sprintf("txs (%v):\n", len(b.Txs)))
	for _, tx := range b.Txs {
		_, _ = sb.WriteString(hex.EncodeToString(tx))
		_, _ = sb.WriteString("\n")
	}

	_, _ = sb.WriteString(fmt.Sprintf("swaps (%v):\n", len(b.Swaps)))
	for _, swap := range b.Swaps {
		_, _ = sb.WriteString(hex.EncodeToString(swap.Id))
		_, _ = sb.WriteString("\n")
	}

	_, _ = sb.WriteString(fmt.Sprintf("mswaps (%v):\n", len(b.MultiSwaps)))
	for _, mswap := range b.MultiSwaps {
		_, _ = sb.WriteString(hex.EncodeToString(mswap.Id))
		_, _ = sb.WriteString("\n")
	}

	_, _ = sb.WriteString(fmt.Sprintf("swaps-keys (%v):\n", len(b.Keys)))
	for _, k := range b.Keys {
		_, _ = sb.WriteString(hex.EncodeToString(k.Id))
		_, _ = sb.WriteString("\n")
	}

	_, _ = sb.WriteString(fmt.Sprintf("mswaps-keys (%v):\n", len(b.MultiKeys)))
	for _, k := range b.MultiKeys {
		_, _ = sb.WriteString(hex.EncodeToString(k.Id))
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
	for _, txresponse := range batchResp.TxResponses {
		if txresponse.Error != nil {
			log.Debugf("tx was executed with error, txID: %s, method: %s, writes: %v, err: %s",
				hex.EncodeToString(txresponse.Id), txresponse.Method, txresponse.Writes, txresponse.Error.Error)

			log.With(
				glog.Field{K: txID, V: hex.EncodeToString(txresponse.Id)},
				glog.Field{K: method, V: txresponse.Method},
				glog.Field{K: keyErrCode, V: txresponse.Error.Code},
			).Warning(txresponse.Error.Error)
		} else {
			log.Debugf("tx was executed successfully, txID: %s, method: %s, writes: %v",
				hex.EncodeToString(txresponse.Id), txresponse.Method, txresponse.Writes)
		}
	}

	// swap key errs
	for _, swapkeyresp := range batchResp.SwapKeyResponses {
		if swapkeyresp.Error != nil {
			log.With(
				glog.Field{K: keyID, V: hex.EncodeToString(swapkeyresp.Id)},
				glog.Field{K: keyErrCode, V: swapkeyresp.Error.Code},
			).Error(swapkeyresp.Error.Error)
		}
	}

	// swaps errs
	for _, swapresp := range batchResp.SwapResponses {
		if swapresp.Error != nil {
			log.With(
				glog.Field{K: keyID, V: hex.EncodeToString(swapresp.Id)},
				glog.Field{K: keyErrCode, V: swapresp.Error.Code},
			).Error(swapresp.Error.Error)
		}
	}
}
