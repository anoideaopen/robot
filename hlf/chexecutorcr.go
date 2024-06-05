package hlf

import (
	"context"
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
		execOpts)
}

func newChExecutorCreator(chName,
	connectionProfile,
	user, org string,
	execOpts ExecuteOptions,
) ChExecutorCreator {
	return func(ctx context.Context) (*chExecutor, error) {
		return createChExecutor(ctx, chName,
			connectionProfile,
			user, org,
			execOpts,
		)
	}
}
