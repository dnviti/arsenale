package main

import (
	"context"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authz"
	"github.com/dnviti/arsenale/backend/internal/catalog"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func main() {
	pdp := authz.NewStaticPDP()
	service := app.StaticService{
		Descriptor: catalog.MustService(contracts.ServiceAuthzPDP),
		Register: func(mux *http.ServeMux) {
			mux.HandleFunc("POST /v1/decide", func(w http.ResponseWriter, r *http.Request) {
				var req contracts.AuthzRequest
				if err := app.ReadJSON(r, &req); err != nil {
					app.ErrorJSON(w, http.StatusBadRequest, err.Error())
					return
				}
				app.WriteJSON(w, http.StatusOK, pdp.Evaluate(req))
			})
		},
	}

	if err := app.Run(context.Background(), service); err != nil {
		panic(err)
	}
}
