package main

import (
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
)

func (d *apiDependencies) handleGatewayRoute(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	path := strings.TrimPrefix(r.URL.Path, "/api/gateways/")
	if path == "" {
		gatewayRouteNotFound(w)
		return
	}
	if path == "tunnel-overview" && !d.features.ZeroTrustEnabled {
		gatewayRouteNotFound(w)
		return
	}

	if strings.HasPrefix(path, "templates/") {
		d.handleGatewayTemplateRoute(w, r, claims, strings.TrimPrefix(path, "templates/"))
		return
	}

	id, rest, hasRest := strings.Cut(path, "/")
	id = strings.TrimSuffix(id, "/")
	if id == "" || strings.Contains(id, "/") {
		gatewayRouteNotFound(w)
		return
	}
	r.SetPathValue("id", id)

	if !hasRest {
		d.handleGatewayRootItemRoute(w, r, claims)
		return
	}
	if strings.HasPrefix(rest, "instances/") {
		d.handleGatewayInstanceRoute(w, r, claims, strings.TrimPrefix(rest, "instances/"))
		return
	}
	if strings.Contains(rest, "/") {
		gatewayRouteNotFound(w)
		return
	}
	d.handleGatewayActionRoute(w, r, claims, rest)
}

func (d *apiDependencies) handleGatewayTemplateRoute(w http.ResponseWriter, r *http.Request, claims authn.Claims, templatePath string) {
	if strings.HasSuffix(templatePath, "/deploy") {
		templateID := strings.TrimSuffix(templatePath, "/deploy")
		templateID = strings.TrimSuffix(templateID, "/")
		if !setGatewayRoutePathValue(w, r, "templateId", templateID) {
			return
		}
		if !gatewayRouteMethod(w, r, http.MethodPost) {
			return
		}
		d.gatewayService.HandleDeployTemplate(w, r, claims)
		return
	}

	templateID := strings.TrimSuffix(templatePath, "/")
	if !setGatewayRoutePathValue(w, r, "templateId", templateID) {
		return
	}
	switch r.Method {
	case http.MethodPut:
		d.gatewayService.HandleUpdateTemplate(w, r, claims)
	case http.MethodDelete:
		d.gatewayService.HandleDeleteTemplate(w, r, claims)
	default:
		gatewayRouteMethodNotAllowed(w, "DELETE, PUT")
	}
}

func (d *apiDependencies) handleGatewayRootItemRoute(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	switch r.Method {
	case http.MethodPut:
		d.gatewayService.HandleUpdate(w, r, claims)
	case http.MethodDelete:
		d.gatewayService.HandleDelete(w, r, claims)
	default:
		gatewayRouteMethodNotAllowed(w, "DELETE, PUT")
	}
}

func (d *apiDependencies) handleGatewayInstanceRoute(w http.ResponseWriter, r *http.Request, claims authn.Claims, instancePath string) {
	instanceID, action, ok := strings.Cut(instancePath, "/")
	instanceID = strings.TrimSuffix(instanceID, "/")
	if !ok || !setGatewayRoutePathValue(w, r, "instanceId", instanceID) {
		return
	}

	switch action {
	case "restart":
		if !gatewayRouteMethod(w, r, http.MethodPost) {
			return
		}
		d.gatewayService.HandleRestartInstance(w, r, claims)
	case "logs":
		if !gatewayRouteMethod(w, r, http.MethodGet) {
			return
		}
		d.gatewayService.HandleGetInstanceLogs(w, r, claims)
	default:
		gatewayRouteNotFound(w)
	}
}

func (d *apiDependencies) handleGatewayActionRoute(w http.ResponseWriter, r *http.Request, claims authn.Claims, action string) {
	switch action {
	case "test":
		if !gatewayRouteMethod(w, r, http.MethodPost) {
			return
		}
		d.gatewayService.HandleTestConnectivity(w, r, claims)
	case "push-key":
		if !gatewayRouteMethod(w, r, http.MethodPost) {
			return
		}
		d.gatewayService.HandlePushSSHKey(w, r, claims)
	case "scaling":
		switch r.Method {
		case http.MethodGet:
			d.gatewayService.HandleGetScalingStatus(w, r, claims)
		case http.MethodPut:
			d.gatewayService.HandleUpdateScalingConfig(w, r, claims)
		default:
			gatewayRouteMethodNotAllowed(w, "GET, PUT")
		}
	case "instances":
		if !gatewayRouteMethod(w, r, http.MethodGet) {
			return
		}
		d.gatewayService.HandleListInstances(w, r, claims)
	case "deploy":
		switch r.Method {
		case http.MethodPost:
			d.gatewayService.HandleDeploy(w, r, claims)
		case http.MethodDelete:
			d.gatewayService.HandleUndeploy(w, r, claims)
		default:
			gatewayRouteMethodNotAllowed(w, "DELETE, POST")
		}
	case "scale":
		if !gatewayRouteMethod(w, r, http.MethodPost) {
			return
		}
		d.gatewayService.HandleScale(w, r, claims)
	case "tunnel-token":
		if !d.features.ZeroTrustEnabled {
			gatewayRouteNotFound(w)
			return
		}
		switch r.Method {
		case http.MethodPost:
			d.gatewayService.HandleGenerateTunnelToken(w, r, claims)
		case http.MethodDelete:
			d.gatewayService.HandleRevokeTunnelToken(w, r, claims)
		default:
			gatewayRouteMethodNotAllowed(w, "DELETE, POST")
		}
	case "tunnel-disconnect":
		if !d.features.ZeroTrustEnabled {
			gatewayRouteNotFound(w)
			return
		}
		if !gatewayRouteMethod(w, r, http.MethodPost) {
			return
		}
		d.gatewayService.HandleForceDisconnectTunnel(w, r, claims)
	case "tunnel-events":
		if !d.features.ZeroTrustEnabled {
			gatewayRouteNotFound(w)
			return
		}
		if !gatewayRouteMethod(w, r, http.MethodGet) {
			return
		}
		d.gatewayService.HandleGetTunnelEvents(w, r, claims)
	case "tunnel-metrics":
		if !d.features.ZeroTrustEnabled {
			gatewayRouteNotFound(w)
			return
		}
		if !gatewayRouteMethod(w, r, http.MethodGet) {
			return
		}
		d.gatewayService.HandleGetTunnelMetrics(w, r, claims)
	default:
		gatewayRouteNotFound(w)
	}
}

func setGatewayRoutePathValue(w http.ResponseWriter, r *http.Request, key, value string) bool {
	if value == "" || strings.Contains(value, "/") {
		gatewayRouteNotFound(w)
		return false
	}
	r.SetPathValue(key, value)
	return true
}

func gatewayRouteMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	gatewayRouteMethodNotAllowed(w, method)
	return false
}

func gatewayRouteMethodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	app.ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
}

func gatewayRouteNotFound(w http.ResponseWriter) {
	app.ErrorJSON(w, http.StatusNotFound, "not found")
}
