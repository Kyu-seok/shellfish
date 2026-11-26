package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kyu-seok/shellfish/internal/app"
)

// input is one thing a user can do: a key or a click.
type input struct {
	name string
	do   func(h *harness)
	quit bool // a quit key, which quits from every screen
}

// everyInput is every key the app binds on any screen, a plain letter, and a
// click. A new key belongs here, so the hidden-screen tests cover it.
var everyInput = func() []input {
	var in []input
	for _, k := range []string{"q", "esc", "ctrl+c"} {
		in = append(in, input{name: k, do: func(h *harness) { h.press(k) }, quit: true})
	}
	for _, k := range []string{"f", " ", "1", "2", "3", "r", "y", "n", "x", "left", "right", "enter", "backspace"} {
		in = append(in, input{name: k, do: func(h *harness) { h.press(k) }})
	}
	in = append(in, input{name: "click", do: func(h *harness) { h.click(10, 3) }})
	return in
}()

// On a screen that hides the Tank, only the quit keys do anything: no key
// or click may act on a Tank or Fish the user can't see.

func TestNothingButQuittingActsOnTheLoadFailureScreen(t *testing.T) {
	t.Parallel()
	for _, in := range everyInput {
		t.Run(in.name, func(t *testing.T) {
			t.Parallel()
			path, h := unreadableSave(t)
			h.resize(100, 24)
			in.do(h)
			if h.quitRequested() != in.quit {
				t.Errorf("%q on the load-failure screen: quit = %v, want %v", in.name, h.quitRequested(), in.quit)
			}
			if got, _ := os.ReadFile(path); string(got) != "{not json" {
				t.Errorf("%q on the load-failure screen wrote over the unreadable save file: %s", in.name, got)
			}
		})
	}
}

func TestMashingKeysOnTheLoadFailureScreenWritesNothing(t *testing.T) {
	t.Parallel()
	path, h := unreadableSave(t)
	h.resize(100, 24)
	for _, in := range everyInput {
		if !in.quit {
			in.do(h)
		}
	}
	h.advance(app.FarewellDuration)
	h.press("enter")
	h.nameFish("Nemo")
	if got, _ := os.ReadFile(path); string(got) != "{not json" {
		t.Errorf("keys on the load-failure screen wrote over the unreadable save file: %s", got)
	}
}

func TestNothingButQuittingActsInAWindowTooSmallForTheTank(t *testing.T) {
	t.Parallel()
	for _, in := range everyInput {
		t.Run(in.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, named("Bubbles"))
			h.resize(20, 5)
			in.do(h)
			if h.quitRequested() != in.quit {
				t.Fatalf("%q in a too-small window: quit = %v, want %v", in.name, h.quitRequested(), in.quit)
			}
			if in.quit {
				return
			}
			untouched := newHarness(t, named("Bubbles"))
			untouched.resize(20, 5)
			h.resize(100, 24)
			untouched.resize(100, 24)
			if got, want := h.screen(), untouched.screen(); got != want {
				t.Errorf("%q in a too-small window changed the Tank shown once it grew:\n%s\nwant:\n%s", in.name, got, want)
			}
		})
	}
}

// unreadableSave starts the app on a save file it can't read, and returns
// the file's path.
func unreadableSave(t *testing.T) (string, *harness) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fish.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path, newHarness(t, withDataDir(dir))
}
