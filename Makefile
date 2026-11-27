# make check is the one gate: run it before committing and merging. CI runs
# it on every push; the pre-commit hook runs the fast part (make hooks once
# per clone to turn the hook on).

.PHONY: check precommit fmt vet rules test windows hooks publish-check goreleaser-version

check: precommit test windows

# The fast checks, also run by .githooks/pre-commit.
precommit: fmt vet rules

fmt:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed on:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...

# Repo rules: injected Clock and random source, one test seam.
rules:
	go run ./tools/checkrules

test:
	go test ./...

# The lock has Windows-only code: make sure it still compiles there.
windows:
	GOOS=windows go build ./...

hooks:
	git config core.hooksPath .githooks

# GoReleaser, pinned here once: CI reads the version with make goreleaser-version.
GORELEASER_VERSION := v2.20.3
GORELEASER ?= go run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)

goreleaser-version:
	@echo $(GORELEASER_VERSION)

# Builds everything a version would publish, publishes nothing, and checks it
# with the tools users install it with: Homebrew's style check on the cask,
# and an npm install that runs the command. CI runs it on every push.
publish-check:
	$(GORELEASER) check
	TAP_GITHUB_TOKEN=unused $(GORELEASER) release --snapshot --clean
	tools/check-cask.sh dist/homebrew/Casks/shellfish.rb
	node npm/build.mjs 0.0.0-snapshot dist dist/npm
	npm/publish.sh dist/npm --dry-run >/dev/null
	npm/smoke.sh dist/npm
