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

	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/auth"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/bridge"
	"github.com/flaviostutz/appo/stutzthings/stutzthings-server/operations"
	"github.com/sirupsen/logrus"
)

const defaultJWTSigningSecretBase64 = "ZGV2X2p3dF9zZWNyZXRfbG9jYWxfb25seV9wbGVhc2VfY2hhbmdl"

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

	jwtSecret, err := auth.DecodeBase64Secret(envOrDefault("JWT_SIGNING_SECRET_BASE64", defaultJWTSigningSecretBase64))
	if err != nil {
		logger.WithError(err).Fatal("load jwt signing secret")
	}
	jwtTTL, err := time.ParseDuration(envOrDefault("JWT_TOKEN_TTL", "24h"))
	if err != nil {
		logger.WithError(err).Fatal("parse jwt token ttl")
	}
	signer, err := auth.NewSigner(jwtSecret, envOrDefault("JWT_ISSUER", "stutzthings-server-dev"), jwtTTL)
	if err != nil {
		logger.WithError(err).Fatal("create jwt signer")
	}
	service := operations.NewService(runtimeBridge, runtimeBridge, signer)
	authenticator := auth.NewBearerAuthenticator(jwtSecret)

	server := &http.Server{
		Addr:              httpAddr,
		Handler:           newHTTPHandler(service, authenticator, runtimeBridge.CheckHealth, baseURLForAddr(httpAddr)),
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

func envOrDefault(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func baseURLForAddr(addr string) string {
	trimmed := strings.TrimSpace(addr)
	switch {
	case strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://"):
		return trimmed
	case strings.HasPrefix(trimmed, ":"):
		return "http://127.0.0.1" + trimmed
	default:
		return "http://" + trimmed
	}
}
