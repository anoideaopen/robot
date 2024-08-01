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

type ChCollector struct {
	Log       glog.Logger
	Metrics   metrics.Metrics
	Events    <-chan *fab.BlockEvent
	DataReady chan<- struct{}
	OutData   chan *collectordto.BlockData
	Parser    ChDataParser
	Cancel    context.CancelFunc
	WgLoop    *sync.WaitGroup
}

func NewCollector(ctx context.Context,
	parser ChDataParser,
	dataReady chan<- struct{},
	events <-chan *fab.BlockEvent,
	bufSize uint,
) (*ChCollector, error) {
	log := glog.FromContext(ctx)

	if bufSize == 0 {
		return nil, errorshlp.WrapWithDetails(
			errors.New("invalid buffer size for collector - must be great than 0"),
			nerrors.ErrTypeInternal,
			nerrors.ComponentCollector)
	}

	clCtx, cancel := context.WithCancel(context.Background())
	cl := &ChCollector{
		Log:       log,
		Metrics:   metrics.FromContext(ctx),
		Events:    events,
		DataReady: dataReady,
		OutData:   make(chan *collectordto.BlockData, bufSize),
		Parser:    parser,
		Cancel:    cancel,
		WgLoop:    &sync.WaitGroup{},
	}
	cl.WgLoop.Add(1)

	go cl.loopExtract(clCtx) //nolint:contextcheck

	return cl, nil
}

func (cc *ChCollector) loopExtract(ctx context.Context) {
	defer func() {
		close(cc.OutData)
		cc.WgLoop.Done()
	}()

	for ctx.Err() == nil {
		var blockData *collectordto.BlockData
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-cc.Events:
			if !ok {
				return
			}
			cc.Log.Debugf("extract data from block: %v", ev.Block.GetHeader().GetNumber())
			d, err := cc.Parser.ExtractData(ev.Block)
			if err != nil {
				cc.Log.Errorf("extract data error: %s", err)
				// don't change state
				continue
			}
			blockData = d
			cc.Metrics.BlockTxCount().Observe(float64(d.ItemsCount()))
			cc.Metrics.CollectorProcessBlockNum().Set(float64(d.BlockNum))
			cc.Metrics.TxWaitingCount().Add(float64(d.ItemsCount()))
		}

		select {
		case <-ctx.Done():
			return
		case cc.OutData <- blockData:
			cc.RaiseReadySignal()
		}
	}
}

func (cc *ChCollector) RaiseReadySignal() {
	// raise ready event
	select {
	case cc.DataReady <- struct{}{}:
	default:
	}
}

func (cc *ChCollector) Close() {
	cc.Cancel()
	cc.WgLoop.Wait()
	cc.RaiseReadySignal()
}

func (cc *ChCollector) GetData() <-chan *collectordto.BlockData {
	return cc.OutData
}
