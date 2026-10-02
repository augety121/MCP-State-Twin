package plugin

import (
	"context"
	"errors"
	"net"
	"regexp"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

func connectControl(ctx context.Context, address string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, address)
}

func listenControl(address string) (net.Listener, error) {
	if !regexp.MustCompile(`^\\\\\.\\pipe\\statetwin-[a-f0-9]{32}$`).MatchString(address) {
		return nil, errors.New("PLUGIN_CONTROL_DENIED")
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, errors.New("PLUGIN_CONTROL_DENIED")
	}
	l, err := winio.ListenPipe(address, &winio.PipeConfig{SecurityDescriptor: "D:P(A;;GA;;;" + user.User.Sid.String() + ")", InputBufferSize: 65536, OutputBufferSize: 65536})
	if err != nil {
		return nil, errors.New("PLUGIN_CONTROL_DENIED")
	}
	return l, nil
}
