package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var errCachedTileNotFound = errors.New("cached tile not found")

type tileKey struct {
	z uint8
	x uint32
	y uint32
}

type cachedTile struct {
	data       []byte
	etag       string
	modifiedAt time.Time
}

func (s *tileService) cachePath(key tileKey) string {
	return filepath.Join(
		s.config.cacheDir,
		strconv.FormatUint(uint64(key.z), 10),
		strconv.FormatUint(uint64(key.x), 10),
		strconv.FormatUint(uint64(key.y), 10)+".png",
	)
}

func (s *tileService) readCachedTile(key tileKey) (cachedTile, error) {
	path := s.cachePath(key)
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cachedTile{}, errCachedTileNotFound
		}
		return cachedTile{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cachedTile{}, err
	}
	return buildCachedTile(data, info.ModTime()), nil
}

func (s *tileService) writeCachedTile(key tileKey, data []byte) (cachedTile, error) {
	path := s.cachePath(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return cachedTile{}, err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return cachedTile{}, err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return cachedTile{}, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return cachedTile{}, err
	}
	return buildCachedTile(data, info.ModTime()), nil
}

func buildCachedTile(data []byte, modifiedAt time.Time) cachedTile {
	sum := sha256.Sum256(data)
	return cachedTile{
		data:       data,
		etag:       `"` + hex.EncodeToString(sum[:]) + `"`,
		modifiedAt: modifiedAt.UTC(),
	}
}
