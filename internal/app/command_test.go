package app_test

import (
	"runtime/debug"
	"strings"
	"testing"

	"github.com/Kyu-seok/shellfish/internal/app"
)

func TestNoArgumentsOpensTheTank(t *testing.T) {
	t.Parallel()
	inv := app.Invoke(nil, "v1.2.3")
	if !inv.Open || inv.Status != 0 || inv.Out != "" || inv.Err != "" {
		t.Errorf("Invoke(no arguments) = %+v, want the Tank opened and nothing printed", inv)
	}
}

func TestVersionPrintsTheVersion(t *testing.T) {
	t.Parallel()
	inv := app.Invoke([]string{"--version"}, "v1.2.3")
	if inv.Open || inv.Status != 0 || inv.Out != "shellfish v1.2.3\n" || inv.Err != "" {
		t.Errorf("Invoke(--version) = %+v, want %q on stdout and the Tank left shut", inv, "shellfish v1.2.3\n")
	}
}

func TestHelpPrintsUsage(t *testing.T) {
	t.Parallel()
	const want = `shellfish: a virtual pet Fish that lives in your terminal.

Usage:
  shellfish            open the Tank
  shellfish --version  print the version
  shellfish --help     print this help

In the Tank, d switches the Fish between Fine and Coarse Detail. Coarse is
for terminals whose font shows the Fine Fish as boxes.
`
	inv := app.Invoke([]string{"--help"}, "v1.2.3")
	if inv.Open || inv.Status != 0 || inv.Out != want || inv.Err != "" {
		t.Errorf("Invoke(--help) = %+v, want exactly this on stdout and the Tank left shut:\n%s", inv, want)
	}
}

// Anything else is a mistake: it says so, shows the usage, and exits with
// the usual status for bad arguments, without opening the Tank.
func TestOtherArgumentsAreRefused(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"-v"}, {"-h"}, {"feed"}, {"--version", "extra"}, {"--help", "--version"}} {
		inv := app.Invoke(args, "v1.2.3")
		if inv.Open || inv.Status != 2 || inv.Out != "" {
			t.Errorf("Invoke(%q) = %+v, want status 2, nothing on stdout and the Tank left shut", args, inv)
		}
		if !strings.Contains(inv.Err, "Usage:") || !strings.Contains(inv.Err, args[0]) {
			t.Errorf("Invoke(%q) printed %q to stderr, want it to name %q and show the usage", args, inv.Err, args[0])
		}
	}
}

// The version GoReleaser stamps in wins; failing that, the module version Go
// recorded when it built the binary, as go install does; failing that, dev.
func TestVersionFallsBackToTheBuildInfoThenDev(t *testing.T) {
	t.Parallel()
	built := func(v string) *debug.BuildInfo { return &debug.BuildInfo{Main: debug.Module{Version: v}} }
	for _, tc := range []struct {
		name    string
		stamped string
		info    *debug.BuildInfo
		want    string
	}{
		{"stamped", "v0.2.0", built("v0.1.0"), "v0.2.0"},
		{"go install", "", built("v0.1.0"), "v0.1.0"},
		{"local build", "", built("(devel)"), "dev"},
		{"no build info", "", nil, "dev"},
	} {
		if got := app.Version(tc.stamped, tc.info); got != tc.want {
			t.Errorf("%s: Version(%q, %v) = %q, want %q", tc.name, tc.stamped, tc.info, got, tc.want)
		}
	}
}
