package hlf

import (
	"context"

	"github.com/anoideaopen/cartridge/manager"
	"github.com/anoideaopen/robot/dto/parserdto"
)

type ChCollectorCreator func(ctx context.Context,
	dataReady chan<- struct{},
	srcChName string, startFrom uint64) (*chCollector, error)

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
