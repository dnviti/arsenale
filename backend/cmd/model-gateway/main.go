package main

import (
	"context"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/catalog"
	"github.com/dnviti/arsenale/backend/internal/modelgateway"
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

	store := modelgateway.NewStore(db)
	if err := store.EnsureSchema(ctx); err != nil {
		panic(err)
	}

	encryptionKey, err := modelgateway.LoadServerEncryptionKey()
	if err != nil {
		panic(err)
	}

	service := app.StaticService{
		Descriptor: catalog.MustService(contracts.ServiceModelGateway),
		Register: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /v1/providers", func(w http.ResponseWriter, _ *http.Request) {
				app.WriteJSON(w, http.StatusOK, map[string]any{
					"providers": modelgateway.Providers(),
				})
			})
			mux.HandleFunc("POST /v1/provider-configs:validate", func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					Config           contracts.TenantAIConfig `json:"config"`
					ApiKeyConfigured bool                     `json:"apiKeyConfigured"`
				}
				if err := app.ReadJSON(r, &req); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				app.WriteJSON(w, http.StatusOK, modelgateway.ValidateConfig(req.Config, req.ApiKeyConfigured))
			})
			mux.HandleFunc("GET /v1/provider-configs/{tenantId}", func(w http.ResponseWriter, r *http.Request) {
				config, err := store.GetConfig(r.Context(), r.PathValue("tenantId"))
				if err != nil {
					if modelgateway.IsNotFound(err) {
						app.WriteJSON(w, http.StatusOK, map[string]any{
							"config": contracts.TenantAIConfig{
								TenantID:            r.PathValue("tenantId"),
								Provider:            contracts.AIProviderNone,
								HasAPIKey:           false,
								ModelID:             "",
								MaxTokensPerRequest: 4000,
								DailyRequestLimit:   100,
								Enabled:             false,
							},
						})
						return
					}
					app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
					return
				}
				app.WriteJSON(w, http.StatusOK, map[string]any{"config": config})
			})
			mux.HandleFunc("PUT /v1/provider-configs/{tenantId}", func(w http.ResponseWriter, r *http.Request) {
				var update contracts.TenantAIConfigUpdate
				if err := app.ReadJSON(r, &update); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				config, err := store.UpsertConfig(r.Context(), r.PathValue("tenantId"), update, encryptionKey)
				if err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				validation := modelgateway.ValidateConfig(config, config.HasAPIKey)
				app.WriteJSON(w, http.StatusOK, map[string]any{
					"config":     config,
					"validation": validation,
				})
			})
		},
	}

	if err := app.Run(ctx, service); err != nil {
		panic(err)
	}
}
