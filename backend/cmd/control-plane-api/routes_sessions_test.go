package main

import (
	"net/http"
	"testing"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/runtimefeatures"
)

func TestRegisterSessionRoutesIncludesPauseAndResume(t *testing.T) {
	deps := &apiDependencies{
		authenticator: &authn.Authenticator{},
		features: runtimefeatures.Manifest{
			ConnectionsEnabled:   true,
			DatabaseProxyEnabled: true,
		},
	}
	mux := http.NewServeMux()
	deps.registerSessionRoutes(mux)

	expectRoutePattern(t, mux, http.MethodPost, "/api/sessions/sess-1/pause", "POST /api/sessions/{sessionId}/pause")
	expectRoutePattern(t, mux, http.MethodPost, "/api/sessions/sess-1/resume", "POST /api/sessions/{sessionId}/resume")
	expectRoutePattern(t, mux, http.MethodPost, "/api/sessions/sess-1/terminate", "POST /api/sessions/{sessionId}/terminate")
}

func TestRegisterSessionRoutesIncludesCLIDesktopLaunchWhenCLIEnabled(t *testing.T) {
	deps := &apiDependencies{
		authenticator: &authn.Authenticator{},
		features: runtimefeatures.Manifest{
			ConnectionsEnabled: true,
			CLIEnabled:         true,
		},
	}
	mux := http.NewServeMux()
	deps.registerSessionRoutes(mux)

	expectRoutePattern(t, mux, http.MethodPost, "/api/cli/connect/desktop/launch", "POST /api/cli/connect/desktop/launch")
	expectRoutePattern(t, mux, http.MethodPost, "/api/cli/connect/desktop/redeem", "POST /api/cli/connect/desktop/redeem")
	expectRoutePattern(t, mux, http.MethodPost, "/api/cli/connect/desktop/sess-1/heartbeat", "POST /api/cli/connect/desktop/{sessionId}/heartbeat")
	expectRoutePattern(t, mux, http.MethodPost, "/api/cli/connect/desktop/sess-1/end", "POST /api/cli/connect/desktop/{sessionId}/end")
}
