//go:build !integration
// +build !integration

//nolint:all
package chrobot

import (
	"context"
	"encoding/binary"
	"errors"
	"log"
	"testing"
	"time"

	"github.com/atomyze-foundation/common-component/errorshlp"
	"github.com/atomyze-foundation/common-component/testshlp"
	"github.com/atomyze-foundation/foundation/proto"
	"github.com/atomyze-foundation/robot/collectorbatch"
	"github.com/atomyze-foundation/robot/dto/collectordto"
	"github.com/atomyze-foundation/robot/dto/executordto"
	"github.com/atomyze-foundation/robot/helpers/nerrors"
	"github.com/stretchr/testify/require"
)

func TestCreate(t *testing.T) {
	t.Parallel()

	ctx, _ := testshlp.CreateCtxLogger(t)

	stor := &stubStor{}
	cr := &stubCollectorCr{}
	chName := "mych"
	chExecutorCr := newStubChExecutorCreator(chName)

	chRobot := NewRobot(ctx, chName, 10, map[string]uint64{},
		func(ctx context.Context, dataReady chan<- struct{}, srcChName string, startFrom uint64) (ChCollector, error) {
			return cr.create(ctx, dataReady, srcChName, startFrom, 1)
		},
		func(ctx context.Context) (ChExecutor, error) {
			return chExecutorCr.create(ctx)
		}, stor, collectorbatch.Limits{LenLimit: 1, SizeLimit: 1, TimeoutLimit: time.Second})

	require.NotNil(t, chRobot)
}

func TestStartStop(t *testing.T) {
	t.Parallel()

	ctx, cancel, _ := testshlp.CreateCtxLoggerWithCancel(t)

	stor := &stubStor{}
	cr := &stubCollectorCr{}
	chName := "mych"
	chExecutorCr := newStubChExecutorCreator(chName)

	chRobot := NewRobot(ctx, chName, 10, map[string]uint64{},
		func(ctx context.Context, dataReady chan<- struct{}, srcChName string, startFrom uint64) (ChCollector, error) {
			return cr.create(ctx, dataReady, srcChName, startFrom, 1)
		},
		func(ctx context.Context) (ChExecutor, error) {
			return chExecutorCr.create(ctx)
		},
		stor, collectorbatch.Limits{LenLimit: 1, SizeLimit: 1, TimeoutLimit: time.Second})
	require.NotNil(t, chRobot)

	go func() {
		<-time.After(time.Second * 2)
		cancel()
	}()

	err := chRobot.Run(ctx)
	require.Error(t, err)
	require.True(t, ctx.Err() != nil && errors.Is(err, context.Canceled))
}

func TestWorkWithOwnCh(t *testing.T) {
	t.Parallel()

	const chName, startBlock, countBlocks, countTxOrSwaps = "mych", 100, 10, 3
	ctx, cancel, _ := testshlp.CreateCtxLoggerWithCancel(t)

	stor := &stubStor{}
	cr := &stubCollectorCr{
		channelsData: map[string][]*collectordto.BlockData{
			chName: createSrcData(chName, chName, startBlock, countBlocks, countTxOrSwaps, 0),
		},
	}
	chExecutorCr := newStubChExecutorCreator(chName)

	chRobot := NewRobot(ctx, chName, 10,
		map[string]uint64{chName: 0},
		func(ctx context.Context, dataReady chan<- struct{}, srcChName string, startFrom uint64) (ChCollector, error) {
			return cr.create(ctx, dataReady, srcChName, startFrom, 1)
		},
		func(ctx context.Context) (ChExecutor, error) {
			return chExecutorCr.create(ctx)
		},
		stor, collectorbatch.Limits{LenLimit: countTxOrSwaps, TimeoutLimit: time.Second})
	require.NotNil(t, chRobot)

	go func() {
		for {
			<-time.After(time.Second * 1)
			chExecutorCr.checkSavedState(func(
				maxTxBlockNum uint64,
				batches []*executordto.Batch,
				allTxs map[uint32][]byte,
				allSwaps map[*proto.Swap]struct{},
			) {
				if maxTxBlockNum == startBlock+countBlocks-1 &&
					len(allTxs) == countBlocks*countTxOrSwaps &&
					len(allSwaps) == 0 {
					cancel()
					return
				}
			})
		}
	}()

	err := chRobot.Run(ctx)
	require.Error(t, err)
	require.Truef(t, ctx.Err() != nil && errors.Is(err, context.Canceled), "err: %s", err)
}

func TestWorkWithManyCh(t *testing.T) {
	t.Parallel()

	const ch1, ch2, ch3 = "ch1", "ch2", "ch3"
	const startBlock1, countBlocks1, countTxOrSwaps1 = 100, 10, 3
	const startBlock2, countBlocks2, countTxOrSwaps2 = 200, 5, 2
	const startBlock3, countBlocks3, countTxOrSwaps3 = 300, 3, 2

	ctx, cancel, _ := testshlp.CreateCtxLoggerWithCancel(t)

	stor := &stubStor{}
	cr := &stubCollectorCr{
		channelsData: map[string][]*collectordto.BlockData{
			ch1: createSrcData(ch1, ch1, startBlock1, countBlocks1, countTxOrSwaps1, 0),
			ch2: createSrcData(ch1, ch2, startBlock2, countBlocks2, countTxOrSwaps2, 0),
			ch3: createSrcData(ch1, ch3, startBlock3, countBlocks3, countTxOrSwaps3, 0),
		},
	}
	chExecutorCr := newStubChExecutorCreator(ch1)

	chRobot := NewRobot(ctx, ch1, 10,
		map[string]uint64{ch1: 0, ch2: 0, ch3: 0},
		func(ctx context.Context, dataReady chan<- struct{}, srcChName string, startFrom uint64) (ChCollector, error) {
			return cr.create(ctx, dataReady, srcChName, startFrom, 1)
		},
		func(ctx context.Context) (ChExecutor, error) {
			return chExecutorCr.create(ctx)
		},
		stor, collectorbatch.Limits{LenLimit: countTxOrSwaps1, TimeoutLimit: time.Second})
	require.NotNil(t, chRobot)

	go func() {
		for {
			<-time.After(time.Second * 1)
			chExecutorCr.checkSavedState(func(
				maxTxBlockNum uint64,
				batches []*executordto.Batch,
				allTxs map[uint32][]byte,
				allSwaps map[*proto.Swap]struct{},
			) {
				if maxTxBlockNum == startBlock1+countBlocks1-1 &&
					len(allTxs) == countBlocks1*countTxOrSwaps1 &&
					len(allSwaps) == countBlocks2*countTxOrSwaps2+countBlocks3*countTxOrSwaps3 {
					cancel()
					return
				}
			})
		}
	}()

	err := chRobot.Run(ctx)
	require.Error(t, err)
	require.Truef(t, ctx.Err() != nil && errors.Is(err, context.Canceled), "err: %s", err)
}

func TestWorkWithErrors(t *testing.T) {
	t.Parallel()

	const ch1, ch2, ch3 = "ch1", "ch2", "ch3"
	const startBlock1, countBlocks1, countTxOrSwaps1 = 100, 10, 3
	const startBlock2, countBlocks2, countTxOrSwaps2 = 200, 5, 2
	const startBlock3, countBlocks3, countTxOrSwaps3 = 300, 3, 2

	var expectedErrors []error
	addExpectedErrors := func(msg string, src errorshlp.ErrType, name errorshlp.ComponentName) error {
		e := errors.New(msg)
		if src != "" && name != "" {
			e = errorshlp.WrapWithDetails(e, src, name)
		}
		expectedErrors = append(expectedErrors, e)
		return e
	}

	ctx, cancel, _ := testshlp.CreateCtxLoggerWithCancel(t)

	stor := &stubStor{}
	stor.callHlp.AddErrMap(stor.LoadCheckPoints, map[int]error{
		1: addExpectedErrors("LoadCheckPoints error 1", "", ""),
	})
	stor.callHlp.AddErrMap(stor.SaveCheckPoints, map[int]error{
		2: addExpectedErrors("SaveCheckPoints error 2", "", ""),
		3: addExpectedErrors("SaveCheckPoints error 3", "", ""),
	})

	cr := &stubCollectorCr{
		channelsData: map[string][]*collectordto.BlockData{
			ch1: createSrcData(ch1, ch1, startBlock1, countBlocks1, countTxOrSwaps1, 0),
			ch2: createSrcData(ch1, ch2, startBlock2, countBlocks2, countTxOrSwaps2, 0),
			ch3: createSrcData(ch1, ch3, startBlock3, countBlocks3, countTxOrSwaps3, 0),
		},
	}
	cr.callHlp.AddErrMap(cr.create, map[int]error{
		0: addExpectedErrors("create collector error 0", nerrors.ErrTypeParsing, nerrors.ComponentBatch),
		2: addExpectedErrors("create collector error 2", nerrors.ErrTypeParsing, nerrors.ComponentBatch),
	})

	chExecutorCr := newStubChExecutorCreator(ch1)
	chExecutorCr.callHlp.AddErrMap(chExecutorCr.CalcBatchSize, map[int]error{
		0: addExpectedErrors("CalcBatchSize error 0", nerrors.ErrTypeParsing, nerrors.ComponentBatch),
		2: addExpectedErrors("CalcBatchSize error 2", nerrors.ErrTypeParsing, nerrors.ComponentBatch),
	})
	chExecutorCr.callHlp.AddErrMap(chExecutorCr.Execute, map[int]error{
		0: addExpectedErrors("Execute error 0", "", ""),
		2: addExpectedErrors("Execute error 2", "", ""),
	})
	chExecutorCr.callHlp.AddErrMap(chExecutorCr.create, map[int]error{
		1: addExpectedErrors("create executor error 1", nerrors.ErrTypeParsing, nerrors.ComponentBatch),
	})

	chRobot := NewRobot(ctx, ch1, 10,
		map[string]uint64{ch1: 0, ch2: 0, ch3: 0},
		func(ctx context.Context, dataReady chan<- struct{}, srcChName string, startFrom uint64) (ChCollector, error) {
			return cr.create(ctx, dataReady, srcChName, startFrom, 1)
		},
		func(ctx context.Context) (ChExecutor, error) {
			return chExecutorCr.create(ctx)
		},
		stor,
		collectorbatch.Limits{LenLimit: countTxOrSwaps1, SizeLimit: 100000, TimeoutLimit: time.Second * 2})
	require.NotNil(t, chRobot)

	go func() {
		for {
			<-time.After(time.Second * 1)
			chExecutorCr.checkSavedState(func(
				maxTxBlockNum uint64,
				batches []*executordto.Batch,
				allTxs map[uint32][]byte,
				allSwaps map[*proto.Swap]struct{},
			) {
				if maxTxBlockNum == startBlock1+countBlocks1-1 &&
					len(allTxs) == countBlocks1*countTxOrSwaps1 &&
					len(allSwaps) == countBlocks2*countTxOrSwaps2+countBlocks3*countTxOrSwaps3 {
					cancel()
					return
				}
			})
		}
	}()

	var caughtErrors []error
	for {
		err := chRobot.Run(ctx)
		if ctx.Err() != nil && errors.Is(err, context.Canceled) {
			break
		}
		log.Println("caught error", err)
		caughtErrors = append(caughtErrors, err)
	}
	require.ElementsMatch(t, expectedErrors, caughtErrors)
}

//nolint:unparam
func createSrcData(dst, src string, startBlock, countBlocks, countTxOrSwaps, emptyBlockEveryN int) []*collectordto.BlockData {
	var res []*collectordto.BlockData
	for i := 0; i < countBlocks; i++ {
		d := &collectordto.BlockData{
			BlockNum: uint64(startBlock + i),
			Size:     0,
		}
		if emptyBlockEveryN != 0 && i%emptyBlockEveryN == 0 {
			continue
		}
		if dst == src {
			for j := 0; j < countTxOrSwaps; j++ {
				tx := make([]byte, 4)
				binary.LittleEndian.PutUint32(tx, uint32(i*countTxOrSwaps+j))
				d.Txs = append(d.Txs, tx)
			}
		} else {
			for j := 0; j < countTxOrSwaps; j++ {
				d.Swaps = append(d.Swaps, &proto.Swap{
					Owner:  nil,
					Token:  "zxcv",
					Amount: []byte{1, 2, 3, 4},
					From:   src,
					To:     dst,
				})
			}
		}
		res = append(res, d)
	}
	return res
}
