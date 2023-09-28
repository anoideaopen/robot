package hlf

import (
	"context"

	"github.com/atomyze-foundation/cartridge/manager"
)

// ChExecutorCreator is a type for channel executor creator
type ChExecutorCreator = func(ctx context.Context) (*chExecutor, error)

// NewChExecutorCreator creates a new ChExecutorCreator
func NewChExecutorCreator(chName,
	connectionProfile,
	user, org string,
	useSmartBFT bool,
	execOpts ExecuteOptions,
) ChExecutorCreator {
	return newChExecutorCreator(chName,
		connectionProfile,
		user, org,
		useSmartBFT,
		execOpts,
		nil)
}

// NewChExecutorCreatorWithCryptoMgr creates a new ChExecutorCreator with crypto manager
func NewChExecutorCreatorWithCryptoMgr(chName,
	connectionProfile,
	user, org string,
	useSmartBFT bool,
	execOpts ExecuteOptions,
	cryptoManager manager.Manager,
) ChExecutorCreator {
	return newChExecutorCreator(chName,
		connectionProfile,
		user, org,
		useSmartBFT,
		execOpts,
		cryptoManager)
}

func newChExecutorCreator(chName,
	connectionProfile,
	user, org string,
	useSmartBFT bool,
	execOpts ExecuteOptions,
	cryptoManager manager.Manager,
) ChExecutorCreator {
	return func(ctx context.Context) (*chExecutor, error) {
		return createChExecutor(ctx, chName,
			connectionProfile,
			user, org,
			useSmartBFT,
			execOpts,
			cryptoManager,
		)
	}
}
