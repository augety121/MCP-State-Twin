//go:build !windows

package plugin

import (
	"os/exec"
	"runtime"
	"syscall"
)

func residentMeasure(cmd *exec.Cmd) func() (int64, bool) {
	return func() (int64, bool) {
		if cmd.ProcessState == nil {
			return 0, false
		}
		r, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage)
		if !ok {
			return 0, false
		}
		rss := r.Maxrss
		if runtime.GOOS != "darwin" {
			rss *= 1024
		}
		return rss, rss > 0
	}
}
