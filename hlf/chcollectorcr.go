package hlf

import (
	"context"

	"github.com/atomyze-foundation/cartridge/manager"
	"github.com/atomyze-foundation/robot/dto/parserdto"
)

// ChCollectorCreator is a type for channel collector creator
type ChCollectorCreator func(ctx context.Context,
	dataReady chan<- struct{},
	srcChName string, startFrom uint64) (*chCollector, error)

// NewChCollectorCreatorWithCryptoMgr creates a new ChCollectorCreator with crypto manager
func NewChCollectorCreatorWithCryptoMgr(
	dstChName,
	connectionProfile, userName, orgName string,
	txPrefixes parserdto.TxPrefixes,
	cryptoManager manager.Manager,
	bufSize uint,
) ChCollectorCreator {
	return newChCollectorCreator(dstChName,
		connectionProfile,
		userName, orgName,
		txPrefixes,
		cryptoManager,
		bufSize)
}

// NewChCollectorCreator creates a new ChCollectorCreator
func NewChCollectorCreator(
	dstChName, connectionProfile,
	userName, orgName string,
	txPrefixes parserdto.TxPrefixes,
	bufSize uint,
) ChCollectorCreator {
	return newChCollectorCreator(dstChName,
		connectionProfile,
		userName, orgName,
		txPrefixes,
		nil,
		bufSize)
}

func newChCollectorCreator(
	dstChName,
	connectionProfile,
	userName, orgName string,
	txPrefixes parserdto.TxPrefixes,
	cryptoManager manager.Manager,
	bufSize uint,
) ChCollectorCreator {
	return func(ctx context.Context, dataReady chan<- struct{}, srcChName string, startFrom uint64) (*chCollector, error) {
		return createChCollector(ctx,
			dstChName, srcChName,
			dataReady, startFrom, bufSize,
			connectionProfile, userName, orgName,
			txPrefixes,
			cryptoManager)
	}
}
