package hlf

import (
	"context"

	"github.com/anoideaopen/cartridge/manager"
)

type ChExecutorCreator = func(ctx context.Context) (*chExecutor, error)

func NewChExecutorCreator(chName,
	connectionProfile,
	user, org string,
	execOpts ExecuteOptions,
) ChExecutorCreator {
	return newChExecutorCreator(chName,
		connectionProfile,
		user, org,
		execOpts,
		nil)
}

func NewChExecutorCreatorWithCryptoMgr(chName,
	connectionProfile,
	user, org string,
	execOpts ExecuteOptions,
	cryptoManager manager.Manager,
) ChExecutorCreator {
	return newChExecutorCreator(chName,
		connectionProfile,
		user, org,
		execOpts,
		cryptoManager)
}

func newChExecutorCreator(chName,
	connectionProfile,
	user, org string,
	execOpts ExecuteOptions,
	cryptoManager manager.Manager,
) ChExecutorCreator {
	return func(ctx context.Context) (*chExecutor, error) {
		return createChExecutor(ctx, chName,
			connectionProfile,
			user, org,
			execOpts,
			cryptoManager,
		)
	}
}
