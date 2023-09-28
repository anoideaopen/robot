package collectordto

import "github.com/atomyze-foundation/foundation/proto"

// BlockData is a struct with data from block
type BlockData struct {
	// BlockNum is a number of block
	BlockNum uint64
	// Txs is a slice with transactions
	Txs [][]byte
	// Swaps is a slice with swap transactions
	Swaps []*proto.Swap
	// MultiSwaps is a slice with multi swap transactions
	MultiSwaps []*proto.MultiSwap
	// SwapsKeys is a slice with swap key transactions
	SwapsKeys []*proto.SwapKey
	// MultiSwapsKeys is a slice with multi swap key transactions
	MultiSwapsKeys []*proto.SwapKey
	// Size is a size of block
	Size uint
}

// IsEmpty returns true if block data is empty
func (d *BlockData) IsEmpty() bool {
	return len(d.Txs) == 0 &&
		len(d.Swaps) == 0 &&
		len(d.MultiSwaps) == 0 &&
		len(d.SwapsKeys) == 0 &&
		len(d.MultiSwapsKeys) == 0
}

// ItemsCount returns count of items in block data
func (d *BlockData) ItemsCount() uint {
	return uint(len(d.Txs) +
		len(d.Swaps) +
		len(d.MultiSwaps) +
		len(d.SwapsKeys) +
		len(d.MultiSwapsKeys))
}
