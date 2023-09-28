package hlf

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/atomyze-foundation/cartridge/manager"
	"github.com/atomyze-foundation/common-component/errorshlp"
	"github.com/atomyze-foundation/robot/chcollector"
	"github.com/atomyze-foundation/robot/dto/collectordto"
	"github.com/atomyze-foundation/robot/dto/parserdto"
	"github.com/atomyze-foundation/robot/helpers/nerrors"
	"github.com/atomyze-foundation/robot/hlf/parser"
	"github.com/atomyze-foundation/robot/logger"
	"github.com/atomyze-foundation/robot/metrics"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/event"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/deliverclient/seek"
	"github.com/newity/glog"
	"github.com/pkg/errors"
)

type realCollector interface {
	GetData() <-chan *collectordto.BlockData
	Close()
}

type chCollector struct {
	log glog.Logger
	m   metrics.Metrics

	srcChName string

	user           string
	org            string
	configProvider core.ConfigProvider
	cryptoManager  manager.Manager

	initStartFrom uint64
	realCollector realCollector
	proxyEvents   chan<- *fab.BlockEvent
	cancel        context.CancelFunc
	wgLoop        *sync.WaitGroup

	createEventsSrc    func(ctx context.Context, startFrom uint64) (eventsSrc, error)
	delayAfterSrcError time.Duration
	awaitEventsTimeout time.Duration
}

type sdkEventsSrc struct {
	sdkComps     *sdkComponents
	reg          fab.Registration
	eventService fab.EventService
	events       <-chan *fab.BlockEvent
}

type eventsSrc interface {
	close()
	getEvents() <-chan *fab.BlockEvent
}

func (ses *sdkEventsSrc) getEvents() <-chan *fab.BlockEvent {
	return ses.events
}

func (ses *sdkEventsSrc) close() {
	if ses == nil {
		return
	}

	if ses.reg != nil {
		// bug from sdk - need to dry channel another way - leak registration
		go func() {
			// just dry
			for range ses.events { //nolint:revive
			}
		}()

		ses.eventService.Unregister(ses.reg)
	}
}

const (
	defaultDelayAfterSrcError = 5 * time.Second
	defaultAwaitEventsTimeout = 60 * time.Second
)

func createChCollector(ctx context.Context,
	dstChName, srcChName string,
	dataReady chan<- struct{}, startFrom uint64, bufSize uint,
	connectionProfile, userName, orgName string,
	txPrefixes parserdto.TxPrefixes,
	cryptoManager manager.Manager,
) (*chCollector, error) {
	log := glog.FromContext(ctx).
		With(logger.Labels{
			Component: logger.ComponentCollector,
			DstChName: dstChName,
			SrcChName: srcChName,
		}.Fields()...)

	m := metrics.FromContext(ctx).CreateChild(
		metrics.Labels().RobotChannel.Create(dstChName),
		metrics.Labels().Channel.Create(srcChName))

	prsr := parser.NewParser(log, dstChName, srcChName, txPrefixes)

	proxyEvents := make(chan *fab.BlockEvent)
	rc, err := chcollector.NewCollector(
		glog.NewContext(metrics.NewContext(ctx, m), log),
		prsr, dataReady, proxyEvents, bufSize)
	if err != nil {
		return nil, errorshlp.WrapWithDetails(err, nerrors.ErrTypeHlf, nerrors.ComponentCollector)
	}

	return createChCollectorAdv(
		ctx, log, m,
		srcChName, startFrom,
		connectionProfile, userName, orgName,
		cryptoManager,
		proxyEvents, rc, nil,
		defaultDelayAfterSrcError,
		defaultAwaitEventsTimeout,
	), nil
}

func createChCollectorAdv(ctx context.Context,
	log glog.Logger, m metrics.Metrics,
	srcChName string, startFrom uint64,
	connectionProfile, userName, orgName string,
	cryptoManager manager.Manager,
	proxyEvents chan<- *fab.BlockEvent,
	rc realCollector,
	createEventsSrc func(ctx context.Context, startFrom uint64) (eventsSrc, error),
	delayAfterSrcError time.Duration,
	awaitEventsTimeout time.Duration,
) *chCollector {
	chColl := &chCollector{
		log:            log,
		m:              m,
		srcChName:      srcChName,
		user:           userName,
		org:            orgName,
		configProvider: config.FromFile(connectionProfile),
		cryptoManager:  cryptoManager,

		initStartFrom:      startFrom,
		proxyEvents:        proxyEvents,
		wgLoop:             &sync.WaitGroup{},
		createEventsSrc:    createEventsSrc,
		realCollector:      rc,
		delayAfterSrcError: delayAfterSrcError,
		awaitEventsTimeout: awaitEventsTimeout,
	}

	if chColl.createEventsSrc == nil {
		chColl.createEventsSrc = func(ctx context.Context, startFrom uint64) (eventsSrc, error) {
			return chColl.createSdkEventsSrc(ctx, startFrom)
		}
	}

	loopCtx, cancel := context.WithCancel(ctx)
	chColl.cancel = cancel
	chColl.wgLoop.Add(1)

	go chColl.loopProxy(loopCtx)

	return chColl
}

func (cc *chCollector) createSdkEventsSrc(ctx context.Context, startFrom uint64) (res *sdkEventsSrc, resErr error) {
	defer func() {
		if resErr != nil {
			res.close()
			res = nil
		}
	}()

	configBackends, err := cc.configProvider()
	if err != nil {
		resErr = errors.WithStack(err)
		return
	}
	res = &sdkEventsSrc{}
	if cc.cryptoManager != nil {
		res.sdkComps, resErr = createSdkComponentsWithCryptoMng(ctx, cc.srcChName, cc.org, configBackends,
			cc.cryptoManager)
	} else {
		res.sdkComps, resErr = createSdkComponentsWithoutCryptoMng(ctx, cc.srcChName, cc.org, cc.user,
			configBackends)
	}
	if resErr != nil {
		return
	}

	// https://github.com/atomyze-foundation/robot/-/issues/89
	// There is an issue in Fabric SDK when we subscribe from the block that was not created yet,
	// so we subscribe from the latest block that was already created and skip it later
	if startFrom > 0 {
		startFrom--
	}

	clientID := fmt.Sprintf("%s-%s", cc.srcChName, uuid.New().String())

	clOpt := []event.ClientOption{
		event.WithBlockEvents(),
		event.WithEventConsumerTimeout(0),
		event.WithSeekType(seek.FromBlock),
		event.WithBlockNum(startFrom),
		event.WithClientID(clientID),
	}
	cc.log.Infof("start collect from %d block in channel %s", startFrom, cc.srcChName)

	res.eventService, err = event.New(res.sdkComps.chProvider, clOpt...)
	if err != nil {
		resErr = errors.WithStack(err)
		return
	}
	cc.log.Debug("event client created")

	res.reg, res.events, err = res.eventService.RegisterBlockEvent()
	if err != nil {
		resErr = errors.WithStack(err)
		return
	}
	cc.log.Debug("blocks events registered")

	return
}

func (cc *chCollector) loopProxy(ctx context.Context) {
	defer func() {
		close(cc.proxyEvents)
		cc.wgLoop.Done()
	}()

	startFrom := cc.initStartFrom
	isFirstAttempt := true
	for ctx.Err() == nil {
		ses, err := cc.createEventsSrc(ctx, startFrom)
		if err != nil {
			cc.sendChErrMetric(isFirstAttempt, false, false)
			cc.log.Errorf("create sdk events source failed: %s", err)
			delayOrCancel(ctx, cc.delayAfterSrcError)
			continue
		}
		isFirstAttempt = false

		lastPushedBlockNum := cc.loopProxyByEvents(ctx, startFrom, ses.getEvents())
		if lastPushedBlockNum != nil {
			startFrom = *lastPushedBlockNum + 1
		}
		ses.close()
		delayOrCancel(ctx, cc.delayAfterSrcError)
	}
}

func (cc *chCollector) loopProxyByEvents(ctx context.Context, startedFrom uint64, events <-chan *fab.BlockEvent) *uint64 {
	var lastPushedBlockNum *uint64

	expectedBlockNum := startedFrom
	isFirstBlockGot := false

	for {
		select {
		case <-ctx.Done():
			return lastPushedBlockNum
		case ev, ok := <-events:
			if !ok {
				cc.sendChErrMetric(false, true, false)
				cc.log.Error("source events channel closed")
				return lastPushedBlockNum
			}

			if ev.Block == nil || ev.Block.Header == nil {
				cc.log.Error("got event block: invalid block!!!")
				return lastPushedBlockNum
			}

			blockNum := ev.Block.Header.Number
			cc.log.Debugf("got event block: %v", blockNum)

			// It is allowed to receive (startedFrom - 1) once when we start receiving blocks
			if !isFirstBlockGot && blockNum != startedFrom && startedFrom > 0 && blockNum == startedFrom-1 {
				cc.log.Infof("got event block: %v, expected: %v. Skip it because we subscribe with -1 gap", blockNum, expectedBlockNum)
				isFirstBlockGot = true
				continue
			}

			if blockNum < expectedBlockNum {
				cc.log.Errorf("got event block: %v, expected: %v. The block was already handled, skip it", blockNum, expectedBlockNum)
				continue
			}

			// It is not allowed, might skip unhandled blocks
			if blockNum > expectedBlockNum {
				cc.log.Errorf("got event block: %v, expected: %v. The block num is greater than expected", blockNum, expectedBlockNum)
				return lastPushedBlockNum
			}

			select {
			case <-ctx.Done():
				return lastPushedBlockNum
			case cc.proxyEvents <- ev:
				lastPushedBlockNum = &ev.Block.Header.Number
				expectedBlockNum++
			}
		}
	}
}

func (cc *chCollector) sendChErrMetric(isFirstAttempt, isSrcChClosed, isTimeout bool) {
	cc.m.TotalSrcChErrors().Inc(
		metrics.Labels().IsFirstAttempt.Create(fmt.Sprint(isFirstAttempt)),
		metrics.Labels().IsSrcChClosed.Create(fmt.Sprint(isSrcChClosed)),
		metrics.Labels().IsTimeout.Create(fmt.Sprint(isTimeout)),
	)
}

func (cc *chCollector) Close() {
	if cc.cancel != nil {
		cc.cancel()
	}
	if cc.realCollector != nil {
		cc.realCollector.Close()
	}
	cc.wgLoop.Wait()
}

func (cc *chCollector) GetData() <-chan *collectordto.BlockData {
	return cc.realCollector.GetData()
}

func delayOrCancel(ctx context.Context, delay time.Duration) {
	select {
	case <-time.After(delay):
	case <-ctx.Done():
	}
}
