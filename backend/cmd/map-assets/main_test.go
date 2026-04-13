package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestWorldMetadataEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	newTestTileService(t, "http://tiles.example.test/{z}/{x}/{y}.png", nil).registerRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/tiles/world/metadata", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	var metadata tileMetadataResponse
	if err := json.Unmarshal(res.Body.Bytes(), &metadata); err != nil {
		t.Fatalf("failed to decode metadata: %v", err)
	}
	if metadata.URLTemplate != "/v1/tiles/world/{z}/{x}/{y}" {
		t.Fatalf("unexpected url template: %q", metadata.URLTemplate)
	}
	if metadata.MaxNativeZoom != maxNativeZoom {
		t.Fatalf("unexpected max native zoom: %d", metadata.MaxNativeZoom)
	}
	if !strings.Contains(metadata.Attribution, "OpenStreetMap") {
		t.Fatalf("unexpected attribution: %q", metadata.Attribution)
	}
}

func TestWorldTileEndpointFetchesAndCachesPNG(t *testing.T) {
	pngData := tinyPNG(t)
	var requestCount atomic.Int32
	var userAgent string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		userAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngData)
	}))
	defer upstream.Close()

	service := newTestTileService(t, upstream.URL+"/{z}/{x}/{y}.png", upstream.Client())
	mux := http.NewServeMux()
	service.registerRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/tiles/world/0/0/0", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if contentType := res.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "image/png") {
		t.Fatalf("unexpected content type: %q", contentType)
	}
	if cacheControl := res.Header().Get("Cache-Control"); cacheControl != tileCacheControl {
		t.Fatalf("unexpected cache control: %q", cacheControl)
	}
	if etag := res.Header().Get("ETag"); etag == "" {
		t.Fatal("expected ETag header")
	}
	if _, err := png.Decode(bytes.NewReader(res.Body.Bytes())); err != nil {
		t.Fatalf("response was not a decodable png: %v", err)
	}
	if got := requestCount.Load(); got != 1 {
		t.Fatalf("expected one upstream request, got %d", got)
	}
	if userAgent != defaultTileUserAgent {
		t.Fatalf("unexpected user agent: %q", userAgent)
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/v1/tiles/world/0/0/0", nil)
	secondRes := httptest.NewRecorder()
	mux.ServeHTTP(secondRes, secondReq)

	if secondRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", secondRes.Code)
	}
	if got := requestCount.Load(); got != 1 {
		t.Fatalf("expected cached second response, got %d upstream requests", got)
	}

	cachePath := filepath.Join(service.config.cacheDir, "0", "0", "0.png")
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("expected cached tile on disk: %v", err)
	}
}

func TestWorldTileEndpointDoesNotEnableBrowserPersistence(t *testing.T) {
	pngData := tinyPNG(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngData)
	}))
	defer upstream.Close()

	mux := http.NewServeMux()
	newTestTileService(t, upstream.URL+"/{z}/{x}/{y}.png", upstream.Client()).registerRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/tiles/world/0/0/0", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected no-store cache control, got %q", res.Header().Get("Cache-Control"))
	}
}

func TestWorldTileEndpointSupportsConditionalRequests(t *testing.T) {
	pngData := tinyPNG(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngData)
	}))
	defer upstream.Close()

	mux := http.NewServeMux()
	newTestTileService(t, upstream.URL+"/{z}/{x}/{y}.png", upstream.Client()).registerRoutes(mux)

	firstReq := httptest.NewRequest(http.MethodGet, "/v1/tiles/world/0/0/0", nil)
	firstRes := httptest.NewRecorder()
	mux.ServeHTTP(firstRes, firstReq)

	etag := firstRes.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header on first response")
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/v1/tiles/world/0/0/0", nil)
	secondReq.Header.Set("If-None-Match", etag)
	secondRes := httptest.NewRecorder()
	mux.ServeHTTP(secondRes, secondReq)

	if secondRes.Code != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", secondRes.Code)
	}
}

func TestWorldTileEndpointRejectsOutOfRangeCoordinates(t *testing.T) {
	mux := http.NewServeMux()
	newTestTileService(t, "http://tiles.example.test/{z}/{x}/{y}.png", nil).registerRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/tiles/world/20/0/0", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func newTestTileService(t *testing.T, tileURL string, client *http.Client) *tileService {
	t.Helper()

	config := loadTileServiceConfig()
	config.cacheDir = t.TempDir()
	config.tileURL = tileURL
	if client != nil {
		config.client = client
	}

	service, err := newTileServiceWithConfig(config)
	if err != nil {
		t.Fatalf("failed to create tile service: %v", err)
	}
	return service
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.NRGBA{R: 30, G: 136, B: 229, A: 255})

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test png: %v", err)
	}
	return buf.Bytes()
}
