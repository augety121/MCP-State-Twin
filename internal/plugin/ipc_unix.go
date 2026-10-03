//go:build !windows

package plugin

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"regexp"
)

func connectControl(ctx context.Context, address string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "unix", address)
}

func listenControl(address string) (net.Listener, error) {
	if !filepath.IsAbs(address) || len(address) > 100 || !regexp.MustCompile(`^statetwin-[a-f0-9]{32}\.sock$`).MatchString(filepath.Base(address)) {
		return nil, errors.New("PLUGIN_CONTROL_DENIED")
	}
	info, err := os.Lstat(filepath.Dir(address))
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("PLUGIN_CONTROL_DENIED")
	}
	if _, err = os.Lstat(address); !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("PLUGIN_CONTROL_DENIED")
	}
	l, err := net.Listen("unix", address)
	if err != nil {
		return nil, errors.New("PLUGIN_CONTROL_DENIED")
	}
	if err = os.Chmod(address, 0600); err != nil {
		l.Close()
		return nil, errors.New("PLUGIN_CONTROL_DENIED")
	}
	return l, nil
}
