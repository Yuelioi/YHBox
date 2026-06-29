# Phase 56 — Controller Trace Error Tests

## Problem

Controller tests covered successful action traces, but backend call failures were not explicitly guarded. Trace consumers depend on error status and message fields to diagnose input/capture failures.

## Scope

- Add Android ADB controller trace test for backend errors.
- Add Browser CDP controller trace test for backend errors.
- Assert error traces retain coordinate steps where coordinate conversion succeeded.

## Non-goals

- No change to preflight coordinate validation errors that fail before action trace recording.
- No trace persistence UI change.
