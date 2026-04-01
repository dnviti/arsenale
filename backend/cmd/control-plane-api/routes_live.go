package main

import "net/http"

func (d *apiDependencies) registerLiveRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/gateways/stream", d.authenticator.Middleware(d.gatewayService.HandleStream))
	mux.HandleFunc("GET /api/gateways/{id}/instances/{instanceId}/logs/stream", d.authenticator.Middleware(d.gatewayService.HandleStreamInstanceLogs))
	mux.HandleFunc("GET /api/notifications/stream", d.authenticator.Middleware(d.notificationService.HandleStream))
	mux.HandleFunc("GET /api/vault/status/stream", d.authenticator.Middleware(d.vaultService.HandleStatusStream))
	mux.HandleFunc("GET /api/sessions/active/stream", d.authenticator.Middleware(d.sessionAdminService.HandleStream))
	mux.HandleFunc("GET /api/audit/stream", d.authenticator.Middleware(d.auditService.HandleStream))
	mux.HandleFunc("GET /api/db-audit/logs/stream", d.authenticator.Middleware(d.dbAuditService.HandleStreamLogs))
}
