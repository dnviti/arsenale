package main

import (
	"net/http"
	"testing"

	"github.com/dnviti/arsenale/backend/internal/authn"
)

func TestRegisterAuthRoutesDoesNotPanic(t *testing.T) {
	deps := &apiDependencies{
		authenticator: &authn.Authenticator{},
	}
	mux := http.NewServeMux()

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("registerAuthRoutes() panicked: %v", recovered)
		}
	}()

	deps.registerAuthRoutes(mux)
}
