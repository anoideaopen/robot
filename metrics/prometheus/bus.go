package prometheus

import (
	"context"

	"github.com/atomyze-foundation/common-component/basemetrics"
	"github.com/atomyze-foundation/common-component/basemetrics/baseprometheus"
	"github.com/atomyze-foundation/robot/metrics"
	"github.com/newity/glog"
	"github.com/prometheus/client_golang/prometheus"
)

// MetricsBus is a prometheus implementation of metrics bus
type MetricsBus struct {
	log glog.Logger

	baseBus *baseprometheus.BaseMetricsBus[MetricsBus]

	mTotalBatchExecuted           *baseprometheus.Counter
	mTotalExecutedTx              *baseprometheus.Counter
	mAppInfo                      *baseprometheus.Counter
	mTotalRobotStarted            *baseprometheus.Counter
	mTotalRobotStopped            *baseprometheus.Counter
	mTotalBatchSize               *baseprometheus.Counter
	mTotalOrderingReqSizeExceeded *baseprometheus.Counter
	mTotalSrcChErrors             *baseprometheus.Counter

	mBatchExecuteInvokeTime *baseprometheus.Histo
	mBatchSize              *baseprometheus.Histo
	mBatchItemsCount        *baseprometheus.Histo
	mBatchCollectTime       *baseprometheus.Histo
	mBatchSizeEstimatedDiff *baseprometheus.Histo
	mBlockTxCount           *baseprometheus.Histo

	mAppInitDuration          *baseprometheus.Gauge
	mTxWaitingCount           *baseprometheus.Gauge
	mHeightLedgerBlocks       *baseprometheus.Gauge
	mCollectorProcessBlockNum *baseprometheus.Gauge
}

// NewMetrics creates a new prometheus metrics bus
//
//nolint:funlen
func NewMetrics(ctx context.Context, mPrefix string) (*MetricsBus, error) {
	l := glog.FromContext(ctx)
	m := &MetricsBus{
		log:     l,
		baseBus: baseprometheus.NewBus[MetricsBus](ctx, mPrefix),
	}

	var err error

	if m.mTotalBatchExecuted, err = m.baseBus.AddCounter(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mTotalBatchExecuted = parent.mTotalBatchExecuted.ChildWith(labels)
		},
		"batches_executed_total", "",
		metrics.Labels().RobotChannel,
		metrics.Labels().IsErr); err != nil {
		return nil, err
	}

	if m.mTotalExecutedTx, err = m.baseBus.AddCounter(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mTotalExecutedTx = parent.mTotalExecutedTx.ChildWith(labels)
		},
		"tx_executed_total", "",
		metrics.Labels().RobotChannel,
		metrics.Labels().TxType); err != nil {
		return nil, err
	}

	if m.mBatchExecuteInvokeTime, err = m.baseBus.AddHisto(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mBatchExecuteInvokeTime = parent.mBatchExecuteInvokeTime.ChildWith(labels)
		},
		"batch_execute_duration_seconds", "",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 20, 30, 40, 50, 60},
		metrics.Labels().RobotChannel); err != nil {
		return nil, err
	}

	if m.mBatchSizeEstimatedDiff, err = m.baseBus.AddHisto(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mBatchSizeEstimatedDiff = parent.mBatchSizeEstimatedDiff.ChildWith(labels)
		},
		"batch_size_estimated_diff",
		"Difference between expected and actual batch size",
		[]float64{.01, .05, .1, .2, .3, .4, 1, 2},
		metrics.Labels().RobotChannel); err != nil {
		return nil, err
	}

	if m.mBatchSize, err = m.baseBus.AddHisto(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mBatchSize = parent.mBatchSize.ChildWith(labels)
		},
		"batch_size_bytes", "", prometheus.DefBuckets,
		metrics.Labels().RobotChannel); err != nil {
		return nil, err
	}

	if m.mTotalBatchSize, err = m.baseBus.AddCounter(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mTotalBatchSize = parent.mTotalBatchSize.ChildWith(labels)
		},
		"batch_size_bytes_total", "",
		metrics.Labels().RobotChannel); err != nil {
		return nil, err
	}

	if m.mTotalOrderingReqSizeExceeded, err = m.baseBus.AddCounter(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mTotalOrderingReqSizeExceeded = parent.mTotalOrderingReqSizeExceeded.ChildWith(labels)
		},
		"ord_reqsize_exceeded_total", "",
		metrics.Labels().RobotChannel,
		metrics.Labels().IsFirstAttempt); err != nil {
		return nil, err
	}

	if m.mBatchItemsCount, err = m.baseBus.AddHisto(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mBatchItemsCount = parent.mBatchItemsCount.ChildWith(labels)
		},
		"batch_tx_count", "",
		prometheus.DefBuckets,
		metrics.Labels().RobotChannel); err != nil {
		return nil, err
	}

	if m.mBatchCollectTime, err = m.baseBus.AddHisto(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mBatchCollectTime = parent.mBatchCollectTime.ChildWith(labels)
		},
		"batch_collect_duration_seconds", "",
		[]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5},
		metrics.Labels().RobotChannel); err != nil {
		return nil, err
	}

	if m.mAppInfo, err = m.baseBus.AddCounter(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mAppInfo = parent.mAppInfo.ChildWith(labels)
		},
		"app_info", "Application info",
		metrics.Labels().AppVer,
		metrics.Labels().AppSdkFabricVer); err != nil {
		return nil, err
	}

	if m.mAppInitDuration, err = m.baseBus.AddGauge(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mAppInitDuration = parent.mAppInitDuration.ChildWith(labels)
		},
		"app_init_duration_seconds",
		"Application init time"); err != nil {
		return nil, err
	}

	if m.mTxWaitingCount, err = m.baseBus.AddGauge(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mTxWaitingCount = parent.mTxWaitingCount.ChildWith(labels)
		},
		"tx_waiting_process_count", "",
		metrics.Labels().RobotChannel,
		metrics.Labels().Channel); err != nil {
		return nil, err
	}

	if m.mHeightLedgerBlocks, err = m.baseBus.AddGauge(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mHeightLedgerBlocks = parent.mHeightLedgerBlocks.ChildWith(labels)
		},
		"height_ledger_blocks", "",
		metrics.Labels().RobotChannel); err != nil {
		return nil, err
	}

	if m.mCollectorProcessBlockNum, err = m.baseBus.AddGauge(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mCollectorProcessBlockNum = parent.mCollectorProcessBlockNum.ChildWith(labels)
		},
		"collector_process_block_num", "",
		metrics.Labels().RobotChannel,
		metrics.Labels().Channel); err != nil {
		return nil, err
	}

	if m.mBlockTxCount, err = m.baseBus.AddHisto(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mBlockTxCount = parent.mBlockTxCount.ChildWith(labels)
		},
		"block_tx_count", "", prometheus.DefBuckets,
		metrics.Labels().RobotChannel,
		metrics.Labels().Channel); err != nil {
		return nil, err
	}

	if m.mTotalSrcChErrors, err = m.baseBus.AddCounter(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mTotalSrcChErrors = parent.mTotalSrcChErrors.ChildWith(labels)
		},
		"src_channel_errors_total", "",
		metrics.Labels().RobotChannel,
		metrics.Labels().Channel,
		metrics.Labels().IsFirstAttempt,
		metrics.Labels().IsSrcChClosed,
		metrics.Labels().IsTimeout,
	); err != nil {
		return nil, err
	}

	if m.mTotalRobotStarted, err = m.baseBus.AddCounter(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mTotalRobotStarted = parent.mTotalRobotStarted.ChildWith(labels)
		},
		"started_total", "",
		metrics.Labels().RobotChannel); err != nil {
		return nil, err
	}

	if m.mTotalRobotStopped, err = m.baseBus.AddCounter(
		func(ch, parent *MetricsBus, labels []basemetrics.Label) {
			ch.mTotalRobotStopped = parent.mTotalRobotStopped.ChildWith(labels)
		},
		"stopped_total", "",
		metrics.Labels().RobotChannel,
		metrics.Labels().IsErr,
		metrics.Labels().ErrType,
		metrics.Labels().Component); err != nil {
		return nil, err
	}

	l.Info("prometheus metrics created")
	return m, nil
}

// CreateChild creates a child metrics bus
func (m *MetricsBus) CreateChild(labels ...basemetrics.Label) metrics.Metrics {
	if len(labels) == 0 {
		return m
	}

	return m.baseBus.CreateChild(func(baseChildBus *baseprometheus.BaseMetricsBus[MetricsBus]) *MetricsBus {
		return &MetricsBus{
			log:     m.log,
			baseBus: baseChildBus,
		}
	}, m, labels...)
}

// TotalBatchExecuted returns a counter of total executed batches
func (m *MetricsBus) TotalBatchExecuted() basemetrics.Counter { return m.mTotalBatchExecuted }

// TotalExecutedTx returns a counter of total executed transactions
func (m *MetricsBus) TotalExecutedTx() basemetrics.Counter { return m.mTotalExecutedTx }

// AppInfo returns a counter of application info
func (m *MetricsBus) AppInfo() basemetrics.Counter { return m.mAppInfo }

// TotalRobotStarted returns a counter of total started robots
func (m *MetricsBus) TotalRobotStarted() basemetrics.Counter { return m.mTotalRobotStarted }

// TotalRobotStopped returns a counter of total stopped robots
func (m *MetricsBus) TotalRobotStopped() basemetrics.Counter { return m.mTotalRobotStopped }

// TotalBatchSize returns a counter of total batch size
func (m *MetricsBus) TotalBatchSize() basemetrics.Counter { return m.mTotalBatchSize }

// TotalOrderingReqSizeExceeded returns a counter of total ordering request size exceeded
func (m *MetricsBus) TotalOrderingReqSizeExceeded() basemetrics.Counter {
	return m.mTotalOrderingReqSizeExceeded
}

// TotalSrcChErrors returns a counter of total source channel errors
func (m *MetricsBus) TotalSrcChErrors() basemetrics.Counter { return m.mTotalSrcChErrors }

// BatchExecuteInvokeTime returns a histogram of batch execute invoke time
func (m *MetricsBus) BatchExecuteInvokeTime() basemetrics.Histogram { return m.mBatchExecuteInvokeTime }

// BatchSize returns a histogram of batch size
func (m *MetricsBus) BatchSize() basemetrics.Histogram { return m.mBatchSize }

// BatchItemsCount returns a histogram of batch items count
func (m *MetricsBus) BatchItemsCount() basemetrics.Histogram { return m.mBatchItemsCount }

// BatchCollectTime returns a histogram of batch collect time
func (m *MetricsBus) BatchCollectTime() basemetrics.Histogram { return m.mBatchCollectTime }

// BatchSizeEstimatedDiff returns a histogram of batch size estimated diff
func (m *MetricsBus) BatchSizeEstimatedDiff() basemetrics.Histogram { return m.mBatchSizeEstimatedDiff }

// BlockTxCount returns a histogram of block tx count
func (m *MetricsBus) BlockTxCount() basemetrics.Histogram { return m.mBlockTxCount }

// AppInitDuration returns a gauge of application init duration
func (m *MetricsBus) AppInitDuration() basemetrics.Gauge { return m.mAppInitDuration }

// TxWaitingCount returns a gauge of transaction waiting count
func (m *MetricsBus) TxWaitingCount() basemetrics.Gauge { return m.mTxWaitingCount }

// HeightLedgerBlocks returns a gauge of height ledger blocks
func (m *MetricsBus) HeightLedgerBlocks() basemetrics.Gauge { return m.mHeightLedgerBlocks }

// CollectorProcessBlockNum returns a gauge of collector process block num
func (m *MetricsBus) CollectorProcessBlockNum() basemetrics.Gauge { return m.mCollectorProcessBlockNum }
