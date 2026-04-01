package gateways

import "testing"

func TestBuildManagedGatewayProbeTargetsPrefersRecordedInstanceAddress(t *testing.T) {
	t.Parallel()

	targets := buildManagedGatewayProbeTargets("", 2222, "127.0.0.1", 36707, "arsenale-gw-example", "10.89.0.44")
	if len(targets) != 3 {
		t.Fatalf("expected 3 targets, got %d", len(targets))
	}

	if targets[0].Host != "10.89.0.44" || targets[0].Port != 2222 {
		t.Fatalf("first target = %#v, want runtime container IP first", targets[0])
	}
	if targets[1].Host != "127.0.0.1" || targets[1].Port != 36707 {
		t.Fatalf("second target = %#v, want recorded instance host/port second", targets[1])
	}
	if targets[2].Host != "arsenale-gw-example" || targets[2].Port != 2222 {
		t.Fatalf("third target = %#v, want container name fallback", targets[2])
	}
}

func TestBuildManagedGatewayProbeTargetsDeduplicatesAndSkipsEmpty(t *testing.T) {
	t.Parallel()

	targets := buildManagedGatewayProbeTargets("10.0.0.5", 4822, "10.0.0.5", 4822, "", "10.0.0.5")
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].Host != "10.0.0.5" || targets[0].Port != 4822 {
		t.Fatalf("unexpected target: %#v", targets[0])
	}
}
