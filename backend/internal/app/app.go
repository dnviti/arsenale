package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/dnviti/arsenale/backend/internal/catalog"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

type Service interface {
	Metadata() contracts.ServiceMetadata
	RegisterRoutes(*http.ServeMux)
}

type StaticService struct {
	Descriptor contracts.ServiceMetadata
	Register   func(*http.ServeMux)
}

func (s StaticService) Metadata() contracts.ServiceMetadata {
	return s.Descriptor
}

func (s StaticService) RegisterRoutes(mux *http.ServeMux) {
	if s.Register != nil {
		s.Register(mux)
	}
}

func Run(ctx context.Context, service Service) error {
	meta := service.Metadata()
	addr := listenAddr(meta.DefaultPort)
	version := getenv("ARSENALE_VERSION", "dev")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	mux := http.NewServeMux()
	registerMetaRoutes(mux, meta, version)
	service.RegisterRoutes(mux)

	server := &http.Server{
		Addr:              addr,
		Handler:           loggingMiddleware(logger, mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	stopCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("service starting", "name", meta.Name, "addr", addr)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-stopCtx.Done():
		logger.Info("service shutting down", "name", meta.Name)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func registerMetaRoutes(mux *http.ServeMux, meta contracts.ServiceMetadata, version string) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"service": meta.Name,
			"plane":   meta.Plane,
		})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]any{
			"status":  "ready",
			"service": meta.Name,
		})
	})
	mux.HandleFunc("GET /v1/meta/service", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]any{
			"version": version,
			"service": meta,
		})
	})
	mux.HandleFunc("GET /v1/meta/architecture", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, catalog.Manifest(version))
	})
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to encode response: %v"}`, err), http.StatusInternalServerError)
	}
}

func ReadJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func ErrorJSON(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]any{
		"error": message,
	})
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start).String())
	})
}

func listenAddr(defaultPort int) string {
	host := getenv("HOST", "0.0.0.0")
	port := getenvInt("PORT", defaultPort)
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}
