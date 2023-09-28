package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/atomyze-foundation/cartridge/manager"
	"github.com/atomyze-foundation/common-component/basemetrics/baseprometheus"
	"github.com/atomyze-foundation/common-component/errorshlp"
	"github.com/atomyze-foundation/common-component/loggerhlp"
	"github.com/atomyze-foundation/robot/chrobot"
	"github.com/atomyze-foundation/robot/collectorbatch"
	"github.com/atomyze-foundation/robot/config"
	"github.com/atomyze-foundation/robot/dto/parserdto"
	"github.com/atomyze-foundation/robot/helpers/nerrors"
	"github.com/atomyze-foundation/robot/hlf"
	"github.com/atomyze-foundation/robot/hlf/hlfprofile"
	"github.com/atomyze-foundation/robot/logger"
	"github.com/atomyze-foundation/robot/metrics"
	"github.com/atomyze-foundation/robot/metrics/prometheus"
	"github.com/atomyze-foundation/robot/server"
	"github.com/atomyze-foundation/robot/storage/redis"
	"github.com/newity/glog"
	"github.com/pkg/errors"
)

// AppInfoVer is a version of the application
var AppInfoVer = "undefined-ver"

//nolint:funlen
func main() {
	startTime := time.Now()

	cfg, err := config.GetConfig()
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	hlfProfile, err := hlfprofile.ParseProfile(cfg.ProfilePath)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	l, err := createLogger(cfg, hlfProfile)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	cfgForLog, _ := json.MarshalIndent(cfg.WithoutSensitiveData(), "", "\t")
	l.Infof("Version: %s", AppInfoVer)
	l.Infof("Robot config: \n%s\n", cfgForLog)

	ctx, cancel := context.WithCancel(glog.NewContext(context.Background(), l))

	cryptoMng, err := createCryptoManager(ctx, cfg, hlfProfile)
	if err != nil {
		l.Panicf("%+v", err)
	}

	// metrics
	var metricsH http.Handler
	if cfg.PromMetrics != nil {
		m, err := prometheus.NewMetrics(ctx, cfg.PromMetrics.PrefixForMetrics)
		if err != nil {
			l.Panicf("%+v", err)
		}
		m.AppInfo().Inc(
			metrics.Labels().AppVer.Create(AppInfoVer),
			metrics.Labels().AppSdkFabricVer.Create(getFabricSdkVersion(l)),
		)
		ctx = metrics.NewContext(ctx, m)
		metricsH = baseprometheus.MetricsHandler(ctx)
		setInitMetricsVals(cfg, m)
	}

	robotsNames := make([]string, 0, len(cfg.Robots))
	for _, r := range cfg.Robots {
		robotsNames = append(robotsNames, r.ChName)
	}

	// start server for host app info, metrics and etc
	lm, shutdownServer, err := server.StartServer(ctx, cfg.ServerPort, &server.AppInfo{
		Ver:          AppInfoVer,
		VerSdkFabric: getFabricSdkVersion(l),
	}, robotsNames, metricsH)
	if err != nil {
		l.Panicf("%+v", err)
	}
	defer shutdownServer()

	// robots

	robots, err := createRobots(ctx, cfg, hlfProfile, cryptoMng)
	if err != nil {
		l.Panicf("%+v", err)
	}

	go func() {
		interruptCh := make(chan os.Signal, 1)
		signal.Notify(interruptCh, os.Interrupt, syscall.SIGTERM)
		s := <-interruptCh
		l.Infof("os signal %s received, shutdown", s)
		cancel()
	}()

	// run robots and wait them
	wg := sync.WaitGroup{}
	wg.Add(len(robots))

	for _, r := range robots {
		go func(r *chrobot.ChRobot) {
			defer wg.Done()
			runRobot(ctx, r, cfg.DelayAfterChRobotError, lm)
		}(r)
	}

	m := metrics.FromContext(ctx)
	m.AppInitDuration().Set(time.Since(startTime).Seconds())
	wg.Wait()
}

func createLogger(cfg *config.Config, hlfProfile *hlfprofile.HlfProfile) (glog.Logger, error) {
	l, err := loggerhlp.CreateLogger(cfg.LogType, cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	l = l.With(logger.Labels{
		Version:   AppInfoVer,
		UserName:  cfg.UserName,
		OrgName:   hlfProfile.OrgName,
		Component: logger.ComponentMain,
	}.Fields()...)

	return l, nil
}

func createRobots(ctx context.Context, cfg *config.Config, hlfProfile *hlfprofile.HlfProfile, cryptoManager manager.Manager) ([]*chrobot.ChRobot, error) {
	robots := make([]*chrobot.ChRobot, 0, len(cfg.Robots))
	for _, rCfg := range cfg.Robots {
		allSrcChannels := map[string]uint64{}
		for _, sc := range rCfg.SrcChannels {
			allSrcChannels[sc.ChName] = *sc.InitBlockNum
		}

		ccr := createChCollectorCreator(cfg, hlfProfile, rCfg, cryptoManager)
		ecr, err := createChExecutorCreator(cfg, hlfProfile, rCfg, cryptoManager)
		if err != nil {
			return nil, err
		}

		stor, err := redis.NewStorage(ctx, cfg.RedisStorage.Addr, cfg.RedisStorage.Password,
			cfg.RedisStorage.WithTLS, cfg.RedisStorage.RootCAs,
			cfg.RedisStorage.DBPrefix, rCfg.ChName)
		if err != nil {
			return nil, err
		}

		bLimits := cfg.DefaultBatchLimits
		if rCfg.BatchLimits != nil {
			bLimits = rCfg.BatchLimits
		}
		if bLimits == nil {
			return nil, errors.Errorf("no configuration for batch limits in %s robot", rCfg.ChName)
		}

		r := chrobot.NewRobot(ctx, rCfg.ChName, rCfg.InitMinExecBlockNum,
			allSrcChannels,
			func(ctx context.Context, dataReady chan<- struct{}, srcChName string, startFrom uint64) (chrobot.ChCollector, error) {
				return ccr(ctx, dataReady, srcChName, startFrom)
			},
			func(ctx context.Context) (chrobot.ChExecutor, error) {
				return ecr(ctx)
			},
			stor,
			collectorbatch.Limits{
				BlocksCountLimit: bLimits.BatchBlocksCountLimit,
				TimeoutLimit:     bLimits.BatchTimeoutLimit,
				LenLimit:         bLimits.BatchLenLimit,
				SizeLimit:        bLimits.BatchSizeLimit,
			})

		robots = append(robots, r)
	}

	return robots, nil
}

func createChCollectorCreator(cfg *config.Config, hlfProfile *hlfprofile.HlfProfile, rCfg *config.Robot,
	cryptoManager manager.Manager,
) hlf.ChCollectorCreator {
	txPrefixes := parserdto.TxPrefixes{
		Tx:        cfg.TxPreimagePrefix,
		Swap:      cfg.TxSwapPrefix,
		MultiSwap: cfg.TxMultiSwapPrefix,
	}
	if cryptoManager == nil {
		return hlf.NewChCollectorCreator(
			rCfg.ChName, cfg.ProfilePath, cfg.UserName, hlfProfile.OrgName,
			txPrefixes,
			rCfg.CollectorsBufSize)
	}
	return hlf.NewChCollectorCreatorWithCryptoMgr(
		rCfg.ChName, cfg.ProfilePath, cfg.UserName, hlfProfile.OrgName,
		txPrefixes, cryptoManager,
		rCfg.CollectorsBufSize)
}

func createChExecutorCreator(cfg *config.Config, hlfProfile *hlfprofile.HlfProfile, rCfg *config.Robot,
	cryptoManager manager.Manager,
) (hlf.ChExecutorCreator, error) {
	execOpts, err := mapExecOpts(cfg, rCfg)
	if err != nil {
		return nil, err
	}

	if cryptoManager == nil {
		return hlf.NewChExecutorCreator(rCfg.ChName, cfg.ProfilePath,
			cfg.UserName, hlfProfile.OrgName, cfg.UseSmartBFT, execOpts), nil
	}

	return hlf.NewChExecutorCreatorWithCryptoMgr(rCfg.ChName, cfg.ProfilePath,
		cfg.UserName, hlfProfile.OrgName, cfg.UseSmartBFT, execOpts, cryptoManager), nil
}

func runRobot(ctx context.Context, r *chrobot.ChRobot, delayAfterError time.Duration, lm *server.LivenessMng) {
	log := glog.FromContext(ctx)
	m := metrics.FromContext(ctx)
	m = m.CreateChild(
		metrics.Labels().RobotChannel.Create(r.ChName()),
	)

	for ctx.Err() == nil {
		log.Infof("robot for [%s] channel started", r.ChName())
		lm.SetRobotState(r.ChName(), server.RobotStarted)
		m.TotalRobotStarted().Inc()
		err := r.Run(ctx)

		if err == nil {
			log.Infof("robot for [%s] channel finished success", r.ChName())
			lm.SetRobotState(r.ChName(), server.RobotStopped)
			incTotalRobotStopped(m, nil)
			return
		}

		if ctx.Err() != nil && errors.Is(err, context.Canceled) {
			log.Infof("robot for [%s] channel finished after cancel context %+v", r.ChName(), err)
			lm.SetRobotState(r.ChName(), server.RobotStopped)
			incTotalRobotStopped(m, nil)
			return
		}

		log.Errorf("robot for [%s] channel finished with: %+v, repeat after delay", r.ChName(), err)
		lm.SetRobotState(r.ChName(), server.RobotStoppedWithErr)
		incTotalRobotStopped(m, err)
		select {
		case <-ctx.Done():
		case <-time.After(delayAfterError):
		}
	}

	log.Info("robot for [%s] channel finished after cancel context", r.ChName())
}

func incTotalRobotStopped(m metrics.Metrics, err error) {
	isErr := fmt.Sprintf("%v", err != nil)
	errType := nerrors.ErrTypeInternal
	componentName := nerrors.ComponentRobot
	dErr, ok := errorshlp.ExtractDetailsError(err)
	if ok {
		errType = dErr.Type
		componentName = dErr.Component
	}

	m.TotalRobotStopped().Inc(
		metrics.Labels().IsErr.Create(isErr),
		metrics.Labels().ErrType.Create(string(errType)),
		metrics.Labels().Component.Create(string(componentName)),
	)
}

func getFabricSdkVersion(log glog.Logger) string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		log.Warning("Failed to read build info")
		return ""
	}

	var m *debug.Module
	for _, dep := range bi.Deps {
		if strings.HasSuffix(dep.Path, "fabric-sdk-go") {
			if dep.Replace != nil {
				m = dep.Replace
			} else {
				m = dep
			}
		}
	}

	if m != nil {
		return fmt.Sprintf("%s %s", m.Path, m.Version)
	}

	return ""
}

//nolint:funlen
func setInitMetricsVals(cfg *config.Config, m *prometheus.MetricsBus) {
	errCases := []string{"true", "false"}
	txTypes := []string{
		metrics.TxTypeTx,
		metrics.TxTypeSwap,
		metrics.TxTypeMultiSwap,
		metrics.TxTypeSwapKey,
		metrics.TxTypeMultiSwapKey,
	}
	errTypes := []errorshlp.ErrType{
		nerrors.ErrTypeHlf,
		nerrors.ErrTypeRedis,
		nerrors.ErrTypeParsing,
		nerrors.ErrTypeInternal,
	}
	components := []errorshlp.ComponentName{
		nerrors.ComponentStorage,
		nerrors.ComponentCollector,
		nerrors.ComponentExecutor,
		nerrors.ComponentRobot,
		nerrors.ComponentParser,
		nerrors.ComponentBatch,
	}
	isFirstAttemptCases := []string{"true", "false"}
	isSrcChClosedCases := []string{"true", "false"}
	isTimeoutCases := []string{"true", "false"}
	for _, r := range cfg.Robots {
		for _, isErrVal := range errCases {
			m.TotalBatchExecuted().Add(0,
				metrics.Labels().RobotChannel.Create(r.ChName),
				metrics.Labels().IsErr.Create(isErrVal))
			for _, txType := range txTypes {
				m.TotalExecutedTx().Add(0,
					metrics.Labels().RobotChannel.Create(r.ChName),
					metrics.Labels().TxType.Create(txType))
			}
			for _, errType := range errTypes {
				for _, component := range components {
					m.TotalRobotStopped().Add(0,
						metrics.Labels().RobotChannel.Create(r.ChName),
						metrics.Labels().IsErr.Create(isErrVal),
						metrics.Labels().ErrType.Create(string(errType)),
						metrics.Labels().Component.Create(string(component)))
				}
			}
		}
		m.TotalRobotStarted().Add(0, metrics.Labels().RobotChannel.Create(r.ChName))
		m.TotalBatchSize().Add(0, metrics.Labels().RobotChannel.Create(r.ChName))

		for _, isFirstAttempt := range isFirstAttemptCases {
			m.TotalOrderingReqSizeExceeded().Add(0,
				metrics.Labels().RobotChannel.Create(r.ChName),
				metrics.Labels().IsFirstAttempt.Create(isFirstAttempt),
			)
		}

		m.HeightLedgerBlocks().Set(0, metrics.Labels().RobotChannel.Create(r.ChName))

		for _, src := range r.SrcChannels {
			m.BlockTxCount().Observe(0,
				metrics.Labels().RobotChannel.Create(r.ChName),
				metrics.Labels().Channel.Create(src.ChName))
			m.TxWaitingCount().Set(0,
				metrics.Labels().RobotChannel.Create(r.ChName),
				metrics.Labels().Channel.Create(src.ChName))
			m.CollectorProcessBlockNum().Set(0,
				metrics.Labels().RobotChannel.Create(r.ChName),
				metrics.Labels().Channel.Create(src.ChName))

			for _, isFirstAttempt := range isFirstAttemptCases {
				for _, isSrcChClosed := range isSrcChClosedCases {
					for _, isTimeout := range isTimeoutCases {
						m.TotalSrcChErrors().Add(0,
							metrics.Labels().RobotChannel.Create(r.ChName),
							metrics.Labels().Channel.Create(src.ChName),
							metrics.Labels().IsFirstAttempt.Create(isFirstAttempt),
							metrics.Labels().IsSrcChClosed.Create(isSrcChClosed),
							metrics.Labels().IsTimeout.Create(isTimeout),
						)
					}
				}
			}
		}
	}
}

func mapExecOpts(cfg *config.Config, rCfg *config.Robot) (hlf.ExecuteOptions, error) {
	execTimeout, err := rCfg.ExecOpts.EffExecuteTimeout(cfg.DefaultRobotExecOpts)
	if err != nil {
		return hlf.ExecuteOptions{}, err
	}

	waitCommitAttempts, err := rCfg.ExecOpts.EffWaitCommitAttempts(cfg.DefaultRobotExecOpts)
	if err != nil {
		return hlf.ExecuteOptions{}, err
	}

	waitCommitAttemptTimeout, err := rCfg.ExecOpts.EffWaitCommitAttemptTimeout(cfg.DefaultRobotExecOpts)
	if err != nil {
		return hlf.ExecuteOptions{}, err
	}

	return hlf.ExecuteOptions{
		ExecuteTimeout:           execTimeout,
		WaitCommitAttempts:       waitCommitAttempts,
		WaitCommitAttemptTimeout: waitCommitAttemptTimeout,
	}, nil
}
