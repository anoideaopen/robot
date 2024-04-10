package executordto

import "github.com/anoideaopen/foundation/proto"

type Batch struct {
	Txs           [][]byte
	Swaps         []*proto.Swap
	MultiSwaps    []*proto.MultiSwap
	Keys          []*proto.SwapKey
	MultiKeys     []*proto.SwapKey
	PredictSize   uint
	TxIndToBlocks map[uint]uint64
}

func (b *Batch) IsEmpty() bool {
	return len(b.Txs) == 0 &&
		len(b.Swaps) == 0 &&
		len(b.MultiSwaps) == 0 &&
		len(b.Keys) == 0 &&
		len(b.MultiKeys) == 0
}
