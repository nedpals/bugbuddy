package lsp_server

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
)

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

var isTempDirInited = false
var tempDirPath = ""

func getTempDir() string {
	if isTempDirInited {
		return tempDirPath
	}

	path := filepath.Join(os.TempDir(), "bugbuddy")
	if !isTempDirInited {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.Mkdir(path, 0700); err == nil {
				isTempDirInited = true
			}
		} else {
			isTempDirInited = true
		}
	}

	tempDirPath = path
	return path
}

func getTempFilePath(name string) string {
	return filepath.Join(getTempDir(), fmt.Sprintf("bugbuddy-%d.md", hash(name)))
}

func getTempFileForFile(name string) (*os.File, error) {
	fullPath := getTempFilePath(name)
	flags := os.O_RDWR | os.O_EXCL
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		flags |= os.O_CREATE
	}
	return os.OpenFile(fullPath, flags, 0600)
}

func removeAllTempFiles() error {
	return os.RemoveAll(getTempDir())
}
