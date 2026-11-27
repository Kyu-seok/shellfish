// Command shellfish is a virtual pet Fish that lives in the terminal.
package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/Kyu-seok/shellfish/internal/app"
)

// version is stamped in when a version is published (see .goreleaser.yaml).
var version string

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

func (realClock) Tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return app.TickMsg{Time: t} })
}

func main() {
	info, _ := debug.ReadBuildInfo()
	inv := app.Invoke(os.Args[1:], app.Version(version, info))
	if !inv.Open {
		fmt.Fprint(os.Stdout, inv.Out)
		fmt.Fprint(os.Stderr, inv.Err)
		os.Exit(inv.Status)
	}
	code, err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "shellfish:", err)
		os.Exit(1)
	}
	os.Exit(code)
}

// run opens the Tank and returns the exit status once it closes.
func run() (int, error) {
	dir, err := dataDir()
	if err != nil {
		return 1, err
	}
	m := app.New(app.Deps{
		Clock:      realClock{},
		Rand:       rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
		DataDir:    dir,
		Monochrome: monochrome(),
		Getenv:     os.Getenv,
	})
	defer m.Close()
	if code := m.ExitStatus(); code != 0 { // refused: no need to open the screen
		fmt.Fprintln(os.Stderr, m.ExitMessage())
		return code, nil
	}
	final, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion()).Run()
	if err != nil {
		return 1, err
	}
	fm := final.(app.Model)
	if msg := fm.ExitMessage(); msg != "" {
		fmt.Fprintln(os.Stderr, msg)
	}
	return fm.ExitStatus(), nil
}

// monochrome reports whether to draw without colour: when NO_COLOR is set, or
// the terminal shows fewer than 256 colours, too few for the Fish's colours.
func monochrome() bool {
	return os.Getenv("NO_COLOR") != "" || lipgloss.ColorProfile() > termenv.ANSI256
}

// dataDir is $XDG_DATA_HOME/shellfish, falling back to the platform's user
// data folder.
func dataDir() (string, error) {
	base, err := userDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "shellfish"), nil
}

func userDataDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(xdg) {
		return xdg, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home folder: %w", err)
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support"), nil
	case "windows":
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return local, nil
		}
		return filepath.Join(home, "AppData", "Local"), nil
	default:
		return filepath.Join(home, ".local", "share"), nil
	}
}
