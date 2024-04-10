//go:build !integration
// +build !integration

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
		lb2 := b.TxIndToBlocks[uint(b.Txs[len(b.Txs)-1][0])] // ToDo проверить, верно ли это?
		require.Equal(t, lb1, lb2)

		for i, txContent := range expectedTxs {
			require.Equal(t, byte(txContent), b.Txs[i][0])
		}
	}

	testCases := []struct {
		executedTxCount, lastSuccessCount, lastErrorCount int
		expected                                          []int
	}{
		// 0 отправлено, получаем половину
		{
			0, 0, 0,
			[]int{0, 1, 2, 3, 4},
		},
		// невозможная ситуация, но все же
		// 2 уже отправлены; !5 успешно отправлено в прошлом батче, получаем следующие 5
		{
			2, 5, 0,
			[]int{2, 3, 4, 5, 6},
		},
		// 5 не пролезло; делим те что не пролезли еще, получаем 2
		{
			0, 0, 5,
			[]int{0, 1},
		},
		// 5 отправлены, 5 не пролезло; делим оставшийся батч еще, получаем 3; 3 < 5 - оставляем 3
		{
			5, 0, 5,
			[]int{5, 6, 7},
		},
		// 9 отправлено, 2 не пролезло; делим те что не пролезли еще, получаем 1
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

// TestExecWithSplitHlpNormal - проверка на то что транзакции влезли в батч и выполнились без ошибок
func TestExecWithSplitHlpNormal(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	// устанавливаем безлимит для батча
	exs := &executorStub{
		maxBatchSize: 0,
	}

	// передаем тестовую функцию executeBatch которая в дальнейшем будет высчитывать результат выполнения батча и сетить результаты выполнения в тестовую мапу
	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	// берем заранее созданный батч
	origBatch := generateOrigBatch()
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.NoError(t, err)

	checkExecAttempts(t, exs, []execAttempt{{
		batchSize:           len(origBatch.Txs),
		isError:             false,
		isSizeExceededError: false,
	}})
}

// TestExecWithSplitHlpExceededError - в тесте создается батч с 11 транзакциями и максимальным размером батча 3
// при попытке выполнения получаем ошибку isSizeExceededError и сплитим батч до 5 транзакций, после попытки выполнения
// снова получаем ошибку isSizeExceededError и сплитим батч до 2 транзакций, далее без ошибок выполняем все executeBatch
func TestExecWithSplitHlpExceededError(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	// устанавливаем лимит 3 транзакции для батча
	exs := &executorStub{
		maxBatchSize: 3,
	}

	// передаем тестовую функцию executeBatch которая в дальнейшем будет высчитывать результат выполнения батча и сетить результаты выполнения в тестовую мапу
	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	// берем заранее созданный батч
	origBatch := generateOrigBatch()
	// вызываем тестируемую функцию execute
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.NoError(t, err)

	checkExecAttempts(t, exs, []execAttempt{
		// проверяем что при 11 транзакциях происходит сплит
		{batchSize: len(origBatch.Txs), isError: true, isSizeExceededError: true},
		// проверяем что при 5 транзакциях происходит сплит
		{batchSize: 5, isError: true, isSizeExceededError: true},
		// далее проверяем что транзакции идут без ошибок по 2 в батче
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 1, isError: false, isSizeExceededError: false},
	})
}

// TestExecWithSplitHlpError - в тесте создается батч с 11 транзакциями и максимальным размером батча 3
// при попытке выполнения получаем ошибку isSizeExceededError и сплитим батч до 5 транзакций, после попытки выполнения
// снова получаем ошибку isSizeExceededError и сплитим батч до 2 транзакций, далее без ошибок идем до 6го вызова executeBatch
// где получаем ошибку отличную от isSizeExceededError после чего робот падает с ошибкой
func TestExecWithSplitHlpError(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	// задан максимальный размер батча 3
	exs := &executorStub{
		maxBatchSize: 3,
	}

	// добавляем кастомную ошибку в возврат executeBatch при 6 выполнении
	testErr := errors.New("test error")
	exs.callHlp.AddErrMap(exs.executeBatch, map[int]error{5: testErr})

	// передаем тестовую функцию executeBatch которая в дальнейшем будет высчитывать результат выполнения батча и сетить результаты выполнения в тестовую мапу
	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	// берем заранее созданный батч
	origBatch := generateOrigBatch()
	// вызываем тестируемую функцию execute
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.Error(t, err)
	require.ErrorIs(t, err, testErr)

	checkExecAttempts(t, exs, []execAttempt{
		{batchSize: len(origBatch.Txs), isError: true, isSizeExceededError: true},
		{batchSize: 5, isError: true, isSizeExceededError: true},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		// проверяем что при 6 вызове executeBatch получаем ожидаемую ошибку
		{batchSize: 2, isError: true, isSizeExceededError: false},
	})
}

// TestExecBatcReqSizeErrorAfterSplit - проверка что после деления батча до 1 тразакции мы получаем ошибку если ее размер не проходит в батч
func TestExecBatcReqSizeErrorAfterSplit(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	// задан максимальный размер батча 3
	exs := &executorStub{
		maxBatchSize: 3,
	}

	// добавляем ошибку размера в возврат executeBatch при 8 выполнении
	testErr := errors.New(errReqSizeMarker)
	exs.callHlp.AddErrMap(exs.executeBatch, map[int]error{7: testErr})

	// передаем тестовую функцию executeBatch которая в дальнейшем будет высчитывать результат выполнения батча и сетить результаты выполнения в тестовую мапу
	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	// берем заранее созданный батч
	origBatch := generateOrigBatch()
	// вызываем тестируемую функцию execute
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
		// проверяем что при 8 вызове executeBatch получаем ожидаемую ошибку
		{batchSize: 1, isError: true, isSizeExceededError: true},
	})
}

// TestExecBatchWithNoReqSizeError - проверка на то что если не все транзакции попадают в батч, при ошибке отличной от errReqSizeMarker робот сразу падает и не сплитит транзакции
func TestExecBatchWithNoReqSizeError(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	// задан максимальный размер батча 3
	exs := &executorStub{
		maxBatchSize: 3,
	}

	// добавляем кастомную ошибку в возврат executeBatch при 1 выполнении
	testErr := errors.New("test error")
	exs.callHlp.AddErrMap(exs.executeBatch, map[int]error{0: testErr})

	// передаем тестовую функцию executeBatch которая в дальнейшем будет высчитывать результат выполнения батча и сетить результаты выполнения в тестовую мапу
	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	// берем заранее созданный батч
	origBatch := generateOrigBatch()
	// вызываем тестируемую функцию execute
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.Error(t, err)
	require.ErrorIs(t, err, testErr)

	checkExecAttempts(t, exs, []execAttempt{
		// проверяем что при 1 вызове executeBatch получили кастомную ошибку
		{batchSize: len(origBatch.Txs), isError: true, isSizeExceededError: false},
	})
}

// TestExecBatchWithNoReqSizeErrorOneTransaction - проверка на то что если все транзакции попадают в батч, при ошибке отличной от errReqSizeMarker робот сразу падает
func TestExecBatchWithNoReqSizeErrorOneTransaction(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	// задаем безлимитный батч
	exs := &executorStub{
		maxBatchSize: 0,
	}
	// добавляем ошибку в возврат executeBatch при 1 выполнении
	testErr := errors.New("test error")
	exs.callHlp.AddErrMap(exs.executeBatch, map[int]error{0: testErr})

	// передаем тестовую функцию executeBatch которая в дальнейшем будет высчитывать результат выполнения батча и сетить результаты выполнения в тестовую мапу
	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	// берем заранее созданный батч
	origBatch := generateOrigBatch()
	// вызываем тестируемую функцию execute
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.Error(t, err)
	require.ErrorIs(t, err, testErr)

	checkExecAttempts(t, exs, []execAttempt{
		// проверяем что при 1 вызове executeBatch получили кастомную ошибку
		{batchSize: len(origBatch.Txs), isError: true, isSizeExceededError: false},
	})
}

// TestExecWithSplitHlpExceededErrorThenExecuteSwaps - в тесте создается батч с 11 транзакциями и 2 свопами и максимальным размером батча 3
// при попытке выполнения получаем ошибку isSizeExceededError и сплитим батч до 5 транзакций, после попытки выполнения
// снова получаем ошибку isSizeExceededError и сплитим батч до 2 транзакций, далее без ошибок выполняем все executeBatch
// далее проверяем
func TestExecWithSplitHlpExceededErrorThenExecuteSwaps(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	// устанавливаем лимит 3 транзакции для батча
	exs := &executorStub{
		maxBatchSize: 3,
	}

	// передаем тестовую функцию executeBatch которая в дальнейшем будет высчитывать результат выполнения батча и сетить результаты выполнения в тестовую мапу
	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	// берем заранее созданный батч с 11 транзакциями и 2 свопами
	origBatch := generateBatchWithTxsAndSwaps()
	// вызываем тестируемую функцию execute
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.NoError(t, err)

	checkExecAttempts(t, exs, []execAttempt{
		// проверяем что при 11 транзакциях происходит сплит
		{batchSize: len(origBatch.Txs), isError: true, isSizeExceededError: true},
		// проверяем что при 5 транзакциях происходит сплит
		{batchSize: 5, isError: true, isSizeExceededError: true},
		// далее проверяем что транзакции идут без ошибок по 2 в батче
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 1, isError: false, isSizeExceededError: false},
		// проверяем что свопы выполнились
		{batchSize: 0, isError: false, isSizeExceededError: false},
	})
}

// TestExecWithSwaps - тест на выполнение батча только со свопами
func TestExecWithSwaps(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	// устанавливаем лимит 3 транзакции для батча
	exs := &executorStub{
		maxBatchSize: 3,
	}

	// передаем тестовую функцию executeBatch которая в дальнейшем будет высчитывать результат выполнения батча и сетить результаты выполнения в тестовую мапу
	ewsHlp := newExecWithSplitHlp(l, exs.executeBatch, nil)

	// берем заранее созданный батч с 2 свопами
	origBatch := generateBatchWithTxsAndSwaps()
	// вызываем тестируемую функцию execute
	_, err := ewsHlp.execute(context.Background(), origBatch)
	require.NoError(t, err)

	checkExecAttempts(t, exs, []execAttempt{
		// проверяем что при 11 транзакциях происходит сплит
		{batchSize: len(origBatch.Txs), isError: true, isSizeExceededError: true},
		// проверяем что при 5 транзакциях происходит сплит
		{batchSize: 5, isError: true, isSizeExceededError: true},
		// далее проверяем что транзакции идут без ошибок по 2 в батче
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 2, isError: false, isSizeExceededError: false},
		{batchSize: 1, isError: false, isSizeExceededError: false},
		// проверяем что свопы выполнились
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
	// вызываем executeBatch через call helper который по номеру вызова executeBatch смотрит в errorsMap и определяет какие значения нужно вернуть
	// если мы не внесли ошибку в мапу пример: exs.callHlp.AddErrMap(exs.executeBatch, map[int]error{0: testErr}) то он вернет nil
	err := ex.callHlp.Call(ex.executeBatch)
	if err != nil {
		// если из executeBatch пришла ошибка, проверяем на ошибку reqSizeExceededErr
		ex.attempts = append(ex.attempts, execAttempt{batchSize: len(b.Txs), isError: true, isSizeExceededError: isOrderingReqSizeExceededErr(err)})
		return 0, err
		// если другая ошибка просто выставляем error true
	}
	// если ошибки нет но длина батча больше макимальной сетим ошибку reqSizeExceededErr
	if ex.maxBatchSize > 0 && len(b.Txs) > int(ex.maxBatchSize) {
		ex.attempts = append(ex.attempts, execAttempt{batchSize: len(b.Txs), isError: true, isSizeExceededError: true})
		return 0, errors.New(errReqSizeMarker)
	}

	// добавляем попытку в массив с попытками
	ex.attempts = append(ex.attempts, execAttempt{batchSize: len(b.Txs)})
	return b.TxIndToBlocks[uint(len(b.Txs))-1] + 1, nil // ToDo перепроверить, верно ли это?
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
