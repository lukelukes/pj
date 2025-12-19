package config

import (
	"os"
	"path/filepath"
)

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
