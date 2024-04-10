package metrics

import "github.com/anoideaopen/common-component/basemetrics"

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

func Labels() LabelNames {
	return allLabels
}

const (
	TxTypeTx           = "tx"
	TxTypeSwap         = "swap"
	TxTypeMultiSwap    = "mswap"
	TxTypeSwapKey      = "swapkey"
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
