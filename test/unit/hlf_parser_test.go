package unit

import (
	"testing"

	"github.com/anoideaopen/common-component/testshlp"
	"github.com/anoideaopen/robot/hlf/parser"
	cmn "github.com/anoideaopen/robot/test/unit/common"
	"github.com/stretchr/testify/require"
)

func TestParserCreate(t *testing.T) {
	_, log := testshlp.CreateCtxLogger(t)

	prs := parser.NewParser(log,
		"dstCh", "srcCh",
		cmn.DefaultPrefixes)
	require.NotNil(t, prs)
}

func TestParserExtractData(t *testing.T) {
	_, log := testshlp.CreateCtxLogger(t)

	t.Run("[NEGATIVE] config block", func(t *testing.T) {
		prs := parser.NewParser(log,
			"ch1", "ch1",
			cmn.DefaultPrefixes)
		require.NotNil(t, prs)

		bl := cmn.GetBlock(t, "../../hlf/parser/test-data/blocks/config_block.block")
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
		prs := parser.NewParser(log,
			"ch1", "ch1",
			cmn.DefaultPrefixes)
		require.NotNil(t, prs)

		bl := cmn.GetBlock(t, "../../hlf/parser/test-data/blocks/block_with_preimages.block")
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
		prs := parser.NewParser(log,
			"", "ch1",
			cmn.DefaultPrefixes)
		require.NotNil(t, prs)

		bl := cmn.GetBlock(t, "../../hlf/parser/test-data/blocks/block_with_keys.block")
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
		prs := parser.NewParser(log,
			"fiat", "cc",
			cmn.DefaultPrefixes)
		require.NotNil(t, prs)

		bl := cmn.GetBlock(t, "../../hlf/parser/test-data/blocks/block_with_swaps.block")
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
		prs := parser.NewParser(log,
			"bac", "ba",
			cmn.DefaultPrefixes)
		require.NotNil(t, prs)

		bl := cmn.GetBlock(t, "../../hlf/parser/test-data/blocks/block_with_multiswaps.block")
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
		prs := parser.NewParser(log,
			"ba", "bac",
			cmn.DefaultPrefixes)
		require.NotNil(t, prs)

		bl := cmn.GetBlock(t, "../../hlf/parser/test-data/blocks/block_with_multikeys.block")
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
