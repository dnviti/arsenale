package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/catalog"
	"github.com/dnviti/arsenale/backend/internal/orchestration"
	"github.com/dnviti/arsenale/backend/internal/storage"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/dnviti/arsenale/backend/pkg/workloadspec"
)

type reconcilePlanRequest struct {
	ConnectionName string                    `json:"connectionName"`
	Workload       workloadspec.WorkloadSpec `json:"workload"`
}

type runtimeAgentValidateRequest struct {
	Kind     contracts.OrchestratorConnectionKind `json:"kind"`
	Workload workloadspec.WorkloadSpec            `json:"workload"`
}

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

	service := app.StaticService{
		Descriptor: catalog.MustService(contracts.ServiceControlController),
		Register: func(mux *http.ServeMux) {
			mux.HandleFunc("GET /v1/reconcile/schema", func(w http.ResponseWriter, _ *http.Request) {
				app.WriteJSON(w, http.StatusOK, map[string]any{
					"controller": "control-plane-controller",
					"responsibilities": []string{
						"placement",
						"reconciliation",
						"gateway lifecycle",
						"token rotation",
						"cleanup jobs",
					},
				})
			})
			mux.HandleFunc("POST /v1/reconcile:plan", func(w http.ResponseWriter, r *http.Request) {
				var req reconcilePlanRequest
				if err := app.ReadJSON(r, &req); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}

				connection, err := store.GetConnection(r.Context(), req.ConnectionName)
				if err != nil {
					if orchestration.IsNotFound(err) {
						app.ErrorJSON(w, http.StatusNotFound, "orchestrator connection not found")
						return
					}
					app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
					return
				}

				validation, err := validateWithRuntimeAgent(r.Context(), connection.Kind, req.Workload)
				if err != nil {
					app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
					return
				}

				app.WriteJSON(w, http.StatusOK, map[string]any{
					"accepted":   validation.Valid,
					"connection": connection,
					"workload":   req.Workload,
					"validation": validation,
				})
			})
		},
	}

	if err := app.Run(ctx, service); err != nil {
		panic(err)
	}
}

func validateWithRuntimeAgent(ctx context.Context, kind contracts.OrchestratorConnectionKind, workload workloadspec.WorkloadSpec) (workloadspec.ValidationResult, error) {
	payload, err := json.Marshal(runtimeAgentValidateRequest{
		Kind:     kind,
		Workload: workload,
	})
	if err != nil {
		return workloadspec.ValidationResult{}, fmt.Errorf("marshal runtime-agent validation request: %w", err)
	}

	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		strings.TrimRight(runtimeAgentURL(), "/")+"/v1/runtime/workloads:validate",
		bytes.NewReader(payload),
	)
	if err != nil {
		return workloadspec.ValidationResult{}, fmt.Errorf("build runtime-agent validation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return workloadspec.ValidationResult{}, fmt.Errorf("call runtime-agent validation endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var failure map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&failure)
		if message, ok := failure["error"].(string); ok && message != "" {
			return workloadspec.ValidationResult{}, fmt.Errorf("runtime-agent rejected validation request: %s", message)
		}
		return workloadspec.ValidationResult{}, fmt.Errorf("runtime-agent returned status %d", resp.StatusCode)
	}

	var result workloadspec.ValidationResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return workloadspec.ValidationResult{}, fmt.Errorf("decode runtime-agent validation response: %w", err)
	}

	return result, nil
}

func runtimeAgentURL() string {
	if value := strings.TrimSpace(os.Getenv("RUNTIME_AGENT_URL")); value != "" {
		return value
	}
	return "http://runtime-agent:8095"
}
