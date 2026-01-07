package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultCatalogPath returns the path to the catalog file.
// It uses XDG_DATA_HOME if set, otherwise falls back to ~/.local/share.
// When HOME is also unset, falls back to relative path ".local/share/pj/catalog.yaml".
func DefaultCatalogPath() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "pj", "catalog.yaml")
}

func DefaultProjectsDir() string {
	home, _ := os.UserHomeDir()
	projectsDir := filepath.Join(home, "projects")
	if info, err := os.Stat(projectsDir); err == nil && info.IsDir() {
		if _, err := os.ReadDir(projectsDir); err == nil {
			return projectsDir
		}
	}
	return home
}

func ShortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func ExpandPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("path cannot be empty")
	}

	if strings.HasPrefix(path, "~") && !strings.HasPrefix(path, "~/") && path != "~" {
		return "", fmt.Errorf("~username expansion is not supported: %s", path)
	}

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot expand ~: %w", err)
		}
		path = filepath.Join(home, path[2:])
	} else if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot expand ~: %w", err)
		}
		path = home
	}

	return filepath.Abs(path)
}
