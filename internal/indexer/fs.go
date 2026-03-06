package indexer

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FS abstracts filesystem operations for testability.
type FS interface {
	Walk(root string, fn fs.WalkDirFunc) error
	ReadFile(path string) ([]byte, error)
	Stat(path string) (FileInfo, error)
}

// FileInfo holds file metadata.
type FileInfo struct {
	ModTime time.Time
	Size    int64
}

// RealFS uses the real filesystem.
type RealFS struct{}

func NewRealFS() *RealFS { return &RealFS{} }

func (r *RealFS) Walk(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

func (r *RealFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (r *RealFS) Stat(path string) (FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{ModTime: info.ModTime(), Size: info.Size()}, nil
}

// MemFile represents a file in MemFS.
type MemFile struct {
	Content []byte
	ModTime time.Time
	Size    int64
}

// MemFS is an in-memory filesystem for testing.
type MemFS struct {
	Files map[string]*MemFile
}

func NewMemFS() *MemFS {
	return &MemFS{Files: make(map[string]*MemFile)}
}

func (m *MemFS) AddFile(path string, content []byte, modTime time.Time) {
	m.Files[path] = &MemFile{Content: content, ModTime: modTime, Size: int64(len(content))}
}

func (m *MemFS) Walk(root string, fn fs.WalkDirFunc) error {
	// Collect and sort paths that are under root
	var paths []string
	for p := range m.Files {
		if strings.HasPrefix(p, root) {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)

	for _, p := range paths {
		f := m.Files[p]
		entry := &memDirEntry{name: filepath.Base(p), size: f.Size}
		if err := fn(p, entry, nil); err != nil {
			if err == fs.SkipDir {
				continue
			}
			return err
		}
	}
	return nil
}

func (m *MemFS) ReadFile(path string) ([]byte, error) {
	f, ok := m.Files[path]
	if !ok {
		return nil, &os.PathError{Op: "open", Path: path, Err: os.ErrNotExist}
	}
	return f.Content, nil
}

func (m *MemFS) Stat(path string) (FileInfo, error) {
	f, ok := m.Files[path]
	if !ok {
		return FileInfo{}, &os.PathError{Op: "stat", Path: path, Err: os.ErrNotExist}
	}
	return FileInfo{ModTime: f.ModTime, Size: f.Size}, nil
}

// memDirEntry implements fs.DirEntry for MemFS.
type memDirEntry struct {
	name string
	size int64
}

func (e *memDirEntry) Name() string               { return e.name }
func (e *memDirEntry) IsDir() bool                { return false }
func (e *memDirEntry) Type() fs.FileMode          { return 0 }
func (e *memDirEntry) Info() (fs.FileInfo, error) { return nil, nil }
