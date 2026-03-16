package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/bridge"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := newLogger()

	config, err := bridge.LoadConfig()
	if err != nil {
		logger.WithError(err).Fatal("load bridge config")
	}

	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	runtimeBridge, err := bridge.NewBridge(config, logger)
	if err != nil {
		logger.WithError(err).Fatal("create bridge")
	}

	ctx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	if err := runtimeBridge.Start(ctx); err != nil {
		logger.WithError(err).Fatal("start bridge")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler(runtimeBridge.CheckHealth))

	server := &http.Server{
		Addr:              httpAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	logger.WithField("addr", httpAddr).Info("health endpoint listening")
	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.WithError(err).Fatal("http server failed")
	}

	if err := runtimeBridge.Stop(); err != nil {
		logger.WithError(err).Error("bridge shutdown completed with error")
	}
}

func newLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	logger.SetOutput(os.Stdout)

	level := strings.TrimSpace(strings.ToLower(os.Getenv("LOG_LEVEL")))
	if level == "" {
		level = "info"
	}
	parsed, err := logrus.ParseLevel(level)
	if err != nil {
		parsed = logrus.InfoLevel
	}
	logger.SetLevel(parsed)

	return logger
}

func healthHandler(getStatus func(context.Context) bridge.BridgeHealthStatus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		status := getStatus(r.Context())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpStatusForHealth(status.Health))
		_ = json.NewEncoder(w).Encode(status)
	}
}

func httpStatusForHealth(health string) int {
	switch health {
	case bridge.HealthOK:
		return http.StatusOK
	case bridge.HealthWarning:
		return 210
	default:
		return http.StatusServiceUnavailable
	}
}
