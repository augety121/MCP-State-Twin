package plugin

import (
	"golang.org/x/sys/windows"
	"os/exec"
	"unsafe"
)

type processCounters struct {
	Size           uint32
	PageFaults     uint32
	PeakWorkingSet uintptr
	WorkingSet     uintptr
	PeakPaged      uintptr
	Paged          uintptr
	PeakNonPaged   uintptr
	NonPaged       uintptr
	Pagefile       uintptr
	PeakPagefile   uintptr
}

func residentMeasure(cmd *exec.Cmd) func() (int64, bool) {
	h, e := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, uint32(cmd.Process.Pid))
	if e != nil {
		return func() (int64, bool) { return 0, false }
	}
	return func() (int64, bool) {
		defer windows.CloseHandle(h)
		c := processCounters{}
		c.Size = uint32(unsafe.Sizeof(c))
		proc := windows.NewLazySystemDLL("psapi.dll").NewProc("GetProcessMemoryInfo")
		ok, _, _ := proc.Call(uintptr(h), uintptr(unsafe.Pointer(&c)), uintptr(c.Size))
		return int64(c.PeakWorkingSet), ok != 0 && c.PeakWorkingSet > 0
	}
}
