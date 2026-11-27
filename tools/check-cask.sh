#!/bin/sh
# Runs Homebrew's own style check on the cask GoReleaser generated, from a
# throwaway local tap, since goreleaser check accepts casks Homebrew rejects.
#
#   tools/check-cask.sh <cask.rb>
#
# Without brew it says so and passes, unless REQUIRE_BREW is set (CI sets it).
set -eu
cask=$1
brew=$(command -v brew || true)
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
tap=local/shellfish-check
"$brew" untap "$tap" >/dev/null 2>&1 || true
"$brew" tap-new --no-git "$tap" >/dev/null
trap '"$brew" untap "$tap" >/dev/null 2>&1 || true' EXIT
mkdir -p "$("$brew" --repo "$tap")/Casks"
cp "$cask" "$("$brew" --repo "$tap")/Casks/"
"$brew" style --cask "$tap/$(basename "$cask" .rb)"
