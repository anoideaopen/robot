//go:build integration
// +build integration

package hlf

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/anoideaopen/common-component/testshlp"
	"github.com/anoideaopen/robot/dto/executordto"
	"github.com/anoideaopen/robot/helpers/ntesting"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/common/selection/fabricselection"
	"github.com/stretchr/testify/require"
)

var defaultExecOpts = ExecuteOptions{
	ExecuteTimeout: 0 * time.Second,
}

func TestChExecutorCreateWithoutChainCode(t *testing.T) {
	ciData := ntesting.CI(t)

	ctx := context.Background()

	che, err := createChExecutor(ctx,
		ciData.HlfNoCcChannel,
		ciData.HlfProfilePath,
		ciData.HlfUserName,
		ciData.HlfProfile.OrgName,
		defaultExecOpts,
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, che)

	_, err = che.Execute(ctx, &executordto.Batch{
		Txs: [][]byte{
			{1, 2, 3, 4},
		},
	}, 0)
	require.Error(t, err)

	var dErr fabricselection.DiscoveryError
	require.True(t, errors.As(err, &dErr))
	require.True(t, dErr.IsTransient())
}

func TestChExecutorExecute(t *testing.T) {
	ciData := ntesting.CI(t)

	ctx, _ := testshlp.CreateCtxLogger(t)

	t.Run("send fake txs, check them committed", func(t *testing.T) {
		che := newFiatChExecutor(ctx, t, ciData)

		firstBlockN, err := che.Execute(context.Background(), &executordto.Batch{
			Txs:        [][]byte{[]byte("123")},
			Swaps:      nil,
			MultiSwaps: nil,
			Keys:       nil,
			MultiKeys:  nil,
		}, 1)
		require.NoError(t, err)

		secondBlockN, err := che.Execute(context.Background(), &executordto.Batch{
			Txs:        [][]byte{[]byte("123")},
			Swaps:      nil,
			MultiSwaps: nil,
			Keys:       nil,
			MultiKeys:  nil,
		}, firstBlockN+1)

		require.NoError(t, err)
		require.Greater(t, secondBlockN, firstBlockN)
	})

	t.Run("emit, check user balance", func(t *testing.T) {
		user, err := ntesting.CreateTestUser(context.Background(), ciData, "user")
		require.NoError(t, err)

		fiatOwner, err := ntesting.GetFiatOwner(context.Background(), ciData)
		require.NoError(t, err)

		var preimages [][]byte
		for i := 0; i < 3; i++ {
			txID, err := ntesting.EmitFiat(context.Background(), fiatOwner, user, 1, ciData.HlfFiatChannel, ciData.HlfFiatChannel)
			require.NoError(t, err)

			idBytes, err := hex.DecodeString(txID)
			require.NoError(t, err)

			preimages = append(preimages, idBytes)
		}
		chl := newFiatChExecutor(ctx, t, ciData)

		_, err = chl.Execute(context.Background(), &executordto.Batch{
			Txs:        preimages,
			Swaps:      nil,
			MultiSwaps: nil,
			Keys:       nil,
			MultiKeys:  nil,
		}, 1)
		require.NoError(t, err)

		require.NoError(t, user.BalanceShouldBe(3))
	})
}

func newFiatChExecutor(ctx context.Context, t *testing.T, ciData ntesting.CiTestData) *chExecutor {
	che, err := createChExecutor(ctx,
		ciData.HlfFiatChannel,
		ciData.HlfProfilePath,
		ciData.HlfUserName,
		ciData.HlfProfile.OrgName,
		defaultExecOpts,
		nil,
	)
	require.NoError(t, err)
	return che
}
