package hlf

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anoideaopen/common-component/testshlp"
	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/robot/dto/executordto"
	"github.com/anoideaopen/robot/metrics"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	"github.com/stretchr/testify/require"
)

type mockExecutor struct {
	callsCount           int
	unRecoverableCallNum int
	successCallNum       int
}

func createMockExecutor(unRecoverableCallNum, successCallNum int) *mockExecutor {
	return &mockExecutor{
		unRecoverableCallNum: unRecoverableCallNum,
		successCallNum:       successCallNum,
	}
}

func (mex *mockExecutor) execute(_ context.Context, _ channel.Request) (channel.Response, error) {
	mex.callsCount++

	switch mex.callsCount {
	case mex.successCallNum:
		return channel.Response{BlockNumber: uint64(mex.callsCount)}, nil
	case mex.unRecoverableCallNum:
		return channel.Response{}, errors.New("unRecoverable error")
	default:
		return channel.Response{}, status.New(status.EndorserClientStatus, status.EndorsementMismatch.ToInt32(),
			"ProposalResponsePayloads do not match", nil)
	}
}

func TestChExecutorRetryExecute(t *testing.T) {
	_, l := testshlp.CreateCtxLogger(t)

	che := &chExecutor{
		log:                  l,
		m:                    metrics.FromContext(context.Background()),
		chName:               "fiat",
		retryExecuteMaxDelay: time.Millisecond,
	}

	t.Run("[POSITIVE] success retry after mismatches", func(t *testing.T) {
		successCallNum := 5
		unRecoverableCallNum := successCallNum + 1
		che.retryExecuteAttempts = uint(successCallNum)

		chc := createMockExecutor(unRecoverableCallNum, successCallNum)
		che.executor = chc

		nnn, err := che.Execute(context.Background(), &executordto.Batch{}, 0)
		require.NoError(t, err)
		require.Equal(t, successCallNum, chc.callsCount)
		require.Equal(t, nnn, uint64(successCallNum))
	})

	t.Run("[NEGATIVE] attempts exceeded", func(t *testing.T) {
		successCallNum := 5
		unRecoverableCallNum := successCallNum + 1
		che.retryExecuteAttempts = uint(successCallNum - 1)

		chc := createMockExecutor(unRecoverableCallNum, successCallNum)
		che.executor = chc

		_, err := che.Execute(context.Background(), &executordto.Batch{}, 0)

		require.Error(t, err)
		require.Equal(t, successCallNum-1, chc.callsCount)
	})

	t.Run("[NEGATIVE] unrecoverable error during retry", func(t *testing.T) {
		successCallNum := 5
		unRecoverableCallNum := successCallNum - 1
		che.retryExecuteAttempts = uint(successCallNum - 1)

		chc := createMockExecutor(unRecoverableCallNum, successCallNum)
		che.executor = chc

		_, err := che.Execute(context.Background(), &executordto.Batch{}, 0)

		require.Error(t, err)
		require.Equal(t, unRecoverableCallNum, chc.callsCount)
	})
}

func TestLogBatchContent(t *testing.T) {
	_, log := testshlp.CreateCtxLogger(t)

	b := &executordto.Batch{}
	logBatchContent(log, b)

	b2 := &executordto.Batch{
		Txs:        [][]byte{[]byte("tx1"), []byte("tx2"), []byte("tx3")},
		Swaps:      []*proto.Swap{{Id: []byte("sw1")}, {Id: []byte("sw2")}},
		MultiSwaps: []*proto.MultiSwap{{Id: []byte("msw1")}, {Id: []byte("msw2")}},
		Keys:       []*proto.SwapKey{{Id: []byte("ksw1")}, {Id: []byte("ksw2")}},
		MultiKeys:  []*proto.SwapKey{{Id: []byte("kmsw1")}, {Id: []byte("kmsw2")}},
	}
	logBatchContent(log, b2)
}
