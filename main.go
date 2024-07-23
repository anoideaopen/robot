package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/anoideaopen/common-component/basemetrics/baseprometheus"
	"github.com/anoideaopen/common-component/errorshlp"
	"github.com/anoideaopen/glog"
	"github.com/anoideaopen/robot/chrobot"
	"github.com/anoideaopen/robot/config"
	"github.com/anoideaopen/robot/helpers/nerrors"
	"github.com/anoideaopen/robot/hlf/hlfprofile"
	"github.com/anoideaopen/robot/logger"
	"github.com/anoideaopen/robot/metrics"
	"github.com/anoideaopen/robot/metrics/prometheus"
	"github.com/anoideaopen/robot/server"
)

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

	l, err := logger.New(cfg, hlfProfile, AppInfoVer)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	cfgForLog, _ := json.MarshalIndent(cfg.WithoutSensitiveData(), "", "\t")
	l.Infof("Version: %s", AppInfoVer)
	l.Infof("Robot config: \n%s\n", cfgForLog)

	ctx, cancel := context.WithCancel(glog.NewContext(context.Background(), l))

	// metrics
	var metricsH http.Handler
	if cfg.PromMetrics != nil {
		m, err := prometheus.NewMetrics(ctx, cfg.PromMetrics.PrefixForMetrics)
		if err != nil {
			panic(err)
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
		panic(err)
	}
	defer shutdownServer()

	// robots

	robots, err := chrobot.CreateRobots(ctx, cfg, hlfProfile)
	if err != nil {
		panic(err)
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
	dur := time.Since(startTime)
	m.AppInitDuration().Set(dur.Seconds())
	l.Infof("Robot started, time - %s\n", dur.String())
	wg.Wait()
}

func runRobot(ctx context.Context, r *chrobot.ChRobot, delayAfterError time.Duration, lm *server.LivenessMng) {
	log := glog.FromContext(ctx)
	m := metrics.FromContext(ctx)
	m = m.CreateChild(
		metrics.Labels().RobotChannel.Create(r.ChName()),
	)

	for ctx.Err() == nil {
		log.Debugf("robot for [%s] channel started", r.ChName())
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

	log.Infof("robot for [%s] channel finished after cancel context", r.ChName())
}

func incTotalRobotStopped(m metrics.Metrics, err error) {
	isErr := strconv.FormatBool(err != nil)
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
