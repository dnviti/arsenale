package dbauditapi

import (
	"net/http"
	"time"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	stream "github.com/dnviti/arsenale/backend/internal/sse"
)

const dbAuditStreamInterval = 10 * time.Second

func (s Service) HandleStreamLogs(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !s.authorized(w, r, claims) {
		return
	}

	query, err := parseDBAuditQuery(r)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.ListLogs(r.Context(), claims.TenantID, query)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	sse, err := stream.Open(w)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	if err := sse.Event("snapshot", result); err != nil {
		return
	}

	ticker := time.NewTicker(dbAuditStreamInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			result, err := s.ListLogs(r.Context(), claims.TenantID, query)
			if err != nil {
				return
			}
			if err := sse.Event("snapshot", result); err != nil {
				return
			}
		}
	}
}
