# make check is the one gate: run it before committing and merging. CI runs
# it on every push; the pre-commit hook runs the fast part (make hooks once
# per clone to turn the hook on).

.PHONY: check precommit fmt vet rules test windows hooks

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
