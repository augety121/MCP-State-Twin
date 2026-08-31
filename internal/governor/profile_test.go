package governor

import (
	"runtime"
	"runtime/debug"
	"testing"
)

func TestResolveDefaultsToQuietOneSlot(t *testing.T) {
	profile, err := Resolve(Options{LogicalCPUs: 16})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Mode != ModeQuiet || profile.MaxProcs != 1 || profile.Source != "default" {
		t.Fatalf("unexpected default profile: %#v", profile)
	}
	if profile.SoftMemoryLimitBytes != 512<<20 || profile.MaxInFlightRequests != 4 {
		t.Fatalf("unexpected quiet safety bounds: %#v", profile)
	}
	if profile.Sources != (ProfileSources{Mode: "default", MaxProcs: "default", SoftMemoryLimit: "default", MaxInFlight: "default"}) {
		t.Fatalf("default field provenance is incomplete: %#v", profile.Sources)
	}
	if profile.HardCPUQuota || profile.HardMemoryQuota || profile.AppliesToChildProcs || !profile.AppliesToGoRuntime {
		t.Fatalf("capability claims are inaccurate: %#v", profile)
	}
}

func TestResolveBalancedIsBounded(t *testing.T) {
	tests := []struct {
		cpus int
		want int
	}{{1, 1}, {2, 1}, {3, 2}, {8, 4}, {64, 4}}
	for _, test := range tests {
		profile, err := Resolve(Options{EnvMode: "balanced", LogicalCPUs: test.cpus})
		if err != nil {
			t.Fatal(err)
		}
		if profile.MaxProcs != test.want {
			t.Fatalf("balanced on %d CPUs = %d, want %d", test.cpus, profile.MaxProcs, test.want)
		}
	}
}

func TestResolvePrecedenceAndValidation(t *testing.T) {
	profile, err := Resolve(Options{
		CLIMode: "quiet", CLIMaxProcs: "2", CLIMemoryMiB: "768", CLIMaxInFlight: "6",
		EnvMode: "throughput", EnvMaxProcs: "3", EnvMemoryMiB: "900", EnvMaxInFlight: "9", LogicalCPUs: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Mode != ModeQuiet || profile.MaxProcs != 2 || profile.SoftMemoryLimitBytes != 768<<20 || profile.MaxInFlightRequests != 6 || profile.Source != "command-line" {
		t.Fatalf("command line did not win: %#v", profile)
	}
	for _, options := range []Options{
		{CLIMode: "fast", LogicalCPUs: 8},
		{CLIMaxProcs: "0", LogicalCPUs: 8},
		{CLIMaxProcs: "9", LogicalCPUs: 8},
		{CLIMaxProcs: "many", LogicalCPUs: 8},
		{CLIMemoryMiB: "63", LogicalCPUs: 8},
		{CLIMemoryMiB: "unlimited", LogicalCPUs: 8},
		{CLIMaxInFlight: "0", LogicalCPUs: 8},
		{CLIMaxInFlight: "1025", LogicalCPUs: 8},
	} {
		if _, err := Resolve(options); err == nil {
			t.Fatalf("expected rejection for %#v", options)
		}
	}
}

func TestResolveReportsMixedProvenance(t *testing.T) {
	profile, err := Resolve(Options{CLIMode: "quiet", EnvMemoryMiB: "700", LogicalCPUs: 8})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Source != "mixed" || profile.SoftMemoryLimitBytes != 700<<20 || profile.Sources.Mode != "command-line" || profile.Sources.SoftMemoryLimit != "environment" {
		t.Fatalf("mixed provenance was hidden: %#v", profile)
	}
}

func TestApplyChangesAndRestoresScheduler(t *testing.T) {
	original := runtime.GOMAXPROCS(0)
	originalMemory := debug.SetMemoryLimit(-1)
	debug.SetMemoryLimit(originalMemory)
	t.Cleanup(func() {
		runtime.GOMAXPROCS(original)
		debug.SetMemoryLimit(originalMemory)
	})
	profile, err := Resolve(Options{CLIMaxProcs: "1", LogicalCPUs: runtime.NumCPU()})
	if err != nil {
		t.Fatal(err)
	}
	previous, err := Apply(profile)
	if err != nil {
		t.Fatal(err)
	}
	if previous.MaxProcs != original || runtime.GOMAXPROCS(0) != 1 {
		t.Fatalf("scheduler setting previous=%d current=%d original=%d", previous.MaxProcs, runtime.GOMAXPROCS(0), original)
	}
	if previous.MemoryLimit != originalMemory {
		t.Fatalf("previous memory limit=%d, want %d", previous.MemoryLimit, originalMemory)
	}
}
