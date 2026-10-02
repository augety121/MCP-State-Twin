package plugin

import (
	"context"
	"github.com/Microsoft/go-winio"
	"net"
)

func dialControl(ctx context.Context, address string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, address)
}
