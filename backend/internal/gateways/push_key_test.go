package gateways

import (
	"context"
	"errors"
	"testing"
)

func TestExecutePushKeyTargets(t *testing.T) {
	targets := []gatewayKeyTarget{
		{InstanceID: "instance-1", Host: "10.0.0.1", Port: 9022},
		{InstanceID: "instance-2", Host: "10.0.0.2", Port: 9022},
		{InstanceID: "direct", Host: "gateway.internal", Port: 9443},
	}

	results := executePushKeyTargets(context.Background(), targets, "ssh-ed25519 AAAA", func(_ context.Context, host string, _ int, _ string) (gatewayKeyPushWireResponse, error) {
		switch host {
		case "10.0.0.1":
			return gatewayKeyPushWireResponse{OK: true}, nil
		case "10.0.0.2":
			return gatewayKeyPushWireResponse{OK: false, Message: "permission denied"}, nil
		default:
			return gatewayKeyPushWireResponse{}, errors.New("dial tcp timeout")
		}
	})

	if len(results) != 3 {
		t.Fatalf("expected 3 push results, got %d", len(results))
	}
	if !results[0].OK {
		t.Fatalf("expected first instance push to succeed: %#v", results[0])
	}
	if results[1].Error != "permission denied" {
		t.Fatalf("expected second instance failure message to be preserved, got %#v", results[1])
	}
	if results[2].Error != "dial tcp timeout" {
		t.Fatalf("expected direct push error to capture transport failure, got %#v", results[2])
	}
}

func TestSummarizePushKeyResults(t *testing.T) {
	results := []pushKeyInstanceResult{
		{InstanceID: "a", OK: true},
		{InstanceID: "b", OK: false},
		{InstanceID: "c", OK: false},
	}

	succeeded, failed := summarizePushKeyResults(results)

	if succeeded != 1 || failed != 2 {
		t.Fatalf("expected summary 1 succeeded / 2 failed, got %d / %d", succeeded, failed)
	}
	if countFailedPushKeyResults(results) != 2 {
		t.Fatalf("expected failed count 2, got %d", countFailedPushKeyResults(results))
	}
}
