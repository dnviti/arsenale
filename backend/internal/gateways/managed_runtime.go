package gateways

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultEdgeNetwork    = "arsenale-net-edge"
	defaultDBNetwork      = "arsenale-net-db"
	defaultGuacdNetwork   = "arsenale-net-guacd"
	defaultGatewayNetwork = "arsenale-net-gateway"

	localSSHGatewayImage  = "localhost/arsenale_ssh-gateway:latest"
	localGuacdImage       = "localhost/arsenale_guacd:latest"
	localDBProxyImage     = "localhost/arsenale_db-proxy:latest"
	localDevDBProxyImage  = "localhost/arsenale_dev-tunnel-db-proxy:latest"
	remoteSSHGatewayImage = "ghcr.io/dnviti/arsenale/ssh-gateway:latest"
	remoteGuacdImage      = "guacamole/guacd:1.6.0"
	remoteDBProxyImage    = "ghcr.io/dnviti/arsenale/db-proxy:latest"
)

type managedContainerPortBinding struct {
	ContainerPort int
	HostPort      int
	Publish       bool
}

type managedContainerHealthcheck struct {
	Test        []string
	IntervalSec int
	TimeoutSec  int
	Retries     int
	StartPeriod int
}

type managedContainerConfig struct {
	Image         string
	Name          string
	Env           map[string]string
	Ports         []managedContainerPortBinding
	Labels        map[string]string
	Healthcheck   *managedContainerHealthcheck
	Networks      []string
	DNSServers    []string
	ResolvConf    string
	Binds         []string
	User          string
	RestartPolicy string
}

type managedContainerInfo struct {
	ID             string
	Name           string
	IPAddress      string
	NetworkIPs     map[string]string
	Status         string
	Health         string
	ContainerPorts map[int]int
	PublishedPorts map[int]int
}

type dockerSocketClient struct {
	kind       string
	socketPath string
	baseURL    string
	httpClient *http.Client
}

func newDockerSocketClient(kind, socketPath string) (*dockerSocketClient, error) {
	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		return nil, errors.New("container socket path is not configured")
	}

	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", socketPath)
		},
	}

	return &dockerSocketClient{
		kind:       strings.ToLower(strings.TrimSpace(kind)),
		socketPath: socketPath,
		baseURL:    "http://d",
		httpClient: &http.Client{Transport: transport, Timeout: 60 * time.Second},
	}, nil
}

func (c *dockerSocketClient) ping(ctx context.Context) error {
	resp, err := c.doRaw(ctx, http.MethodGet, "/_ping", nil, http.StatusOK)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}

func (c *dockerSocketClient) ensureImage(ctx context.Context, image string) error {
	image = strings.TrimSpace(image)
	if image == "" || strings.HasPrefix(image, "localhost/") {
		return nil
	}

	query := url.Values{}
	query.Set("fromImage", image)
	resp, err := c.doRaw(ctx, http.MethodPost, "/images/create?"+query.Encode(), nil, http.StatusOK)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func (c *dockerSocketClient) deployContainer(ctx context.Context, cfg managedContainerConfig) (managedContainerInfo, error) {
	if err := c.ensureImage(ctx, cfg.Image); err != nil {
		return managedContainerInfo{}, err
	}

	networks := normalizedStrings(cfg.Networks)
	dnsServers := normalizedStrings(cfg.DNSServers)
	primaryNetwork := ""
	if len(networks) > 0 {
		primaryNetwork = networks[0]
	}

	exposedPorts := make(map[string]map[string]struct{})
	portBindings := make(map[string][]map[string]string)
	containerPorts := make(map[int]int)
	for _, port := range cfg.Ports {
		if port.ContainerPort <= 0 {
			continue
		}
		key := fmt.Sprintf("%d/tcp", port.ContainerPort)
		exposedPorts[key] = map[string]struct{}{}
		containerPorts[port.ContainerPort] = port.ContainerPort
		if port.Publish && port.HostPort > 0 {
			portBindings[key] = []map[string]string{{
				"HostIp":   "127.0.0.1",
				"HostPort": strconv.Itoa(port.HostPort),
			}}
		}
	}

	envPairs := make([]string, 0, len(cfg.Env))
	envKeys := make([]string, 0, len(cfg.Env))
	for key := range cfg.Env {
		envKeys = append(envKeys, key)
	}
	sort.Strings(envKeys)
	for _, key := range envKeys {
		envPairs = append(envPairs, key+"="+cfg.Env[key])
	}

	restartPolicy := strings.TrimSpace(cfg.RestartPolicy)
	if restartPolicy == "" {
		restartPolicy = "always"
	}

	payload := map[string]any{
		"Image":  cfg.Image,
		"Env":    envPairs,
		"Labels": cfg.Labels,
		"HostConfig": map[string]any{
			"PortBindings": portBindings,
			"RestartPolicy": map[string]string{
				"Name": restartPolicy,
			},
			"Binds": cfg.Binds,
		},
	}
	if cfg.User != "" {
		payload["User"] = cfg.User
	}
	if len(exposedPorts) > 0 {
		payload["ExposedPorts"] = exposedPorts
	}
	if primaryNetwork != "" {
		payload["HostConfig"].(map[string]any)["NetworkMode"] = primaryNetwork
	}
	if len(dnsServers) > 0 {
		payload["HostConfig"].(map[string]any)["Dns"] = dnsServers
	}
	if cfg.Healthcheck != nil {
		payload["Healthcheck"] = map[string]any{
			"Test":        cfg.Healthcheck.Test,
			"Interval":    int64(cfg.Healthcheck.IntervalSec) * int64(time.Second),
			"Timeout":     int64(cfg.Healthcheck.TimeoutSec) * int64(time.Second),
			"Retries":     cfg.Healthcheck.Retries,
			"StartPeriod": int64(cfg.Healthcheck.StartPeriod) * int64(time.Second),
		}
	}

	var created struct {
		ID string `json:"Id"`
	}
	query := url.Values{}
	query.Set("name", cfg.Name)
	if err := c.doJSON(ctx, http.MethodPost, "/containers/create?"+query.Encode(), payload, &created, http.StatusCreated); err != nil {
		return managedContainerInfo{}, err
	}
	if created.ID == "" {
		return managedContainerInfo{}, errors.New("container create returned an empty id")
	}

	for _, network := range networks[1:] {
		connectPayload := map[string]any{
			"Container": created.ID,
			"EndpointConfig": map[string]any{
				"Aliases": []string{cfg.Name},
			},
		}
		if err := c.doJSON(ctx, http.MethodPost, "/networks/"+url.PathEscape(network)+"/connect", connectPayload, nil, http.StatusOK); err != nil {
			_ = c.removeContainer(ctx, created.ID)
			return managedContainerInfo{}, err
		}
	}

	if err := c.doJSON(ctx, http.MethodPost, "/containers/"+created.ID+"/start", nil, nil, http.StatusNoContent); err != nil {
		_ = c.removeContainer(ctx, created.ID)
		return managedContainerInfo{}, err
	}

	info, err := c.inspectContainer(ctx, created.ID)
	if err != nil {
		return managedContainerInfo{}, err
	}
	if len(info.ContainerPorts) == 0 {
		info.ContainerPorts = containerPorts
	}
	return info, nil
}

func (c *dockerSocketClient) inspectContainer(ctx context.Context, containerID string) (managedContainerInfo, error) {
	var payload dockerContainerInspect
	if err := c.doJSON(ctx, http.MethodGet, "/containers/"+url.PathEscape(strings.TrimSpace(containerID))+"/json", nil, &payload, http.StatusOK); err != nil {
		return managedContainerInfo{}, err
	}

	info := managedContainerInfo{
		ID:             payload.ID,
		Name:           strings.TrimPrefix(payload.Name, "/"),
		NetworkIPs:     make(map[string]string),
		Status:         strings.ToLower(strings.TrimSpace(payload.State.Status)),
		Health:         "none",
		ContainerPorts: make(map[int]int),
		PublishedPorts: make(map[int]int),
	}
	for networkName, network := range payload.NetworkSettings.Networks {
		if ip := strings.TrimSpace(network.IPAddress); ip != "" {
			info.NetworkIPs[networkName] = ip
		}
	}
	for _, networkName := range sortedKeys(info.NetworkIPs) {
		if ip := strings.TrimSpace(info.NetworkIPs[networkName]); ip != "" {
			info.IPAddress = ip
			break
		}
	}
	if payload.State.Health != nil && strings.TrimSpace(payload.State.Health.Status) != "" {
		info.Health = strings.ToLower(strings.TrimSpace(payload.State.Health.Status))
	}
	for portKey, bindings := range payload.NetworkSettings.Ports {
		containerPort, err := parseDockerPortKey(portKey)
		if err != nil {
			continue
		}
		info.ContainerPorts[containerPort] = containerPort
		for _, binding := range bindings {
			hostPort, err := strconv.Atoi(strings.TrimSpace(binding.HostPort))
			if err == nil && hostPort > 0 {
				info.PublishedPorts[containerPort] = hostPort
				break
			}
		}
	}
	if len(info.ContainerPorts) == 0 {
		for portKey := range payload.Config.ExposedPorts {
			containerPort, err := parseDockerPortKey(portKey)
			if err == nil {
				info.ContainerPorts[containerPort] = containerPort
			}
		}
	}
	return info, nil
}

func (c *dockerSocketClient) removeContainer(ctx context.Context, containerID string) error {
	containerID = strings.TrimSpace(containerID)
	if containerID == "" || strings.HasPrefix(containerID, "failed-") {
		return nil
	}
	resp, err := c.doRaw(ctx, http.MethodPost, "/containers/"+url.PathEscape(containerID)+"/stop?t=10", nil, http.StatusNoContent, http.StatusNotModified, http.StatusNotFound)
	if err == nil && resp != nil {
		_ = resp.Body.Close()
	}
	resp, err = c.doRaw(ctx, http.MethodDelete, "/containers/"+url.PathEscape(containerID)+"?force=1", nil, http.StatusNoContent, http.StatusNotFound)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}

func (c *dockerSocketClient) restartContainer(ctx context.Context, containerID string) error {
	return c.doJSON(ctx, http.MethodPost, "/containers/"+url.PathEscape(strings.TrimSpace(containerID))+"/restart?t=10", nil, nil, http.StatusNoContent)
}

func (c *dockerSocketClient) getContainerLogs(ctx context.Context, containerID string, tail int) (string, error) {
	query := url.Values{}
	query.Set("stdout", "1")
	query.Set("stderr", "1")
	query.Set("tail", strconv.Itoa(tail))
	resp, err := c.doRaw(ctx, http.MethodGet, "/containers/"+url.PathEscape(strings.TrimSpace(containerID))+"/logs?"+query.Encode(), nil, http.StatusOK)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read container logs: %w", err)
	}
	return demuxDockerLogStream(body), nil
}

func (c *dockerSocketClient) doJSON(ctx context.Context, method, path string, body any, out any, expectedStatuses ...int) error {
	resp, err := c.doRaw(ctx, method, path, body, expectedStatuses...)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode container runtime response: %w", err)
	}
	return nil
}

func (c *dockerSocketClient) doRaw(ctx context.Context, method, path string, body any, expectedStatuses ...int) (*http.Response, error) {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal container runtime request: %w", err)
		}
		payload = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, payload)
	if err != nil {
		return nil, fmt.Errorf("create container runtime request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s container runtime request failed via %s (%s): %w", strings.ToUpper(c.kind), c.socketPath, path, err)
	}

	for _, status := range expectedStatuses {
		if resp.StatusCode == status {
			return resp, nil
		}
	}

	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	message := strings.TrimSpace(string(bodyBytes))
	if message == "" {
		message = resp.Status
	}
	return nil, fmt.Errorf("%s container runtime request %s %s failed: %s", strings.ToUpper(c.kind), method, path, message)
}

type dockerContainerInspect struct {
	ID    string `json:"Id"`
	Name  string `json:"Name"`
	State struct {
		Status string `json:"Status"`
		Health *struct {
			Status string `json:"Status"`
		} `json:"Health"`
	} `json:"State"`
	Config struct {
		ExposedPorts map[string]any `json:"ExposedPorts"`
	} `json:"Config"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
		Networks map[string]struct {
			IPAddress string `json:"IPAddress"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

func parseDockerPortKey(raw string) (int, error) {
	parts := strings.Split(strings.TrimSpace(raw), "/")
	if len(parts) == 0 {
		return 0, fmt.Errorf("invalid docker port key %q", raw)
	}
	value, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, fmt.Errorf("invalid docker port key %q: %w", raw, err)
	}
	return value, nil
}

func demuxDockerLogStream(payload []byte) string {
	if len(payload) < 8 {
		return string(payload)
	}

	var plain strings.Builder
	for len(payload) >= 8 {
		streamType := payload[0]
		frameSize := int(binary.BigEndian.Uint32(payload[4:8]))
		if frameSize < 0 || len(payload) < 8+frameSize {
			return string(payload)
		}
		if streamType >= 1 && streamType <= 3 {
			plain.Write(payload[8 : 8+frameSize])
		}
		payload = payload[8+frameSize:]
	}
	if plain.Len() == 0 {
		return string(payload)
	}
	return plain.String()
}

func (s Service) managedGatewayRuntime(ctx context.Context) (*dockerSocketClient, string, error) {
	requested := strings.ToLower(strings.TrimSpace(s.OrchestratorType))
	switch requested {
	case "none":
		return nil, "", &requestError{status: http.StatusNotImplemented, message: unavailableOrchestratorMsg}
	case "docker":
		client, err := newDockerSocketClient("docker", s.DockerSocketPath)
		if err != nil {
			return nil, "", err
		}
		if err := client.ping(ctx); err != nil {
			return nil, "", &requestError{status: http.StatusServiceUnavailable, message: unavailableOrchestratorMsg}
		}
		return client, "DOCKER", nil
	case "podman":
		client, err := newDockerSocketClient("podman", s.PodmanSocketPath)
		if err != nil {
			return nil, "", err
		}
		if err := client.ping(ctx); err != nil {
			return nil, "", &requestError{status: http.StatusServiceUnavailable, message: unavailableOrchestratorMsg}
		}
		return client, "PODMAN", nil
	case "", "auto":
		candidates := []struct {
			kind       string
			socketPath string
		}{
			{kind: "podman", socketPath: s.PodmanSocketPath},
			{kind: "docker", socketPath: s.DockerSocketPath},
		}
		for _, candidate := range candidates {
			client, err := newDockerSocketClient(candidate.kind, candidate.socketPath)
			if err != nil {
				continue
			}
			if err := client.ping(ctx); err == nil {
				return client, strings.ToUpper(candidate.kind), nil
			}
		}
		return nil, "", &requestError{status: http.StatusNotImplemented, message: unavailableOrchestratorMsg}
	default:
		return nil, "", &requestError{status: http.StatusNotImplemented, message: unavailableOrchestratorMsg}
	}
}

func (s Service) managedGatewayNetworks(record gatewayRecord) []string {
	edgeNetwork := firstNonEmpty(s.EdgeNetwork, defaultEdgeNetwork)
	dbNetwork := firstNonEmpty(s.DBNetwork, defaultDBNetwork)
	guacdNetwork := firstNonEmpty(s.GuacdNetwork, defaultGuacdNetwork)
	gatewayNetwork := firstNonEmpty(s.GatewayNetwork, defaultGatewayNetwork)

	switch strings.ToUpper(strings.TrimSpace(record.Type)) {
	case "MANAGED_SSH":
		return normalizedStrings([]string{edgeNetwork, gatewayNetwork})
	case "GUACD":
		networks := []string{guacdNetwork}
		if record.TunnelEnabled {
			networks = append(networks, gatewayNetwork)
		}
		return normalizedStrings(networks)
	case "DB_PROXY":
		networks := []string{edgeNetwork, dbNetwork}
		if record.TunnelEnabled {
			networks = append(networks, gatewayNetwork)
		}
		return normalizedStrings(networks)
	default:
		return nil
	}
}

func (s Service) managedGatewayAttachNetworks(record gatewayRecord) []string {
	networks := make([]string, 0, 4)
	if egressNetwork := strings.TrimSpace(s.EgressNetwork); egressNetwork != "" {
		networks = append(networks, egressNetwork)
	}
	networks = append(networks, s.managedGatewayNetworks(record)...)
	return normalizedStrings(networks)
}

func (s Service) managedGatewayImageCandidates(gatewayType string) []string {
	switch strings.ToUpper(strings.TrimSpace(gatewayType)) {
	case "MANAGED_SSH":
		return normalizedStrings([]string{
			firstNonEmpty(s.SSHGatewayImage, localSSHGatewayImage),
			localSSHGatewayImage,
			remoteSSHGatewayImage,
		})
	case "GUACD":
		return normalizedStrings([]string{
			firstNonEmpty(s.GuacdImage, localGuacdImage),
			localGuacdImage,
			remoteGuacdImage,
		})
	case "DB_PROXY":
		return normalizedStrings([]string{
			firstNonEmpty(s.DBProxyImage, remoteDBProxyImage),
			localDBProxyImage,
			localDevDBProxyImage,
			remoteDBProxyImage,
		})
	default:
		return nil
	}
}

func (s Service) managedGatewayPrimaryPort(gatewayType string) int {
	switch strings.ToUpper(strings.TrimSpace(gatewayType)) {
	case "MANAGED_SSH":
		return 2222
	case "GUACD":
		return 4822
	case "DB_PROXY":
		return 5432
	default:
		return 0
	}
}

func (s Service) managedGatewayPublishedPorts(record gatewayRecord) ([]managedContainerPortBinding, error) {
	containerPort := s.managedGatewayPrimaryPort(record.Type)
	if containerPort <= 0 {
		return nil, fmt.Errorf("unsupported managed gateway type %q", record.Type)
	}

	publish := record.PublishPorts && !record.TunnelEnabled
	ports := []managedContainerPortBinding{{
		ContainerPort: containerPort,
		Publish:       publish,
	}}
	if publish {
		hostPort, err := findAvailableLoopbackPort()
		if err != nil {
			return nil, err
		}
		ports[0].HostPort = hostPort
	}
	return ports, nil
}

func (s Service) buildManagedGatewayContainerConfig(ctx context.Context, record gatewayRecord, instanceIndex int) ([]managedContainerConfig, error) {
	env := map[string]string{}

	switch strings.ToUpper(strings.TrimSpace(record.Type)) {
	case "MANAGED_SSH":
		keyPair, err := s.loadSSHKeyPair(ctx, record.TenantID)
		if err != nil {
			return nil, err
		}
		env["SSH_AUTHORIZED_KEYS"] = keyPair.PublicKey

		grpcEnv, err := s.buildManagedSSHGRPCEnv()
		if err != nil {
			return nil, err
		}
		for key, value := range grpcEnv {
			env[key] = value
		}

	case "GUACD":
		guacdEnv, err := s.buildManagedGuacdTLSEnv()
		if err != nil {
			return nil, err
		}
		for key, value := range guacdEnv {
			env[key] = value
		}
	case "DB_PROXY":
		env["DB_LISTEN_PORT"] = "5432"
	default:
		return nil, &requestError{status: http.StatusBadRequest, message: "Only MANAGED_SSH, GUACD, and DB_PROXY gateways can be deployed as managed containers"}
	}

	tunnelEnv, err := s.buildManagedGatewayTunnelEnv(ctx, record)
	if err != nil {
		return nil, err
	}
	for key, value := range tunnelEnv {
		env[key] = value
	}

	labels := map[string]string{
		"arsenale.managed":      "true",
		"arsenale.gateway-id":   record.ID,
		"arsenale.tenant-id":    record.TenantID,
		"arsenale.gateway-type": strings.ToUpper(strings.TrimSpace(record.Type)),
	}

	networks := s.managedGatewayAttachNetworks(record)
	ports, err := s.managedGatewayPublishedPorts(record)
	if err != nil {
		return nil, err
	}

	baseConfig := managedContainerConfig{
		Name:          buildManagedGatewayContainerName(record, instanceIndex),
		Env:           env,
		Ports:         ports,
		Labels:        labels,
		Networks:      networks,
		DNSServers:    normalizedStrings(s.DNSServers),
		ResolvConf:    strings.TrimSpace(s.ResolvConfPath),
		RestartPolicy: "always",
	}
	if baseConfig.ResolvConf != "" {
		baseConfig.Binds = append(baseConfig.Binds, fmt.Sprintf("%s:/etc/resolv.conf:ro", baseConfig.ResolvConf))
	}

	switch strings.ToUpper(strings.TrimSpace(record.Type)) {
	case "GUACD":
		baseConfig.Healthcheck = &managedContainerHealthcheck{
			Test:        []string{"NONE"},
			IntervalSec: 0,
			TimeoutSec:  0,
			Retries:     0,
			StartPeriod: 0,
		}
	case "DB_PROXY":
	}

	configs := make([]managedContainerConfig, 0)
	for _, image := range s.managedGatewayImageCandidates(record.Type) {
		cfg := baseConfig
		cfg.Image = image
		configs = append(configs, cfg)
	}
	return configs, nil
}

func (s Service) buildManagedGatewayTunnelEnv(ctx context.Context, record gatewayRecord) (map[string]string, error) {
	if !record.TunnelEnabled {
		return nil, nil
	}

	gateway := record
	if gateway.TunnelClientCert == nil || gateway.TunnelClientKey == nil || gateway.TunnelClientKeyIV == nil || gateway.TunnelClientKeyTag == nil {
		if err := s.ensureTunnelMaterial(ctx, record.TenantID, record.ID); err != nil {
			return nil, err
		}
		reloaded, err := s.loadGateway(ctx, record.TenantID, record.ID)
		if err != nil {
			return nil, err
		}
		gateway = reloaded
	}

	if gateway.EncryptedTunnelToken == nil || gateway.TunnelTokenIV == nil || gateway.TunnelTokenTag == nil {
		return nil, &requestError{status: http.StatusBadRequest, message: "Tunnel is enabled but no tunnel token is configured for this gateway"}
	}

	token, err := decryptEncryptedField(s.ServerEncryptionKey, encryptedField{
		Ciphertext: *gateway.EncryptedTunnelToken,
		IV:         *gateway.TunnelTokenIV,
		Tag:        *gateway.TunnelTokenTag,
	})
	if err != nil {
		return nil, fmt.Errorf("decrypt tunnel token: %w", err)
	}

	clientKey, err := decryptEncryptedField(s.ServerEncryptionKey, encryptedField{
		Ciphertext: derefString(gateway.TunnelClientKey),
		IV:         derefString(gateway.TunnelClientKeyIV),
		Tag:        derefString(gateway.TunnelClientKeyTag),
	})
	if err != nil {
		return nil, fmt.Errorf("decrypt tunnel client key: %w", err)
	}

	env := map[string]string{
		"TUNNEL_SERVER_URL":  buildManagedTunnelConnectURL(s.TunnelBrokerURL),
		"TUNNEL_TOKEN":       token,
		"TUNNEL_GATEWAY_ID":  gateway.ID,
		"TUNNEL_LOCAL_HOST":  "127.0.0.1",
		"TUNNEL_LOCAL_PORT":  strconv.Itoa(s.managedGatewayPrimaryPort(gateway.Type)),
		"TUNNEL_CLIENT_CERT": derefString(gateway.TunnelClientCert),
		"TUNNEL_CLIENT_KEY":  clientKey,
	}
	return env, nil
}

func (s Service) ensureTunnelMaterial(ctx context.Context, tenantID, gatewayID string) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tunnel material transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := s.ensureTunnelMTLSMaterialTx(ctx, tx, tenantID, gatewayID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tunnel material transaction: %w", err)
	}
	return nil
}

func (s Service) buildManagedSSHGRPCEnv() (map[string]string, error) {
	if strings.TrimSpace(s.GatewayGRPCTLSCA) == "" || strings.TrimSpace(s.GatewayGRPCServerCert) == "" || strings.TrimSpace(s.GatewayGRPCServerKey) == "" {
		return nil, &requestError{status: http.StatusInternalServerError, message: "Managed SSH gateways require GATEWAY_GRPC_TLS_CA plus GATEWAY_GRPC_SERVER_CERT/KEY to enable gRPC key management"}
	}

	caPEM, err := os.ReadFile(s.GatewayGRPCTLSCA)
	if err != nil {
		return nil, fmt.Errorf("read gateway gRPC CA: %w", err)
	}
	certPEM, err := os.ReadFile(s.GatewayGRPCServerCert)
	if err != nil {
		return nil, fmt.Errorf("read gateway gRPC server certificate: %w", err)
	}
	keyPEM, err := os.ReadFile(s.GatewayGRPCServerKey)
	if err != nil {
		return nil, fmt.Errorf("read gateway gRPC server key: %w", err)
	}

	clientCAPEM := caPEM
	clientCAPath := strings.TrimSpace(s.GatewayGRPCClientCA)
	if clientCAPath != "" {
		clientCAPEM, err = os.ReadFile(clientCAPath)
		if err != nil {
			return nil, fmt.Errorf("read gateway gRPC client CA: %w", err)
		}
	}

	return map[string]string{
		"SPIFFE_TRUST_DOMAIN":             firstNonEmpty(s.TunnelTrustDomain, defaultTunnelTrustDomain),
		"GATEWAY_GRPC_TLS_CA_PEM":         strings.TrimSpace(string(caPEM)),
		"GATEWAY_GRPC_TLS_CERT_PEM":       strings.TrimSpace(string(certPEM)),
		"GATEWAY_GRPC_TLS_KEY_PEM":        strings.TrimSpace(string(keyPEM)),
		"GATEWAY_GRPC_CLIENT_CA_PEM":      strings.TrimSpace(string(clientCAPEM)),
		"GATEWAY_GRPC_EXPECTED_SPIFFE_ID": buildServiceSPIFFEID(firstNonEmpty(s.TunnelTrustDomain, defaultTunnelTrustDomain), "control-plane-api"),
	}, nil
}

func (s Service) buildManagedGuacdTLSEnv() (map[string]string, error) {
	if strings.TrimSpace(s.GuacdTLSCert) == "" || strings.TrimSpace(s.GuacdTLSKey) == "" {
		return nil, &requestError{status: http.StatusInternalServerError, message: "Managed GUACD gateways require ORCHESTRATOR_GUACD_TLS_CERT/KEY so desktop-broker TLS can reach guacd"}
	}

	certPEM, err := os.ReadFile(s.GuacdTLSCert)
	if err != nil {
		return nil, fmt.Errorf("read managed guacd certificate: %w", err)
	}
	keyPEM, err := os.ReadFile(s.GuacdTLSKey)
	if err != nil {
		return nil, fmt.Errorf("read managed guacd private key: %w", err)
	}

	return map[string]string{
		"GUACD_SSL":          "true",
		"GUACD_SSL_CERT_PEM": strings.TrimSpace(string(certPEM)),
		"GUACD_SSL_KEY_PEM":  strings.TrimSpace(string(keyPEM)),
	}, nil
}

func buildManagedGatewayContainerName(record gatewayRecord, instanceIndex int) string {
	tenantSlug := sanitizeGatewayName(record.TenantID)
	if len(tenantSlug) > 8 {
		tenantSlug = tenantSlug[:8]
	}
	nameSlug := sanitizeGatewayName(record.Name)
	if len(nameSlug) > 32 {
		nameSlug = nameSlug[:32]
	}
	idSuffix := sanitizeGatewayName(record.ID)
	if len(idSuffix) > 8 {
		idSuffix = idSuffix[:8]
	}
	return fmt.Sprintf("arsenale-gw-%s-%s-%s-%d", tenantSlug, nameSlug, idSuffix, instanceIndex)
}

func buildManagedTunnelConnectURL(rawBaseURL string) string {
	rawBaseURL = strings.TrimSpace(rawBaseURL)
	if rawBaseURL == "" {
		return "ws://tunnel-broker:8092/api/tunnel/connect"
	}

	parsed, err := url.Parse(rawBaseURL)
	if err != nil {
		return "ws://tunnel-broker:8092/api/tunnel/connect"
	}

	switch parsed.Scheme {
	case "https":
		parsed.Scheme = "wss"
	case "http":
		parsed.Scheme = "ws"
	case "ws", "wss":
	default:
		parsed.Scheme = "ws"
	}

	path := strings.TrimRight(parsed.Path, "/")
	if !strings.HasSuffix(path, "/api/tunnel/connect") {
		if path == "" {
			path = "/api/tunnel/connect"
		} else {
			path += "/api/tunnel/connect"
		}
	}
	parsed.Path = path
	parsed.RawPath = ""
	return parsed.String()
}

func buildServiceSPIFFEID(trustDomain, service string) string {
	trustDomain = strings.TrimSpace(trustDomain)
	if trustDomain == "" {
		trustDomain = defaultTunnelTrustDomain
	}
	service = strings.TrimSpace(service)
	if service == "" {
		service = "control-plane-api"
	}
	return "spiffe://" + trustDomain + "/service/" + service
}

func sanitizeGatewayName(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return "gateway"
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range raw {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "gateway"
	}
	return result
}

func findAvailableLoopbackPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("reserve loopback port: %w", err)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok || addr.Port <= 0 {
		return 0, fmt.Errorf("allocate loopback port: unexpected address %T", listener.Addr())
	}
	return addr.Port, nil
}

func normalizedStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || slices.Contains(result, value) {
			continue
		}
		result = append(result, value)
	}
	return result
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func inferInstanceHealth(status, health string) string {
	switch strings.ToLower(strings.TrimSpace(health)) {
	case "healthy", "unhealthy", "starting", "restarting":
		return strings.ToLower(strings.TrimSpace(health))
	}
	if strings.EqualFold(strings.TrimSpace(status), "running") {
		return "healthy"
	}
	return "unhealthy"
}

func inferInstanceStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "created", "configured":
		return "PROVISIONING"
	case "running":
		return "RUNNING"
	case "restarting", "paused", "exited", "dead", "stopped":
		return "STOPPED"
	default:
		return "ERROR"
	}
}

func inferPrimaryInstanceHost(record gatewayRecord, containerName string) string {
	if strings.TrimSpace(containerName) != "" {
		return strings.TrimSpace(containerName)
	}
	if strings.TrimSpace(record.Host) != "" {
		return strings.TrimSpace(record.Host)
	}
	return "localhost"
}

func managedGatewayAPIPort(record gatewayRecord, defaultGRPCPort int) *int {
	if !strings.EqualFold(strings.TrimSpace(record.Type), "MANAGED_SSH") {
		return nil
	}
	value := defaultGRPCPort
	return &value
}

func defaultGatewayGRPCClientCAPath(certPath string) string {
	certPath = strings.TrimSpace(certPath)
	if certPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(certPath), "client-ca.pem")
}
