package chrobot

import (
	"context"
	"fmt"
	"time"

	"github.com/anoideaopen/glog"
	"github.com/anoideaopen/robot/collectorbatch"
	"github.com/anoideaopen/robot/dto/collectordto"
	"github.com/anoideaopen/robot/dto/executordto"
	"github.com/anoideaopen/robot/dto/stordto"
	"github.com/anoideaopen/robot/logger"
	"github.com/anoideaopen/robot/metrics"
)

type ChStorage interface {
	SaveCheckPoints(ctx context.Context, cp *stordto.ChCheckPoint) (*stordto.ChCheckPoint, error)
	LoadCheckPoints(ctx context.Context) (*stordto.ChCheckPoint, bool, error)
}

type ChExecutorCreator func(ctx context.Context) (ChExecutor, error)

type ChCollectorCreator func(ctx context.Context,
	dataReady chan<- struct{},
	srcChName string, startFrom uint64) (ChCollector, error)

// ChExecutor is component that performs actions related to the execution of a batch.
type ChExecutor interface {
	// CalcBatchSize calculates and returns batch size (in bytes) which will be sent to chaincode,
	// in method Execute
	CalcBatchSize(b *executordto.Batch) (uint, error)

	// Execute sends batch to chaincode and returns num of block in which batch was committed.
	// minHeight - the minimum height that peers must have in order to execute batch
	Execute(ctx context.Context, b *executordto.Batch, minHeight uint64) (uint64, error)
	Close()
}

type ChCollector interface {
	GetData() <-chan *collectordto.BlockData
	Close()
}

type ChRobot struct {
	log glog.Logger
	m   metrics.Metrics

	chName              string
	initMinExecBlockNum uint64
	chsSources          map[string]uint64
	collectorCr         ChCollectorCreator
	chExecCr            ChExecutorCreator
	chStor              ChStorage

	batchLimits collectorbatch.Limits

	checkPoints      *stordto.ChCheckPoint
	collectors       []*chCollector
	lastCollectorInd int
	dataReady        <-chan struct{}
}

type chCollector struct {
	chName    string
	collector ChCollector
	bufData   *collectordto.BlockData
}

func NewRobot(ctx context.Context,
	chName string, initMinExecBlockNum uint64,
	chsSources map[string]uint64,
	collectorCr ChCollectorCreator,
	chExecutorCr ChExecutorCreator,
	chStor ChStorage,
	batchLimits collectorbatch.Limits,
) *ChRobot {
	log := glog.FromContext(ctx).
		With(logger.Labels{
			Component: logger.ComponentRobot,
			ChName:    chName,
		}.Fields()...)

	m := metrics.FromContext(ctx)
	m = m.CreateChild(
		metrics.Labels().RobotChannel.Create(chName),
	)

	return &ChRobot{
		log:                 log,
		m:                   m,
		chName:              chName,
		initMinExecBlockNum: initMinExecBlockNum,
		chsSources:          chsSources,
		collectorCr:         collectorCr,
		chExecCr:            chExecutorCr,
		chStor:              chStor,
		batchLimits:         batchLimits,
	}
}

func (chr *ChRobot) ChName() string {
	return chr.chName
}

func (chr *ChRobot) Run(ctx context.Context) error {
	chExec, err := chr.chExecCr(ctx)
	if err != nil {
		return err
	}
	defer chExec.Close()

	defer chr.closeCollectors()

	if err = chr.createCollectors(ctx); err != nil {
		return err
	}

	for ctx.Err() == nil {
		now := time.Now()
		bfe, limInfo, err := chr.collectBatch(ctx, chExec.CalcBatchSize)
		if err != nil {
			return err
		}
		chr.m.BatchCollectTime().Observe(time.Since(now).Seconds())

		if err = chr.executeBatch(ctx, chExec, bfe, limInfo); err != nil {
			return err
		}
	}

	return ctx.Err()
}

func (chr *ChRobot) closeCollectors() {
	if chr.collectors != nil {
		for _, cl := range chr.collectors {
			cl.collector.Close()
		}
		chr.collectors = nil
	}
	chr.dataReady = nil
}

func (chr *ChRobot) createCollectors(ctx context.Context) error {
	for chName := range chr.chsSources {
		chr.m.TxWaitingCount().Set(0,
			metrics.Labels().Channel.Create(chName))
	}

	chps, ok, err := chr.chStor.LoadCheckPoints(ctx)
	if err != nil {
		return err
	}

	if !ok {
		chps = &stordto.ChCheckPoint{
			Ver:                     0,
			MinExecBlockNum:         0,
			SrcCollectFromBlockNums: make(map[string]uint64),
		}
	}
	if chps.MinExecBlockNum < chr.initMinExecBlockNum {
		chps.MinExecBlockNum = chr.initMinExecBlockNum
	}

	// add to the checkpoints the missing channels for which the blockNum is specified in init config
	// or if blockNum in config for channel is greater than checkpoint blockNum
	for chName, blockNumFromCfg := range chr.chsSources {
		if blockNumFromChp, ok := chps.SrcCollectFromBlockNums[chName]; !ok || blockNumFromChp < blockNumFromCfg {
			chps.SrcCollectFromBlockNums[chName] = blockNumFromCfg
		}
	}

	// create collectors
	chr.checkPoints = chps
	dataReady := make(chan struct{}, 1)
	chr.dataReady = dataReady
	chr.collectors = nil
	chr.lastCollectorInd = 0

	for chName := range chr.chsSources {
		startFrom := uint64(0)
		if h, ok := chps.SrcCollectFromBlockNums[chName]; ok {
			startFrom = h
		}
		cl, err := chr.collectorCr(ctx, dataReady, chName, startFrom)
		if err != nil {
			return err
		}
		chr.collectors = append(chr.collectors, &chCollector{
			chName:    chName,
			collector: cl,
		})
	}

	return nil
}

// collectBatch collects data from all collectors
// and returns batch if some limit is reached (time of collection, size in bytes, size in items, etc).
//
// It takes context param and callback that calculates size of the batch in bytes.
func (chr *ChRobot) collectBatch(ctx context.Context, calcBatchSize func(b *executordto.Batch) (uint, error)) (*executordto.Batch, *collectorbatch.BatchInfo, error) { //nolint:gocognit
	b := collectorbatch.NewBatch(ctx, chr.chName,
		chr.batchLimits, calcBatchSize)
	for ctx.Err() == nil {
		isFound := false
		chr.log.Debugf("collect batch: %v", b)

		for i := 0; i < len(chr.collectors); chr.lastCollectorInd, i = chr.lastCollectorInd+1, i+1 {
			if chr.lastCollectorInd >= len(chr.collectors) {
				chr.lastCollectorInd = 0
			}

			chc := chr.collectors[chr.lastCollectorInd]

			if chc.bufData == nil {
				select {
				case bd, ok := <-chc.collector.GetData():
					if !ok {
						return nil, nil, fmt.Errorf("end of collector channel src:%s, dst:%s", chc.chName, chr.chName)
					}
					chc.bufData = bd
				default:
				}
			}

			if chc.bufData == nil {
				continue
			}

			isAdded, err := b.AddIfInLimit(chc.chName, chc.bufData)
			chr.log.Debugf("collect batch isAdded: %t, chName: %s", isAdded, chc.chName)
			if err != nil {
				return nil, nil, err
			}
			if !isAdded {
				bfe, limInfo := b.GetBatchForExec()
				chr.log.Debugf("batch collected, batch for exec: %v, limInfo: %v", bfe, limInfo)
				return bfe, limInfo, nil
			}

			chc.bufData = nil
			isFound = true
		}

		if !isFound {
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-b.Deadline():
				bfe, limInfo := b.GetBatchForExec()
				chr.log.Debugf("batch collected, cbatch deadline, batch for exec: %v, limInfo: %v", bfe, limInfo)
				return bfe, limInfo, nil
			case <-chr.dataReady:
			}
		}
	}
	return nil, nil, ctx.Err()
}

func (chr *ChRobot) executeBatch(ctx context.Context, chExec ChExecutor, bfe *executordto.Batch, batchInfo *collectorbatch.BatchInfo) error {
	var isCheckpointChanged bool

	chr.log.Debugf("got next batch: \n%s", batchExecInfoMsg(chr.chName, bfe, batchInfo))

	src, ok := batchInfo.Sources[chr.chName]
	if ok && chr.checkPoints.MinExecBlockNum < src.LastBlockNum {
		isCheckpointChanged = true
		chr.checkPoints.MinExecBlockNum = src.LastBlockNum
	}

	for chName, srcInfo := range batchInfo.Sources {
		if bn, ok := chr.checkPoints.SrcCollectFromBlockNums[chName]; !ok || bn < (srcInfo.LastBlockNum+1) {
			isCheckpointChanged = true
			chr.checkPoints.SrcCollectFromBlockNums[chName] = srcInfo.LastBlockNum + 1
		}
	}

	if !bfe.IsEmpty() {
		committedBlockNum, err := chExec.Execute(ctx, bfe, chr.checkPoints.MinExecBlockNum+1)
		if err != nil {
			return err
		}

		chr.log.Info("batch committed")

		if chr.checkPoints.MinExecBlockNum < committedBlockNum {
			chr.checkPoints.MinExecBlockNum = committedBlockNum
		}

		for chName, srcInfo := range batchInfo.Sources {
			if srcInfo.ItemsCount == 0 {
				continue
			}
			chr.m.TxWaitingCount().Add(-float64(srcInfo.ItemsCount),
				metrics.Labels().Channel.Create(chName))
		}
	}

	if !isCheckpointChanged {
		return nil
	}

	checkPoints, err := chr.chStor.SaveCheckPoints(ctx, chr.checkPoints)
	if err != nil {
		return err
	}

	chr.log.Debugf(storedCheckpointsMsg(checkPoints))

	chr.checkPoints = checkPoints
	return nil
}

func batchExecInfoMsg(chName string, bfe *executordto.Batch, batchInfo *collectorbatch.BatchInfo) string {
	msg := "batch info:"
	if len(bfe.Txs) > 0 {
		msg += fmt.Sprintf("\nlast preimage from block %d", batchInfo.Sources[chName].LastBlockNum)
		msg += fmt.Sprintf("\n%d txs", len(bfe.Txs))
	}
	if len(bfe.Swaps) > 0 {
		msg += fmt.Sprintf("\n%d swaps", len(bfe.Swaps))
	}
	if len(bfe.Keys) > 0 {
		msg += fmt.Sprintf("\n%d keys", len(bfe.Swaps))
	}
	if len(bfe.MultiSwaps) > 0 {
		msg += fmt.Sprintf("\n%d multiswaps", len(bfe.MultiSwaps))
	}
	if len(bfe.MultiKeys) > 0 {
		msg += fmt.Sprintf("\n%d multikeys", len(bfe.MultiKeys))
	}
	msg += fmt.Sprintf("\nlimit: %s ", batchInfo.Kind)
	msg += fmt.Sprintf("\nlen: %v", batchInfo.Len)
	msg += fmt.Sprintf("\nblocks: %v", batchInfo.BlocksCount)
	msg += fmt.Sprintf("\nsize: %v", batchInfo.Size)

	return msg
}

func storedCheckpointsMsg(checkPoints *stordto.ChCheckPoint) string {
	msg := "saved checkpoints:\n"
	for ch, n := range checkPoints.SrcCollectFromBlockNums {
		msg += fmt.Sprintf("collect from %d block for channel %s\n", n, ch)
	}
	msg += fmt.Sprintf("exec on peers with minimum blocknum in ledger %d\n", checkPoints.MinExecBlockNum)
	return msg
}
