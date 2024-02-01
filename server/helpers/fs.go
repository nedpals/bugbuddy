package helpers

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/liamg/memoryfs"
)

type SharedFS struct {
	memfs *memoryfs.FS
}

func NewSharedFS() *SharedFS {
	return &SharedFS{
		memfs: memoryfs.New(),
	}
}

func (sfs *SharedFS) Remove(name string) error {
	return sfs.memfs.Remove(name)
}

func (sfs *SharedFS) WriteFile(name string, content []byte) error {
	return sfs.memfs.WriteFile(name, content, 0o700)
}

func (sfs *SharedFS) Open(name string) (fs.File, error) {
	file, err := sfs.memfs.Open(name)
	if err != nil {
		fmt.Println(err)

		file, err := os.Open(name)
		if err != nil {
			return nil, err
		}

		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}

		if err := sfs.memfs.MkdirAll(filepath.Dir(name), 0o700); err != nil {
			return nil, err
		}

		if err := sfs.WriteFile(name, content); err != nil {
			return nil, err
		}

		return sfs.memfs.Open(name)
	}
	return file, nil
}

func (sfs *SharedFS) ReadFile(name string) ([]byte, error) {
	file, err := sfs.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return io.ReadAll(file)
}
