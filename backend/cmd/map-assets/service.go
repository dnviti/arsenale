package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/dnviti/arsenale/backend/internal/app"
)

const (
	tileSize         = 256
	minZoom          = 0
	maxNativeZoom    = 7
	maxDisplayZoom   = 12
	maxMercatorLat   = 85.05112878
	tileCacheControl = "public, max-age=86400, stale-while-revalidate=604800"
)

type coordinate [2]float64

type pixelPoint struct {
	x float64
	y float64
}

type tileKey struct {
	z uint8
	x uint32
	y uint32
}

type cachedTile struct {
	data []byte
	etag string
}

type tileMetadataResponse struct {
	Name          string     `json:"name"`
	Format        string     `json:"format"`
	TileSize      int        `json:"tileSize"`
	MinZoom       int        `json:"minZoom"`
	MaxNativeZoom int        `json:"maxNativeZoom"`
	MaxZoom       int        `json:"maxZoom"`
	Bounds        [4]float64 `json:"bounds"`
	URLTemplate   string     `json:"urlTemplate"`
	Graticule     []float64  `json:"graticule"`
	Attribution   string     `json:"attribution"`
}

type tileService struct {
	mu      sync.RWMutex
	cache   map[tileKey]cachedTile
	renders singleflight.Group
}

func newTileService() *tileService {
	return &tileService{
		cache: make(map[tileKey]cachedTile),
	}
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
		Graticule:     []float64{30},
		Attribution:   "Arsenale IP geolocation basemap",
	})
}

func (s *tileService) handleWorldTile(w http.ResponseWriter, r *http.Request) {
	z, x, y, ok := parseTileRequest(r)
	if !ok {
		http.NotFound(w, r)
		return
	}

	tile, err := s.getTile(z, x, y)
	if err != nil {
		http.Error(w, "failed to render tile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", tileCacheControl)
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("ETag", tile.etag)
	http.ServeContent(w, r, "", time.UnixMilli(0), bytes.NewReader(tile.data))
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

func (s *tileService) getTile(z uint8, x uint32, y uint32) (cachedTile, error) {
	key := tileKey{z: z, x: x, y: y}

	s.mu.RLock()
	if tile, ok := s.cache[key]; ok {
		s.mu.RUnlock()
		return tile, nil
	}
	s.mu.RUnlock()

	result, err, _ := s.renders.Do(tileKeyString(key), func() (any, error) {
		s.mu.RLock()
		if tile, ok := s.cache[key]; ok {
			s.mu.RUnlock()
			return tile, nil
		}
		s.mu.RUnlock()

		rendered, renderErr := renderWorldTile(int(z), int(x), int(y))
		if renderErr != nil {
			return cachedTile{}, renderErr
		}
		sum := sha256.Sum256(rendered)
		tile := cachedTile{
			data: rendered,
			etag: `"` + hex.EncodeToString(sum[:]) + `"`,
		}

		s.mu.Lock()
		s.cache[key] = tile
		s.mu.Unlock()
		return tile, nil
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
