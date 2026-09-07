package task

import (
	"errors"
	"io"
	"os"
	"strings"
)

// ReadFile bounds IO before parsing and confines resolution to a trusted root.
// OpenRoot prevents escapes even if a component is swapped after Lstat. This
// does not protect against hostile writers changing files within that root.
func ReadFile(root, name string, maxBytes int) ([]byte, error) {
	if err := PortablePath(name); err != nil {
		return nil, err
	}
	if maxBytes < 1 {
		return nil, errors.New("invalid file bound")
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, errors.New("cannot open task root")
	}
	defer r.Close()
	parts := strings.Split(name, "/")
	for i := range parts {
		info, err := r.Lstat(strings.Join(parts[:i+1], "/"))
		if err != nil {
			return nil, errors.New("cannot inspect task member")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("task member traverses symlink")
		}
		if i == len(parts)-1 && (!info.Mode().IsRegular() || info.Size() > int64(maxBytes)) {
			return nil, errors.New("task member is non-regular or oversized")
		}
	}
	f, err := r.Open(name)
	if err != nil {
		return nil, errors.New("cannot open task member")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("task member is not regular")
	}
	data, err := io.ReadAll(io.LimitReader(f, int64(maxBytes)+1))
	if err != nil || len(data) > maxBytes {
		return nil, errors.New("task member read failed or exceeds bound")
	}
	return data, nil
}
