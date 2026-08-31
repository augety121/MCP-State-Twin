package governor

import (
	"runtime"
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
	if profile.HardCPUQuota || profile.AppliesToChildProcs || !profile.AppliesToGoRuntime {
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
		CLIMode: "quiet", CLIMaxProcs: "2",
		EnvMode: "throughput", EnvMaxProcs: "3", LogicalCPUs: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Mode != ModeQuiet || profile.MaxProcs != 2 || profile.Source != "command-line" {
		t.Fatalf("command line did not win: %#v", profile)
	}
	for _, options := range []Options{
		{CLIMode: "fast", LogicalCPUs: 8},
		{CLIMaxProcs: "0", LogicalCPUs: 8},
		{CLIMaxProcs: "9", LogicalCPUs: 8},
		{CLIMaxProcs: "many", LogicalCPUs: 8},
	} {
		if _, err := Resolve(options); err == nil {
			t.Fatalf("expected rejection for %#v", options)
		}
	}
}

func TestApplyChangesAndRestoresScheduler(t *testing.T) {
	original := runtime.GOMAXPROCS(0)
	t.Cleanup(func() { runtime.GOMAXPROCS(original) })
	profile, err := Resolve(Options{CLIMaxProcs: "1", LogicalCPUs: runtime.NumCPU()})
	if err != nil {
		t.Fatal(err)
	}
	previous, err := Apply(profile)
	if err != nil {
		t.Fatal(err)
	}
	if previous != original || runtime.GOMAXPROCS(0) != 1 {
		t.Fatalf("scheduler setting previous=%d current=%d original=%d", previous, runtime.GOMAXPROCS(0), original)
	}
}
