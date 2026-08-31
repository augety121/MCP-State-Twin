// Package governor applies host-local execution policy without changing
// deterministic world semantics. It deliberately does not claim an OS-level
// CPU quota: GOMAXPROCS bounds simultaneous Go execution, while the operating
// system remains the final scheduler.
package governor

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

const (
	Format  = "statetwin.dev/execution-profile/v1alpha1"
	Version = "local-v1"
)

type Mode string

const (
	ModeQuiet      Mode = "quiet"
	ModeBalanced   Mode = "balanced"
	ModeThroughput Mode = "throughput"
)

// Options are resolved with command-line values taking precedence over
// environment values. Empty values select the conservative local default.
type Options struct {
	CLIMode     string
	CLIMaxProcs string
	EnvMode     string
	EnvMaxProcs string
	LogicalCPUs int
}

// Profile is operational evidence, not part of deterministic environment
// identity. HardCPUQuota is false until an OS-specific isolation backend is
// implemented and verified.
type Profile struct {
	Format              string `json:"format"`
	Version             string `json:"version"`
	Mode                Mode   `json:"mode"`
	LogicalCPUs         int    `json:"logicalCpus"`
	MaxProcs            int    `json:"maxProcs"`
	WorkerConcurrency   int    `json:"workerConcurrency"`
	HardCPUQuota        bool   `json:"hardCpuQuota"`
	AppliesToGoRuntime  bool   `json:"appliesToGoRuntime"`
	AppliesToChildProcs bool   `json:"appliesToChildProcesses"`
	Source              string `json:"source"`
}

// Resolve constructs a bounded execution profile. The default is quiet and
// uses one Go execution slot so a local run cannot occupy every logical CPU.
func Resolve(options Options) (Profile, error) {
	logical := options.LogicalCPUs
	if logical <= 0 {
		logical = runtime.NumCPU()
	}
	if logical <= 0 {
		logical = 1
	}

	modeText, source := first(options.CLIMode, options.EnvMode)
	if modeText == "" {
		modeText, source = string(ModeQuiet), "default"
	}
	mode := Mode(strings.ToLower(strings.TrimSpace(modeText)))
	var maxProcs int
	switch mode {
	case ModeQuiet:
		maxProcs = 1
	case ModeBalanced:
		maxProcs = (logical + 1) / 2
		if maxProcs > 4 {
			maxProcs = 4
		}
	case ModeThroughput:
		maxProcs = logical
	default:
		return Profile{}, fmt.Errorf("execution mode must be quiet, balanced, or throughput")
	}

	override, overrideSource := first(options.CLIMaxProcs, options.EnvMaxProcs)
	if override != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(override))
		if err != nil || parsed < 1 || parsed > logical {
			return Profile{}, fmt.Errorf("max procs must be an integer within 1..%d", logical)
		}
		maxProcs, source = parsed, overrideSource
	}

	return Profile{
		Format:              Format,
		Version:             Version,
		Mode:                mode,
		LogicalCPUs:         logical,
		MaxProcs:            maxProcs,
		WorkerConcurrency:   1,
		HardCPUQuota:        false,
		AppliesToGoRuntime:  true,
		AppliesToChildProcs: false,
		Source:              source,
	}, nil
}

// Apply updates the Go scheduler and returns its previous setting. It must be
// called before starting servers, workers, or evaluation goroutines.
func Apply(profile Profile) (int, error) {
	if profile.Format != Format || profile.Version != Version {
		return 0, fmt.Errorf("unsupported execution profile")
	}
	if profile.MaxProcs < 1 || profile.MaxProcs > profile.LogicalCPUs {
		return 0, fmt.Errorf("max procs must be within 1..logical CPUs")
	}
	return runtime.GOMAXPROCS(profile.MaxProcs), nil
}

func first(cli, environment string) (string, string) {
	if strings.TrimSpace(cli) != "" {
		return cli, "command-line"
	}
	if strings.TrimSpace(environment) != "" {
		return environment, "environment"
	}
	return "", ""
}
