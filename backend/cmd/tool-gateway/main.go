package main

import (
	"context"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/catalog"
	"github.com/dnviti/arsenale/backend/internal/tooling"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func main() {
	service := app.StaticService{
		Descriptor: catalog.MustService(contracts.ServiceToolGateway),
		Register: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /v1/capabilities", func(w http.ResponseWriter, _ *http.Request) {
				app.WriteJSON(w, http.StatusOK, map[string]any{"capabilities": catalog.Capabilities()})
			})
			mux.HandleFunc("POST /v1/tool-calls:plan", func(w http.ResponseWriter, r *http.Request) {
				var req contracts.ToolCallPlanRequest
				if err := app.ReadJSON(r, &req); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				plan, err := tooling.PlanToolCall(req)
				if err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				app.WriteJSON(w, http.StatusOK, plan)
			})
			mux.HandleFunc("POST /v1/tool-calls:execute", func(w http.ResponseWriter, r *http.Request) {
				var req contracts.ToolCallExecuteRequest
				if err := app.ReadJSON(r, &req); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				result, err := tooling.ExecuteToolCall(r.Context(), req)
				if err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				app.WriteJSON(w, http.StatusOK, result)
			})
		},
	}

	if err := app.Run(context.Background(), service); err != nil {
		panic(err)
	}
}
