# Coding standards

Judgement calls for review. Mechanical rules are enforced by `tools/checkrules` and `make check` instead.

## Tests keep their strength

An existing test's assertion stays as strong as it was. A diff that weakens one (a wider bound, an added wait, a skipped case, polling where it checked at once, an exemption) carries its proof in the commit message: that the test fails on the change without the weakening, and why the old assertion was wrong about the behaviour, not merely inconvenient for the new code. Review each weakened assertion against that proof. One without it gets reverted, or the code changes instead.

Why: during spec #11, four tests were loosened to get merges green. Three turned out unneeded, and one hid a real regression: the welcome-back greeting covering Food dropped with `f`.

## Each module does its own job

Code goes in the module that owns its job:

| Module | Owns |
| --- | --- |
| `internal/fish` | What a Fish is and the rules about it: Hunger, Mood, Full, Age, eating, Food Kinds, Species. Pure logic. |
| `internal/tank` | The simulation: where the Fish, Food, bubbles and Scenery are and how they move, one time step at a time. |
| `internal/render` | Everything drawn: the Tank from a snapshot, the status line, the greeting, and every screen that takes the Tank's place. Colour and monochrome are decided here. |
| `internal/store` | The save file and the single-instance lock. |
| `internal/app` | Wiring only: turning the command line, terminal messages and ticks into calls on the others, deciding when to save, and handing `render` a `Scene`. |

Flag a diff that draws, applies a Fish rule or simulates movement inside `internal/app`, or that makes one module reach into another's internals. `internal/app/app.go` is where this drift collects.

Why: after spec #1's parallel tickets, the faces, status line, messages, greeting and monochrome choice had all landed in `internal/app` (371 lines), and the final review had to move them out.

## Tuning numbers live in tuning files

A number someone might want to adjust (a speed, a duration, a size, a row, a threshold) is a named constant in its module's tuning file: `tuning.go`, or `constants.go` in `internal/fish`. A quantity used in more than one place has one constant, like `fish.Day`. Tests are the exception: their expected values are written out on purpose, so they stay independent of the code.

Why: spec #1's review found the status-line height, the greeting's row and the bubble rise step hard-coded where they were used, and the day length written out three times.

## One table per kind

Facts about each value of a kind (each Food Kind, Species or Mood) sit in one table in the module that uses them. Adding a value means adding one row, not finding parallel switches and lists. Two modules can each keep their own facts about the same kind when sharing them would couple the modules: `tank` keeps each Mood's energy in its `energies` table, and `render` decides how each Mood looks.

Why: adding a Food Kind in spec #1 meant editing four places across `fish`, `tank` and `render`.

## A changed meaning updates the glossary

Names in code use `GLOSSARY.md`'s terms (see `docs/agents/domain.md`). A diff that changes what a term means, such as when the Fish counts as Full, updates its glossary entry in the same change. When the decision is hard to reverse and surprising, it also gets an ADR in `docs/adr/`.

Why: in spec #1, "Full" quietly became "Hunger below 1", and only the final review noticed the code and glossary disagreed.

## Failures never punish the user

When something fails but the Fish's life can carry on, such as a failed save, the Tank stays open, Hunger and time keep running, the problem shows on the status line, and the next attempt retries. A failure ends the session only when going on would lose or overwrite the user's Fish, such as an unreadable save file or a second Tank already open.

Why: the spec promises a pet that never punishes neglect. In spec #1, one failed periodic save replaced the Tank with an error screen and stopped every later save, including the save on quit.

## Test helpers live in the harness

A helper more than one test uses (reading a cell, the status line, counting a glyph, waiting for a meal) goes in `internal/app/harness_test.go`, and one helper does each job. Strings the tests share, like the Fish's faces, are defined there once.

Why: in spec #1 helpers were scattered across four test files and the face strings were copied. It happened again in spec #11, which needed cleanup commits for a second colour parser and a second wait-for-a-meal.

## A test that X changes nothing shows X could change something

A twin test (a Tank given X runs the same as one without it) only proves X is ignored if, in that same setup, X would act. Show it in the test or the commit message: the twins differ when the rule is broken.

Why: in #24's review fixes, the Release twin test passed with the bug back in, because the Fish sat under the pointer and never started a trip.

## Names the outside world keeps are decided in the spec

A name that can't be changed once it ships (a published package, a tag format, a field or value in the save file) is spelled out in the spec. A diff that introduces one the spec leaves open is flagged, so the user decides it before merge.

Why: in #25 the spec said `@kyu-seok/shellfish-<os>-<arch>`, and the build chose Node's names (`darwin-x64`, `win32-x64`) itself. Only the spec review raised it, and npm names are permanent once published.
