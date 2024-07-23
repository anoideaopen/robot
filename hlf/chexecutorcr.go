package hlf

import (
	"context"

	"github.com/anoideaopen/robot/config"
	"github.com/anoideaopen/robot/hlf/hlfprofile"
)

type ChExecutorCreator = func(ctx context.Context) (*ChExecutor, error)

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
	return func(ctx context.Context) (*ChExecutor, error) {
		return createChExecutor(ctx, chName,
			connectionProfile,
			user, org,
			execOpts,
		)
	}
}

func mapExecOpts(cfg *config.Config, rCfg *config.Robot) (ExecuteOptions, error) {
	execTimeout, err := rCfg.ExecOpts.EffExecuteTimeout(cfg.DefaultRobotExecOpts)
	if err != nil {
		return ExecuteOptions{}, err
	}

	return ExecuteOptions{
		ExecuteTimeout: execTimeout,
	}, nil
}

// CreateChExecutorCreator - creates ChExecutorCreator specified by config
func CreateChExecutorCreator(cfg *config.Config, hlfProfile *hlfprofile.HlfProfile,
	rCfg *config.Robot,
) (ChExecutorCreator, error) {
	execOpts, err := mapExecOpts(cfg, rCfg)
	if err != nil {
		return nil, err
	}

	return NewChExecutorCreator(rCfg.ChName, cfg.ProfilePath,
		cfg.UserName, hlfProfile.OrgName, execOpts), nil
}
