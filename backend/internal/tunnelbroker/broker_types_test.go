package tunnelbroker

import "testing"

func TestNewBrokerDefaultsProxyAdvertiseHostToCanonicalServiceName(t *testing.T) {
	t.Setenv("HOSTNAME", "")

	broker := NewBroker(BrokerConfig{})

	if broker.config.ProxyAdvertiseHost != "tunnel-broker" {
		t.Fatalf("ProxyAdvertiseHost = %q, want tunnel-broker", broker.config.ProxyAdvertiseHost)
	}
}
