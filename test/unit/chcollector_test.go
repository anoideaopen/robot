package unit

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"testing"
	"time"

	"github.com/anoideaopen/common-component/testshlp"
	"github.com/anoideaopen/robot/chcollector"
	"github.com/anoideaopen/robot/dto/collectordto"
	"github.com/anoideaopen/robot/hlf/parser"
	cmn "github.com/anoideaopen/robot/test/unit/common"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/stretchr/testify/require"
)

type stubDataParser struct {
	callHlp testshlp.CallHlp
}

func (sdp *stubDataParser) ExtractData(block *common.Block) (*collectordto.BlockData, error) {
	if err := sdp.callHlp.Call(sdp.ExtractData); err != nil {
		return nil, err
	}
	return &collectordto.BlockData{}, nil
}

func TestCollectorCreate(t *testing.T) {
	ctx := context.Background()
	dataReady := make(chan struct{}, 1)
	events := make(chan *fab.BlockEvent)
	prsr := &stubDataParser{}

	dc, err := chcollector.NewCollector(ctx, prsr, dataReady, events, 1)
	require.NoError(t, err)
	require.NotNil(t, dc)
}

func TestEmptyStartFinish(t *testing.T) {
	ctx := context.Background()
	dataReady := make(chan struct{}, 1)
	events := make(chan *fab.BlockEvent)
	prsr := &stubDataParser{}

	dc, err := chcollector.NewCollector(ctx, prsr, dataReady, events, 1)
	require.NoError(t, err)

	dch := dc.GetData()

	go func() {
		<-time.After(2 * time.Second)
		dc.Close()
	}()

	_, ok := <-dch
	require.False(t, ok)
}

func TestCollectingTheSameBlock(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)
	ctx := context.Background()
	dataReady := make(chan struct{}, 1)

	prs := parser.NewParser(l,
		"ch1", "ch1",
		cmn.DefaultPrefixes)
	require.NotNil(t, prs)

	bltx := cmn.GetBlock(t, "test-data/parser/block_with_preimages.block")

	const totalCountBlocks = 2
	// add two identical blocks with transactions to the event channel
	events := make(chan *fab.BlockEvent, totalCountBlocks)
	for i := 0; i < totalCountBlocks; i++ {
		events <- &fab.BlockEvent{Block: bltx}
	}

	// create a collector and pass an event channel to it
	dc, err := chcollector.NewCollector(ctx, prs, dataReady, events, 1)
	require.NoError(t, err)
	countBlocks := 0

	// unmarshaled array
	var bd []*collectordto.BlockData
loop:
	for {
		select {
		case d, ok := <-dc.GetData():
			// add blocks to the slice
			bd = append(bd, d)
			require.True(t, ok)
			require.NotNil(t, d)
			countBlocks++
			log.Println("got data count:", countBlocks)
			if countBlocks == totalCountBlocks {
				break loop
			}
		default:
			<-dataReady
		}
	}

	// make sure they're the same.
	require.True(t, reflect.DeepEqual(bd[0], bd[1]))
	dc.Close()
	_, ok := <-dc.GetData()
	require.False(t, ok)
}

func TestCollectingTheSameBlockMultiswaps(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)
	ctx := context.Background()
	dataReady := make(chan struct{}, 1)

	prs := parser.NewParser(l,
		"ch1", "ch1",
		cmn.DefaultPrefixes)
	require.NotNil(t, prs)

	bltx := cmn.GetBlock(t, "test-data/parser/block_with_multiswaps.block")

	const totalCountBlocks = 2
	// add two identical blocks with transactions to the event channel
	events := make(chan *fab.BlockEvent, totalCountBlocks)
	for i := 0; i < totalCountBlocks; i++ {
		events <- &fab.BlockEvent{Block: bltx}
	}

	// create a collector and pass an event channel to it
	dc, err := chcollector.NewCollector(ctx, prs, dataReady, events, 1)
	require.NoError(t, err)
	countBlocks := 0

	// unmarshaled array
	var bd []*collectordto.BlockData
loop:
	for {
		select {
		case d, ok := <-dc.GetData():
			// add blocks to the slice
			bd = append(bd, d)
			require.True(t, ok)
			require.NotNil(t, d)
			countBlocks++
			log.Println("got data count:", countBlocks)
			if countBlocks == totalCountBlocks {
				break loop
			}
		default:
			<-dataReady
		}
	}

	// make sure they're the same.
	require.True(t, reflect.DeepEqual(bd[0], bd[1]))
	dc.Close()
	_, ok := <-dc.GetData()
	require.False(t, ok)
}

func TestCollectingTheSameBlockKeys(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)
	ctx := context.Background()
	dataReady := make(chan struct{}, 1)

	prs := parser.NewParser(l,
		"ch1", "ch1",
		cmn.DefaultPrefixes)
	require.NotNil(t, prs)

	bltx := cmn.GetBlock(t, "test-data/parser/block_with_keys.block")

	const totalCountBlocks = 2
	// add two identical blocks with transactions to the event channel
	events := make(chan *fab.BlockEvent, totalCountBlocks)
	for i := 0; i < totalCountBlocks; i++ {
		events <- &fab.BlockEvent{Block: bltx}
	}

	// create a collector and pass an event channel to it
	dc, err := chcollector.NewCollector(ctx, prs, dataReady, events, 1)
	require.NoError(t, err)
	countBlocks := 0

	// unmarshaled array
	var bd []*collectordto.BlockData
loop:
	for {
		select {
		case d, ok := <-dc.GetData():
			// add blocks to the slice
			bd = append(bd, d)
			require.True(t, ok)
			require.NotNil(t, d)
			countBlocks++
			log.Println("got data count:", countBlocks)
			if countBlocks == totalCountBlocks {
				break loop
			}
		default:
			<-dataReady
		}
	}

	// make sure they're the same.
	require.True(t, reflect.DeepEqual(bd[0], bd[1]))
	dc.Close()
	_, ok := <-dc.GetData()
	require.False(t, ok)
}

func TestCollectingTheSameBlockMultikeys(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)
	ctx := context.Background()
	dataReady := make(chan struct{}, 1)

	prs := parser.NewParser(l,
		"ch1", "ch1",
		cmn.DefaultPrefixes)
	require.NotNil(t, prs)

	bltx := cmn.GetBlock(t, "test-data/parser/block_with_multikeys.block")

	const totalCountBlocks = 2
	// add two identical blocks with transactions to the event channel
	events := make(chan *fab.BlockEvent, totalCountBlocks)
	for i := 0; i < totalCountBlocks; i++ {
		events <- &fab.BlockEvent{Block: bltx}
	}

	// create a collector and pass an event channel to it
	dc, err := chcollector.NewCollector(ctx, prs, dataReady, events, 1)
	require.NoError(t, err)
	countBlocks := 0

	// unmarshaled array
	var bd []*collectordto.BlockData
loop:
	for {
		select {
		case d, ok := <-dc.GetData():
			// add blocks to the slice
			bd = append(bd, d)
			require.True(t, ok)
			require.NotNil(t, d)
			countBlocks++
			log.Println("got data count:", countBlocks)
			if countBlocks == totalCountBlocks {
				break loop
			}
		default:
			<-dataReady
		}
	}

	// make sure they're the same.
	require.True(t, reflect.DeepEqual(bd[0], bd[1]))
	dc.Close()
	_, ok := <-dc.GetData()
	require.False(t, ok)
}

func TestCollectingTheSameBlockSwaps(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)
	ctx := context.Background()
	dataReady := make(chan struct{}, 1)

	prs := parser.NewParser(l,
		"ch1", "ch1",
		cmn.DefaultPrefixes)
	require.NotNil(t, prs)

	bltx := cmn.GetBlock(t, "test-data/parser/block_with_swaps.block")

	const totalCountBlocks = 2
	// add two identical blocks with transactions to the event channel
	events := make(chan *fab.BlockEvent, totalCountBlocks)
	for i := 0; i < totalCountBlocks; i++ {
		events <- &fab.BlockEvent{Block: bltx}
	}

	// create a collector and pass an event channel to it
	dc, err := chcollector.NewCollector(ctx, prs, dataReady, events, 1)
	require.NoError(t, err)
	countBlocks := 0

	// unmarshaled array
	var bd []*collectordto.BlockData
loop:
	for {
		select {
		case d, ok := <-dc.GetData():
			// add blocks to the slice
			bd = append(bd, d)
			require.True(t, ok)
			require.NotNil(t, d)
			countBlocks++
			log.Println("got data count:", countBlocks)
			if countBlocks == totalCountBlocks {
				break loop
			}
		default:
			<-dataReady
		}
	}

	// make sure they're the same.
	require.True(t, reflect.DeepEqual(bd[0], bd[1]))
	dc.Close()
	_, ok := <-dc.GetData()
	require.False(t, ok)
}

func TestSimpleWork(t *testing.T) {
	ctx := context.Background()
	dataReady := make(chan struct{}, 1)
	prsr := &stubDataParser{}

	const totalCountBlocks = 100
	events := make(chan *fab.BlockEvent, totalCountBlocks)
	for i := 0; i < totalCountBlocks; i++ {
		events <- &fab.BlockEvent{Block: &common.Block{
			Header: &common.BlockHeader{
				Number: uint64(i),
			},
		}}
	}

	dc, err := chcollector.NewCollector(ctx, prsr, dataReady, events, 1)
	require.NoError(t, err)
	countBlocks := 0
loop:
	for {
		select {
		case d, ok := <-dc.GetData():
			require.True(t, ok)
			require.NotNil(t, d)
			countBlocks++
			log.Println("got data count:", countBlocks)
			if countBlocks == totalCountBlocks {
				break loop
			}
		default:
			<-dataReady
		}
	}

	dc.Close()
	_, ok := <-dc.GetData()
	require.False(t, ok)
}

func TestCollectorWorkWithErrors(t *testing.T) {
	ctx := context.Background()
	dataReady := make(chan struct{}, 1)
	prsr := &stubDataParser{}

	const totalCountBlocks = 100
	prsrErrors := map[int]error{
		2:                    errors.New("stub parse 2 error"),
		5:                    errors.New("stub parse 5 error"),
		50:                   errors.New("stub parse 50 error"),
		totalCountBlocks - 1: fmt.Errorf("stub parse %v error", totalCountBlocks-1),
	}
	prsr.callHlp.AddErrMap(prsr.ExtractData, prsrErrors)

	events := make(chan *fab.BlockEvent, totalCountBlocks)
	for i := 0; i < totalCountBlocks; i++ {
		events <- &fab.BlockEvent{Block: &common.Block{
			Header: &common.BlockHeader{
				Number: uint64(i),
			},
		}}
	}

	dc, err := chcollector.NewCollector(ctx, prsr, dataReady, events, 1)
	require.NoError(t, err)
	countBlocks := 0
loop:
	for {
		select {
		case d, ok := <-dc.GetData():
			require.True(t, ok)
			require.NotNil(t, d)
			countBlocks++
			log.Println("got data count:", countBlocks)
			if countBlocks == totalCountBlocks-len(prsrErrors) {
				break loop
			}
		default:
			log.Println("data not ready - will wait")
			<-dataReady
		}
	}

	dc.Close()

	_, ok := <-dc.GetData()
	require.False(t, ok)
}
