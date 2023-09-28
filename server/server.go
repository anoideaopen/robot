package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/newity/glog"
)

const (
	serverShutdownTimeout = 3 * time.Second
	appInfoURLPattern     = "/info"
	metricsURLPattern     = "/metrics"
	livenessURLPattern    = "/healthz"
	readinessURLPattern   = "/readyz"
)

// StartServer starts http server with app info, metrics, liveness and readiness handlers
func StartServer(ctx context.Context, serverPort uint,
	appInfo *AppInfo, robots []string, metricsH http.Handler,
) (*LivenessMng, func(), error) {
	appInfoH, err := appInfoHandler(ctx, appInfo)
	if err != nil {
		return nil, nil, err
	}

	livenessH, readinessH, lm := healthChecksHandler(ctx, robots)

	stopServer := startServer(ctx, serverPort,
		appInfoH,
		metricsH,
		livenessH, readinessH)

	return lm, stopServer, nil
}

func startServer(ctx context.Context, serverPort uint,
	appInfoH, metricsH, livenessH, readinessH http.Handler,
) func() {
	log := glog.FromContext(ctx)

	router := http.NewServeMux()
	if appInfoH != nil {
		router.Handle(appInfoURLPattern, appInfoH)
		log.Infof("http server handle app info on %s", appInfoURLPattern)
	}

	if metricsH != nil {
		router.Handle(metricsURLPattern, metricsH)
		log.Infof("http server handle metrics on %s", metricsURLPattern)
	}

	if livenessH != nil {
		router.Handle(livenessURLPattern, livenessH)
		log.Infof("http server handle liveness on %s", livenessURLPattern)
	}

	if readinessH != nil {
		router.Handle(readinessURLPattern, readinessH)
		log.Infof("http server handle readiness on %s", readinessURLPattern)
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", serverPort),
		Handler:           router,
		ReadHeaderTimeout: 0,
	}

	go func() {
		log.Info("server ListenAndServe starting")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorf("server ListenAndServe error: %s", err)
		}
		log.Info("server ListenAndServe finished")
	}()

	return func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Errorf("server Shutdown  error: %s", err)
		}
	}
}
