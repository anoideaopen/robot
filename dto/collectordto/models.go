package collectordto

import "github.com/anoideaopen/foundation/proto"

type BlockData struct {
	BlockNum       uint64
	Txs            [][]byte
	Swaps          []*proto.Swap
	MultiSwaps     []*proto.MultiSwap
	SwapsKeys      []*proto.SwapKey
	MultiSwapsKeys []*proto.SwapKey
	Size           uint
}

func (d *BlockData) IsEmpty() bool {
	return len(d.Txs) == 0 &&
		len(d.Swaps) == 0 &&
		len(d.MultiSwaps) == 0 &&
		len(d.SwapsKeys) == 0 &&
		len(d.MultiSwapsKeys) == 0
}

func (d *BlockData) ItemsCount() uint {
	return uint(len(d.Txs) +
		len(d.Swaps) +
		len(d.MultiSwaps) +
		len(d.SwapsKeys) +
		len(d.MultiSwapsKeys))
}
