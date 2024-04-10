//go:build !integration
// +build !integration

package chcollector

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/anoideaopen/common-component/testshlp"
	"github.com/anoideaopen/robot/dto/collectordto"
	"github.com/anoideaopen/robot/dto/parserdto"
	"github.com/anoideaopen/robot/hlf/parser"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/stretchr/testify/require"
)

var defaultPrefixes = parserdto.TxPrefixes{
	Tx:        "batchTransactions",
	Swap:      "swaps",
	MultiSwap: "multi_swap",
}

type stubDataParser struct {
	callHlp testshlp.CallHlp
}

func (sdp *stubDataParser) ExtractData(block *common.Block) (*collectordto.BlockData, error) {
	if err := sdp.callHlp.Call(sdp.ExtractData); err != nil {
		return nil, err
	}
	return &collectordto.BlockData{}, nil
}

func TestCreate(t *testing.T) {
	ctx := context.Background()
	dataReady := make(chan struct{}, 1)
	events := make(chan *fab.BlockEvent)
	prsr := &stubDataParser{}

	dc, err := NewCollector(ctx, prsr, dataReady, events, 1)
	require.NoError(t, err)
	require.NotNil(t, dc)
}

func TestEmptyStartFinish(t *testing.T) {
	ctx := context.Background()
	dataReady := make(chan struct{}, 1)
	events := make(chan *fab.BlockEvent)
	prsr := &stubDataParser{}

	dc, err := NewCollector(ctx, prsr, dataReady, events, 1)
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
		defaultPrefixes)
	require.NotNil(t, prs)

	bltx := getBlock(t, "../hlf/parser/test-data/blocks/block_with_preimages.block")

	const totalCountBlocks = 2
	// добавляем в канал событий, два одинаковых блока с транзакциями
	events := make(chan *fab.BlockEvent, totalCountBlocks)
	for i := 0; i < totalCountBlocks; i++ {
		events <- &fab.BlockEvent{Block: bltx}
	}

	// создаем коллектор и передаем туда канал с событиями
	dc, err := NewCollector(ctx, prs, dataReady, events, 1)
	require.NoError(t, err)
	countBlocks := 0

	// массив с распаршенными блоками
	var bd []*collectordto.BlockData
loop:
	for {
		select {
		case d, ok := <-dc.GetData():
			// добавляем блоки в слайс
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

	// проверяем что они одинаковые
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
		defaultPrefixes)
	require.NotNil(t, prs)

	bltx := getBlock(t, "../hlf/parser/test-data/blocks/block_with_multiswaps.block")

	const totalCountBlocks = 2
	// добавляем в канал событий, два одинаковых блока с транзакциями
	events := make(chan *fab.BlockEvent, totalCountBlocks)
	for i := 0; i < totalCountBlocks; i++ {
		events <- &fab.BlockEvent{Block: bltx}
	}

	// создаем коллектор и передаем туда канал с событиями
	dc, err := NewCollector(ctx, prs, dataReady, events, 1)
	require.NoError(t, err)
	countBlocks := 0

	// массив с распаршенными блоками
	var bd []*collectordto.BlockData
loop:
	for {
		select {
		case d, ok := <-dc.GetData():
			// добавляем блоки в слайс
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

	// проверяем что они одинаковые
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
		defaultPrefixes)
	require.NotNil(t, prs)

	bltx := getBlock(t, "../hlf/parser/test-data/blocks/block_with_keys.block")

	const totalCountBlocks = 2
	// добавляем в канал событий, два одинаковых блока с транзакциями
	events := make(chan *fab.BlockEvent, totalCountBlocks)
	for i := 0; i < totalCountBlocks; i++ {
		events <- &fab.BlockEvent{Block: bltx}
	}

	// создаем коллектор и передаем туда канал с событиями
	dc, err := NewCollector(ctx, prs, dataReady, events, 1)
	require.NoError(t, err)
	countBlocks := 0

	// массив с распаршенными блоками
	var bd []*collectordto.BlockData
loop:
	for {
		select {
		case d, ok := <-dc.GetData():
			// добавляем блоки в слайс
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

	// проверяем что они одинаковые
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
		defaultPrefixes)
	require.NotNil(t, prs)

	bltx := getBlock(t, "../hlf/parser/test-data/blocks/block_with_multikeys.block")

	const totalCountBlocks = 2
	// добавляем в канал событий, два одинаковых блока с транзакциями
	events := make(chan *fab.BlockEvent, totalCountBlocks)
	for i := 0; i < totalCountBlocks; i++ {
		events <- &fab.BlockEvent{Block: bltx}
	}

	// создаем коллектор и передаем туда канал с событиями
	dc, err := NewCollector(ctx, prs, dataReady, events, 1)
	require.NoError(t, err)
	countBlocks := 0

	// массив с распаршенными блоками
	var bd []*collectordto.BlockData
loop:
	for {
		select {
		case d, ok := <-dc.GetData():
			// добавляем блоки в слайс
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

	// проверяем что они одинаковые
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
		defaultPrefixes)
	require.NotNil(t, prs)

	bltx := getBlock(t, "../hlf/parser/test-data/blocks/block_with_swaps.block")

	const totalCountBlocks = 2
	// добавляем в канал событий, два одинаковых блока с транзакциями
	events := make(chan *fab.BlockEvent, totalCountBlocks)
	for i := 0; i < totalCountBlocks; i++ {
		events <- &fab.BlockEvent{Block: bltx}
	}

	// создаем коллектор и передаем туда канал с событиями
	dc, err := NewCollector(ctx, prs, dataReady, events, 1)
	require.NoError(t, err)
	countBlocks := 0

	// массив с распаршенными блоками
	var bd []*collectordto.BlockData
loop:
	for {
		select {
		case d, ok := <-dc.GetData():
			// добавляем блоки в слайс
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

	// проверяем что они одинаковые
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

	dc, err := NewCollector(ctx, prsr, dataReady, events, 1)
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

func TestWorkWithErrors(t *testing.T) {
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

	dc, err := NewCollector(ctx, prsr, dataReady, events, 1)
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

func getBlock(t *testing.T, pathToBlock string) *common.Block {
	file, err := os.ReadFile(pathToBlock)
	require.NoError(t, err)

	fabBlock := &common.Block{}
	err = proto.Unmarshal(file, fabBlock)
	require.NoError(t, err)

	return fabBlock
}
