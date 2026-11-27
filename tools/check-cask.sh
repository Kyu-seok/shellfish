#!/bin/sh
# Runs Homebrew's own style check on the cask GoReleaser generated, from a
# throwaway local tap, since goreleaser check accepts casks Homebrew rejects.
# Making the tap switches on Homebrew's developer mode, so it is switched back
# off afterwards if it was off before.
#
#   tools/check-cask.sh <cask.rb>
#
# Without brew it says so and passes, unless REQUIRE_BREW is set (CI sets it).
set -eu
cask=$1
brew=$(command -v brew || true)
# GitHub's Ubuntu runners have Homebrew here but leave it off PATH.
if [ -z "$brew" ] && [ -x /home/linuxbrew/.linuxbrew/bin/brew ]; then
  brew=/home/linuxbrew/.linuxbrew/bin/brew
fi
if [ -z "$brew" ]; then
  if [ -n "${REQUIRE_BREW:-}" ]; then
    echo "check-cask: brew not found" >&2
    exit 1
  fi
  echo "check-cask: brew not found, skipping the cask check"
  exit 0
fi
export HOMEBREW_NO_AUTO_UPDATE=1 HOMEBREW_NO_ANALYTICS=1 HOMEBREW_NO_ENV_HINTS=1
developer=off
if "$brew" developer | grep -q "is enabled"; then
  developer=on
fi
tap=local/shellfish-check
cleanup() {
  "$brew" untap "$tap" >/dev/null 2>&1 || true
  if [ "$developer" = off ]; then
    "$brew" developer off >/dev/null
  fi
}
"$brew" untap "$tap" >/dev/null 2>&1 || true
trap cleanup EXIT
"$brew" tap-new --no-git "$tap" >/dev/null
dir=$("$brew" --repo "$tap")/Casks
mkdir -p "$dir"
cp "$cask" "$dir/"
"$brew" style --cask "$tap/$(basename "$cask" .rb)"
