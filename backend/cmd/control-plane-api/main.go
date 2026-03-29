package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/catalog"
	"github.com/dnviti/arsenale/backend/internal/orchestration"
	"github.com/dnviti/arsenale/backend/internal/storage"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/jackc/pgx/v5"
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

	store := orchestration.NewStore(db)
	if err := store.EnsureSchema(ctx); err != nil {
		panic(err)
	}

	service := app.StaticService{
		Descriptor: catalog.MustService(contracts.ServiceControlPlaneAPI),
		Register: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /v1/services", func(w http.ResponseWriter, _ *http.Request) {
				app.WriteJSON(w, http.StatusOK, map[string]any{"services": catalog.Services()})
			})
			mux.HandleFunc("GET /v1/capabilities", func(w http.ResponseWriter, _ *http.Request) {
				app.WriteJSON(w, http.StatusOK, map[string]any{"capabilities": catalog.Capabilities()})
			})
			mux.HandleFunc("POST /v1/orchestrators:validate", func(w http.ResponseWriter, r *http.Request) {
				var conn contracts.OrchestratorConnection
				if err := app.ReadJSON(r, &conn); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				app.WriteJSON(w, http.StatusOK, orchestration.ValidateConnection(conn))
			})
			mux.HandleFunc("GET /v1/orchestrators", func(w http.ResponseWriter, r *http.Request) {
				connections, err := store.ListConnections(r.Context())
				if err != nil {
					status := http.StatusServiceUnavailable
					app.ErrorJSON(w, status, err.Error())
					return
				}
				app.WriteJSON(w, http.StatusOK, map[string]any{"connections": connections})
			})
			mux.HandleFunc("GET /v1/orchestrators/{name}", func(w http.ResponseWriter, r *http.Request) {
				connection, err := store.GetConnection(r.Context(), r.PathValue("name"))
				if err != nil {
					switch {
					case errors.Is(err, pgx.ErrNoRows):
						app.ErrorJSON(w, http.StatusNotFound, "orchestrator connection not found")
					default:
						app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
					}
					return
				}
				app.WriteJSON(w, http.StatusOK, map[string]any{"connection": connection})
			})
			mux.HandleFunc("PUT /v1/orchestrators/{name}", func(w http.ResponseWriter, r *http.Request) {
				var conn contracts.OrchestratorConnection
				if err := app.ReadJSON(r, &conn); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}

				name := r.PathValue("name")
				if conn.Name != "" && conn.Name != name {
					app.ErrorJSON(w, http.StatusBadRequest, "connection name must match the URL path")
					return
				}
				conn.Name = name

				validation := orchestration.ValidateConnection(conn)
				if !validation.Valid {
					app.WriteJSON(w, http.StatusBadRequest, validation)
					return
				}

				stored, err := store.UpsertConnection(r.Context(), conn)
				if err != nil {
					app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
					return
				}

				app.WriteJSON(w, http.StatusOK, map[string]any{
					"connection": stored,
					"validation": validation,
				})
			})
		},
	}

	if err := app.Run(ctx, service); err != nil {
		panic(err)
	}
}
