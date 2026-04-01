package main

import (
	"context"
	"os"
)

func main() {
	ctx := context.Background()

	runtime, err := newAPIRuntime(ctx)
	if err != nil {
		panic(err)
	}
	defer runtime.Close()

	if len(os.Args) > 1 && os.Args[1] == "dev-bootstrap" {
		if err := runtime.DevBootstrap(ctx); err != nil {
			panic(err)
		}
		return
	}

	if err := runtime.Run(ctx); err != nil {
		panic(err)
	}
}
