package prometheus

import (
	"context"

	"github.com/anoideaopen/common-component/basemetrics"
	"github.com/anoideaopen/common-component/basemetrics/baseprometheus"
	"github.com/anoideaopen/robot/metrics"
	"github.com/newity/glog"
	"github.com/prometheus/client_golang/prometheus"
)

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

func (m *MetricsBus) TotalBatchExecuted() basemetrics.Counter { return m.mTotalBatchExecuted }
func (m *MetricsBus) TotalExecutedTx() basemetrics.Counter    { return m.mTotalExecutedTx }
func (m *MetricsBus) AppInfo() basemetrics.Counter            { return m.mAppInfo }
func (m *MetricsBus) TotalRobotStarted() basemetrics.Counter  { return m.mTotalRobotStarted }
func (m *MetricsBus) TotalRobotStopped() basemetrics.Counter  { return m.mTotalRobotStopped }
func (m *MetricsBus) TotalBatchSize() basemetrics.Counter     { return m.mTotalBatchSize }
func (m *MetricsBus) TotalOrderingReqSizeExceeded() basemetrics.Counter {
	return m.mTotalOrderingReqSizeExceeded
}
func (m *MetricsBus) TotalSrcChErrors() basemetrics.Counter { return m.mTotalSrcChErrors }

func (m *MetricsBus) BatchExecuteInvokeTime() basemetrics.Histogram { return m.mBatchExecuteInvokeTime }
func (m *MetricsBus) BatchSize() basemetrics.Histogram              { return m.mBatchSize }
func (m *MetricsBus) BatchItemsCount() basemetrics.Histogram        { return m.mBatchItemsCount }
func (m *MetricsBus) BatchCollectTime() basemetrics.Histogram       { return m.mBatchCollectTime }
func (m *MetricsBus) BatchSizeEstimatedDiff() basemetrics.Histogram { return m.mBatchSizeEstimatedDiff }
func (m *MetricsBus) BlockTxCount() basemetrics.Histogram           { return m.mBlockTxCount }

func (m *MetricsBus) AppInitDuration() basemetrics.Gauge          { return m.mAppInitDuration }
func (m *MetricsBus) TxWaitingCount() basemetrics.Gauge           { return m.mTxWaitingCount }
func (m *MetricsBus) HeightLedgerBlocks() basemetrics.Gauge       { return m.mHeightLedgerBlocks }
func (m *MetricsBus) CollectorProcessBlockNum() basemetrics.Gauge { return m.mCollectorProcessBlockNum }
