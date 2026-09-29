package agenteval

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

// Read-only views keep archive admission and audit on the same frozen bytes.
// No mutable filesystem methods are available to replay inspection.
type evidenceReadFile interface {
	io.Reader
	io.Closer
	Stat() (os.FileInfo, error)
	ReadDir(int) ([]os.DirEntry, error)
}
type evidenceReadRoot interface {
	Lstat(string) (os.FileInfo, error)
	Open(string) (evidenceReadFile, error)
}
type diskReadRoot struct{ root *os.Root }

func (d diskReadRoot) Lstat(n string) (os.FileInfo, error)     { return d.root.Lstat(n) }
func (d diskReadRoot) Open(n string) (evidenceReadFile, error) { return d.root.Open(n) }

type subReadRoot struct {
	root   evidenceReadRoot
	prefix string
}

func (s subReadRoot) Lstat(n string) (os.FileInfo, error) {
	return s.root.Lstat(path.Join(s.prefix, n))
}
func (s subReadRoot) Open(n string) (evidenceReadFile, error) {
	return s.root.Open(path.Join(s.prefix, n))
}

type frozenFiles map[string][]byte
type frozenInfo struct {
	name string
	size int64
	dir  bool
}

func (f frozenInfo) Name() string { return path.Base(f.name) }
func (f frozenInfo) Size() int64  { return f.size }
func (f frozenInfo) Mode() fs.FileMode {
	if f.dir {
		return fs.ModeDir | 0700
	}
	return 0600
}
func (f frozenInfo) ModTime() time.Time         { return time.Time{} }
func (f frozenInfo) IsDir() bool                { return f.dir }
func (f frozenInfo) Sys() any                   { return nil }
func (f frozenInfo) Type() fs.FileMode          { return f.Mode().Type() }
func (f frozenInfo) Info() (os.FileInfo, error) { return f, nil }
func (m frozenFiles) Lstat(n string) (os.FileInfo, error) {
	if !fs.ValidPath(n) {
		return nil, fs.ErrInvalid
	}
	if b, ok := m[n]; ok {
		return frozenInfo{n, int64(len(b)), false}, nil
	}
	for name := range m {
		if n == "." || strings.HasPrefix(name, n+"/") {
			return frozenInfo{name: n, dir: true}, nil
		}
	}
	return nil, fs.ErrNotExist
}

type frozenFile struct {
	*bytes.Reader
	info    os.FileInfo
	entries []os.DirEntry
	cursor  int
}

func (f *frozenFile) Close() error               { return nil }
func (f *frozenFile) Stat() (os.FileInfo, error) { return f.info, nil }
func (f *frozenFile) ReadDir(n int) ([]os.DirEntry, error) {
	if !f.info.IsDir() {
		return nil, fs.ErrInvalid
	}
	if n <= 0 {
		r := f.entries[f.cursor:]
		f.cursor = len(f.entries)
		return r, nil
	}
	if f.cursor >= len(f.entries) {
		return nil, io.EOF
	}
	end := min(f.cursor+n, len(f.entries))
	r := f.entries[f.cursor:end]
	f.cursor = end
	return r, nil
}
func (m frozenFiles) Open(n string) (evidenceReadFile, error) {
	info, err := m.Lstat(n)
	if err != nil {
		return nil, err
	}
	f := &frozenFile{Reader: bytes.NewReader(m[n]), info: info}
	if info.IsDir() {
		seen := map[string]bool{}
		prefix := n + "/"
		if n == "." {
			prefix = ""
		}
		for name := range m {
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			part := strings.Split(strings.TrimPrefix(name, prefix), "/")[0]
			seen[part] = true
		}
		names := make([]string, 0, len(seen))
		for name := range seen {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			i, _ := m.Lstat(path.Join(n, name))
			f.entries = append(f.entries, i.(frozenInfo))
		}
	}
	return f, nil
}
