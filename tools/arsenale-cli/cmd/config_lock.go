package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var (
	configLockTimeout    = 10 * time.Second
	configLockStaleAfter = 2 * time.Minute
)

func acquireConfigLock(configPath string) (func(), error) {
	if configPath == "" {
		return func() {}, nil
	}

	lockPath := configPath + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0700); err != nil {
		return nil, fmt.Errorf("create config lock dir: %w", err)
	}

	deadline := time.Now().Add(configLockTimeout)
	for {
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			_, _ = fmt.Fprintf(file, "pid=%d\ncreated_at=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
			_ = file.Close()
			return func() { _ = os.Remove(lockPath) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("create config lock: %w", err)
		}

		removeStaleConfigLock(lockPath)
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for config lock %s", lockPath)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func removeStaleConfigLock(lockPath string) {
	info, err := os.Stat(lockPath)
	if err != nil {
		return
	}
	if time.Since(info.ModTime()) > configLockStaleAfter {
		_ = os.Remove(lockPath)
	}
}
