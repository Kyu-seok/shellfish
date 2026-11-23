package app_test

import (
	"os"
	"strings"
	"testing"
)

func TestFirstLaunchAsksForTheFishsName(t *testing.T) {
	h := newHarness(t)
	h.resize(40, 10)

	screen := strings.Join(h.view(), "\n")
	if !strings.Contains(screen, "What is your Fish called?") {
		t.Fatalf("first launch shows no naming prompt:\n%s", screen)
	}
	if strings.Contains(screen, "◕") {
		t.Errorf("the Tank opened before the Fish was named:\n%s", screen)
	}
}

func TestNamingTheFishOpensTheTank(t *testing.T) {
	h := newHarness(t)
	h.resize(40, 10)
	h.nameFish("Bubbles")

	screen := strings.Join(h.view(), "\n")
	if strings.Contains(screen, "What is your Fish called?") {
		t.Fatalf("naming prompt is still shown:\n%s", screen)
	}
	if !strings.Contains(screen, "◕") {
		t.Errorf("the Tank shows no Fish after naming:\n%s", screen)
	}
}

func TestNamingPromptShowsWhatIsTyped(t *testing.T) {
	h := newHarness(t)
	h.resize(40, 10)
	h.typeText("Bubbx")
	h.press("backspace")
	h.typeText("les")

	screen := strings.Join(h.view(), "\n")
	if !strings.Contains(screen, "> Bubbles") {
		t.Errorf("prompt does not echo the typed name:\n%s", screen)
	}
}

func TestEmptyOrBlankNameIsRejected(t *testing.T) {
	for _, name := range []string{"", "   "} {
		t.Run(strings.ReplaceAll(name, " ", "␠"), func(t *testing.T) {
			h := newHarness(t)
			h.resize(40, 10)
			h.nameFish(name)

			screen := strings.Join(h.view(), "\n")
			if !strings.Contains(screen, "What is your Fish called?") {
				t.Fatalf("a blank name was accepted:\n%s", screen)
			}
			if !strings.Contains(screen, "Your Fish needs a name.") {
				t.Errorf("no reason given for rejecting the name:\n%s", screen)
			}
			if entries, _ := os.ReadDir(h.dataDir); len(entries) != 0 {
				t.Errorf("a blank name saved files: %v", entries)
			}

			h.nameFish("Bubbles")
			if screen := strings.Join(h.view(), "\n"); !strings.Contains(screen, "◕") {
				t.Errorf("a real name after a rejected one did not open the Tank:\n%s", screen)
			}
		})
	}
}
