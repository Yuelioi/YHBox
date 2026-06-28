# Phase 43 — Build Knowledge Baseline Refresh

## Problem

The build knowledge and cockpit still recorded old pre-existing red baselines for Go runtime fixtures, frontend i18n residue, and `pnpm -C frontend test`. Current verification shows those notes are stale and would mislead future regression triage.

## Scope

- Refresh `flightdeck/knowledge/build/build.md` with the current green verification commands.
- Update cockpit's pre-existing failure note so it no longer treats passing suites as expected red.

## Non-goals

- No build command changes.
- No test implementation changes.
