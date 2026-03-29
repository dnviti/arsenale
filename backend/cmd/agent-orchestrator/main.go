package main

import (
	"context"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/agents"
	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/catalog"
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

	store := agents.NewStore(db)
	if err := store.EnsureSchema(ctx); err != nil {
		panic(err)
	}

	service := app.StaticService{
		Descriptor: catalog.MustService(contracts.ServiceAgentOrchestrator),
		Register: func(mux *http.ServeMux) {
			mux.HandleFunc("POST /v1/agent-runs:validate", func(w http.ResponseWriter, r *http.Request) {
				var req contracts.AgentRunRequest
				if err := app.ReadJSON(r, &req); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				if err := agents.ValidateRunRequest(req); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				app.WriteJSON(w, http.StatusOK, map[string]any{
					"valid": true,
					"run":   req,
				})
			})
			mux.HandleFunc("POST /v1/agent-runs", func(w http.ResponseWriter, r *http.Request) {
				var req contracts.AgentRunRequest
				if err := app.ReadJSON(r, &req); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				run, err := store.CreateRun(r.Context(), req)
				if err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				app.WriteJSON(w, http.StatusCreated, map[string]any{"run": run})
			})
			mux.HandleFunc("GET /v1/agent-runs", func(w http.ResponseWriter, r *http.Request) {
				tenantID := r.URL.Query().Get("tenantId")
				runs, err := store.ListRuns(r.Context(), tenantID)
				if err != nil {
					status := http.StatusServiceUnavailable
					if err.Error() == "tenantId is required" {
						status = http.StatusBadRequest
					}
					app.ErrorJSON(w, status, err.Error())
					return
				}
				app.WriteJSON(w, http.StatusOK, map[string]any{"runs": runs})
			})
			mux.HandleFunc("GET /v1/agent-runs/{id}", func(w http.ResponseWriter, r *http.Request) {
				run, err := store.GetRun(r.Context(), r.PathValue("id"))
				if err != nil {
					if agents.IsNotFound(err) {
						app.ErrorJSON(w, http.StatusNotFound, "agent run not found")
						return
					}
					app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
					return
				}
				app.WriteJSON(w, http.StatusOK, map[string]any{"run": run})
			})
		},
	}

	if err := app.Run(ctx, service); err != nil {
		panic(err)
	}
}
