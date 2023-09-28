//go:build !integration
// +build !integration

package chcollector

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/atomyze-foundation/common-component/testshlp"
	"github.com/atomyze-foundation/robot/dto/collectordto"
	"github.com/hyperledger/fabric-protos-go/common"
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
