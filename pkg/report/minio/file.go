package report

import (
	"io"
	"os"
	"time"
)

type memoryFile struct {
	at      int64
	name    string
	read    io.Reader
	size    int64
	modTime time.Time
}

func (f *memoryFile) Close() error {
	return nil
}

func (f *memoryFile) Stat() (os.FileInfo, error) {
	return &memoryFileInfo{f}, nil
}

func (f *memoryFile) Readdir(_ int) ([]os.FileInfo, error) {
	return make([]os.FileInfo, 0), nil
}

func (f *memoryFile) Read(b []byte) (int, error) {
	return f.read.Read(b)
}

func (f *memoryFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		f.at = offset
	case io.SeekCurrent:
		f.at += offset
	case io.SeekEnd:
		f.at = f.size + offset
	}
	return f.at, nil
}

type memoryFileInfo struct {
	f *memoryFile
}

func (s *memoryFileInfo) Name() string       { return s.f.name }
func (s *memoryFileInfo) Size() int64        { return s.f.size }
func (s *memoryFileInfo) ModTime() time.Time { return s.f.modTime }
func (s *memoryFileInfo) Mode() os.FileMode  { return os.ModeTemporary }
func (s *memoryFileInfo) IsDir() bool        { return false }
func (s *memoryFileInfo) Sys() interface{}   { return nil }
