package migrations

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type Definition struct {
	Version int64
	Name    string
	Path    string
}

//go:embed *.sql
var files embed.FS

func Definitions() ([]Definition, error) {
	paths, err := fs.Glob(files, "*.sql")
	if err != nil {
		return nil, fmt.Errorf("glob migrations: %w", err)
	}
	if len(paths) == 0 {
		return nil, nil
	}

	defs := make([]Definition, 0, len(paths))
	for _, path := range paths {
		base := filepath.Base(path)
		version, err := parseVersion(base)
		if err != nil {
			return nil, fmt.Errorf("parse migration %q: %w", base, err)
		}
		defs = append(defs, Definition{
			Version: version,
			Name:    base,
			Path:    path,
		})
	}

	slices.SortFunc(defs, func(left, right Definition) int {
		switch {
		case left.Version < right.Version:
			return -1
		case left.Version > right.Version:
			return 1
		default:
			return strings.Compare(left.Name, right.Name)
		}
	})

	return defs, nil
}

func LatestVersion() (int64, error) {
	defs, err := Definitions()
	if err != nil {
		return 0, err
	}
	if len(defs) == 0 {
		return 0, nil
	}
	return defs[len(defs)-1].Version, nil
}

func BaselineVersion() (int64, error) {
	defs, err := Definitions()
	if err != nil {
		return 0, err
	}
	if len(defs) == 0 {
		return 0, nil
	}
	return defs[0].Version, nil
}

func Read(def Definition) ([]byte, error) {
	payload, err := files.ReadFile(def.Path)
	if err != nil {
		return nil, fmt.Errorf("read migration %q: %w", def.Name, err)
	}
	return payload, nil
}

func parseVersion(name string) (int64, error) {
	base := strings.TrimSpace(strings.TrimSuffix(name, filepath.Ext(name)))
	if base == "" {
		return 0, fmt.Errorf("missing version prefix")
	}

	prefix := strings.Builder{}
	for _, r := range base {
		if r < '0' || r > '9' {
			break
		}
		prefix.WriteRune(r)
	}
	if prefix.Len() == 0 {
		return 0, fmt.Errorf("missing numeric prefix")
	}

	version, err := strconv.ParseInt(prefix.String(), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric prefix: %w", err)
	}
	return version, nil
}
