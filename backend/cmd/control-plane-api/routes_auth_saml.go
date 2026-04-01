package main

import "net/http"

func (d *apiDependencies) registerAuthSAMLRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/saml/metadata", d.oauthService.HandleSAMLMetadata)
	mux.HandleFunc("GET /api/auth/saml", d.oauthService.HandleInitiateSAML)
	mux.HandleFunc("GET /api/auth/saml/link", d.oauthService.HandleInitiateSAMLLink)
	mux.HandleFunc("POST /api/auth/saml/callback", d.oauthService.HandleSAMLCallback)
}
