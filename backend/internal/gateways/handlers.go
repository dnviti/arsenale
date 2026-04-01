package gateways

import (
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
)

func (s Service) HandleList(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	result, err := s.ListGateways(r.Context(), claims.TenantID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleCreate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	var payload createPayload
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.CreateGateway(r.Context(), claims, payload, requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusCreated, result)
}

func (s Service) HandleUpdate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	var payload updatePayload
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.UpdateGateway(r.Context(), claims, r.PathValue("id"), payload, requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleDelete(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	force := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("force")), "true")
	result, err := s.DeleteGateway(r.Context(), claims, r.PathValue("id"), force, requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleTestConnectivity(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	result, err := s.TestGatewayConnectivity(r.Context(), claims.TenantID, r.PathValue("id"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleGetScalingStatus(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	result, err := s.GetScalingStatus(r.Context(), claims.TenantID, r.PathValue("id"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandlePushSSHKey(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
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

func (s Service) HandleListInstances(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	result, err := s.ListGatewayInstances(r.Context(), claims.TenantID, r.PathValue("id"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleDeploy(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	result, err := s.DeployGatewayInstance(r.Context(), claims, r.PathValue("id"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusCreated, result)
}

func (s Service) HandleUndeploy(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	result, err := s.UndeployGateway(r.Context(), claims, r.PathValue("id"), requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleScale(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	var payload scalePayload
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.ScaleGateway(r.Context(), claims, r.PathValue("id"), payload.Replicas, requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleRestartInstance(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	result, err := s.RestartGatewayInstance(r.Context(), claims, r.PathValue("id"), r.PathValue("instanceId"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleGetInstanceLogs(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	result, err := s.GetGatewayInstanceLogs(r.Context(), claims, r.PathValue("id"), r.PathValue("instanceId"), parseGatewayLogTail(r.URL.Query().Get("tail")))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleUpdateScalingConfig(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	var payload scalingConfigPayload
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.UpdateScalingConfig(r.Context(), claims, r.PathValue("id"), payload, requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleGenerateSSHKeyPair(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
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
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
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
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
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
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
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
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
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
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	result, err := s.GetSSHKeyRotationStatus(r.Context(), claims.TenantID)
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleListTemplates(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	result, err := s.ListGatewayTemplates(r.Context(), claims.TenantID)
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleCreateTemplate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	var payload createTemplatePayload
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.CreateGatewayTemplate(r.Context(), claims, payload, requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusCreated, result)
}

func (s Service) HandleUpdateTemplate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	var payload updateTemplatePayload
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.UpdateGatewayTemplate(r.Context(), claims, r.PathValue("templateId"), payload, requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleDeleteTemplate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	result, err := s.DeleteGatewayTemplate(r.Context(), claims, r.PathValue("templateId"), requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleDeployTemplate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	result, err := s.DeployGatewayTemplate(r.Context(), claims, r.PathValue("templateId"), requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusCreated, result)
}

func (s Service) HandleTunnelOverview(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	result, err := s.GetTunnelOverview(r.Context(), claims.TenantID)
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleGenerateTunnelToken(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	result, err := s.GenerateTunnelToken(r.Context(), claims, r.PathValue("id"), requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusCreated, result)
}

func (s Service) HandleRevokeTunnelToken(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	result, err := s.RevokeTunnelToken(r.Context(), claims, r.PathValue("id"), requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleForceDisconnectTunnel(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	result, err := s.ForceDisconnectTunnel(r.Context(), claims, r.PathValue("id"), requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleGetTunnelEvents(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	result, err := s.GetTunnelEvents(r.Context(), claims.TenantID, r.PathValue("id"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleGetTunnelMetrics(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageGateways(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	result, err := s.GetTunnelMetrics(r.Context(), claims.TenantID, r.PathValue("id"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func claimsCanManageGateways(role string) bool {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case "OWNER", "ADMIN", "OPERATOR":
		return true
	default:
		return false
	}
}
