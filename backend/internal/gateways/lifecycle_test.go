package gateways

import "testing"

func TestIsManagedLifecycleGatewayType(t *testing.T) {
	cases := map[string]bool{
		"MANAGED_SSH": true,
		"GUACD":       true,
		"DB_PROXY":    true,
		"SSH_BASTION": false,
		"":            false,
	}

	for input, want := range cases {
		if got := isManagedLifecycleGatewayType(input); got != want {
			t.Fatalf("isManagedLifecycleGatewayType(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestParseGatewayLogTail(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  int
	}{
		{name: "default on empty", input: "", want: defaultGatewayLogTailLines},
		{name: "default on invalid", input: "abc", want: defaultGatewayLogTailLines},
		{name: "min clamp", input: "-5", want: 1},
		{name: "pass through", input: "25", want: 25},
		{name: "max clamp", input: "9000", want: maxGatewayLogTailLines},
	}

	for _, tc := range cases {
		if got := parseGatewayLogTail(tc.input); got != tc.want {
			t.Fatalf("%s: parseGatewayLogTail(%q) = %d, want %d", tc.name, tc.input, got, tc.want)
		}
	}
}

func TestManagedGatewayPublishedPortsSuppressesTunnelPublishing(t *testing.T) {
	service := Service{}

	ports, err := service.managedGatewayPublishedPorts(gatewayRecord{
		Type:          "MANAGED_SSH",
		PublishPorts:  true,
		TunnelEnabled: true,
	})
	if err != nil {
		t.Fatalf("managedGatewayPublishedPorts returned error: %v", err)
	}
	if len(ports) != 1 {
		t.Fatalf("expected 1 port binding, got %d", len(ports))
	}
	if ports[0].Publish {
		t.Fatal("expected tunnel-enabled managed gateway to suppress host port publishing")
	}
	if ports[0].HostPort != 0 {
		t.Fatalf("expected no host port assignment, got %d", ports[0].HostPort)
	}
}

func TestManagedGatewayInstanceAddressUsesPublishedHostPort(t *testing.T) {
	host, port := managedGatewayInstanceAddress(
		gatewayRecord{Host: "gateway.example"},
		managedContainerInfo{
			Name:           "arsenale-gw-demo",
			PublishedPorts: map[int]int{2222: 40022},
		},
		2222,
	)

	if host != "gateway.example" {
		t.Fatalf("expected published host gateway.example, got %q", host)
	}
	if port != 40022 {
		t.Fatalf("expected published port 40022, got %d", port)
	}
}

func TestManagedGatewayInstanceAddressUsesContainerNameInternally(t *testing.T) {
	host, port := managedGatewayInstanceAddress(
		gatewayRecord{},
		managedContainerInfo{
			Name:           "arsenale-gw-demo",
			ContainerPorts: map[int]int{4822: 4822},
		},
		4822,
	)

	if host != "arsenale-gw-demo" {
		t.Fatalf("expected internal host arsenale-gw-demo, got %q", host)
	}
	if port != 4822 {
		t.Fatalf("expected internal port 4822, got %d", port)
	}
}
