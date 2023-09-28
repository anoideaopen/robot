package collectorbatch

import (
	"context"
	"time"

	"github.com/atomyze-foundation/common-component/errorshlp"
	"github.com/atomyze-foundation/robot/dto/collectordto"
	"github.com/atomyze-foundation/robot/dto/executordto"
	"github.com/atomyze-foundation/robot/helpers/nerrors"
	"github.com/atomyze-foundation/robot/logger"
	"github.com/atomyze-foundation/robot/metrics"
	"github.com/newity/glog"
	"github.com/pkg/errors"
)

// ErrBlockDataOutOfLimit is returned when the block data is out of limit
var ErrBlockDataOutOfLimit = errors.New("block data out of limit")

// Limits is a struct with limits for batch
type Limits struct {
	// BlocksCountLimit is a limit for blocks count in batch
	BlocksCountLimit uint
	// TimeoutLimit is a limit for batch execution time
	TimeoutLimit time.Duration
	// LenLimit is a limit for batch length
	LenLimit uint
	// SizeLimit is a limit for batch size
	SizeLimit uint
}

// LimitKind is a type for batch limit kind
type LimitKind string

const (
	// NoneLimitKind is a kind of batch limit when batch is not limited
	NoneLimitKind LimitKind = "None"
	// BlocksCountLimitKind is a kind of batch limit when batch is limited by blocks count
	BlocksCountLimitKind LimitKind = "BlocksCountLimit"
	// TimeoutLimitKind is a kind of batch limit when batch is limited by timeout
	TimeoutLimitKind LimitKind = "TimeoutLimit"
	// LenLimitKind is a kind of batch limit when batch is limited by length
	LenLimitKind LimitKind = "LenLimit"
	// SizeLimitKind is a kind of batch limit when batch is limited by size
	SizeLimitKind LimitKind = "SizeLimit"
)

// SrcInfo is a struct with info about source
type SrcInfo struct {
	// LastBlockNum is a number of last block from source
	LastBlockNum uint64
	// ItemsCount is a count of items from source
	ItemsCount uint
}

// BatchInfo is a struct with info about batch
type BatchInfo struct {
	// Kind is a kind of batch limit
	Kind LimitKind
	// BlocksCount is a count of blocks in batch
	BlocksCount uint
	// Len is a length of batch
	Len uint
	// Size is a size of batch
	Size uint
	// Sources is a map with info about sources
	Sources map[string]*SrcInfo
}

// CBatch is a struct with info about batch
type CBatch struct {
	log glog.Logger
	m   metrics.Metrics

	chName string

	calcBatchSize func(b *executordto.Batch) (uint, error)

	deadlineCh <-chan time.Time
	startTime  time.Time

	limits Limits

	batch, prevBatch *internalBatch
}

type internalBatch struct {
	batch       *executordto.Batch
	len         uint
	countBlocks uint
	size        uint
	limitKind   LimitKind
	sources     map[string]*SrcInfo
}

// NewBatch creates a new batch
func NewBatch(
	ctx context.Context,
	chName string,
	limits Limits,
	calcBatchSize func(b *executordto.Batch) (uint, error),
) *CBatch {
	log := glog.FromContext(ctx).
		With(logger.Labels{
			Component: logger.ComponentBatch,
			ChName:    chName,
		}.Fields()...)

	m := metrics.FromContext(ctx)
	m = m.CreateChild(
		metrics.Labels().RobotChannel.Create(chName),
	)

	var deadlineCh <-chan time.Time
	if limits.TimeoutLimit == 0 {
		deadlineCh = make(chan time.Time)
	} else {
		deadlineCh = time.After(limits.TimeoutLimit)
	}

	return &CBatch{
		log:           log,
		m:             m,
		chName:        chName,
		startTime:     time.Now(),
		limits:        limits,
		calcBatchSize: calcBatchSize,
		deadlineCh:    deadlineCh,
		batch: &internalBatch{
			limitKind: NoneLimitKind,
			batch: &executordto.Batch{
				TxIndToBlocks: make(map[uint]uint64),
			},
			sources: make(map[string]*SrcInfo),
		},
		prevBatch: &internalBatch{
			limitKind: NoneLimitKind,
			batch: &executordto.Batch{
				TxIndToBlocks: make(map[uint]uint64),
			},
			sources: make(map[string]*SrcInfo),
		},
	}
}

// AddIfInLimit adds block data to batch if batch is not limited
func (b *CBatch) AddIfInLimit(chName string, d *collectordto.BlockData) (bool, error) {
	if b.batch.limitKind != NoneLimitKind {
		return false, nil
	}

	if b.isDeadline() {
		b.batch.limitKind = TimeoutLimitKind
		return false, nil
	}

	b.addBlockToBatch(b.prevBatch, chName, d)

	if b.limits.BlocksCountLimit > 0 && b.prevBatch.countBlocks > b.limits.BlocksCountLimit {
		b.batch.limitKind = BlocksCountLimitKind
		return false, nil
	}

	if b.limits.LenLimit > 0 && b.prevBatch.len > b.limits.LenLimit {
		b.batch.limitKind = LenLimitKind
		if b.prevBatch.countBlocks == 1 {
			return false,
				errorshlp.WrapWithDetails(
					errors.Wrapf(
						ErrBlockDataOutOfLimit,
						"ch: %s, block: %v, current len: %v, batch len limit: %v",
						chName, d.BlockNum, b.prevBatch.len, b.limits.LenLimit),
					nerrors.ErrTypeInternal, nerrors.ComponentBatch)
		}
		return false, nil
	}

	if b.limits.SizeLimit > 0 {
		realSize, err := b.calcBatchSize(b.prevBatch.batch)
		if err != nil {
			return false,
				errorshlp.WrapWithDetails(err,
					nerrors.ErrTypeParsing, nerrors.ComponentBatch)
		}

		// compare predictSize with totalSize
		b.logSizeMetrics(realSize, b.prevBatch.batch.PredictSize)

		if realSize > b.limits.SizeLimit {
			b.batch.limitKind = SizeLimitKind
			if b.prevBatch.countBlocks == 1 {
				return false,
					errorshlp.WrapWithDetails(
						errors.Wrapf(ErrBlockDataOutOfLimit,
							"ch: %s, block: %v, total size: %v, batch size limit: %v",
							chName, d.BlockNum, realSize, b.limits.SizeLimit),
						nerrors.ErrTypeInternal, nerrors.ComponentBatch)
			}
			return false, nil
		}
		b.batch.size = realSize
		b.prevBatch.size = realSize
	}

	b.addBlockToBatch(b.batch, chName, d)

	return true, nil
}

// GetBatchForExec returns batch for execution
func (b *CBatch) GetBatchForExec() (*executordto.Batch, *BatchInfo) {
	b.setLimitBeforeReturn()
	return b.batch.batch, &BatchInfo{
		Kind:        b.batch.limitKind,
		BlocksCount: b.batch.countBlocks,
		Len:         b.batch.len,
		Size:        b.batch.size,
		Sources:     b.batch.sources,
	}
}

func (b *CBatch) setLimitBeforeReturn() {
	if b.batch.limitKind != NoneLimitKind {
		return
	}

	if b.isDeadline() {
		b.batch.limitKind = TimeoutLimitKind
		return
	}

	if b.limits.BlocksCountLimit > 0 && b.batch.countBlocks >= b.limits.BlocksCountLimit {
		b.batch.limitKind = BlocksCountLimitKind
		return
	}

	if b.limits.LenLimit > 0 && b.batch.len >= b.limits.LenLimit {
		b.batch.limitKind = LenLimitKind
		return
	}

	if b.limits.SizeLimit > 0 && b.batch.size >= b.limits.SizeLimit {
		b.batch.limitKind = SizeLimitKind
		return
	}
}

func (b *CBatch) isDeadline() bool {
	if b.limits.TimeoutLimit == 0 {
		return false
	}
	return b.startTime.Add(b.limits.TimeoutLimit).Before(time.Now())
}

// Deadline returns deadline channel
func (b *CBatch) Deadline() <-chan time.Time {
	return b.deadlineCh
}

func (b *CBatch) logSizeMetrics(realSize, predictSize uint) {
	if realSize == 0 {
		return
	}
	absDiff := float64(realSize) - float64(predictSize)
	diff := absDiff / float64(predictSize)

	b.log.Debugf("totalSize: %v predictSize: %v absDiff: %v diff: %v",
		realSize, predictSize, absDiff, diff)
	b.m.BatchSizeEstimatedDiff().Observe(diff)
}

func (b *CBatch) addBlockToBatch(ab *internalBatch,
	chName string, d *collectordto.BlockData,
) {
	srcInfo, ok := ab.sources[chName]
	if !ok {
		srcInfo = &SrcInfo{}
		ab.sources[chName] = srcInfo
	}
	srcInfo.LastBlockNum = d.BlockNum
	srcInfo.ItemsCount += d.ItemsCount()

	ab.countBlocks++
	ab.batch.PredictSize += d.Size
	ab.batch.Txs = append(ab.batch.Txs, d.Txs...)
	ab.batch.Keys = append(ab.batch.Keys, d.SwapsKeys...)
	ab.batch.MultiKeys = append(ab.batch.MultiKeys, d.MultiSwapsKeys...)
	ab.batch.Swaps = append(ab.batch.Swaps, d.Swaps...)
	ab.batch.MultiSwaps = append(ab.batch.MultiSwaps, d.MultiSwaps...)
	ab.len += d.ItemsCount()

	if chName == b.chName {
		startIndex := len(ab.batch.Txs) - len(d.Txs)
		for i := 0; i < len(d.Txs); i++ {
			ab.batch.TxIndToBlocks[uint(i+startIndex)] = d.BlockNum
		}
	}
}
