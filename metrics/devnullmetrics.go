package metrics

import "github.com/atomyze-foundation/common-component/basemetrics"

type devNullMetrics struct{}

func (m *devNullMetrics) Inc(_ ...basemetrics.Label) { /* nothing just stub */ }

func (m *devNullMetrics) Add(_ float64, _ ...basemetrics.Label) { /* nothing just stub */ }

func (m *devNullMetrics) Set(_ float64, _ ...basemetrics.Label) { /* nothing just stub */ }

func (m *devNullMetrics) Observe(_ float64, _ ...basemetrics.Label) { /* nothing just stub */ }

func (m *devNullMetrics) TotalBatchExecuted() basemetrics.Counter { return m }

func (m *devNullMetrics) TotalExecutedTx() basemetrics.Counter { return m }

func (m *devNullMetrics) AppInfo() basemetrics.Counter { return m }

func (m *devNullMetrics) TotalRobotStarted() basemetrics.Counter { return m }

func (m *devNullMetrics) TotalRobotStopped() basemetrics.Counter { return m }

func (m *devNullMetrics) TotalBatchSize() basemetrics.Counter { return m }

func (m *devNullMetrics) TotalOrderingReqSizeExceeded() basemetrics.Counter { return m }

func (m *devNullMetrics) BatchExecuteInvokeTime() basemetrics.Histogram { return m }

func (m *devNullMetrics) BatchSize() basemetrics.Histogram { return m }

func (m *devNullMetrics) BatchItemsCount() basemetrics.Histogram { return m }

func (m *devNullMetrics) BatchCollectTime() basemetrics.Histogram { return m }

func (m *devNullMetrics) BatchSizeEstimatedDiff() basemetrics.Histogram { return m }

func (m *devNullMetrics) BlockTxCount() basemetrics.Histogram { return m }

func (m *devNullMetrics) AppInitDuration() basemetrics.Gauge { return m }

func (m *devNullMetrics) TxWaitingCount() basemetrics.Gauge { return m }

func (m *devNullMetrics) HeightLedgerBlocks() basemetrics.Gauge { return m }

func (m *devNullMetrics) CollectorProcessBlockNum() basemetrics.Gauge { return m }

func (m *devNullMetrics) TotalSrcChErrors() basemetrics.Counter { return m }

func (m *devNullMetrics) CreateChild(_ ...basemetrics.Label) Metrics { return m }
