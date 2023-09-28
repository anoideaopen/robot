package executordto

import "github.com/atomyze-foundation/foundation/proto"

// Batch is a struct with data for batch
type Batch struct {
	// Txs is a slice with transactions
	Txs [][]byte
	// Swaps is a slice with swap transactions
	Swaps []*proto.Swap
	// MultiSwaps is a slice with multi swap transactions
	MultiSwaps []*proto.MultiSwap
	// Keys is a slice with swap key transactions
	Keys []*proto.SwapKey
	// MultiKeys is a slice with multi swap key transactions
	MultiKeys []*proto.SwapKey
	// Size is a size of batch
	PredictSize uint
	// TxIndToBlocks is a map with indexes of transactions to blocks
	TxIndToBlocks map[uint]uint64
}

// IsEmpty returns true if batch is empty
func (b *Batch) IsEmpty() bool {
	return len(b.Txs) == 0 &&
		len(b.Swaps) == 0 &&
		len(b.MultiSwaps) == 0 &&
		len(b.Keys) == 0 &&
		len(b.MultiKeys) == 0
}
