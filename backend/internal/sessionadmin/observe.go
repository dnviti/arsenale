package sessionadmin

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/sessions"
	"github.com/dnviti/arsenale/backend/internal/tenantauth"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

type SSHObserveGrantResponse struct {
	SessionID     string                        `json:"sessionId"`
	Token         string                        `json:"token"`
	ExpiresAt     time.Time                     `json:"expiresAt"`
	WebSocketPath string                        `json:"webSocketPath"`
	WebSocketURL  string                        `json:"webSocketUrl,omitempty"`
	Mode          contracts.TerminalSessionMode `json:"mode"`
	ReadOnly      bool                          `json:"readOnly"`
}

type sshObserverGrantIssuer interface {
	IssueSSHObserverGrant(ctx context.Context, sessionID, observerUserID string, request *http.Request) (SSHObserveGrantResponse, error)
}

type sshProxyObserverGrantIssuer interface {
	IssueSSHProxyObserverGrant(ctx context.Context, sessionID, observerUserID string, request *http.Request) (SSHObserveGrantResponse, error)
}

type statusCoder interface {
	StatusCode() int
}

func (s Service) HandleObserveSSH(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !s.authorized(w, r, claims, tenantauth.CanObserveSessions) {
		return
	}

	sessionID := strings.TrimSpace(r.PathValue("sessionId"))
	if sessionID == "" {
		app.ErrorJSON(w, http.StatusBadRequest, "sessionId is required")
		return
	}

	target, err := s.Store.LoadTenantSessionSummary(r.Context(), sessionID, claims.TenantID)
	if err != nil {
		s.writeObserveError(w, err)
		return
	}
	if target == nil {
		app.ErrorJSON(w, http.StatusNotFound, "Session not found")
		return
	}
	if target.Status == sessions.SessionStatusClosed {
		app.ErrorJSON(w, http.StatusConflict, "Session already closed")
		return
	}

	var response SSHObserveGrantResponse
	switch target.Protocol {
	case "SSH":
		if s.SSHObserverGrants == nil {
			app.ErrorJSON(w, http.StatusServiceUnavailable, "SSH observer grants are unavailable")
			return
		}
		response, err = s.SSHObserverGrants.IssueSSHObserverGrant(r.Context(), target.ID, claims.UserID, r)
	case "SSH_PROXY":
		if s.SSHProxyObserverGrants == nil {
			app.ErrorJSON(w, http.StatusServiceUnavailable, "SSH proxy observer grants are unavailable")
			return
		}
		response, err = s.SSHProxyObserverGrants.IssueSSHProxyObserverGrant(r.Context(), target.ID, claims.UserID, r)
	default:
		app.ErrorJSON(w, http.StatusBadRequest, "Only SSH sessions can be observed")
		return
	}
	if err != nil {
		s.writeObserveError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, response)
}

func (s Service) writeObserveError(w http.ResponseWriter, err error) {
	var statusErr statusCoder
	if errors.As(err, &statusErr) {
		app.ErrorJSON(w, statusErr.StatusCode(), err.Error())
		return
	}
	s.writeLifecycleError(w, err)
}
