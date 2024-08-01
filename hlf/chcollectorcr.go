package hlf

import (
	"context"

	"github.com/anoideaopen/robot/config"
	"github.com/anoideaopen/robot/dto/parserdto"
	"github.com/anoideaopen/robot/hlf/hlfprofile"
)

type ChCollectorCreator func(ctx context.Context,
	dataReady chan<- struct{},
	srcChName string, startFrom uint64) (*ChCollector, error)

func NewChCollectorCreator(
	dstChName,
	connectionProfile,
	userName,
	orgName string,
	txPrefixes parserdto.TxPrefixes,
	bufSize uint,
) ChCollectorCreator {
	return func(ctx context.Context, dataReady chan<- struct{}, srcChName string, startFrom uint64) (*ChCollector, error) {
		return createChCollector(ctx,
			dstChName, srcChName,
			dataReady, startFrom, bufSize,
			connectionProfile, userName, orgName,
			txPrefixes)
	}
}

// CreateChCollectorCreatorFromConfig creates ChCollectorCreator specified by robot config
func CreateChCollectorCreatorFromConfig(
	cfg *config.Config,
	hlfProfile *hlfprofile.HlfProfile,
	rCfg *config.Robot,
) ChCollectorCreator {
	txPrefixes := parserdto.TxPrefixes{
		Tx:        cfg.TxPreimagePrefix,
		Swap:      cfg.TxSwapPrefix,
		MultiSwap: cfg.TxMultiSwapPrefix,
	}
	return NewChCollectorCreator(
		rCfg.ChName, cfg.ProfilePath, cfg.UserName, hlfProfile.OrgName,
		txPrefixes,
		rCfg.CollectorsBufSize)
}
