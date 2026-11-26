# Coding standards

Judgement calls for review. Mechanical rules are enforced by `tools/checkrules` and `make check` instead.

## Tests keep their strength

An existing test's assertion stays as strong as it was. A diff that weakens one (a wider bound, an added wait, a skipped case, polling where it checked at once, an exemption) carries its proof in the commit message: that the test fails on the change without the weakening, and why the old assertion was wrong about the behaviour, not merely inconvenient for the new code. Review each weakened assertion against that proof. One without it gets reverted, or the code changes instead.

Why: during spec #11, four tests were loosened to get merges green. Three turned out unneeded, and one hid a real regression: the welcome-back greeting covering Food dropped with `f`.
