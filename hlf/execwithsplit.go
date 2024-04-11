package hlf

import (
	"context"
	"math"

	"github.com/anoideaopen/glog"
	"github.com/anoideaopen/robot/dto/executordto"
)

type execWithSplitHlp struct {
	log                        glog.Logger
	executeBatch               func(ctx context.Context, b *executordto.Batch) (uint64, error)
	reqSizeExceededErrCallBack func(b *executordto.Batch, num int)
	reqSizeExceededErrNum      int
}

func newExecWithSplitHlp(
	log glog.Logger,
	executeBatch func(ctx context.Context, b *executordto.Batch) (uint64, error),
	reqSizeExceededErrCallBack func(b *executordto.Batch, num int),
) *execWithSplitHlp {
	return &execWithSplitHlp{
		log:                        log,
		executeBatch:               executeBatch,
		reqSizeExceededErrCallBack: reqSizeExceededErrCallBack,
	}
}

func (hlp *execWithSplitHlp) execute(ctx context.Context, b *executordto.Batch) (uint64, error) {
	blockNum, err := hlp.executeBatch(ctx, b)
	hlp.raiseReqSizeExceededIfErr(b, err)

	if len(b.Txs) < 2 || !isOrderingReqSizeExceededErr(err) {
		return blockNum, err
	}

	hlp.log.Warning("executeBatch: request size exceeded and txs more than 1, will split batch")

	executedTxsCount, lastSuccessCount, lastErrorCount := 0, 0, 0
	for executedTxsCount < len(b.Txs) {
		tmpBatch := splitBatchForExec(executedTxsCount, lastSuccessCount, lastErrorCount, b, hlp.log)
		hlp.log.Warningf("try to exec part of original batch, txs: %v executed: %v from %v", len(tmpBatch.Txs), executedTxsCount, len(b.Txs))
		blockNum, err = hlp.executeBatch(ctx, tmpBatch)
		hlp.raiseReqSizeExceededIfErr(tmpBatch, err)
		if err == nil {
			lastSuccessCount = int(math.Max(float64(len(tmpBatch.Txs)), float64(lastSuccessCount)))
			executedTxsCount += len(tmpBatch.Txs)
			continue
		}

		if len(tmpBatch.Txs) < 2 || !isOrderingReqSizeExceededErr(err) {
			return 0, err
		}
		hlp.log.Errorf("exec part of original batch request size exceeded error: %s", err)
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

	hlp.log.Info("try to exec other parts of original batch (swaps, multiSwaps, keys, multiKeys)")
	blockNum, err = hlp.executeBatch(ctx, restBatch)
	hlp.raiseReqSizeExceededIfErr(restBatch, err)
	return blockNum, err
}

func (hlp *execWithSplitHlp) raiseReqSizeExceededIfErr(b *executordto.Batch, err error) {
	if !isOrderingReqSizeExceededErr(err) {
		return
	}

	defer func() { hlp.reqSizeExceededErrNum++ }()

	if hlp.reqSizeExceededErrCallBack == nil {
		return
	}
	hlp.reqSizeExceededErrCallBack(b, hlp.reqSizeExceededErrNum)
}

// splitBatchForExec splits transactions, based on results of previous  executions of a batch
// executedTxsCount - amount of transactions that were successfully executed and must be skipped
// lastSuccessCount - amount of transactions that were successfully executed previously, next batch must be the same or less
// lastErrorCount - amount of transactions that were not successfully executed previously, next batch must be less
func splitBatchForExec(executedTxsCount, lastSuccessCount, lastErrorCount int, origBatch *executordto.Batch, log glog.Logger) *executordto.Batch {
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
