package chcollector

import (
	"context"
	"sync"

	"github.com/atomyze-foundation/common-component/errorshlp"
	"github.com/atomyze-foundation/robot/dto/collectordto"
	"github.com/atomyze-foundation/robot/helpers/nerrors"
	"github.com/atomyze-foundation/robot/metrics"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/newity/glog"
	"github.com/pkg/errors"
)

// ChDataParser is a interface for extract data from block
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

// NewCollector creates a new collector
func NewCollector(ctx context.Context,
	prsr ChDataParser,
	dataReady chan<- struct{},
	events <-chan *fab.BlockEvent,
	bufSize uint,
) (*chCollector, error) { //nolint:revive
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
			cc.log.Debugf("extract data from block: %v", ev.Block.Header.Number)
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

// Close closes collector
func (cc *chCollector) Close() {
	cc.cancel()
	cc.wgLoop.Wait()
	cc.raiseReadySignal()
}

// GetData returns channel with data
func (cc *chCollector) GetData() <-chan *collectordto.BlockData {
	return cc.outData
}
