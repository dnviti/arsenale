package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/catalog"
	"github.com/dnviti/arsenale/backend/internal/storage"
	"github.com/dnviti/arsenale/backend/internal/terminalbroker"
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

	secret, err := terminalbroker.LoadSecret()
	if err != nil {
		panic(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	broker := terminalbroker.NewBroker(terminalbroker.BrokerConfig{
		Secret:       secret,
		SessionStore: terminalbroker.NewPostgresSessionStore(db),
		Logger:       logger,
	})

	service := app.StaticService{
		Descriptor: catalog.MustService(contracts.ServiceTerminalBroker),
		Register: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /v1/session-protocol", func(w http.ResponseWriter, _ *http.Request) {
				app.WriteJSON(w, http.StatusOK, terminalbroker.ProtocolDescriptor())
			})
			mux.HandleFunc("POST /v1/session-grants:issue", broker.HandleGrantIssue)
			mux.HandleFunc("POST /v1/session-grants:validate", broker.HandleGrantValidate)
			mux.HandleFunc("GET /ws/terminal", broker.HandleWebSocket)
		},
	}

	if err := app.Run(ctx, service); err != nil {
		panic(err)
	}
}
