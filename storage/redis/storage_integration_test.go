//go:build integration
// +build integration

//nolint:unused
package redis

import (
	"context"
	"testing"

	"github.com/anoideaopen/robot/dto/stordto"
	"github.com/anoideaopen/robot/helpers/ntesting"
	"github.com/stretchr/testify/require"
)

const (
	dbPrefix = "my1"
	channel1 = "ch1"
	channel2 = "ch2"
	channel3 = "ch3"
)

func TestStorageSaveLoadCheckPoint(t *testing.T) {
	ciData := ntesting.CI(t)

	clearAll(t)
	ctx := context.Background()
	stor, err := NewStorage(
		ctx,
		[]string{ciData.RedisAddr}, ciData.RedisPass,
		false, nil,
		dbPrefix, channel1)

	require.NoError(t, err)
	require.NotNil(t, stor)

	// 1. Empty storage
	v, ok, err := stor.LoadCheckPoints(ctx)
	require.Nil(t, v)
	require.False(t, ok)
	require.Nil(t, err)

	// 2. Save into empty storage
	cp1 := &stordto.ChCheckPoint{
		Ver:                     10,
		SrcCollectFromBlockNums: map[string]uint64{"fiat": 123},
		MinExecBlockNum:         456,
	}
	v, err = stor.SaveCheckPoints(ctx, cp1)
	require.NotNil(t, v)
	require.Nil(t, err)
	require.EqualValues(t, cp1.Ver, v.Ver)

	// 3. Trying update with the other version
	cp2 := &stordto.ChCheckPoint{
		Ver:                     2,
		SrcCollectFromBlockNums: map[string]uint64{"fiat": 234},
		MinExecBlockNum:         567,
	}
	v, err = stor.SaveCheckPoints(ctx, cp2)
	require.Nil(t, v)
	require.NotNil(t, err)
	require.ErrorIs(t, err, ErrStorVersionMismatch)

	// 4. Update with the same version
	cp3 := &stordto.ChCheckPoint{
		Ver:                     cp1.Ver,
		SrcCollectFromBlockNums: map[string]uint64{"fiat": 1234},
		MinExecBlockNum:         5678,
	}
	v, err = stor.SaveCheckPoints(ctx, cp3)
	require.NotNil(t, v)
	require.Nil(t, err)
	require.Greater(t, v.Ver, cp1.Ver)

	// 5. Load existed value
	v, ok, err = stor.LoadCheckPoints(ctx)
	require.NotNil(t, v)
	require.True(t, ok)
	require.Nil(t, err)
	require.EqualValues(t, v.SrcCollectFromBlockNums, cp3.SrcCollectFromBlockNums)
	require.EqualValues(t, v.MinExecBlockNum, cp3.MinExecBlockNum)
}

func TestStorageDifferentChannelsCheckPoints(t *testing.T) {
	ciData := ntesting.CI(t)

	clearAll(t)
	ctx := context.Background()

	stor1, err := NewStorage(
		ctx,
		[]string{ciData.RedisAddr}, ciData.RedisPass,
		false, nil,
		dbPrefix, channel1)
	require.NoError(t, err)
	require.NotNil(t, stor1)

	stor2, err := NewStorage(
		ctx,
		[]string{ciData.RedisAddr}, ciData.RedisPass,
		false, nil,
		dbPrefix, channel2)
	require.NoError(t, err)
	require.NotNil(t, stor2)

	chp1, err := stor1.SaveCheckPoints(ctx, &stordto.ChCheckPoint{
		Ver:                     1,
		SrcCollectFromBlockNums: map[string]uint64{"fiat": 123},
		MinExecBlockNum:         111,
	})
	require.NoError(t, err)
	require.NotNil(t, chp1)

	chp2, err := stor2.SaveCheckPoints(ctx, &stordto.ChCheckPoint{
		Ver:                     20,
		SrcCollectFromBlockNums: map[string]uint64{"fiat": 123},
		MinExecBlockNum:         222,
	})
	require.NoError(t, err)
	require.NotNil(t, chp2)

	chp1, ok, err := stor1.LoadCheckPoints(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, chp1)

	chp2, ok, err = stor2.LoadCheckPoints(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, chp2)

	require.NotEqual(t, chp1.Ver, chp2.Ver)
	require.NotEqual(t, chp1.MinExecBlockNum, chp2.MinExecBlockNum)
}

func clearAll(t *testing.T) {
	ciData := ntesting.CI(t)
	ctx := context.Background()
	for _, chName := range []string{channel1, channel2, channel3} {
		stor, err := NewStorage(
			ctx,
			[]string{ciData.RedisAddr}, ciData.RedisPass,
			false, nil,
			dbPrefix, chName)

		require.NoError(t, err)
		require.NotNil(t, stor)

		err = stor.RemoveAllData(ctx)
		require.NoErrorf(t, err, "%+v", err)
	}
}
