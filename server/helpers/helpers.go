package helpers

import (
	"os"
	"path/filepath"
)

func GetDirPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		if homeEnv := os.Getenv("HOME"); len(homeEnv) != 0 {
			homeDir = homeEnv
		} else {
			homeDir = os.TempDir()
		}
	}

	return filepath.Join(homeDir, ".bugbuddy")
}

func GetOrInitializeDir() (string, error) {
	dirPath := GetDirPath()
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return "", err
		}
	}

	return dirPath, nil
}
