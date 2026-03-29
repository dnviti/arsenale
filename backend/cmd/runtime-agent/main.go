package main

import (
	"context"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/catalog"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/dnviti/arsenale/backend/pkg/workloadspec"
)

type validateWorkloadRequest struct {
	Kind     contracts.OrchestratorConnectionKind `json:"kind"`
	Workload workloadspec.WorkloadSpec            `json:"workload"`
}

func main() {
	service := app.StaticService{
		Descriptor: catalog.MustService(contracts.ServiceRuntimeAgent),
		Register: func(mux *http.ServeMux) {
			mux.HandleFunc("POST /v1/runtime/workloads:validate", func(w http.ResponseWriter, r *http.Request) {
				var req validateWorkloadRequest
				if err := app.ReadJSON(r, &req); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				app.WriteJSON(w, http.StatusOK, req.Workload.ValidateFor(req.Kind))
			})
		},
	}

	if err := app.Run(context.Background(), service); err != nil {
		panic(err)
	}
}
