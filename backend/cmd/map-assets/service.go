package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sync/singleflight"

	"github.com/dnviti/arsenale/backend/internal/app"
)

type tileService struct {
	config  tileServiceConfig
	fetches singleflight.Group
}

func newTileService() (*tileService, error) {
	return newTileServiceWithConfig(loadTileServiceConfig())
}

func newTileServiceWithConfig(config tileServiceConfig) (*tileService, error) {
	if config.client == nil {
		config.client = http.DefaultClient
	}
	if config.maxTileBytes <= 0 {
		config.maxTileBytes = 5 << 20
	}
	if err := os.MkdirAll(config.cacheDir, 0o755); err != nil {
		return nil, err
	}
	return &tileService{config: config}, nil
}

func (s *tileService) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/tiles/world/metadata", s.handleWorldMetadata)
	mux.HandleFunc("GET /v1/tiles/world/{z}/{x}/{y}", s.handleWorldTile)
}

func (s *tileService) handleWorldMetadata(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=3600")
	app.WriteJSON(w, http.StatusOK, tileMetadataResponse{
		Name:          "world",
		Format:        "png",
		TileSize:      tileSize,
		MinZoom:       minZoom,
		MaxNativeZoom: maxNativeZoom,
		MaxZoom:       maxDisplayZoom,
		Bounds:        worldBounds,
		URLTemplate:   "/v1/tiles/world/{z}/{x}/{y}",
		Graticule:     []float64{},
		Attribution:   s.config.attribution,
	})
}

func (s *tileService) handleWorldTile(w http.ResponseWriter, r *http.Request) {
	z, x, y, ok := parseTileRequest(r)
	if !ok {
		http.NotFound(w, r)
		return
	}

	tile, err := s.getTile(r.Context(), tileKey{z: z, x: x, y: y})
	if err != nil {
		http.Error(w, "failed to load tile", http.StatusBadGateway)
		return
	}
	if etagMatches(r.Header.Get("If-None-Match"), tile.etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Cache-Control", tileCacheControl)
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("ETag", tile.etag)
	http.ServeContent(w, r, "", tile.modifiedAt, bytes.NewReader(tile.data))
}

func parseTileRequest(r *http.Request) (uint8, uint32, uint32, bool) {
	zValue, err := strconv.ParseUint(r.PathValue("z"), 10, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	xValue, err := strconv.ParseUint(r.PathValue("x"), 10, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	yValue, err := strconv.ParseUint(r.PathValue("y"), 10, 32)
	if err != nil {
		return 0, 0, 0, false
	}

	z := uint8(zValue)
	if z < minZoom || z > maxNativeZoom {
		return 0, 0, 0, false
	}
	edge := uint32(1) << z
	x := uint32(xValue)
	y := uint32(yValue)
	if x >= edge || y >= edge {
		return 0, 0, 0, false
	}
	return z, x, y, true
}

func (s *tileService) getTile(ctx context.Context, key tileKey) (cachedTile, error) {
	tile, err := s.readCachedTile(key)
	if err == nil {
		return tile, nil
	}
	if !errors.Is(err, errCachedTileNotFound) {
		return cachedTile{}, err
	}

	result, err, _ := s.fetches.Do(tileKeyString(key), func() (any, error) {
		cached, cacheErr := s.readCachedTile(key)
		if cacheErr == nil {
			return cached, nil
		}
		if !errors.Is(cacheErr, errCachedTileNotFound) {
			return cachedTile{}, cacheErr
		}

		data, fetchErr := s.fetchUpstreamTile(ctx, key)
		if fetchErr != nil {
			return cachedTile{}, fetchErr
		}
		return s.writeCachedTile(key, data)
	})
	if err != nil {
		return cachedTile{}, err
	}
	return result.(cachedTile), nil
}

func tileKeyString(key tileKey) string {
	return strconv.FormatUint(uint64(key.z), 10) + "/" +
		strconv.FormatUint(uint64(key.x), 10) + "/" +
		strconv.FormatUint(uint64(key.y), 10)
}

func etagMatches(headerValue string, etag string) bool {
	for _, value := range strings.Split(headerValue, ",") {
		candidate := strings.TrimSpace(value)
		if candidate == "*" || candidate == etag {
			return true
		}
	}
	return false
}
