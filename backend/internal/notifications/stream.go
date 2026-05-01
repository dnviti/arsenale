package notifications

import (
	"net/http"
	"time"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	stream "github.com/dnviti/arsenale/backend/internal/sse"
)

const notificationStreamInterval = 15 * time.Second

func (s Service) HandleStream(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	snapshot, err := s.ListNotifications(r.Context(), claims.UserID, 50, 0)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	sse, err := stream.Open(w)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	if err := sse.Event("snapshot", snapshot); err != nil {
		return
	}

	ticker := time.NewTicker(notificationStreamInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			snapshot, err := s.ListNotifications(r.Context(), claims.UserID, 50, 0)
			if err != nil {
				return
			}
			if err := sse.Event("snapshot", snapshot); err != nil {
				return
			}
		}
	}
}
