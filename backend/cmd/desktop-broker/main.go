package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/catalog"
	"github.com/dnviti/arsenale/backend/internal/desktopbroker"
	"github.com/dnviti/arsenale/backend/internal/storage"
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

	secret, err := desktopbroker.LoadSecret("GUACAMOLE_SECRET", "GUACAMOLE_SECRET_FILE")
	if err != nil {
		panic(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	broker := desktopbroker.NewBroker(desktopbroker.BrokerConfig{
		GuacamoleSecret:  secret,
		DefaultGuacdHost: getenv("GUACD_HOST", "guacd"),
		DefaultGuacdPort: getenvInt("GUACD_PORT", 4822),
		GuacdTLS:         os.Getenv("GUACD_SSL") == "true",
		GuacdCAPath:      os.Getenv("GUACD_CA_CERT"),
		SessionStore:     desktopbroker.NewPostgresSessionStore(db),
		Logger:           logger,
	})

	service := app.StaticService{
		Descriptor: catalog.MustService(contracts.ServiceDesktopBroker),
		Register: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /", broker.HandleWebSocket)
			mux.HandleFunc("GET /guacamole/", broker.HandleWebSocket)
		},
	}

	if err := app.Run(ctx, service); err != nil {
		panic(err)
	}
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
