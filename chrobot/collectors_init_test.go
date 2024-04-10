//go:build !integration
// +build !integration

package chrobot

import (
	"context"
	"testing"

	"github.com/anoideaopen/robot/collectorbatch"
	"github.com/anoideaopen/robot/dto/collectordto"
	"github.com/anoideaopen/robot/dto/stordto"
	"github.com/stretchr/testify/require"
)

type stubChStorage struct {
	chps map[string]uint64
}

func (stor *stubChStorage) SaveCheckPoints(_ context.Context, _ *stordto.ChCheckPoint) (*stordto.ChCheckPoint, error) {
	panic("need to implement SaveCheckPoints")
}

func (stor *stubChStorage) LoadCheckPoints(_ context.Context) (*stordto.ChCheckPoint, bool, error) {
	if stor.chps == nil {
		return nil, false, nil
	}

	return &stordto.ChCheckPoint{
		Ver:                     0,
		SrcCollectFromBlockNums: stor.chps,
		MinExecBlockNum:         0,
	}, true, nil
}

type stubCollectorFixFrom struct {
	srcChName string
	startFrom uint64
}

func (cc *stubCollectorFixFrom) GetData() <-chan *collectordto.BlockData {
	panic("need to implement GetData")
}

func (cc *stubCollectorFixFrom) Close() {
	panic("need to implement Close")
}

func createStubCollector(_ context.Context,
	_ chan<- struct{},
	srcChName string, startFrom uint64,
) (ChCollector, error) {
	return &stubCollectorFixFrom{
		srcChName: srcChName,
		startFrom: startFrom,
	}, nil
}

func TestCollectorsInitPositions(t *testing.T) {
	t.Run("case 1 - all empty", func(t *testing.T) {
		checkEffectiveInitPositions(t,
			map[string]uint64{},
			map[string]uint64{},
			map[string]uint64{},
		)
	})

	t.Run("case 2 - only config", func(t *testing.T) {
		checkEffectiveInitPositions(t,
			map[string]uint64{
				"ch1": 0,
				"ch2": 1,
			},
			nil,
			map[string]uint64{
				"ch1": 0,
				"ch2": 1,
			},
		)
	})

	t.Run("case 3 - only storage", func(t *testing.T) {
		checkEffectiveInitPositions(t,
			map[string]uint64{
				"ch1": 0,
				"ch2": 0,
			},
			map[string]uint64{
				"ch1": 0,
				"ch2": 1,
			},
			map[string]uint64{
				"ch1": 0,
				"ch2": 1,
			},
		)
	})

	t.Run("case 4 - storage override config", func(t *testing.T) {
		checkEffectiveInitPositions(t,
			map[string]uint64{
				"ch1": 0,
				"ch2": 1,
			},
			map[string]uint64{
				"ch1": 100,
			},
			map[string]uint64{
				"ch1": 100,
				"ch2": 1,
			},
		)
	})

	t.Run("case 5 - ignore chp if not in config", func(t *testing.T) {
		checkEffectiveInitPositions(t,
			map[string]uint64{
				"ch1": 0,
				"ch2": 1,
			},
			map[string]uint64{
				"ch3": 100,
			},
			map[string]uint64{
				"ch1": 0,
				"ch2": 1,
			},
		)
	})

	t.Run("case 6 - ch less than config", func(t *testing.T) {
		checkEffectiveInitPositions(t,
			map[string]uint64{
				"ch1": 100,
			},
			map[string]uint64{
				"ch1": 2,
			},
			map[string]uint64{
				"ch1": 100,
			},
		)
	})

	t.Run("case 7 - readme example", func(t *testing.T) {
		checkEffectiveInitPositions(t,
			map[string]uint64{
				"ch1": 50,
				"ch2": 100,
				"ch3": 0,
				"ch4": 0,
			},
			map[string]uint64{
				"ch1": 75,
				"ch2": 75,
				"ch3": 75,
			},
			map[string]uint64{
				"ch1": 75,
				"ch2": 100,
				"ch3": 75,
				"ch4": 0,
			},
		)
	})
}

func checkEffectiveInitPositions(t *testing.T, configData map[string]uint64, storageData map[string]uint64, expectedData map[string]uint64) {
	r := NewRobot(context.Background(), "robotCh", 0,
		configData,
		createStubCollector,
		nil,
		&stubChStorage{chps: storageData},
		collectorbatch.Limits{})

	err := r.createCollectors(context.Background())
	require.NoError(t, err)

	require.True(t, len(r.collectors) == len(expectedData))

	for _, c := range r.collectors {
		ep, ok := expectedData[c.chName]
		require.True(t, ok)

		collector, ok := c.collector.(*stubCollectorFixFrom)
		require.True(t, ok)

		require.Equal(t, c.chName, collector.srcChName)
		require.True(t,
			(collector.startFrom == ep) ||
				(collector.startFrom == 0 && ep == 0))
	}
}
