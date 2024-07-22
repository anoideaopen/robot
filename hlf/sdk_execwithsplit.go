package hlf

import (
	"context"
	"math"

	"github.com/anoideaopen/glog"
	"github.com/anoideaopen/robot/dto/executordto"
)

// ExecWithSplitHlp - struct for execution with split
type ExecWithSplitHlp struct {
	Log                        glog.Logger
	ExecuteBatch               func(ctx context.Context, b *executordto.Batch) (uint64, error)
	ReqSizeExceededErrCallBack func(b *executordto.Batch, num int)
	ReqSizeExceededErrNum      int
}

func NewExecWithSplitHlp(
	log glog.Logger,
	executeBatch func(ctx context.Context, b *executordto.Batch) (uint64, error),
	reqSizeExceededErrCallBack func(b *executordto.Batch, num int),
) *ExecWithSplitHlp {
	return &ExecWithSplitHlp{
		Log:                        log,
		ExecuteBatch:               executeBatch,
		ReqSizeExceededErrCallBack: reqSizeExceededErrCallBack,
	}
}

func (hlp *ExecWithSplitHlp) Execute(ctx context.Context, b *executordto.Batch) (uint64, error) {
	blockNum, err := hlp.ExecuteBatch(ctx, b)
	hlp.RaiseReqSizeExceededIfErr(b, err)

	if len(b.Txs) < 2 || !IsOrderingReqSizeExceededErr(err) {
		return blockNum, err
	}

	hlp.Log.Warning("executeBatch: request size exceeded and txs more than 1, will split batch")

	executedTxsCount, lastSuccessCount, lastErrorCount := 0, 0, 0
	for executedTxsCount < len(b.Txs) {
		tmpBatch := SplitBatchForExec(executedTxsCount, lastSuccessCount, lastErrorCount, b, hlp.Log)
		hlp.Log.Warningf("try to exec part of original batch, txs: %v executed: %v from %v", len(tmpBatch.Txs), executedTxsCount, len(b.Txs))
		blockNum, err = hlp.ExecuteBatch(ctx, tmpBatch)
		hlp.RaiseReqSizeExceededIfErr(tmpBatch, err)
		if err == nil {
			lastSuccessCount = int(math.Max(float64(len(tmpBatch.Txs)), float64(lastSuccessCount)))
			executedTxsCount += len(tmpBatch.Txs)
			continue
		}

		if len(tmpBatch.Txs) < 2 || !IsOrderingReqSizeExceededErr(err) {
			return 0, err
		}
		hlp.Log.Errorf("exec part of original batch request size exceeded error: %s", err)
		lastErrorCount = len(tmpBatch.Txs)
		lastSuccessCount = 0
	}

	// batch for other parts of original batch
	restBatch := &executordto.Batch{
		Txs:        nil,
		Swaps:      b.Swaps,
		MultiSwaps: b.MultiSwaps,
		Keys:       b.Keys,
		MultiKeys:  b.MultiKeys,
	}
	if restBatch.IsEmpty() {
		return blockNum, nil
	}

	hlp.Log.Info("try to exec other parts of original batch (swaps, multiSwaps, keys, multiKeys)")
	blockNum, err = hlp.ExecuteBatch(ctx, restBatch)
	hlp.RaiseReqSizeExceededIfErr(restBatch, err)
	return blockNum, err
}

func (hlp *ExecWithSplitHlp) RaiseReqSizeExceededIfErr(b *executordto.Batch, err error) {
	if !IsOrderingReqSizeExceededErr(err) {
		return
	}

	defer func() { hlp.ReqSizeExceededErrNum++ }()

	if hlp.ReqSizeExceededErrCallBack == nil {
		return
	}
	hlp.ReqSizeExceededErrCallBack(b, hlp.ReqSizeExceededErrNum)
}

// SplitBatchForExec splits transactions, based on results of previous  executions of a batch
// executedTxsCount - amount of transactions that were successfully executed and must be skipped
// lastSuccessCount - amount of transactions that were successfully executed previously, next batch must be the same or less
// lastErrorCount - amount of transactions that were not successfully executed previously, next batch must be less
func SplitBatchForExec(executedTxsCount, lastSuccessCount, lastErrorCount int, origBatch *executordto.Batch, log glog.Logger) *executordto.Batch {
	const lessK = 2

	txCount := len(origBatch.Txs) - executedTxsCount
	splitCount := lastSuccessCount
	if txCount <= lastSuccessCount || txCount == 1 {
		splitCount = txCount
	} else if lastSuccessCount == 0 {
		splitCount = txCount / lessK
	}

	if lastErrorCount > 0 && splitCount >= lastErrorCount {
		splitCount = lastErrorCount / lessK
	}

	log.Debugf("splitBatchForExec: executedTxsCount=%d, lastSuccessCount=%d, lastErrorCount=%d, txCount=%d, splitCount=%d", executedTxsCount, lastSuccessCount, lastErrorCount, txCount, splitCount)

	return &executordto.Batch{
		Txs:           origBatch.Txs[executedTxsCount : executedTxsCount+splitCount],
		TxIndToBlocks: origBatch.TxIndToBlocks,
	}
}
