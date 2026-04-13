package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	tileSize               = 256
	minZoom                = 0
	maxNativeZoom          = 19
	maxDisplayZoom         = 19
	maxMercatorLat         = 85.05112878
	tileCacheControl       = "no-store"
	defaultTileURLTemplate = "https://tile.openstreetmap.org/{z}/{x}/{y}.png"
	defaultTileCacheDir    = "/var/lib/map-assets-cache/world"
	defaultTileUserAgent   = "arsenale-map-assets/1.0 (+https://github.com/dnviti/arsenale)"
)

var worldBounds = [4]float64{-180, -maxMercatorLat, 180, maxMercatorLat}

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

type tileServiceConfig struct {
	cacheDir     string
	tileURL      string
	attribution  string
	userAgent    string
	client       *http.Client
	maxTileBytes int64
}

func loadTileServiceConfig() tileServiceConfig {
	cacheDir := strings.TrimSpace(os.Getenv("MAP_ASSETS_CACHE_DIR"))
	if cacheDir == "" {
		cacheDir = defaultTileCacheDir
	}

	tileURL := strings.TrimSpace(os.Getenv("MAP_ASSETS_TILE_URL_TEMPLATE"))
	if tileURL == "" {
		tileURL = defaultTileURLTemplate
	}

	userAgent := strings.TrimSpace(os.Getenv("MAP_ASSETS_USER_AGENT"))
	if userAgent == "" {
		userAgent = defaultTileUserAgent
	}

	return tileServiceConfig{
		cacheDir:    filepath.Clean(cacheDir),
		tileURL:     tileURL,
		attribution: `&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors`,
		userAgent:   userAgent,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		maxTileBytes: 5 << 20,
	}
}
