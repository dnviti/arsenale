package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/runtimefeatures"
)

func TestGatewayRouteTypesCatalog(t *testing.T) {
	deps := &apiDependencies{features: runtimefeatures.Manifest{ZeroTrustEnabled: true}}

	rec := runGatewayRoute(deps, http.MethodGet, "/api/gateways/types")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Types []struct {
			Type            string   `json:"type"`
			DisplayName     string   `json:"displayName"`
			Summary         string   `json:"summary"`
			DeploymentModel string   `json:"deploymentModel"`
			DeploymentModes []string `json:"deploymentModes"`
		} `json:"types"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Types) == 0 {
		t.Fatal("empty type catalog")
	}
	for _, ti := range resp.Types {
		if ti.DisplayName == "" || ti.Summary == "" || ti.DeploymentModel == "" || len(ti.DeploymentModes) == 0 {
			t.Fatalf("type %q has incomplete metadata: %+v", ti.Type, ti)
		}
	}
}

func TestGatewayRouteTypesRequiresGET(t *testing.T) {
	deps := &apiDependencies{features: runtimefeatures.Manifest{ZeroTrustEnabled: true}}
	rec := runGatewayRoute(deps, http.MethodPost, "/api/gateways/types")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestGatewayRouteFeatureGatesTunnelActions(t *testing.T) {
	deps := &apiDependencies{
		features: runtimefeatures.Manifest{ZeroTrustEnabled: false},
	}

	rec := runGatewayRoute(deps, http.MethodPost, "/api/gateways/gw-1/tunnel-token")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGatewayRouteMethodGatesActions(t *testing.T) {
	deps := &apiDependencies{
		features: runtimefeatures.Manifest{ZeroTrustEnabled: true},
	}

	tests := []struct {
		name      string
		method    string
		path      string
		wantAllow string
	}{
		{
			name:      "test requires POST",
			method:    http.MethodGet,
			path:      "/api/gateways/gw-1/test",
			wantAllow: http.MethodPost,
		},
		{
			name:      "instance logs require GET",
			method:    http.MethodPost,
			path:      "/api/gateways/gw-1/instances/inst-1/logs",
			wantAllow: http.MethodGet,
		},
		{
			name:      "template deploy requires POST",
			method:    http.MethodGet,
			path:      "/api/gateways/templates/template-1/deploy",
			wantAllow: http.MethodPost,
		},
		{
			name:      "tunnel token supports create and revoke",
			method:    http.MethodGet,
			path:      "/api/gateways/gw-1/tunnel-token",
			wantAllow: "DELETE, POST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := runGatewayRoute(deps, tt.method, tt.path)
			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}
			if got := rec.Header().Get("Allow"); got != tt.wantAllow {
				t.Fatalf("Allow = %q, want %q", got, tt.wantAllow)
			}
		})
	}
}

func TestGatewayRouteRejectsNestedIDs(t *testing.T) {
	deps := &apiDependencies{
		features: runtimefeatures.Manifest{ZeroTrustEnabled: true},
	}

	rec := runGatewayRoute(deps, http.MethodGet, "/api/gateways/templates/template-1/extra/deploy")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func runGatewayRoute(deps *apiDependencies, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	deps.handleGatewayRoute(rec, req, authn.Claims{})
	return rec
}
