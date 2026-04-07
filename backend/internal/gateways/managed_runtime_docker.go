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
	"sort"
	"strconv"
	"strings"
	"time"
)

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
