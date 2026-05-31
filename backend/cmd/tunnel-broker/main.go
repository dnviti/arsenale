package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/catalog"
	"github.com/dnviti/arsenale/backend/internal/storage"
	"github.com/dnviti/arsenale/backend/internal/tunnelbroker"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func main() {
	ctx := context.Background()

	db, err := storage.OpenPostgres(ctx)
	if err != nil {
		panic(err)
	}
	if db != nil {
		defer db.Close()
	}

	key, err := tunnelbroker.LoadServerEncryptionKey()
	if err != nil {
		panic(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	quicTLS, err := loadQUICTLSConfig()
	if err != nil {
		panic(err)
	}

	broker := tunnelbroker.NewBroker(tunnelbroker.BrokerConfig{
		Store:               tunnelbroker.NewPostgresStore(db),
		Logger:              logger,
		ServerEncryptionKey: key,
		SpiffeTrustDomain:   getenv("SPIFFE_TRUST_DOMAIN", "arsenale.local"),
		ProxyBindHost:       getenv("TUNNEL_TCP_PROXY_BIND_HOST", "0.0.0.0"),
		ProxyAdvertiseHost:  getenv("TUNNEL_TCP_PROXY_ADVERTISE_HOST", "tunnel-broker"),
		QUICTLSConfig:       quicTLS,
		// QUIC listens on the broker's port over UDP, mirroring the HTTP control
		// surface on TCP (the HTTP/3-alongside-HTTP convention).
		QUICListenAddr: getenv("TUNNEL_QUIC_LISTEN_ADDR", ":8092"),
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// QUIC tunnel listener runs alongside the HTTP control surface. It is a
	// no-op unless a server certificate is configured, so the broker stays
	// WebSocket-only by default.
	if quicTLS != nil {
		go func() {
			if err := broker.ListenQUIC(ctx); err != nil {
				logger.Error("tunnel QUIC listener stopped", "error", err)
			}
		}()
	}

	service := app.StaticService{
		Descriptor: catalog.MustService(contracts.ServiceTunnelBroker),
		Register: func(mux *http.ServeMux) {
			broker.RegisterRoutes(mux)
		},
	}
	if err := app.Run(ctx, service); err != nil {
		panic(err)
	}
}

// loadQUICTLSConfig builds the broker's QUIC server TLS config from the
// environment. QUIC is enabled only when both a server certificate and key are
// provided (PEM via *_PEM env or *_FILE path); otherwise it returns nil and the
// broker serves WebSocket tunnels exclusively.
func loadQUICTLSConfig() (*tls.Config, error) {
	certPEM, err := readPEMEnv("TUNNEL_QUIC_SERVER_CERT", "TUNNEL_QUIC_SERVER_CERT_FILE")
	if err != nil {
		return nil, err
	}
	keyPEM, err := readPEMEnv("TUNNEL_QUIC_SERVER_KEY", "TUNNEL_QUIC_SERVER_KEY_FILE")
	if err != nil {
		return nil, err
	}
	if certPEM == "" || keyPEM == "" {
		return nil, nil
	}
	cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		return nil, fmt.Errorf("load QUIC server certificate: %w", err)
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}

func readPEMEnv(valueKey, fileKey string) (string, error) {
	if value := os.Getenv(valueKey); value != "" {
		return value, nil
	}
	if path := os.Getenv(fileKey); path != "" {
		payload, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", fileKey, err)
		}
		return string(payload), nil
	}
	return "", nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
