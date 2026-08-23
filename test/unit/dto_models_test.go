package unit

import (
	"math/rand"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/proto"
	"github.com/stretchr/testify/require"
	pb "google.golang.org/protobuf/proto"
)

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func TestSize(t *testing.T) {
	protoData := generateBatches()

	// Test proto
	size1 := 0
	for _, pd := range protoData {
		size1 += pb.Size(pd)
	}

	size2 := 0
	for _, pd := range protoData {
		m, err := pb.Marshal(pd)
		if err != nil {
			t.Error(err)
		}
		size2 += len(m)
	}
	require.Equal(t, size1, size2)
}

func BenchmarkProtoMarshalVsSize(b *testing.B) {
	protoData := generateBatches()

	b.Run("pb.Marshal", func(b *testing.B) {
		for _, pd := range protoData {
			_, _ = pb.Marshal(pd)
		}
	})

	b.Run("pb.Size", func(b *testing.B) {
		for _, pd := range protoData {
			_ = pb.Size(pd)
		}
	})
}

func generateBatches() []*proto.Batch {
	res := make([]*proto.Batch, 0, 1000)
	for range 1000 {
		b := &proto.Batch{
			TxIDs:          generateTxses(),
			Swaps:          generateSwaps(),
			MultiSwaps:     generateMultiSwaps(),
			Keys:           generateSwapKeys(),
			MultiSwapsKeys: generateSwapKeys(),
		}

		res = append(res, b)
	}
	return res
}

func generateTxses() [][]byte {
	count := rnd.Intn(100)
	res := make([][]byte, 0, count)
	for range count {
		res = append(res, generateBytes())
	}
	return res
}

func generateSwaps() []*proto.Swap {
	count := rnd.Intn(100)
	res := make([]*proto.Swap, 0, count)
	for range count {
		s := &proto.Swap{
			Id:      generateBytes(),
			Creator: generateBytes(),
			Owner:   generateBytes(),
			Token:   generateString(),
			Amount:  generateBytes(),
			From:    generateString(),
			To:      generateString(),
			Hash:    generateBytes(),
			Timeout: 0,
		}
		res = append(res, s)
	}
	return res
}

func generateMultiSwaps() []*proto.MultiSwap {
	count := rnd.Intn(100)
	res := make([]*proto.MultiSwap, 0, count)
	for range count {
		s := &proto.MultiSwap{
			Id:      generateBytes(),
			Creator: generateBytes(),
			Owner:   generateBytes(),
			Token:   generateString(),
			From:    generateString(),
			To:      generateString(),
			Hash:    generateBytes(),
			Timeout: 0,
			Assets:  generateAssets(),
		}
		res = append(res, s)
	}
	return res
}

func generateSwapKeys() []*proto.SwapKey {
	count := rnd.Intn(100)
	res := make([]*proto.SwapKey, 0, count)
	for range count {
		s := &proto.SwapKey{
			Id:  generateBytes(),
			Key: generateString(),
		}
		res = append(res, s)
	}
	return res
}

func generateBytes() []byte {
	count := rnd.Intn(100)
	res := make([]byte, 0, count)
	for range count {
		res = append(res, 0)
	}
	return res
}

func generateString() string {
	return string(generateBytes())
}

func generateAssets() []*proto.Asset {
	count := rnd.Intn(100)
	res := make([]*proto.Asset, 0, count)
	for range count {
		s := &proto.Asset{
			Group:  generateString(),
			Amount: generateBytes(),
		}
		res = append(res, s)
	}
	return res
}
