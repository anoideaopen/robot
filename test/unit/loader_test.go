//nolint:all
package unit

import (
	"context"
	"testing"
	"time"

	"github.com/anoideaopen/common-component/testshlp"
	"github.com/anoideaopen/glog"
	"github.com/anoideaopen/robot/dto/parserdto"
	"github.com/anoideaopen/robot/helpers/ntesting"
	"github.com/anoideaopen/robot/hlf"
	"github.com/stretchr/testify/require"
)

/*
Just pseudo util for generate preimages and test some hypotheses - its no test
*/

//nolint:deadcode
func NoTestPutPreimages(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	ctx, cancel := context.WithCancel(context.Background())
	ctx = glog.NewContext(ctx, l)
	defer cancel()

	ciData := ntesting.CI(t)

	l.Infof("get fiat owner")
	fiatOwner, err := ntesting.GetFiatOwner(ctx, ciData)
	require.NoError(t, err)
	l.Infof("fiat owner ready")

	oldFiatB, err := ntesting.GetBalance(ctx, fiatOwner, ciData.HlfFiatChannel, ciData.HlfFiatChannel)
	require.NoError(t, err)
	l.Infof("old user balance %v", oldFiatB)

	const (
		countEmits = 1000
		amountEmit = 1
	)
	totalB := oldFiatB
	for i := 0; i < countEmits; i++ {
		_, err = ntesting.EmitFiat(ctx, fiatOwner, fiatOwner, amountEmit, ciData.HlfFiatChannel, ciData.HlfFiatChannel)
		if err != nil {
			l.Infof("emit error: %s", err)
			time.Sleep(2 * time.Second)
		} else {
			totalB += amountEmit
		}
		l.Infof("attempts left: %v, last total balance: %v", countEmits-i, totalB)
	}
}

//nolint:deadcode
func NoTestCheckBalance(t *testing.T) {
	ctx, cancel, l := testshlp.CreateCtxLoggerWithCancel(t)

	ctx = glog.NewContext(ctx, l)
	defer cancel()

	ciData := ntesting.CI(t)

	l.Infof("get fiat owner")
	fiatOwner, err := ntesting.GetFiatOwner(ctx, ciData)
	require.NoError(t, err)
	l.Infof("fiat owner ready")

	b, err := ntesting.GetBalance(ctx, fiatOwner, ciData.HlfFiatChannel, ciData.HlfFiatChannel)
	require.NoError(t, err)
	l.Infof("fiat owner balance %v", b)
}

//nolint:deadcode
func NoTestGetBlocks(t *testing.T) {
	ciData := ntesting.CI(t)

	cr := hlf.NewChCollectorCreator(
		ciData.HlfFiatChannel,
		ciData.HlfProfilePath,
		ciData.HlfUserName,
		ciData.HlfProfile.OrgName, parserdto.TxPrefixes{
			Tx:        "swaps",
			MultiSwap: "multi_swap",
			Swap:      "batchTransactions",
		}, 1)

	ctx, l := testshlp.CreateCtxLogger(t)

	chc, err := cr(ctx, nil, ciData.HlfFiatChannel, 0)
	require.NoError(t, err)

	defer chc.Close()
	for {
		bd, ok := <-chc.GetData()
		if !ok {
			l.Info("collector channel closed!!!!!!")
			break
		}
		_ = bd
		if bd.BlockNum < 30 {
			time.Sleep(1 * time.Second)
		}
	}
}
