//go:build !integration
// +build !integration

package chrobot

import (
	"context"
	"sync"

	"github.com/anoideaopen/common-component/testshlp"
	"github.com/anoideaopen/robot/dto/collectordto"
)

type stubCollectorCr struct {
	callHlp testshlp.CallHlp

	dstChName    string
	channelsData map[string][]*collectordto.BlockData
}

func (cr *stubCollectorCr) create(_ context.Context,
	dataReady chan<- struct{},
	srcChName string, startFrom uint64,
	bufSize uint,
) (*stubCollector, error) {
	if err := cr.callHlp.Call(cr.create); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	cl := &stubCollector{
		dstChName: cr.dstChName,
		srcChName: srcChName,
		startFrom: startFrom,
		data:      cr.channelsData[srcChName],
		dataReady: dataReady,
		outData:   make(chan *collectordto.BlockData),
		cancel:    cancel,
		bufSize:   bufSize,
	}

	cl.wgLoop.Add(1)
	go cl.run(ctx)

	return cl, nil
}

type stubCollector struct {
	dstChName string
	srcChName string
	startFrom uint64
	dataReady chan<- struct{}
	outData   chan *collectordto.BlockData
	cancel    context.CancelFunc
	wgLoop    sync.WaitGroup
	data      []*collectordto.BlockData
	bufSize   uint
}

func (cc *stubCollector) GetData() <-chan *collectordto.BlockData {
	return cc.outData
}

func (cc *stubCollector) Close() {
	cc.cancel()
	cc.wgLoop.Wait()
	cc.raiseReadySignal()
}

func (cc *stubCollector) run(ctx context.Context) {
	defer func() {
		cc.wgLoop.Done()
		close(cc.outData)
	}()

	for i := 0; i < len(cc.data) && ctx.Err() == nil; i++ {
		bd := cc.data[i]
		if bd.BlockNum < cc.startFrom {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case cc.outData <- bd:
			cc.raiseReadySignal()
		}
	}
	<-ctx.Done()
}

func (cc *stubCollector) raiseReadySignal() {
	// raise ready event
	select {
	case cc.dataReady <- struct{}{}:
	default:
	}
}
