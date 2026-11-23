## Checks

`make check` is the gate (gofmt, vet, repo rules, tests, Windows build); CI runs it on every push. Run `make hooks` once per clone to turn on the pre-commit hook. Repo rules live in `tools/checkrules`.

## Merging

Merge command: `git merge --ff-only <branch>`. If it refuses, run `git merge --no-ff <branch> && make check`.

## Agent skills

### Issue tracker

Issues are tracked in this repo's GitHub Issues through the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Uses the five default triage labels (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: one `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.
