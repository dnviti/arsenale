package gateways

import (
	"testing"
	"time"
)

func TestComputeScalingStatusAutoScale(t *testing.T) {
	lastScaleAction := time.Date(2026, 3, 31, 1, 0, 0, 0, time.UTC)
	now := lastScaleAction.Add(2 * time.Minute)
	gateway := gatewayRecord{
		ID:                       "gw-1",
		AutoScale:                true,
		DesiredReplicas:          1,
		MinReplicas:              1,
		MaxReplicas:              5,
		SessionsPerInstance:      3,
		ScaleDownCooldownSeconds: 300,
		LastScaleAction:          &lastScaleAction,
	}

	result := computeScalingStatus(gateway, 10, 2, []instanceSessionSummary{
		{InstanceID: "i-1", ContainerName: "gw-1-1", Count: 4},
	}, now)

	if result.TargetReplicas != 4 {
		t.Fatalf("expected target replicas 4, got %d", result.TargetReplicas)
	}
	if result.Recommendation != "scale-up" {
		t.Fatalf("expected scale-up recommendation, got %q", result.Recommendation)
	}
	if result.CooldownRemaining != 180 {
		t.Fatalf("expected cooldownRemaining 180, got %d", result.CooldownRemaining)
	}
}

func TestComputeScalingStatusManualMode(t *testing.T) {
	gateway := gatewayRecord{
		ID:                       "gw-2",
		AutoScale:                false,
		DesiredReplicas:          3,
		MinReplicas:              1,
		MaxReplicas:              5,
		SessionsPerInstance:      10,
		ScaleDownCooldownSeconds: 300,
	}

	result := computeScalingStatus(gateway, 99, 1, nil, time.Now())

	if result.TargetReplicas != 3 {
		t.Fatalf("expected target replicas to follow desiredReplicas, got %d", result.TargetReplicas)
	}
	if result.Recommendation != "stable" {
		t.Fatalf("expected stable recommendation, got %q", result.Recommendation)
	}
	if result.CooldownRemaining != 0 {
		t.Fatalf("expected cooldownRemaining 0, got %d", result.CooldownRemaining)
	}
}

func TestValidateScalingConfigPayload(t *testing.T) {
	gateway := gatewayRecord{
		MinReplicas:              2,
		MaxReplicas:              5,
		SessionsPerInstance:      10,
		ScaleDownCooldownSeconds: 300,
	}

	maxReplicas := 1
	err := validateScalingConfigPayload(gateway, scalingConfigPayload{
		MaxReplicas: optionalInt{Present: true, Value: &maxReplicas},
	})
	if err == nil {
		t.Fatal("expected validation error for maxReplicas below current minReplicas")
	}
}
