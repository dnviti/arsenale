package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func (s *tileService) fetchUpstreamTile(ctx context.Context, key tileKey) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, expandTileURL(s.config.tileURL, key), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", s.config.userAgent)

	res, err := s.config.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned status %d", res.StatusCode)
	}
	if contentType := strings.ToLower(strings.TrimSpace(res.Header.Get("Content-Type"))); contentType != "" && !strings.HasPrefix(contentType, "image/png") {
		return nil, fmt.Errorf("upstream returned unsupported content type %q", contentType)
	}

	data, err := io.ReadAll(io.LimitReader(res.Body, s.config.maxTileBytes))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("upstream returned empty tile")
	}
	return data, nil
}

func expandTileURL(template string, key tileKey) string {
	replacer := strings.NewReplacer(
		"{z}", strconv.FormatUint(uint64(key.z), 10),
		"{x}", strconv.FormatUint(uint64(key.x), 10),
		"{y}", strconv.FormatUint(uint64(key.y), 10),
	)
	return replacer.Replace(template)
}
