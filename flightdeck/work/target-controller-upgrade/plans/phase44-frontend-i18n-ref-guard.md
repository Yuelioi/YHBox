# Phase 44 — Frontend I18n Static Reference Guard

## Problem

The existing frontend i18n check covered zh/en parity, message compilation, and Chinese literal residue, but it did not verify that static `t('key')` / `te('key')` references actually exist. Missing keys could pass checks and render as raw key strings in the UI.

## Scope

- Fix the checker to load the actual default-exported message object from `zh.ts` / `en.ts`.
- Add a static source scan for literal i18n key references.
- Add the missing variable promote/delete modal keys surfaced by the new guard.

## Non-goals

- No dynamic node key scanning in the frontend checker; backend catalog i18n guards cover generated node label/hint/option keys.
- No component behavior changes.
