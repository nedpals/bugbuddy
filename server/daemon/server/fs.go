package server

import (
	"io"
	"io/fs"
	"os"

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
	if file, err := sfs.memfs.Open(name); err != nil {
		if err := sfs.memfs.WriteLazyFile(name, func() (io.Reader, error) {
			return os.Open(name)
		}, 0o700); err != nil {
			return nil, err
		}
		return sfs.memfs.Open(name)
	} else {
		return file, nil
	}
}

func (sfs *SharedFS) ReadFile(name string) ([]byte, error) {
	file, err := sfs.Open(name)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(file)
}
