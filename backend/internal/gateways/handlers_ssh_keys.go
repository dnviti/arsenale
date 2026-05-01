package gateways

import (
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
)

func (s Service) HandlePushSSHKey(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}

	result, err := s.PushSSHKeyToGateway(r.Context(), claims.TenantID, r.PathValue("id"))
	if err != nil {
		s.writeError(w, err)
		return
	}

	succeeded, failed := summarizePushKeyResults(result.Instances)
	if err := s.insertGatewayAuditLog(r.Context(), claims.UserID, "SSH_KEY_PUSH", r.PathValue("id"), map[string]any{
		"instances": len(result.Instances),
		"succeeded": succeeded,
		"failed":    failed,
	}, requestIP(r)); err != nil {
		s.writeError(w, err)
		return
	}

	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleGenerateSSHKeyPair(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	result, err := s.GenerateSSHKeyPair(r.Context(), claims.UserID, claims.TenantID, requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusCreated, result)
}

func (s Service) HandleGetSSHKeyPair(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	result, err := s.GetSSHKeyPair(r.Context(), claims.TenantID)
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleDownloadSSHPrivateKey(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	privateKey, err := s.GetSSHPrivateKey(r.Context(), claims.TenantID)
	if err != nil {
		s.writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", `attachment; filename="tenant_ed25519"`)
	_, _ = w.Write([]byte(privateKey))
}

func (s Service) HandleRotateSSHKeyPair(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	result, err := s.RotateSSHKeyPair(r.Context(), claims.UserID, claims.TenantID, requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}

	pushResults, err := s.PushSSHKeyToAllManagedGateways(r.Context(), claims.TenantID)
	if err != nil {
		s.writeError(w, err)
		return
	}
	for _, item := range pushResults {
		if !item.OK {
			continue
		}
		if err := s.insertGatewayAuditLog(r.Context(), claims.UserID, "SSH_KEY_PUSH", item.GatewayID, map[string]any{
			"auto":    true,
			"trigger": "rotate",
		}, requestIP(r)); err != nil {
			s.writeError(w, err)
			return
		}
	}

	app.WriteJSON(w, http.StatusOK, rotateSSHKeyPairResponse{
		sshKeyPairResponse: result,
		PushResults:        pushResults,
	})
}

func (s Service) HandleUpdateSSHKeyRotationPolicy(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	var payload rotationPolicyPayload
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.UpdateSSHKeyRotationPolicy(r.Context(), claims.UserID, claims.TenantID, requestIP(r), payload)
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleGetSSHKeyRotationStatus(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	result, err := s.GetSSHKeyRotationStatus(r.Context(), claims.TenantID)
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}
