//go:build integration
// +build integration

package hlf

import (
	"context"
	"testing"

	"github.com/anoideaopen/common-component/testshlp"
	"github.com/anoideaopen/robot/dto/parserdto"
	"github.com/anoideaopen/robot/helpers/ntesting"
	"github.com/stretchr/testify/require"
)

var defaultPrefixes = parserdto.TxPrefixes{
	Tx:        "batchTransactions",
	Swap:      "swaps",
	MultiSwap: "multi_swap",
}

func TestChCollectorCreate(t *testing.T) {
	ciData := ntesting.CI(t)

	ctx := context.Background()
	cr := NewChCollectorCreator(
		ciData.HlfFiatChannel,
		ciData.HlfProfilePath,
		ciData.HlfUserName,
		ciData.HlfProfile.OrgName,
		defaultPrefixes,
		1)
	require.NotNil(t, cr)

	dataReady := make(chan struct{}, 1)

	cl, err := cr(ctx, dataReady, ciData.HlfFiatChannel, 10)
	require.NoError(t, err)
	require.NotNil(t, cl)
}

func TestChCollectorCreateWithoutChainCode(t *testing.T) {
	ciData := ntesting.CI(t)

	ctx := context.Background()
	cr := NewChCollectorCreator(
		ciData.HlfNoCcChannel,
		ciData.HlfProfilePath,
		ciData.HlfUserName, ciData.HlfProfile.OrgName,
		defaultPrefixes, 1)
	require.NotNil(t, cr)

	dataReady := make(chan struct{}, 1)

	cl, err := cr(ctx, dataReady, ciData.HlfNoCcChannel, 0)
	require.NoError(t, err)
	require.NotNil(t, cl)

	bd, ok := <-cl.GetData()
	require.True(t, ok)
	require.Equal(t, uint64(0), bd.BlockNum)
}

func TestGetData(t *testing.T) {
	ciData := ntesting.CI(t)

	ctx, log := testshlp.CreateCtxLogger(t)

	cr := NewChCollectorCreator(
		ciData.HlfFiatChannel,
		ciData.HlfProfilePath,
		ciData.HlfUserName, ciData.HlfProfile.OrgName,
		defaultPrefixes, 1)
	require.NotNil(t, cr)

	dataReady := make(chan struct{}, 1)

	cl, err := cr(ctx, dataReady, ciData.HlfFiatChannel, 0)
	require.NoError(t, err)
	require.NotNil(t, cl)

	const closeAfterNBlocks = 1

	for i := 0; ; i++ {
		select {
		case d, ok := <-cl.GetData():
			if ok {
				log.Info("data from block:", d.BlockNum)
			}

			if !ok && i >= closeAfterNBlocks {
				log.Info("channel closed i:", i)
				return
			}

			require.True(t, ok)
			require.NotNil(t, d)

			if !d.IsEmpty() {
				log.Info("data for block:", d.BlockNum,
					" lens:", len(d.Txs), len(d.Swaps), len(d.MultiSwaps), len(d.SwapsKeys), len(d.MultiSwapsKeys))
			}

		default:
			log.Debug("need to wait")
			<-dataReady
		}

		// after closeAfterNBlocks - init stop
		if i == closeAfterNBlocks {
			log.Info("schedule close")

			// don't wait
			go func() {
				cl.Close()
				log.Info("was closed")
			}()
		}
	}
}

func TestBagSdkSubscrEvents(t *testing.T) {
	ciData := ntesting.CI(t)

	logCtx, _ := testshlp.CreateCtxLogger(t)

	cr1 := NewChCollectorCreator(
		ciData.HlfFiatChannel,
		ciData.HlfProfilePath,
		ciData.HlfUserName,
		ciData.HlfProfile.OrgName,
		defaultPrefixes,
		10,
	)
	require.NotNil(t, cr1)

	dataReady1 := make(chan struct{}, 1)

	cl1, err := cr1(logCtx, dataReady1, ciData.HlfFiatChannel, 3)
	require.NoError(t, err)
	require.NotNil(t, cl1)

	b1, ok := <-cl1.GetData()
	require.True(t, ok)
	require.Equal(t, uint64(3), b1.BlockNum)

	cr2 := NewChCollectorCreator(
		"dst-2",
		ciData.HlfProfilePath,
		ciData.HlfUserName, ciData.HlfProfile.OrgName,
		defaultPrefixes, 1)
	require.NotNil(t, cr1)

	dataReady2 := make(chan struct{}, 1)
	cl2, err := cr2(logCtx, dataReady2, ciData.HlfFiatChannel, 1)

	b2, ok := <-cl2.GetData()
	require.True(t, ok)
	require.Equal(t, uint64(1), b2.BlockNum)
}
