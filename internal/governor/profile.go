// Package governor applies host-local execution policy without changing
// deterministic world semantics. It deliberately does not claim an OS-level
// CPU quota: GOMAXPROCS bounds simultaneous Go execution, while the operating
// system remains the final scheduler.
package governor

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

const (
	Format  = "statetwin.dev/execution-profile/v1alpha1"
	Version = "local-v2"

	minMemoryLimitMiB = 64
	maxMemoryLimitMiB = 1 << 20
	maxInFlightLimit  = 1024
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
	CLIMode        string
	CLIMaxProcs    string
	CLIMemoryMiB   string
	CLIMaxInFlight string
	EnvMode        string
	EnvMaxProcs    string
	EnvMemoryMiB   string
	EnvMaxInFlight string
	LogicalCPUs    int
}

// Profile is operational evidence, not part of deterministic environment
// identity. HardCPUQuota is false until an OS-specific isolation backend is
// implemented and verified.
type Profile struct {
	Format               string         `json:"format"`
	Version              string         `json:"version"`
	Mode                 Mode           `json:"mode"`
	LogicalCPUs          int            `json:"logicalCpus"`
	MaxProcs             int            `json:"maxProcs"`
	WorkerConcurrency    int            `json:"workerConcurrency"`
	SoftMemoryLimitBytes int64          `json:"softMemoryLimitBytes"`
	MaxInFlightRequests  int            `json:"maxInFlightRequests"`
	HardCPUQuota         bool           `json:"hardCpuQuota"`
	HardMemoryQuota      bool           `json:"hardMemoryQuota"`
	AppliesToGoRuntime   bool           `json:"appliesToGoRuntime"`
	AppliesToChildProcs  bool           `json:"appliesToChildProcesses"`
	Source               string         `json:"source"`
	Sources              ProfileSources `json:"sources"`
}

type ProfileSources struct {
	Mode            string `json:"mode"`
	MaxProcs        string `json:"maxProcs"`
	SoftMemoryLimit string `json:"softMemoryLimit"`
	MaxInFlight     string `json:"maxInFlight"`
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

	modeText, modeSource := first(options.CLIMode, options.EnvMode)
	if modeText == "" {
		modeText, modeSource = string(ModeQuiet), "default"
	}
	maxProcsSource, memorySource, inFlightSource := modeSource, modeSource, modeSource
	mode := Mode(strings.ToLower(strings.TrimSpace(modeText)))
	var maxProcs, maxInFlight, memoryMiB int
	switch mode {
	case ModeQuiet:
		maxProcs = 1
		memoryMiB = 512
		maxInFlight = 4
	case ModeBalanced:
		maxProcs = (logical + 1) / 2
		if maxProcs > 4 {
			maxProcs = 4
		}
		memoryMiB = 1024
		maxInFlight = 16
	case ModeThroughput:
		maxProcs = logical
		memoryMiB = 2048
		maxInFlight = 64
	default:
		return Profile{}, fmt.Errorf("execution mode must be quiet, balanced, or throughput")
	}

	override, overrideSource := first(options.CLIMaxProcs, options.EnvMaxProcs)
	if override != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(override))
		if err != nil || parsed < 1 || parsed > logical {
			return Profile{}, fmt.Errorf("max procs must be an integer within 1..%d", logical)
		}
		maxProcs, maxProcsSource = parsed, overrideSource
	}
	memoryOverride, memoryOverrideSource := first(options.CLIMemoryMiB, options.EnvMemoryMiB)
	if memoryOverride != "" {
		parsed, err := parseBounded(memoryOverride, minMemoryLimitMiB, maxMemoryLimitMiB, "memory limit MiB")
		if err != nil {
			return Profile{}, err
		}
		memoryMiB, memorySource = parsed, memoryOverrideSource
	}
	inFlightOverride, inFlightOverrideSource := first(options.CLIMaxInFlight, options.EnvMaxInFlight)
	if inFlightOverride != "" {
		parsed, err := parseBounded(inFlightOverride, 1, maxInFlightLimit, "max in-flight requests")
		if err != nil {
			return Profile{}, err
		}
		maxInFlight, inFlightSource = parsed, inFlightOverrideSource
	}
	source := mergeSource(mergeSource(modeSource, maxProcsSource), mergeSource(memorySource, inFlightSource))

	return Profile{
		Format:               Format,
		Version:              Version,
		Mode:                 mode,
		LogicalCPUs:          logical,
		MaxProcs:             maxProcs,
		WorkerConcurrency:    1,
		SoftMemoryLimitBytes: int64(memoryMiB) << 20,
		MaxInFlightRequests:  maxInFlight,
		HardCPUQuota:         false,
		HardMemoryQuota:      false,
		AppliesToGoRuntime:   true,
		AppliesToChildProcs:  false,
		Source:               source,
		Sources: ProfileSources{
			Mode: modeSource, MaxProcs: maxProcsSource,
			SoftMemoryLimit: memorySource, MaxInFlight: inFlightSource,
		},
	}, nil
}

// Apply updates the Go scheduler and returns its previous setting. It must be
// called before starting servers, workers, or evaluation goroutines.
type Previous struct {
	MaxProcs    int
	MemoryLimit int64
}

func Apply(profile Profile) (Previous, error) {
	if profile.Format != Format || profile.Version != Version {
		return Previous{}, fmt.Errorf("unsupported execution profile")
	}
	if profile.MaxProcs < 1 || profile.MaxProcs > profile.LogicalCPUs {
		return Previous{}, fmt.Errorf("max procs must be within 1..logical CPUs")
	}
	if profile.SoftMemoryLimitBytes < int64(minMemoryLimitMiB)<<20 {
		return Previous{}, fmt.Errorf("soft memory limit is below supported minimum")
	}
	if profile.MaxInFlightRequests < 1 || profile.MaxInFlightRequests > maxInFlightLimit {
		return Previous{}, fmt.Errorf("max in-flight requests is outside supported range")
	}
	previous := Previous{MaxProcs: runtime.GOMAXPROCS(profile.MaxProcs)}
	previous.MemoryLimit = debug.SetMemoryLimit(profile.SoftMemoryLimitBytes)
	return previous, nil
}

func parseBounded(value string, minimum, maximum int, label string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < minimum || parsed > maximum {
		return 0, fmt.Errorf("%s must be an integer within %d..%d", label, minimum, maximum)
	}
	return parsed, nil
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

func mergeSource(current, next string) string {
	if current == "" {
		return next
	}
	if next == "" || current == next {
		return current
	}
	return "mixed"
}
