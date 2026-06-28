# Phase65 - config-dependent target capabilities

## Why

Phase64 validates static node capability requirements, but some requirements are triggered only by node config. `ClickAt` and `ClickTemplate` normally need click/move/screenshot, but when `Keys` is set they also call `KeyDown/KeyUp` for modifier keys. Android ADB can tap, but it does not provide `key-state`, so those graphs should fail during validation.

## Contract

- Static `TargetCapabilities` stays on `node.Spec`.
- Validator may derive extra per-node capabilities from config.
- Runtime capability checks remain as fail-closed backup.
- Config-dependent checks must stay centralized in container validation, not scattered into controller profiles or UI-only rules.

## Tasks

- [x] Add failing tests for Android `ClickAt.Keys` and `ClickTemplate.Keys`.
- [x] Add config-derived target capability helper in validator.
- [x] Verify plain Android `ClickAt` still passes.
- [x] Update Flightdeck notes.
- [x] Run targeted and broad verification.
