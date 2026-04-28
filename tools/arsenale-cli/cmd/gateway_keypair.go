package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

func runGwSSHKeypairGet(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiGet("/api/gateways/ssh-keypair", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "FINGERPRINT", Field: "fingerprint"},
		{Header: "ALGORITHM", Field: "algorithm"},
		{Header: "CREATED_AT", Field: "createdAt"},
	})
}

func runGwSSHKeypairGenerate(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiPost("/api/gateways/ssh-keypair", nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "fingerprint")
}

func runGwSSHKeypairDownload(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	destPath := filepath.Join(gwSSHKeypairDest, "id_arsenale")

	status, err := apiDownload("/api/gateways/ssh-keypair/private", destPath, cfg)
	if err != nil {
		fatal("%v", err)
	}
	if status != 200 {
		fatal("download failed (HTTP %d)", status)
	}

	if !quiet {
		fmt.Printf("SSH private key downloaded to %s\n", destPath)
	}
}

func runGwSSHKeypairRotate(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiPost("/api/gateways/ssh-keypair/rotate", nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if !quiet {
		fmt.Println("SSH keypair rotated")
	}
}

func runGwSSHKeypairRotationPolicyGet(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiGet("/api/gateways/ssh-keypair/rotation", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "ENABLED", Field: "enabled"},
		{Header: "INTERVAL_DAYS", Field: "intervalDays"},
		{Header: "LAST_ROTATION", Field: "lastRotation"},
	})
}

func runGwSSHKeypairRotationPolicySet(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	data, err := readResourceFromFileOrStdin(gwRotationFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPatch("/api/gateways/ssh-keypair/rotation", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if !quiet {
		fmt.Println("SSH keypair rotation policy updated")
	}
}
