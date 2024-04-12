package hlf

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/anoideaopen/common-component/testshlp"
	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/robot/dto/executordto"
	"github.com/stretchr/testify/require"
)

func TestSplitBatchForExec(t *testing.T) {
	origBatch := generateOrigBatch()
	_, l := testshlp.CreateCtxLogger(t)

	getBlockForTx := func(tx []byte) uint64 {
		bn, ok := origBatch.TxIndToBlocks[uint(tx[0])]
		require.True(t, ok)
		return bn
	}

	checkBatch := func(b *executordto.Batch, expectedTxs []int) {
		require.NotNil(t, b)
		require.Equal(t, len(expectedTxs), len(b.Txs))

		lb1 := getBlockForTx(b.Txs[len(b.Txs)-1])
		lb2 := b.TxIndToBlocks[uint(b.Txs[len(b.Txs)-1][0])] // ToDo to see if that's right?
		require.Equal(t, lb1, lb2)

		for i, txContent := range expectedTxs {
			require.Equal(t, byte(txContent), b.Txs[i][0])
		}
	}

	testCases := []struct {
		executedTxCount, lastSuccessCount, lastErrorCount int
		expected                                          []int
	}{
		// 0 sent, we're getting half
		{
			0, 0, 0,
			[]int{0, 1, 2, 3, 4},
		},
		// an impossible situation, but still
		// 2 already sent; !5 successfully sent in the last batch, get the next 5
		{
			2, 5, 0,
			[]int{2, 3, 4, 5, 6},
		},
		// 5 didn't fit; divide the ones that haven't fit yet, we get 2.
		{
			0, 0, 5,
			[]int{0, 1},
		},
		// 5 sent, 5 didn't get through; divide the remaining batch more, get 3; 3 < 5 - leave 3
		{
			5, 0, 5,
			[]int{5, 6, 7},
		},
		// 9 sent, 2 didn't go through; divide those that haven't gone through yet, get 1.
		{
			9, 0, 2,
			[]int{9},
		},
	}

	for _, tc := range testCases {
		name := fmt.Sprintf("exec:%d success:%d error:%d",
			tc.executedTxCount, tc.lastSuccessCount, tc.lastErrorCount)
		t.Run(name, func(t *testing.T) {
			b := splitBatchForExec(tc.executedTxCount, tc.lastSuccessCount, tc.lastErrorCount, origBatch, l)
			checkBatch(b, tc.expected)
		})
	}
}

// TestExecWithSplitHlpNormal - check that the transactions fit into the batches and were executed without errors
func TestExecWithSplitHlpNormal(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	// set unlimited for the batch
	exs := &executorStub{
		maxBatchSize: 0,
	}

	// pass the test function executeBatch, which will further calculate
	// the result of the execution of the batches and network
	// the results of the execution into the test dump.
	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	// take a pre-created batch
	origBatch := generateOrigBatch()
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.NoError(t, err)

	checkExecAttempts(t, exs, []execAttempt{{
		batchSize:           len(origBatch.Txs),
		isError:             false,
		isSizeExceededError: false,
	}})
}

// TestExecWithSplitHlpExceededError - in the test we create a battle with 11 transactions and
// the maximum size of the battle is 3. when trying to execute, we get an isSizeExceededError and
// split the batch to 5 transactions, after trying to execute again we get an isSizeExceededError and
// split the batch to 2 transactions, then execute all executeBatch without errors
func TestExecWithSplitHlpExceededError(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	// set the limit of 3 transactions for the batch
	exs := &executorStub{
		maxBatchSize: 3,
	}

	// pass the test function executeBatch, which will further calculate
	// the result of the execution of the batches and network
	// the results of the execution into the test dump.
	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	// take a pre-created batch
	origBatch := generateOrigBatch()
	// call the function under test execute
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.NoError(t, err)

	checkExecAttempts(t, exs, []execAttempt{
		// check that 11 transactions result in a split.
		{batchSize: len(origBatch.Txs), isError: true, isSizeExceededError: true},
		// check that at 5 transactions the split occurs
		{batchSize: 5, isError: true, isSizeExceededError: true},
		// then we check that transactions are running without errors, 2 in batches.
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 1, isError: false, isSizeExceededError: false},
	})
}

// TestExecWithSplitHlpError - in the test, we create a battle with 11 transactions and
// the maximum size of the battle is 3 when trying to execute we get isSizeExceededError and
// split the batch to 5 transactions, after trying to execute again we get isSizeExceededError and
// split the batch to 2 transactions, then without errors we go to
// the 6th call of executeBatch where we get an error different from isSizeExceededError after which
// the robot crashes with an error.
func TestExecWithSplitHlpError(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	// maximum batch size is set 3
	exs := &executorStub{
		maxBatchSize: 3,
	}

	// add a custom error to executeBatch return on execution 6
	testErr := errors.New("test error")
	exs.callHlp.AddErrMap(exs.executeBatch, map[int]error{5: testErr})

	// pass the test function executeBatch, which will further calculate the result of
	// the execution of the batches and network the results of the execution into the test dump.
	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	// take a pre-created batch
	origBatch := generateOrigBatch()
	// call the function under test execute
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.Error(t, err)
	require.ErrorIs(t, err, testErr)

	checkExecAttempts(t, exs, []execAttempt{
		{batchSize: len(origBatch.Txs), isError: true, isSizeExceededError: true},
		{batchSize: 5, isError: true, isSizeExceededError: true},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		// check that at the 6th call of executeBatch we get the expected error
		{batchSize: 2, isError: true, isSizeExceededError: false},
	})
}

// TestExecBatcReqSizeErrorAfterSplit - check that after dividing the batches
// up to 1 transaction we get an error if the size of the transaction does not pass into the batches.
func TestExecBatcReqSizeErrorAfterSplit(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	// the maximum batch size is set to 3
	exs := &executorStub{
		maxBatchSize: 3,
	}

	// add size error to executeBatch return at 8 execution
	testErr := errors.New(errReqSizeMarker)
	exs.callHlp.AddErrMap(exs.executeBatch, map[int]error{7: testErr})

	// pass the test function executeBatch, which will further calculate
	// the result of the execution of the batches and network the results
	// of the execution into the test dump.
	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	// take a pre-created batch
	origBatch := generateOrigBatch()
	// call the function under test execute
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.Error(t, err)
	require.ErrorIs(t, err, testErr)

	checkExecAttempts(t, exs, []execAttempt{
		{batchSize: len(origBatch.Txs), isError: true, isSizeExceededError: true},
		{batchSize: 5, isError: true, isSizeExceededError: true},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		// check that at 8 calls of executeBatch we get the expected error
		{batchSize: 1, isError: true, isSizeExceededError: true},
	})
}

// TestExecBatchWithNoReqSizeError - check that if not all transactions get into
// the batches, in case of an error other than errReqSizeMarker the robot crashes immediately
// and does not split transactions
func TestExecBatchWithNoReqSizeError(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	// the maximum batch size is set to 3
	exs := &executorStub{
		maxBatchSize: 3,
	}

	// add custom error to executeBatch return at 1 execution
	testErr := errors.New("test error")
	exs.callHlp.AddErrMap(exs.executeBatch, map[int]error{0: testErr})

	// pass the test function executeBatch, which will further calculate the result of
	// the execution of the batches and network the results of the execution into the test dump.
	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	// take a pre-created batch
	origBatch := generateOrigBatch()
	// call the function under test execute
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.Error(t, err)
	require.ErrorIs(t, err, testErr)

	checkExecAttempts(t, exs, []execAttempt{
		// check that at 1 call of executeBatch we got a custom error
		{batchSize: len(origBatch.Txs), isError: true, isSizeExceededError: false},
	})
}

// TestExecBatchWithNoReqSizeErrorOneTransaction - check that if all transactions are in the batch,
// the robot crashes immediately if an error other than errReqSizeMarker occurs.
func TestExecBatchWithNoReqSizeErrorOneTransaction(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	// unlimited batch
	exs := &executorStub{
		maxBatchSize: 0,
	}
	// add an error to the return of executeBatch at 1 execution
	testErr := errors.New("test error")
	exs.callHlp.AddErrMap(exs.executeBatch, map[int]error{0: testErr})

	// pass the test function executeBatch, which will further calculate the result of
	// the execution of the batches and network the results of the execution into the test dump.
	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	// take a pre-created batch
	origBatch := generateOrigBatch()
	// call the function under test execute
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.Error(t, err)
	require.ErrorIs(t, err, testErr)

	checkExecAttempts(t, exs, []execAttempt{
		// check that at 1 call of executeBatch we got a custom error
		{batchSize: len(origBatch.Txs), isError: true, isSizeExceededError: false},
	})
}

// TestExecWithSplitHlpExceededErrorThenExecuteSwaps - in the test, we create a batches with 11 transactions
// and 2 swaps and the maximum size of the batches is 3 when trying to execute we get
// an isSizeExceededError error and split the batch to 5 transactions, after trying to execute again
// we get an isSizeExceededError error and split the batch to 2 transactions,
// then without errors execute all executeBatch and then check
func TestExecWithSplitHlpExceededErrorThenExecuteSwaps(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	// set the limit of 3 transactions for the batch
	exs := &executorStub{
		maxBatchSize: 3,
	}

	// pass the test function executeBatch, which will further calculate the result of
	// the execution of the batches and network the results of the execution into the test dump.
	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	// we take a pre-created batches with 11 transactions and 2 swaps
	origBatch := generateBatchWithTxsAndSwaps()
	// call the function under test execute
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.NoError(t, err)

	checkExecAttempts(t, exs, []execAttempt{
		// check that 11 transactions result in a split.
		{batchSize: len(origBatch.Txs), isError: true, isSizeExceededError: true},
		// check that at 5 transactions the split occurs
		{batchSize: 5, isError: true, isSizeExceededError: true},
		// then we check that transactions are running without errors, 2 in batches.
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 1, isError: false, isSizeExceededError: false},
		// check that the swaps have been executed
		{batchSize: 0, isError: false, isSizeExceededError: false},
	})
}

// TestExecWithSwaps - test for executing a swap-only batch
func TestExecWithSwaps(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	// set the limit of 3 transactions for the batch
	exs := &executorStub{
		maxBatchSize: 3,
	}

	// pass the test function executeBatch, which will further calculate
	// the result of the execution of the batches and network the results of the execution into the test dump.
	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	// we take a pre-created batch with 2 swaps
	origBatch := generateBatchWithTxsAndSwaps()
	// call the function under test execute
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.NoError(t, err)

	checkExecAttempts(t, exs, []execAttempt{
		// check that 11 transactions result in a split.
		{batchSize: len(origBatch.Txs), isError: true, isSizeExceededError: true},
		// check that at 5 transactions the split occurs
		{batchSize: 5, isError: true, isSizeExceededError: true},
		// then we check that transactions are running without errors, 2 in batches.
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 1, isError: false, isSizeExceededError: false},
		// check that the swaps have been executed
		{batchSize: 0, isError: false, isSizeExceededError: false},
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
	// call executeBatch through the call helper, which looks into errorsMap by the number of
	// the executeBatch call and determines what values should be returned.
	// if we did not put an error into the map example:
	// exs.callHlp.AddErrMap(exs.executeBatch, map[int]error{0: testErr}) it will return nil.
	err := ex.callHlp.Call(ex.executeBatch)
	if err != nil {
		// if an error came from executeBatch, check for error reqSizeExceededErr
		ex.attempts = append(ex.attempts, execAttempt{batchSize: len(b.Txs), isError: true, isSizeExceededError: isOrderingReqSizeExceededErr(err)})
		return 0, err
		// If another error occurs, just set error true
	}
	// if there is no error, but the length of the batches is greater
	// than the maximum, we net the error reqSizeExceededErr.
	if ex.maxBatchSize > 0 && len(b.Txs) > int(ex.maxBatchSize) {
		ex.attempts = append(ex.attempts, execAttempt{batchSize: len(b.Txs), isError: true, isSizeExceededError: true})
		return 0, errors.New(errReqSizeMarker)
	}

	// add the attempt to the array with attempts
	ex.attempts = append(ex.attempts, execAttempt{batchSize: len(b.Txs)})
	return b.TxIndToBlocks[uint(len(b.Txs))-1] + 1, nil // ToDo double-check to see if that's right?
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

func generateBatchWithTxsAndSwaps() *executordto.Batch {
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
		Swaps: []*proto.Swap{{Id: []byte("sw1")}, {Id: []byte("sw2")}},
	}
}
