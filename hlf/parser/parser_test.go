//go:build !integration
// +build !integration

package parser

import (
	"os"
	"testing"

	"github.com/anoideaopen/common-component/testshlp"
	"github.com/anoideaopen/robot/dto/parserdto"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/stretchr/testify/require"
)

var defaultPrefixes = parserdto.TxPrefixes{
	Tx:        "batchTransactions",
	Swap:      "swaps",
	MultiSwap: "multi_swap",
}

func TestCreate(t *testing.T) {
	_, log := testshlp.CreateCtxLogger(t)

	prs := NewParser(log,
		"dstCh", "srcCh",
		defaultPrefixes)
	require.NotNil(t, prs)
}

func TestParserExtractData(t *testing.T) {
	_, log := testshlp.CreateCtxLogger(t)

	t.Run("[NEGATIVE] config block", func(t *testing.T) {
		prs := NewParser(log,
			"ch1", "ch1",
			defaultPrefixes)
		require.NotNil(t, prs)

		bl := getBlock(t, "test-data/blocks/config_block.block")
		d, err := prs.ExtractData(bl)
		require.NoError(t, err)
		require.NotNil(t, d)
		require.Empty(t, d.Txs)
		require.Empty(t, d.Swaps)
		require.Empty(t, d.MultiSwaps)
		require.Empty(t, d.SwapsKeys)
		require.Empty(t, d.MultiSwapsKeys)
	})

	t.Run("extract txs", func(t *testing.T) {
		prs := NewParser(log,
			"ch1", "ch1",
			defaultPrefixes)
		require.NotNil(t, prs)

		bl := getBlock(t, "test-data/blocks/block_with_preimages.block")
		d, err := prs.ExtractData(bl)
		require.NoError(t, err)
		require.NotNil(t, d)
		require.NotEmpty(t, d.Txs)
		require.Empty(t, d.Swaps)
		require.Empty(t, d.MultiSwaps)
		require.Empty(t, d.SwapsKeys)
		require.Empty(t, d.MultiSwapsKeys)
	})

	t.Run("extract swap keys", func(t *testing.T) {
		prs := NewParser(log,
			"", "ch1",
			defaultPrefixes)
		require.NotNil(t, prs)

		bl := getBlock(t, "test-data/blocks/block_with_keys.block")
		d, err := prs.ExtractData(bl)
		require.NoError(t, err)
		require.NotNil(t, d)
		require.Empty(t, d.Txs)
		require.Empty(t, d.Swaps)
		require.Empty(t, d.MultiSwaps)
		require.NotEmpty(t, d.SwapsKeys)
		require.Empty(t, d.MultiSwapsKeys)
	})

	t.Run("extract swap", func(t *testing.T) {
		prs := NewParser(log,
			"fiat", "cc",
			defaultPrefixes)
		require.NotNil(t, prs)

		bl := getBlock(t, "test-data/blocks/block_with_swaps.block")
		d, err := prs.ExtractData(bl)
		require.NoError(t, err)
		require.NotNil(t, d)
		require.Empty(t, d.Txs)
		require.NotEmpty(t, d.Swaps)
		require.Empty(t, d.MultiSwaps)
		require.Empty(t, d.SwapsKeys)
		require.Empty(t, d.MultiSwapsKeys)
	})

	t.Run("extract multiswap", func(t *testing.T) {
		prs := NewParser(log,
			"bac", "ba",
			defaultPrefixes)
		require.NotNil(t, prs)

		bl := getBlock(t, "test-data/blocks/block_with_multiswaps.block")
		d, err := prs.ExtractData(bl)
		require.NoError(t, err)
		require.NotNil(t, d)
		require.Empty(t, d.Txs)
		require.Empty(t, d.Swaps)
		require.NotEmpty(t, d.MultiSwaps)
		require.Empty(t, d.SwapsKeys)
		require.Empty(t, d.MultiSwapsKeys)
	})

	t.Run("extract multiswap keys", func(t *testing.T) {
		prs := NewParser(log,
			"ba", "bac",
			defaultPrefixes)
		require.NotNil(t, prs)

		bl := getBlock(t, "test-data/blocks/block_with_multikeys.block")
		d, err := prs.ExtractData(bl)
		require.NoError(t, err)
		require.NotNil(t, d)
		require.Empty(t, d.Txs)
		require.Empty(t, d.Swaps)
		require.Empty(t, d.MultiSwaps)
		require.Empty(t, d.SwapsKeys)
		require.NotEmpty(t, d.MultiSwapsKeys)
	})
}

func getBlock(t *testing.T, pathToBlock string) *common.Block {
	file, err := os.ReadFile(pathToBlock)
	require.NoError(t, err)

	fabBlock := &common.Block{}
	err = proto.Unmarshal(file, fabBlock)
	require.NoError(t, err)

	return fabBlock
}
