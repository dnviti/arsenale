package main

import (
	"context"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/catalog"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func main() {
	tiles, err := newTileService()
	if err != nil {
		panic(err)
	}
	service := app.StaticService{
		Descriptor: catalog.MustService(contracts.ServiceMapAssets),
		Register: func(mux *http.ServeMux) {
			tiles.registerRoutes(mux)
		},
	}

	if err := app.Run(context.Background(), service); err != nil {
		panic(err)
	}
}
