//go:build !integration
// +build !integration

package chrobot

import (
	"context"
	"encoding/binary"
	"sync"

	"github.com/atomyze-foundation/common-component/testshlp"
	fp "github.com/atomyze-foundation/foundation/proto"
	"github.com/atomyze-foundation/robot/dto/executordto"
	"google.golang.org/protobuf/proto"
)

type stubChExecutor struct {
	callHlp testshlp.CallHlp

	chName        string
	lock          sync.RWMutex
	blockNum      uint64
	maxTxBlockNum uint64
	savedBatches  []*executordto.Batch
	allTxs        map[uint32][]byte
	allSwaps      map[*fp.Swap]struct{}
}

func newStubChExecutorCreator(chName string) *stubChExecutor {
	return &stubChExecutor{
		chName:   chName,
		allTxs:   make(map[uint32][]byte),
		allSwaps: make(map[*fp.Swap]struct{}),
	}
}

func (che *stubChExecutor) create(_ context.Context) (*stubChExecutor, error) {
	if err := che.callHlp.Call(che.create); err != nil {
		return nil, err
	}

	return che, nil
}

func (che *stubChExecutor) Close() {
	// nothing just stub
}

func (che *stubChExecutor) CalcBatchSize(b *executordto.Batch) (uint, error) {
	if err := che.callHlp.Call(che.CalcBatchSize); err != nil {
		return 0, err
	}

	batch := &fp.Batch{
		TxIDs:          b.Txs,
		Swaps:          b.Swaps,
		Keys:           b.Keys,
		MultiSwapsKeys: b.MultiKeys,
		MultiSwaps:     b.MultiSwaps,
	}

	data, err := proto.Marshal(batch)
	if err != nil {
		return 0, err
	}
	return uint(len(data)), nil
}

func (che *stubChExecutor) Execute(_ context.Context, b *executordto.Batch, h uint64) (uint64, error) {
	if err := che.callHlp.Call(che.Execute); err != nil {
		return 0, err
	}

	che.lock.Lock()
	defer che.lock.Unlock()

	che.blockNum++
	if che.blockNum < h {
		che.blockNum = h - 1
	}

	lastTxBlockNum := b.TxIndToBlocks[uint(len(b.Txs))-1]
	if che.blockNum < lastTxBlockNum {
		che.blockNum = lastTxBlockNum
	}
	che.blockNum++
	che.savedBatches = append(che.savedBatches, b)
	for _, tx := range b.Txs {
		che.allTxs[binary.LittleEndian.Uint32(tx)] = tx
	}
	for _, sw := range b.Swaps {
		che.allSwaps[sw] = struct{}{}
	}

	if che.maxTxBlockNum < lastTxBlockNum {
		che.maxTxBlockNum = lastTxBlockNum
	}
	return che.blockNum, nil
}

func (che *stubChExecutor) checkSavedState(check func(
	maxTxBlockNum uint64,
	batches []*executordto.Batch,
	allTxs map[uint32][]byte,
	allSwaps map[*fp.Swap]struct{}),
) {
	che.lock.RLock()
	defer che.lock.RUnlock()

	check(che.maxTxBlockNum, che.savedBatches, che.allTxs, che.allSwaps)
}
