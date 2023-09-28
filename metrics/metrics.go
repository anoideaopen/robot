package metrics

import "github.com/atomyze-foundation/common-component/basemetrics"

// LabelNames is a struct with label names
type LabelNames struct {
	TxType          basemetrics.LabelName
	RobotChannel    basemetrics.LabelName
	Channel         basemetrics.LabelName
	AppVer          basemetrics.LabelName
	AppSdkFabricVer basemetrics.LabelName
	ErrType         basemetrics.LabelName
	Component       basemetrics.LabelName
	IsErr           basemetrics.LabelName
	IsFirstAttempt  basemetrics.LabelName
	IsSrcChClosed   basemetrics.LabelName
	IsTimeout       basemetrics.LabelName
}

// Labels returns all label names
func Labels() LabelNames {
	return allLabels
}

const (
	// TxTypeTx is a type of transaction
	TxTypeTx = "tx"
	// TxTypeSwap is a type of swap transaction
	TxTypeSwap = "swap"
	// TxTypeMultiSwap is a type of multi swap transaction
	TxTypeMultiSwap = "mswap"
	// TxTypeSwapKey is a type of swap key transaction
	TxTypeSwapKey = "swapkey"
	// TxTypeMultiSwapKey is a type of multi swap key transaction
	TxTypeMultiSwapKey = "mswapkey"
)

var allLabels = createLabels()

func createLabels() LabelNames {
	return LabelNames{
		RobotChannel:    "robot",
		Channel:         "channel",
		AppVer:          "ver",
		AppSdkFabricVer: "ver_sdk_fabric",
		ErrType:         "err_type",
		Component:       "component",
		TxType:          "txtype",
		IsErr:           "iserr",
		IsFirstAttempt:  "is_first_attempt",
		IsSrcChClosed:   "is_src_ch_closed",
		IsTimeout:       "is_timeout",
	}
}

// Metrics is a metrics interface for robot
type Metrics interface { //nolint:interfacebloat
	TotalBatchExecuted() basemetrics.Counter
	TotalExecutedTx() basemetrics.Counter
	AppInfo() basemetrics.Counter
	TotalRobotStarted() basemetrics.Counter
	TotalRobotStopped() basemetrics.Counter
	TotalBatchSize() basemetrics.Counter
	TotalOrderingReqSizeExceeded() basemetrics.Counter
	TotalSrcChErrors() basemetrics.Counter

	BatchExecuteInvokeTime() basemetrics.Histogram
	BatchSize() basemetrics.Histogram
	BatchItemsCount() basemetrics.Histogram
	BatchCollectTime() basemetrics.Histogram
	BatchSizeEstimatedDiff() basemetrics.Histogram
	BlockTxCount() basemetrics.Histogram

	AppInitDuration() basemetrics.Gauge
	TxWaitingCount() basemetrics.Gauge
	HeightLedgerBlocks() basemetrics.Gauge
	CollectorProcessBlockNum() basemetrics.Gauge

	CreateChild(labels ...basemetrics.Label) Metrics
}
