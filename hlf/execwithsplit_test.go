//go:build !integration
// +build !integration

package hlf

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/atomyze-foundation/common-component/testshlp"
	"github.com/atomyze-foundation/robot/dto/executordto"
	"github.com/stretchr/testify/require"
)

func TestSplitBatchForExec(t *testing.T) {
	origBatch := generateOrigBatch()

	getBlockForTx := func(tx []byte) uint64 {
		bn, ok := origBatch.TxIndToBlocks[uint(tx[0])]
		require.True(t, ok)
		return bn
	}

	checkBatch := func(b *executordto.Batch, expectedTxs []int) {
		require.NotNil(t, b)
		require.Equal(t, len(expectedTxs), len(b.Txs))

		lb1 := getBlockForTx(b.Txs[len(b.Txs)-1])
		lb2 := b.TxIndToBlocks[uint(b.Txs[len(b.Txs)-1][0])]
		require.Equal(t, lb1, lb2)

		for i, txContent := range expectedTxs {
			require.Equal(t, byte(txContent), b.Txs[i][0])
		}
	}

	testCases := []struct {
		executedTxCount, lastSuccessCount, lastErrorCount int
		expected                                          []int
	}{
		// 0 sent, get half
		{
			0, 0, 0,
			[]int{0, 1, 2, 3, 4},
		},
		// impossible situation, but still
		// 2 have already been sent; !5 successfully sent in the last battle, get the next 5
		{
			2, 5, 0,
			[]int{2, 3, 4, 5, 6},
		},
		// 5 didn't go through; divide those that didn't go through yet, get 2.
		{
			0, 0, 5,
			[]int{0, 1},
		},
		// 5 sent, 5 didn't get through; divide the remaining batch more, get 3; 3 < 5 - leave 3
		{
			5, 0, 5,
			[]int{5, 6, 7},
		},
		// 9 sent, 2 didn't go through; divide those that haven't gone through yet, get 1
		{
			9, 0, 2,
			[]int{9},
		},
	}

	for _, tc := range testCases {
		name := fmt.Sprintf("exec:%d success:%d error:%d",
			tc.executedTxCount, tc.lastSuccessCount, tc.lastErrorCount)
		t.Run(name, func(t *testing.T) {
			b := splitBatchForExec(tc.executedTxCount, tc.lastSuccessCount, tc.lastErrorCount, origBatch)
			checkBatch(b, tc.expected)
		})
	}
}

func TestExecWithSplitHlpNormal(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	exs := &executorStub{
		maxBatchSize: 0,
	}

	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	origBatch := generateOrigBatch()
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.NoError(t, err)

	checkExecAttempts(t, exs, []execAttempt{{
		batchSize:           len(origBatch.Txs),
		isError:             false,
		isSizeExceededError: false,
	}})
}

func TestExecWithSplitHlpExceededError(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)
	exs := &executorStub{
		maxBatchSize: 3,
	}

	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	origBatch := generateOrigBatch()
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.NoError(t, err)

	checkExecAttempts(t, exs, []execAttempt{
		{batchSize: len(origBatch.Txs), isError: true, isSizeExceededError: true},
		{batchSize: 5, isError: true, isSizeExceededError: true},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 1, isError: false, isSizeExceededError: false},
	})
}

func TestExecWithSplitHlpError(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	exs := &executorStub{
		maxBatchSize: 3,
	}
	testErr := errors.New("test error")
	exs.callHlp.AddErrMap(exs.executeBatch, map[int]error{5: testErr})

	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	origBatch := generateOrigBatch()
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.Error(t, err)
	require.ErrorIs(t, err, testErr)

	checkExecAttempts(t, exs, []execAttempt{
		{batchSize: len(origBatch.Txs), isError: true, isSizeExceededError: true},
		{batchSize: 5, isError: true, isSizeExceededError: true},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: true, isSizeExceededError: false},
	})
}

type execAttempt struct {
	batchSize           int
	isError             bool
	isSizeExceededError bool
}

type executorStub struct {
	callHlp      testshlp.CallHlp
	maxBatchSize uint

	attempts []execAttempt
}

func checkExecAttempts(t *testing.T, ex *executorStub, expectedAttempts []execAttempt) {
	require.Equal(t, len(expectedAttempts), len(ex.attempts))
	for i, exp := range expectedAttempts {
		require.Equal(t, exp, ex.attempts[i])
	}
}

func (ex *executorStub) executeBatch(_ context.Context, b *executordto.Batch) (uint64, error) {
	if err := ex.callHlp.Call(ex.executeBatch); err != nil {
		ex.attempts = append(ex.attempts, execAttempt{batchSize: len(b.Txs), isError: true})
		return 0, err
	}
	if ex.maxBatchSize > 0 && len(b.Txs) > int(ex.maxBatchSize) {
		ex.attempts = append(ex.attempts, execAttempt{batchSize: len(b.Txs), isError: true, isSizeExceededError: true})
		return 0, errors.New(errReqSizeMarker)
	}

	ex.attempts = append(ex.attempts, execAttempt{batchSize: len(b.Txs)})
	return b.TxIndToBlocks[uint(len(b.Txs))-1] + 1, nil
}

func generateOrigBatch() *executordto.Batch {
	return &executordto.Batch{
		Txs: [][]byte{
			{0, 0, 0},
			{1, 1, 1},
			{2, 2, 2},
			{3, 3, 3},
			{4, 4, 4},
			{5, 5, 5},
			{6, 6, 6},
			{7, 7, 7},
			{8, 8, 8},
			{9, 9, 9},
			{10, 10, 10},
		},
		TxIndToBlocks: map[uint]uint64{
			0: 0, 1: 0, 2: 0,
			3: 1, 4: 1, 5: 1,
			6: 2, 7: 2,
			8: 3,
			9: 4, 10: 4,
		},
	}
}
