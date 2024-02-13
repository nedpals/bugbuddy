package helpers

import (
	"os"
	"path/filepath"
)

var bbDirPath = ""

func SetDirPath(newPath string) error {
	// NOTE: when path does not exist, use GetOrInitializeDir

	// set the new path
	bbDirPath = newPath
	return nil
}

func GetDirPath() string {
	// check if BUGBUDDY_DIR is set. if present, set it as the directory path
	if envPath := os.Getenv("BUGBUDDY_DIR"); len(envPath) != 0 && bbDirPath != envPath {
		SetDirPath(envPath)
	}

	if len(bbDirPath) != 0 {
		return bbDirPath
	}

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
