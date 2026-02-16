package unit

import (
	"context"
	"testing"

	"github.com/anoideaopen/robot/chcollector"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/stretchr/testify/require"
)

func BenchmarkChCollectorGetData(b *testing.B) {
	b.Run("benchmark with 1 buffer capacity", func(b *testing.B) {
		collectNBlocks(b, 10000, 1)
	})

	b.Run("benchmark with 100000 buffer capacity", func(b *testing.B) {
		collectNBlocks(b, 10000, 100000)
	})
}

func collectNBlocks(b *testing.B, totalCountBlocks int, bufSize uint) {
	b.Helper()

	ctx := context.Background()
	dataReady := make(chan struct{}, 1)
	prsr := &stubDataParser{}

	for i := 0; i < b.N; i++ {
		events := make(chan *fab.BlockEvent, totalCountBlocks)
		for range totalCountBlocks {
			events <- &fab.BlockEvent{Block: &common.Block{}}
		}

		dc, err := chcollector.NewCollector(ctx, prsr, dataReady, events, bufSize)
		require.NoError(b, err)
		countBlocks := 0
	loop:
		for {
			select {
			case d, ok := <-dc.GetData():
				require.True(b, ok)
				require.NotNil(b, d)
				countBlocks++
				if countBlocks == totalCountBlocks {
					break loop
				}
			default:
				<-dataReady
			}
		}

		dc.Close()
	}
}
