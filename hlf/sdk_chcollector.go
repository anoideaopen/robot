package hlf

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/anoideaopen/common-component/errorshlp"
	"github.com/anoideaopen/glog"
	"github.com/anoideaopen/robot/chcollector"
	"github.com/anoideaopen/robot/dto/collectordto"
	"github.com/anoideaopen/robot/dto/parserdto"
	"github.com/anoideaopen/robot/helpers/nerrors"
	"github.com/anoideaopen/robot/hlf/parser"
	"github.com/anoideaopen/robot/logger"
	"github.com/anoideaopen/robot/metrics"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/event"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/options"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/client"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/deliverclient/seek"
)

type realCollector interface {
	GetData() <-chan *collectordto.BlockData
	Close()
}

type ChCollector struct {
	log glog.Logger
	m   metrics.Metrics

	srcChName string

	user           string
	org            string
	configProvider core.ConfigProvider

	initStartFrom uint64
	realCollector realCollector
	proxyEvents   chan<- *fab.BlockEvent
	cancel        context.CancelFunc
	wgLoop        *sync.WaitGroup

	createEventsSrc    func(ctx context.Context, startFrom uint64) (EventsSrc, error)
	delayAfterSrcError time.Duration
	awaitEventsTimeout time.Duration
}

type sdkEventsSrc struct {
	sdkComps     *sdkComponents
	reg          fab.Registration
	eventService fab.EventService
	events       <-chan *fab.BlockEvent
	clientID     string
	log          glog.Logger
}

// EventsSrc - event source
type EventsSrc interface {
	Close()
	GetEvents() <-chan *fab.BlockEvent
}

func (ses *sdkEventsSrc) GetEvents() <-chan *fab.BlockEvent {
	return ses.events
}

func (ses *sdkEventsSrc) Close() {
	if ses == nil {
		return
	}

	if ses.reg != nil {
		// bug from sdk - need to dry channel another way - leak registration
		go func() {
			// just dry
			for range ses.events {
			}
		}()

		ses.eventService.Unregister(ses.reg)

		ses.deleteEventService()
	}
}

func (ses *sdkEventsSrc) deleteEventService() {
	channelContext, err := ses.sdkComps.chProvider()
	if err != nil {
		ses.log.Infof("sdkComps chProvider return error: %s", err)
		return
	}
	cs := channelContext.ChannelService()

	var opts []options.Opt
	opts = append(opts, client.WithBlockEvents())
	if ses.clientID != "" {
		opts = append(opts, client.WithID(ses.clientID))
	}
	err = cs.DeleteEventService(opts...)
	if err != nil {
		ses.log.Infof("delete event service error: %s", err)
		return
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
) (*ChCollector, error) {
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

	return CreateChCollectorAdv(
		ctx, log, m,
		srcChName, startFrom,
		connectionProfile, userName, orgName,
		proxyEvents, rc, nil,
		defaultDelayAfterSrcError,
		defaultAwaitEventsTimeout,
	), nil
}

// CreateChCollectorAdv - creates advanced channel collector
func CreateChCollectorAdv(ctx context.Context,
	log glog.Logger, m metrics.Metrics,
	srcChName string, startFrom uint64,
	connectionProfile, userName, orgName string,
	proxyEvents chan<- *fab.BlockEvent,
	rc realCollector,
	createEventsSrc func(ctx context.Context, startFrom uint64) (EventsSrc, error),
	delayAfterSrcError time.Duration,
	awaitEventsTimeout time.Duration,
) *ChCollector {
	chColl := &ChCollector{
		log:            log,
		m:              m,
		srcChName:      srcChName,
		user:           userName,
		org:            orgName,
		configProvider: config.FromFile(connectionProfile),

		initStartFrom:      startFrom,
		proxyEvents:        proxyEvents,
		wgLoop:             &sync.WaitGroup{},
		createEventsSrc:    createEventsSrc,
		realCollector:      rc,
		delayAfterSrcError: delayAfterSrcError,
		awaitEventsTimeout: awaitEventsTimeout,
	}

	if chColl.createEventsSrc == nil {
		chColl.createEventsSrc = func(ctx context.Context, startFrom uint64) (EventsSrc, error) {
			return chColl.createSdkEventsSrc(ctx, startFrom)
		}
	}

	loopCtx, cancel := context.WithCancel(ctx)
	chColl.cancel = cancel
	chColl.wgLoop.Add(1)

	go chColl.loopProxy(loopCtx)

	return chColl
}

func (cc *ChCollector) createSdkEventsSrc(ctx context.Context, startFrom uint64) (res *sdkEventsSrc, resErr error) {
	defer func() {
		if resErr != nil {
			res.Close()
			res = nil
		}
	}()

	configBackends, err := cc.configProvider()
	if err != nil {
		resErr = err
		return
	}
	res = &sdkEventsSrc{
		log: cc.log,
	}
	res.sdkComps, resErr = createSdkComponents(ctx, cc.srcChName, cc.org, cc.user, configBackends)
	if resErr != nil {
		return
	}

	// https://github.com/anoideaopen/robot/-/issues/89
	// There is an issue in Fabric SDK when we subscribe from the block that was not created yet,
	// so we subscribe from the latest block that was already created and skip it later
	if startFrom > 0 {
		startFrom--
	}

	clientID := fmt.Sprintf("%s-%s", cc.srcChName, uuid.New().String())
	res.clientID = clientID
	clOpt := []event.ClientOption{
		event.WithBlockEvents(),
		event.WithEventConsumerTimeout(0),
		event.WithSeekType(seek.FromBlock),
		event.WithBlockNum(startFrom),
		event.WithClientID(clientID),
	}
	cc.log.Debugf("start collect from %d block in channel %s", startFrom, cc.srcChName)

	res.eventService, err = event.New(res.sdkComps.chProvider, clOpt...)
	if err != nil {
		resErr = err
		return
	}
	cc.log.Debug("event client created")

	res.reg, res.events, err = res.eventService.RegisterBlockEvent()
	if err != nil {
		resErr = err
		return
	}
	cc.log.Debug("blocks events registered")

	return
}

func (cc *ChCollector) loopProxy(ctx context.Context) {
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

		lastPushedBlockNum := cc.loopProxyByEvents(ctx, startFrom, ses.GetEvents())
		if lastPushedBlockNum != nil {
			startFrom = *lastPushedBlockNum + 1
		}
		ses.Close()
		delayOrCancel(ctx, cc.delayAfterSrcError)
	}
}

func (cc *ChCollector) loopProxyByEvents(ctx context.Context, startedFrom uint64, events <-chan *fab.BlockEvent) *uint64 {
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

			if ev.Block == nil || ev.Block.GetHeader() == nil {
				cc.log.Error("got event block: invalid block!!!")
				return lastPushedBlockNum
			}

			blockNum := ev.Block.GetHeader().GetNumber()
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

func (cc *ChCollector) sendChErrMetric(isFirstAttempt, isSrcChClosed, isTimeout bool) {
	cc.m.TotalSrcChErrors().Inc(
		metrics.Labels().IsFirstAttempt.Create(strconv.FormatBool(isFirstAttempt)),
		metrics.Labels().IsSrcChClosed.Create(strconv.FormatBool(isSrcChClosed)),
		metrics.Labels().IsTimeout.Create(strconv.FormatBool(isTimeout)),
	)
}

func (cc *ChCollector) Close() {
	if cc.cancel != nil {
		cc.cancel()
	}
	if cc.realCollector != nil {
		cc.realCollector.Close()
	}
	cc.wgLoop.Wait()
}

func (cc *ChCollector) GetData() <-chan *collectordto.BlockData {
	return cc.realCollector.GetData()
}

func delayOrCancel(ctx context.Context, delay time.Duration) {
	select {
	case <-time.After(delay):
	case <-ctx.Done():
	}
}
