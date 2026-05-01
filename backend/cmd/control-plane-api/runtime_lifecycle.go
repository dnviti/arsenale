package main

import (
	"context"

	"github.com/dnviti/arsenale/backend/internal/app"
)

type apiRuntime struct {
	service  app.StaticService
	deps     *apiDependencies
	closeFns []func()
}

func (r *apiRuntime) Close() {
	closeRuntimeResources(r.closeFns)
}

func (r *apiRuntime) Run(ctx context.Context) error {
	return app.Run(ctx, r.service)
}

func (r *apiRuntime) DevBootstrap(ctx context.Context) error {
	return runDevBootstrap(ctx, r.deps)
}
