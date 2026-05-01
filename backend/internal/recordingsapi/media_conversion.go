package recordingsapi

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func (s Service) ConvertToVideo(ctx context.Context, item recordingResponse) (string, int64, error) {
	if item.Status != "COMPLETE" {
		return "", 0, &requestError{status: http.StatusBadRequest, message: "Recording is not complete"}
	}
	if item.Format != "guac" && item.Format != "asciicast" {
		return "", 0, &requestError{status: http.StatusBadRequest, message: "Video export is only available for RDP/VNC/SSH recordings"}
	}

	videoExt := ".m4v"
	if item.Format == "asciicast" {
		videoExt = ".mp4"
	}
	videoPath := item.FilePath + videoExt
	if info, err := os.Stat(videoPath); err == nil {
		return videoPath, info.Size(), nil
	}

	serviceURL := s.GuacencServiceURL
	endpoint := "/convert"
	if item.Format == "asciicast" {
		if strings.TrimSpace(s.AsciicastConverterURL) != "" {
			serviceURL = s.AsciicastConverterURL
		}
		endpoint = "/convert-asciicast"
	}
	serviceURL = s.resolveGuacencURL(serviceURL)
	if strings.TrimSpace(serviceURL) == "" {
		return "", 0, &requestError{status: http.StatusServiceUnavailable, message: "Video conversion service unavailable"}
	}

	client, err := s.guacencClient()
	if err != nil {
		return "", 0, fmt.Errorf("configure guacenc client: %w", err)
	}

	body := map[string]any{
		"filePath": s.toContainerPath(item.FilePath),
	}
	if item.Format != "asciicast" {
		width := defaultVideoWidth
		height := defaultVideoHeight
		if item.Width != nil && *item.Width > 0 {
			width = *item.Width
		}
		if item.Height != nil && *item.Height > 0 {
			height = *item.Height
		}
		body["width"] = width
		body["height"] = height
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return "", 0, fmt.Errorf("marshal conversion request: %w", err)
	}

	submitCtx, cancelSubmit := context.WithTimeout(ctx, guacencSubmitTimeout)
	defer cancelSubmit()

	req, err := http.NewRequestWithContext(submitCtx, http.MethodPost, serviceURL+endpoint, strings.NewReader(string(rawBody)))
	if err != nil {
		return "", 0, fmt.Errorf("create conversion request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token := strings.TrimSpace(s.GuacencAuthToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, s.mapFetchError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var payload map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		detail, _ := payload["error"].(string)
		if detail == "" {
			detail = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		status := http.StatusServiceUnavailable
		if resp.StatusCode < 500 {
			status = http.StatusBadGateway
		}
		return "", 0, &requestError{status: status, message: "Video conversion failed: " + detail}
	}

	var submitResult struct {
		JobID string `json:"jobId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&submitResult); err != nil {
		return "", 0, fmt.Errorf("decode conversion response: %w", err)
	}
	if strings.TrimSpace(submitResult.JobID) == "" {
		return "", 0, &requestError{status: http.StatusBadGateway, message: "Video conversion failed: missing job id"}
	}

	deadline := time.Now().Add(s.guacencTimeout())
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", 0, ctx.Err()
		case <-time.After(guacencPollInterval):
		}

		statusCtx, cancelStatus := context.WithTimeout(ctx, guacencStatusTimeout)
		statusReq, err := http.NewRequestWithContext(statusCtx, http.MethodGet, serviceURL+"/status/"+submitResult.JobID, nil)
		if err != nil {
			cancelStatus()
			return "", 0, fmt.Errorf("create status request: %w", err)
		}
		if token := strings.TrimSpace(s.GuacencAuthToken); token != "" {
			statusReq.Header.Set("Authorization", "Bearer "+token)
		}

		statusResp, err := client.Do(statusReq)
		if err != nil {
			cancelStatus()
			return "", 0, s.mapFetchError(err)
		}

		var job struct {
			Status     string `json:"status"`
			OutputPath string `json:"outputPath"`
			FileSize   int64  `json:"fileSize"`
			Error      string `json:"error"`
		}
		decodeErr := json.NewDecoder(statusResp.Body).Decode(&job)
		statusResp.Body.Close()
		cancelStatus()
		if decodeErr != nil {
			return "", 0, fmt.Errorf("decode status response: %w", decodeErr)
		}
		if statusResp.StatusCode < 200 || statusResp.StatusCode >= 300 {
			return "", 0, &requestError{status: http.StatusBadGateway, message: "Failed to check conversion status"}
		}

		switch job.Status {
		case "complete":
			return s.toHostPath(job.OutputPath), job.FileSize, nil
		case "error":
			detail := strings.TrimSpace(job.Error)
			if detail == "" {
				detail = "unknown"
			}
			return "", 0, &requestError{status: http.StatusBadGateway, message: "Video conversion failed: " + detail}
		}
	}

	return "", 0, &requestError{status: http.StatusGatewayTimeout, message: "Video conversion timed out"}
}

func (s Service) resolveGuacencURL(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return ""
	}
	if s.GuacencUseTLS {
		return strings.Replace(baseURL, "http://", "https://", 1)
	}
	return baseURL
}

func (s Service) toContainerPath(hostPath string) string {
	recordingPath := strings.TrimRight(strings.TrimSpace(s.RecordingPath), "/")
	containerPath := strings.TrimRight(strings.TrimSpace(s.GuacencRecordingPath), "/")
	if recordingPath == "" || containerPath == "" {
		return hostPath
	}
	if hostPath == recordingPath {
		return containerPath
	}
	if strings.HasPrefix(hostPath, recordingPath+"/") {
		return containerPath + strings.TrimPrefix(hostPath, recordingPath)
	}
	return hostPath
}

func (s Service) toHostPath(containerPath string) string {
	recordingPath := strings.TrimRight(strings.TrimSpace(s.RecordingPath), "/")
	guacPath := strings.TrimRight(strings.TrimSpace(s.GuacencRecordingPath), "/")
	if recordingPath == "" || guacPath == "" {
		return containerPath
	}
	if containerPath == guacPath {
		return recordingPath
	}
	if strings.HasPrefix(containerPath, guacPath+"/") {
		return recordingPath + strings.TrimPrefix(containerPath, guacPath)
	}
	return containerPath
}

func (s Service) guacencTimeout() time.Duration {
	if s.GuacencTimeout > 0 {
		return s.GuacencTimeout
	}
	return defaultGuacencTimeout
}

func (s Service) guacencClient() (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if !s.GuacencUseTLS {
		return &http.Client{Transport: transport}, nil
	}

	rootCAs, err := x509.SystemCertPool()
	if err != nil || rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	if certPath := strings.TrimSpace(s.GuacencTLSCA); certPath != "" {
		pem, readErr := os.ReadFile(certPath)
		if readErr != nil {
			return nil, fmt.Errorf("read guacenc CA: %w", readErr)
		}
		if ok := rootCAs.AppendCertsFromPEM(pem); !ok {
			return nil, errors.New("failed to append guacenc CA certificate")
		}
	}

	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    rootCAs,
	}
	return &http.Client{Transport: transport}, nil
}

func (s Service) mapFetchError(err error) error {
	var reqErr *requestError
	if errors.As(err, &reqErr) {
		return reqErr
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &requestError{status: http.StatusGatewayTimeout, message: "Video conversion timed out"}
	}
	return &requestError{status: http.StatusServiceUnavailable, message: "Video conversion service unavailable"}
}
