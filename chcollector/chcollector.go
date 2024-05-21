package chcollector

import (
	"context"
	"errors"
	"sync"

	"github.com/anoideaopen/common-component/errorshlp"
	"github.com/anoideaopen/glog"
	"github.com/anoideaopen/robot/dto/collectordto"
	"github.com/anoideaopen/robot/helpers/nerrors"
	"github.com/anoideaopen/robot/metrics"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
)

type ChDataParser interface {
	ExtractData(block *common.Block) (*collectordto.BlockData, error)
}

type chCollector struct {
	log       glog.Logger
	m         metrics.Metrics
	events    <-chan *fab.BlockEvent
	dataReady chan<- struct{}
	outData   chan *collectordto.BlockData
	prsr      ChDataParser
	cancel    context.CancelFunc
	wgLoop    *sync.WaitGroup
}

func NewCollector(ctx context.Context,
	prsr ChDataParser,
	dataReady chan<- struct{},
	events <-chan *fab.BlockEvent,
	bufSize uint,
) (*chCollector, error) {
	log := glog.FromContext(ctx)

	if bufSize == 0 {
		return nil, errorshlp.WrapWithDetails(
			errors.New("invalid buffer size for collector - must be great than 0"),
			nerrors.ErrTypeInternal,
			nerrors.ComponentCollector)
	}

	clCtx, cancel := context.WithCancel(context.Background())
	cl := &chCollector{
		log:       log,
		m:         metrics.FromContext(ctx),
		events:    events,
		dataReady: dataReady,
		outData:   make(chan *collectordto.BlockData, bufSize),
		prsr:      prsr,
		cancel:    cancel,
		wgLoop:    &sync.WaitGroup{},
	}
	cl.wgLoop.Add(1)

	go cl.loopExtract(clCtx) //nolint:contextcheck

	return cl, nil
}

func (cc *chCollector) loopExtract(ctx context.Context) {
	defer func() {
		close(cc.outData)
		cc.wgLoop.Done()
	}()

	for ctx.Err() == nil {
		var blockData *collectordto.BlockData
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-cc.events:
			if !ok {
				return
			}
			cc.log.Debugf("extract data from block: %v", ev.Block.GetHeader().GetNumber())
			d, err := cc.prsr.ExtractData(ev.Block)
			if err != nil {
				cc.log.Errorf("extract data error: %s", err)
				// don't change state
				continue
			}
			blockData = d
			cc.m.BlockTxCount().Observe(float64(d.ItemsCount()))
			cc.m.CollectorProcessBlockNum().Set(float64(d.BlockNum))
			cc.m.TxWaitingCount().Add(float64(d.ItemsCount()))
		}

		select {
		case <-ctx.Done():
			return
		case cc.outData <- blockData:
			cc.raiseReadySignal()
		}
	}
}

func (cc *chCollector) raiseReadySignal() {
	// raise ready event
	select {
	case cc.dataReady <- struct{}{}:
	default:
	}
}

func (cc *chCollector) Close() {
	cc.cancel()
	cc.wgLoop.Wait()
	cc.raiseReadySignal()
}

func (cc *chCollector) GetData() <-chan *collectordto.BlockData {
	return cc.outData
}
