package unit

import (
	"testing"
	"time"

	"github.com/anoideaopen/common-component/testshlp"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/robot/collectorbatch"
	"github.com/anoideaopen/robot/dto/collectordto"
	"github.com/anoideaopen/robot/dto/executordto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestEmpty(t *testing.T) {
	ctx, _ := testshlp.CreateCtxLogger(t)
	b := collectorbatch.NewBatch(ctx, "ch1",
		collectorbatch.Limits{}, calcBatchSize)
	batch, bi := b.GetBatchForExec()
	require.Empty(t, bi.Sources)
	require.EqualValues(t, &collectorbatch.BatchInfo{
		Kind:        collectorbatch.NoneLimitKind,
		BlocksCount: 0,
		Sources:     map[string]*collectorbatch.SrcInfo{},
	}, bi)
	require.True(t, batch.IsEmpty())
}

func TestReachingLimitWithoutExceeding(t *testing.T) {
	ctx, _ := testshlp.CreateCtxLogger(t)
	b := collectorbatch.NewBatch(ctx, "ch1",
		collectorbatch.Limits{
			BlocksCountLimit: 2,
		}, func(b *executordto.Batch) (uint, error) {
			return 0, nil
		})

	ok, err := b.AddIfInLimit("ch1", &collectordto.BlockData{
		BlockNum: 1,
	})
	require.NoError(t, err)
	require.True(t, ok)

	_, linfo := b.GetBatchForExec()
	require.Equal(t, collectorbatch.NoneLimitKind, linfo.Kind)

	ok, err = b.AddIfInLimit("ch1", &collectordto.BlockData{
		BlockNum: 2,
	})
	require.NoError(t, err)
	require.True(t, ok)

	_, linfo = b.GetBatchForExec()
	require.Equal(t, collectorbatch.BlocksCountLimitKind, linfo.Kind)

	ok, err = b.AddIfInLimit("ch1", &collectordto.BlockData{
		BlockNum: 3,
	})
	require.NoError(t, err)
	require.False(t, ok)

	_, linfo = b.GetBatchForExec()
	require.Equal(t, collectorbatch.BlocksCountLimitKind, linfo.Kind)
}

func TestEmptyContent(t *testing.T) {
	ctx, _ := testshlp.CreateCtxLogger(t)
	b := collectorbatch.NewBatch(ctx, "ch1",
		collectorbatch.Limits{}, calcBatchSize)

	for chName, bn := range map[string]uint64{"ch1": 100, "ch2": 200, "ch3": 300} {
		ok, err := b.AddIfInLimit(chName, &collectordto.BlockData{
			BlockNum: bn,
		})
		require.NoError(t, err)
		require.True(t, ok)
	}

	batch, bi := b.GetBatchForExec()
	require.EqualValues(t, &collectorbatch.BatchInfo{
		Kind:        collectorbatch.NoneLimitKind,
		BlocksCount: 3,
		Sources: map[string]*collectorbatch.SrcInfo{
			"ch1": {
				LastBlockNum: 100,
				ItemsCount:   0,
			},
			"ch2": {
				LastBlockNum: 200,
				ItemsCount:   0,
			},
			"ch3": {
				LastBlockNum: 300,
				ItemsCount:   0,
			},
		},
	}, bi)

	ch2Src, ok := bi.Sources["ch2"]
	require.True(t, ok)
	require.Equal(t, uint64(200), ch2Src.LastBlockNum)

	ch3Src, ok := bi.Sources["ch3"]
	require.True(t, ok)
	require.Equal(t, uint64(300), ch3Src.LastBlockNum)

	require.Empty(t, batch.Txs)
	require.Empty(t, batch.MultiSwaps)
	require.Empty(t, batch.Swaps)
	require.Empty(t, batch.MultiKeys)
	require.Empty(t, batch.Keys)

	require.True(t, batch.IsEmpty())
}

func TestBasicBehavior(t *testing.T) {
	ch1Name, ch1StartBlockNum, ch1CountBlocks := "ch1", 5, 10
	ch2Name, ch2StartBlockNum, ch2CountBlocks := "ch2", 50, 8
	ch3Name, ch3StartBlockNum, ch3CountBlocks := "ch3", 500, 7

	ctx, _ := testshlp.CreateCtxLogger(t)
	b := collectorbatch.NewBatch(ctx, ch1Name,
		collectorbatch.Limits{}, calcBatchSize)
	require.NotNil(t, b)

	addBlocks := func(chName string, fbn, bc int) {
		for i := range bc {
			if chName == ch1Name {
				ok, err := b.AddIfInLimit(chName, &collectordto.BlockData{
					BlockNum: uint64(i + fbn),
					Txs:      [][]byte{{1, 2, 3}, {1, 2, 3}},
				})
				require.NoError(t, err)
				require.True(t, ok)
				continue
			}
			ok, err := b.AddIfInLimit(chName, &collectordto.BlockData{
				BlockNum:       uint64(i + fbn),
				MultiSwaps:     []*pb.MultiSwap{{}, {}, {}},
				Swaps:          []*pb.Swap{{}, {}, {}, {}},
				MultiSwapsKeys: []*pb.SwapKey{{}, {}, {}, {}, {}},
				SwapsKeys:      []*pb.SwapKey{{}, {}, {}, {}, {}, {}},
			})
			require.NoError(t, err)
			require.True(t, ok)
		}
	}
	addBlocks(ch1Name, ch1StartBlockNum, ch1CountBlocks)
	addBlocks(ch2Name, ch2StartBlockNum, ch2CountBlocks)
	addBlocks(ch3Name, ch3StartBlockNum, ch3CountBlocks)

	batch, bi := b.GetBatchForExec()
	require.EqualValues(t, &collectorbatch.BatchInfo{
		Kind:        collectorbatch.NoneLimitKind,
		BlocksCount: 25,
		Len:         290,
		Sources: map[string]*collectorbatch.SrcInfo{
			"ch1": {
				LastBlockNum: 14,
				ItemsCount:   10 * 2,
			},
			"ch2": {
				LastBlockNum: 57,
				ItemsCount:   8 * 18,
			},
			"ch3": {
				LastBlockNum: 506,
				ItemsCount:   7 * 18,
			},
		},
	}, bi)
	require.False(t, batch.IsEmpty())

	ch1Src, ok := bi.Sources[ch1Name]
	require.True(t, ok)
	require.Equal(t, uint64(ch1StartBlockNum+ch1CountBlocks-1), ch1Src.LastBlockNum)

	ch2Src, ok := bi.Sources[ch2Name]
	require.True(t, ok)
	require.Equal(t, uint64(ch2StartBlockNum+ch2CountBlocks-1), ch2Src.LastBlockNum)

	ch3Src, ok := bi.Sources[ch3Name]
	require.True(t, ok)
	require.Equal(t, uint64(ch3StartBlockNum+ch3CountBlocks-1), ch3Src.LastBlockNum)

	require.Len(t, batch.Txs, 2*ch1CountBlocks)
	require.Len(t, batch.MultiSwaps, 3*ch2CountBlocks+3*ch3CountBlocks)
	require.Len(t, batch.Swaps, 4*ch2CountBlocks+4*ch3CountBlocks)
	require.Len(t, batch.MultiKeys, 5*ch2CountBlocks+5*ch3CountBlocks)
	require.Len(t, batch.Keys, 6*ch2CountBlocks+6*ch3CountBlocks)
}

func TestDeadlineBehavior(t *testing.T) {
	ctx, _ := testshlp.CreateCtxLogger(t)
	const timeout = time.Second * 2
	b := collectorbatch.NewBatch(ctx, "ch1",
		collectorbatch.Limits{TimeoutLimit: timeout}, calcBatchSize)
	require.NotNil(t, b)

	ok, err := b.AddIfInLimit("ch1", &collectordto.BlockData{
		BlockNum: uint64(100),
		Txs:      [][]byte{{1, 2, 3}, {1, 2, 3}, {1, 2, 3}},
	})
	require.NoError(t, err)
	require.True(t, ok)

	<-time.After(timeout)

	ok, err = b.AddIfInLimit("ch1", &collectordto.BlockData{
		BlockNum: uint64(101),
		Txs:      [][]byte{{1, 2, 3}, {1, 2, 3}, {1, 2, 3}},
	})
	require.NoError(t, err)
	require.False(t, ok)

	_, li := b.GetBatchForExec()
	require.EqualValues(t, &collectorbatch.BatchInfo{
		Kind:        collectorbatch.TimeoutLimitKind,
		BlocksCount: 1,
		Len:         3,
		Sources: map[string]*collectorbatch.SrcInfo{
			"ch1": {
				LastBlockNum: 100,
				ItemsCount:   3,
			},
		},
	}, li)
}

func TestLenOverLimit(t *testing.T) {
	ctx, _ := testshlp.CreateCtxLogger(t)
	b := collectorbatch.NewBatch(ctx, "ch1",
		collectorbatch.Limits{
			LenLimit: 10,
		}, calcBatchSize)
	require.NotNil(t, b)

	ok, err := b.AddIfInLimit("ch1", &collectordto.BlockData{
		BlockNum: uint64(101),
		Txs:      [][]byte{{1, 2, 3}, {1, 2, 3}, {1, 2, 3}},
	})
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = b.AddIfInLimit("ch2", &collectordto.BlockData{
		BlockNum:  uint64(101),
		SwapsKeys: []*pb.SwapKey{{}, {}, {}},
	})
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = b.AddIfInLimit("ch1", &collectordto.BlockData{
		BlockNum: uint64(101),
		Txs:      [][]byte{{1, 2, 3}, {1, 2, 3}, {1, 2, 3}, {1, 2, 3}, {1, 2, 3}},
	})
	require.NoError(t, err)
	require.False(t, ok)

	_, li := b.GetBatchForExec()
	require.EqualValues(t, &collectorbatch.BatchInfo{
		Kind:        collectorbatch.LenLimitKind,
		BlocksCount: 2,
		Len:         6,
		Sources: map[string]*collectorbatch.SrcInfo{
			"ch1": {
				LastBlockNum: 101,
				ItemsCount:   3,
			},
			"ch2": {
				LastBlockNum: 101,
				ItemsCount:   3,
			},
		},
	}, li)
}

func TestSizeOverLimit(t *testing.T) {
	ctx, _ := testshlp.CreateCtxLogger(t)
	b := collectorbatch.NewBatch(ctx, "ch1", collectorbatch.Limits{
		SizeLimit: 40,
	}, calcBatchSize)
	require.NotNil(t, b)

	ok, err := b.AddIfInLimit("ch1", &collectordto.BlockData{
		BlockNum: uint64(101),
		Txs:      [][]byte{{1, 2, 3}, {1, 2, 3}, {1, 2, 3}},
	})
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = b.AddIfInLimit("ch2", &collectordto.BlockData{
		BlockNum:  uint64(101),
		SwapsKeys: []*pb.SwapKey{{}, {}, {}},
	})
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = b.AddIfInLimit("ch1", &collectordto.BlockData{
		BlockNum: uint64(101),
		Txs:      [][]byte{{1, 2, 3}, {1, 2, 3}, {1, 2, 3}, {1, 2, 3}, {1, 2, 3}},
	})
	require.NoError(t, err)
	require.False(t, ok)

	_, li := b.GetBatchForExec()
	require.EqualValues(t, &collectorbatch.BatchInfo{
		Kind:        collectorbatch.SizeLimitKind,
		BlocksCount: 2,
		Len:         6,
		Size:        21,
		Sources: map[string]*collectorbatch.SrcInfo{
			"ch1": {
				LastBlockNum: 101,
				ItemsCount:   3,
			},
			"ch2": {
				LastBlockNum: 101,
				ItemsCount:   3,
			},
		},
	}, li)
}

func TestBlocksCountOverLimit(t *testing.T) {
	ctx, _ := testshlp.CreateCtxLogger(t)
	b := collectorbatch.NewBatch(ctx, "ch1",
		collectorbatch.Limits{
			BlocksCountLimit: 2,
		}, calcBatchSize)
	require.NotNil(t, b)

	ok, err := b.AddIfInLimit("ch1", &collectordto.BlockData{
		BlockNum: uint64(101),
	})
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = b.AddIfInLimit("ch2", &collectordto.BlockData{
		BlockNum: uint64(102),
	})
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = b.AddIfInLimit("ch1", &collectordto.BlockData{
		BlockNum: uint64(103),
	})
	require.NoError(t, err)
	require.False(t, ok)

	_, li := b.GetBatchForExec()
	require.EqualValues(t, &collectorbatch.BatchInfo{
		Kind:        collectorbatch.BlocksCountLimitKind,
		BlocksCount: 2,
		Sources: map[string]*collectorbatch.SrcInfo{
			"ch1": {
				LastBlockNum: 101,
				ItemsCount:   0,
			},
			"ch2": {
				LastBlockNum: 102,
				ItemsCount:   0,
			},
		},
	}, li)
}

func calcBatchSize(b *executordto.Batch) (uint, error) {
	batch := &pb.Batch{
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
