//go:build !windows

package plugin

import (
	"context"
	"net"
)

func dialControl(ctx context.Context, address string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "unix", address)
}
