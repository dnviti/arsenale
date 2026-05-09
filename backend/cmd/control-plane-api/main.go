package main

import (
	"context"
	"os"
)

func main() {
	ctx := context.Background()
	isDevBootstrap := len(os.Args) > 1 && os.Args[1] == "dev-bootstrap"
	if isDevBootstrap {
		// dev-bootstrap is a one-off maintenance command run inside the live API
		// container. It must not start listeners owned by the server process.
		_ = os.Setenv("SSH_PROXY_ENABLED", "false")
	}

	runtime, err := newAPIRuntime(ctx)
	if err != nil {
		panic(err)
	}
	defer runtime.Close()

	if isDevBootstrap {
		if err := runtime.DevBootstrap(ctx); err != nil {
			panic(err)
		}
		return
	}

	if err := runtime.Run(ctx); err != nil {
		panic(err)
	}
}
