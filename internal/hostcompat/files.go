package hostcompat

import (
	"errors"
	"io"
	"os"
)

// Declarations come from trusted, quiescent local directories. These checks
// refuse observed symlinks/special files, but do not harden parent components
// or eliminate concurrent replacement races. Errors never echo input paths.
func readDeclaration(path string, limit int64, prefix string) ([]byte, error) {
	invalid := errors.New(prefix + "_FILE_INVALID")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil, invalid
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, invalid
	}
	defer f.Close()
	info, err = f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil, invalid
	}
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, invalid
	}
	if int64(len(raw)) > limit {
		return nil, errors.New(prefix + "_RESOURCE_LIMIT")
	}
	return raw, nil
}
