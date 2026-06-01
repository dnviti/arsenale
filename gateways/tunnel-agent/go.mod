module github.com/dnviti/arsenale/gateways/tunnel-agent

go 1.25.0

require (
	github.com/dnviti/arsenale/gateways/gateway-core v0.0.0
	github.com/gorilla/websocket v1.5.3
	github.com/quic-go/quic-go v0.59.1
)

require (
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
)

replace github.com/dnviti/arsenale/gateways/gateway-core => ../gateway-core
