package main

import (
	"context"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/catalog"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func main() {
	service := app.StaticService{
		Descriptor: catalog.MustService(contracts.ServiceTunnelBroker),
	}
	if err := app.Run(context.Background(), service); err != nil {
		panic(err)
	}
}
