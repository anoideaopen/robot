package hlf

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anoideaopen/common-component/testshlp"
	"github.com/anoideaopen/robot/dto/collectordto"
	"github.com/anoideaopen/robot/metrics"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/stretchr/testify/require"
)

func TestProxyLoopCreate(t *testing.T) {
	ctx, log := testshlp.CreateCtxLogger(t)

	proxyEvents := make(chan *fab.BlockEvent)
	stubRealColl := newStubRealCollector(ctx, proxyEvents)
	eventsSrcCreator := newStubEventsSrcCreator(5)

	createEventSrcErr := errors.New("createEventsSrc error")
	eventsSrcCreator.callHlp.AddErrMap(eventsSrcCreator.createEventsSrc, map[int]error{
		0: createEventSrcErr,
		2: createEventSrcErr,
		4: createEventSrcErr,
	})

	delayAfterSrcError := 1 * time.Second
	awaitEventsTimeout := 60 * time.Second

	chColl := createChCollectorAdv(ctx,
		log, metrics.FromContext(ctx),
		"srcCh", 0,
		"fakeConnectionProfile", "fakeUser", "fakeOrg",
		nil,
		proxyEvents,
		stubRealColl,
		eventsSrcCreator.createEventsSrc,
		delayAfterSrcError,
		awaitEventsTimeout,
	)
	require.NotNil(t, chColl)

	const targetBlockNum = 14
	for {
		bd, ok := <-stubRealColl.GetData()
		require.True(t, ok)
		log.Info("got data from src: ", bd.BlockNum)
		if bd.BlockNum == targetBlockNum {
			break
		}
	}

	chColl.Close()
	<-stubRealColl.wasClosed

	require.EqualValues(t, 3, *eventsSrcCreator.createdCount)
	require.EqualValues(t, 3, *eventsSrcCreator.closedCount)
}

// stubRealCollector reads from proxyEvents and writes to blockData
type stubRealCollector struct {
	proxyEvents <-chan *fab.BlockEvent
	blockData   chan *collectordto.BlockData
	cancel      context.CancelFunc
	wasClosed   chan struct{}
}

func newStubRealCollector(ctx context.Context, proxyEvents chan *fab.BlockEvent) *stubRealCollector {
	ctx, cancel := context.WithCancel(ctx)
	stub := &stubRealCollector{
		proxyEvents: proxyEvents,
		blockData:   make(chan *collectordto.BlockData),
		wasClosed:   make(chan struct{}),
		cancel:      cancel,
	}

	go stub.collectLoop(ctx)

	return stub
}

func (r *stubRealCollector) collectLoop(ctx context.Context) {
	defer close(r.blockData)

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-r.proxyEvents:
			if !ok {
				return
			}
			select {
			case <-ctx.Done():
				return
			case r.blockData <- &collectordto.BlockData{BlockNum: e.Block.Header.Number}:
			}
		}
	}
}

func (r *stubRealCollector) GetData() <-chan *collectordto.BlockData {
	return r.blockData
}

func (r *stubRealCollector) Close() {
	r.cancel()
	close(r.wasClosed)
}

type stubEventsSrc struct {
	events  <-chan *fab.BlockEvent
	closeCb func()
}

func (r *stubEventsSrc) close() {
	r.closeCb()
}

func (r *stubEventsSrc) getEvents() <-chan *fab.BlockEvent {
	return r.events
}

type stubEventsSrcCreator struct {
	callHlp testshlp.CallHlp

	chunkLen     int
	createdCount *uint32
	closedCount  *uint32
}

func newStubEventsSrcCreator(chunkLen int) *stubEventsSrcCreator {
	created := uint32(0)
	closed := uint32(0)
	return &stubEventsSrcCreator{
		chunkLen:     chunkLen,
		createdCount: &created,
		closedCount:  &closed,
	}
}

func (r *stubEventsSrcCreator) createEventsSrc(_ context.Context, startFrom uint64) (eventsSrc, error) {
	if err := r.callHlp.Call(r.createEventsSrc); err != nil {
		return nil, err
	}
	atomic.AddUint32(r.createdCount, 1)

	events := make(chan *fab.BlockEvent, r.chunkLen)
	for i := 0; i < r.chunkLen; i++ {
		events <- &fab.BlockEvent{
			Block: &common.Block{Header: &common.BlockHeader{Number: startFrom + uint64(i)}},
		}
	}
	close(events)

	return &stubEventsSrc{
		events: events,
		closeCb: func() {
			atomic.AddUint32(r.closedCount, 1)
		},
	}, nil
}

/*
func TestParallelCollecting(t *testing.T) {
	ciData := ntesting.CI(t)
	const countThreads = 2
	const batchSize = 100
	const maxbn = 7000

	channels := []string{"curaed"}
	wg := sync.WaitGroup{}
	wg.Add(countThreads * len(channels))

	logCtx, log := testshlp.CreateCtxLogger(t)

	for i := 0; i < countThreads; i++ {
		num := i
		for _, ch := range channels {
			chName := ch
			go func() {
				defer wg.Done()
				defer func() {
					log.Info("th:", num, " finished")
				}()

				cr := NewChCollectorCreator(
					fmt.Sprintf("th-%s-%v", chName, num),
					ciData.HlfProfilePath,
					ciData.HlfUserName, ciData.HlfProfile.OrgName,
					defaultPrefixes, 1)
				require.NotNil(t, cr)

				dataReady := make(chan struct{}, 1)

				cl, err := cr(logCtx, dataReady, chName, uint64(10+num*10))
				require.NoError(t, err)
				require.NotNil(t, cl)

				batchBlockNum := 0
				batchStartTime := time.Now()

				totalBlocks := 0

				for {
					bd, ok := <-cl.GetData()
					batchBlockNum++
					totalBlocks++
					if batchBlockNum == batchSize {
						diff := time.Since(batchStartTime)
						log.Info("ch:", chName, "th:", num, ", total:", totalBlocks, ",blocks per second:", batchSize/diff.Seconds())
						// start new
						batchBlockNum = 0
						batchStartTime = time.Now()
					}
					require.True(t, ok)
					if maxbn == bd.BlockNum {
						break
					}
				}
			}()
		}

	}
	wg.Wait()
}

*/
