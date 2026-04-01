package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/storage"
)

func main() {
	ctx := context.Background()

	command := "up"
	if len(os.Args) > 1 {
		command = strings.TrimSpace(os.Args[1])
	}

	switch command {
	case "up":
		report, err := storage.RunMigrations(ctx)
		if err != nil {
			fail(err)
		}
		if report.LegacyStamped {
			fmt.Println("migrate: stamped legacy database at baseline")
		}
		fmt.Printf("migrate: applied_now=%d total=%d pending=%d\n", report.AppliedNow, len(report.Applied), len(report.Pending))
	case "status":
		report, err := storage.MigrationStatus(ctx)
		if err != nil {
			fail(err)
		}
		fmt.Printf("migrate: applied=%d pending=%d\n", len(report.Applied), len(report.Pending))
		for _, item := range report.Applied {
			fmt.Printf("applied %06d %s\n", item.Version, item.Name)
		}
		for _, item := range report.Pending {
			fmt.Printf("pending %06d %s\n", item.Version, item.Name)
		}
	default:
		fail(fmt.Errorf("unknown command %q (supported: up, status)", command))
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
	os.Exit(1)
}
