package vaultapi

import (
	"net/http"
	"time"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	stream "github.com/dnviti/arsenale/backend/internal/sse"
)

const vaultStatusStreamInterval = 30 * time.Second

func (s Service) HandleStatusStream(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	status, err := s.GetStatus(r.Context(), claims.UserID)
	if err != nil {
		s.writeError(w, err)
		return
	}

	sse, err := stream.Open(w)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	if err := sse.Event("snapshot", status); err != nil {
		return
	}

	ticker := time.NewTicker(vaultStatusStreamInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			status, err := s.GetStatus(r.Context(), claims.UserID)
			if err != nil {
				return
			}
			if err := sse.Event("snapshot", status); err != nil {
				return
			}
		}
	}
}
