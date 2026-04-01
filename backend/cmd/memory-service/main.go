package main

import (
	"context"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/catalog"
	"github.com/dnviti/arsenale/backend/internal/memory"
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

	store := memory.NewStore(db)

	service := app.StaticService{
		Descriptor: catalog.MustService(contracts.ServiceMemoryService),
		Register: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /v1/memory/schema", func(w http.ResponseWriter, _ *http.Request) {
				manifest := catalog.Manifest("dev")
				app.WriteJSON(w, http.StatusOK, map[string]any{
					"types":  manifest.SupportedMemoryTypes,
					"scopes": manifest.SupportedMemoryScopes,
				})
			})
			mux.HandleFunc("POST /v1/memory/namespaces:validate", func(w http.ResponseWriter, r *http.Request) {
				var ns contracts.MemoryNamespace
				if err := app.ReadJSON(r, &ns); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				if err := memory.ValidateNamespace(ns); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				app.WriteJSON(w, http.StatusOK, map[string]any{
					"valid": true,
					"key":   memory.NamespaceKey(ns),
				})
			})
			mux.HandleFunc("PUT /v1/memory/namespaces", func(w http.ResponseWriter, r *http.Request) {
				var ns contracts.MemoryNamespace
				if err := app.ReadJSON(r, &ns); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				if err := memory.ValidateNamespace(ns); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				record, err := store.UpsertNamespace(r.Context(), ns)
				if err != nil {
					app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
					return
				}
				app.WriteJSON(w, http.StatusOK, map[string]any{"namespace": record})
			})
			mux.HandleFunc("GET /v1/memory/namespaces", func(w http.ResponseWriter, r *http.Request) {
				tenantID := r.URL.Query().Get("tenantId")
				records, err := store.ListNamespaces(r.Context(), tenantID)
				if err != nil {
					status := http.StatusServiceUnavailable
					if err.Error() == "tenantId is required" {
						status = http.StatusBadRequest
					}
					app.ErrorJSON(w, status, err.Error())
					return
				}
				app.WriteJSON(w, http.StatusOK, map[string]any{"namespaces": records})
			})
			mux.HandleFunc("POST /v1/memory/items", func(w http.ResponseWriter, r *http.Request) {
				var req contracts.MemoryWriteRequest
				if err := app.ReadJSON(r, &req); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				if err := memory.ValidateNamespace(req.Namespace); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				item, err := store.AppendItem(r.Context(), req)
				if err != nil {
					app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
					return
				}
				app.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
			})
			mux.HandleFunc("GET /v1/memory/items", func(w http.ResponseWriter, r *http.Request) {
				namespaceKey := r.URL.Query().Get("namespaceKey")
				items, err := store.ListItems(r.Context(), namespaceKey)
				if err != nil {
					status := http.StatusServiceUnavailable
					if err.Error() == "namespaceKey is required" {
						status = http.StatusBadRequest
					}
					app.ErrorJSON(w, status, err.Error())
					return
				}
				app.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
			})
		},
	}

	if err := app.Run(ctx, service); err != nil {
		panic(err)
	}
}
