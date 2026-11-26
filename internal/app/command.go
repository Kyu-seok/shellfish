package app

import (
	"fmt"
	"runtime/debug"
)

// Invocation is what one run of the command does, given its arguments.
type Invocation struct {
	// Open is set when the Tank should open; nothing is printed then.
	Open bool
	// Out and Err are printed to stdout and stderr before exiting with Status.
	Out, Err string
	Status   int
}

const usage = `shellfish: a virtual pet Fish that lives in your terminal.

Usage:
  shellfish            open the Tank
  shellfish --version  print the version
  shellfish --help     print this help
`

// Invoke decides what the command line asks for. With no arguments the Tank
// opens; --version and --help print and exit without touching the Tank or its
// lock, so they work while a Tank is open; anything else is refused with
// status 2.
func Invoke(args []string, version string) Invocation {
	switch {
	case len(args) == 0:
		return Invocation{Open: true}
	case len(args) == 1 && args[0] == "--version":
		return Invocation{Out: "shellfish " + version + "\n"}
	case len(args) == 1 && args[0] == "--help":
		return Invocation{Out: usage}
	}
	return Invocation{Err: fmt.Sprintf("shellfish: unknown argument %q\n\n%s", args[0], usage), Status: 2}
}

// Version is the version to report: the one stamped in when the binary was
// published, else the module version Go recorded in info (as go install
// does), else dev for a local build.
func Version(stamped string, info *debug.BuildInfo) string {
	if stamped != "" {
		return stamped
	}
	if info != nil && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
