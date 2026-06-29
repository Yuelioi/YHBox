# Phase66 - click button semantics

## Why

Android ADB `input tap` has no right/middle button semantics. After Phase64/65, `ClickAt` and `ClickTemplate` still only require generic `click`, so Android graphs with `Button=right` or `Button=middle` pass validation but execute as ordinary taps.

## Contract

- Plain left/touch click requires `click`.
- Non-left mouse button click requires `mouse-button` in addition to `click`.
- Win32 and Browser CDP can support non-left button clicks; Android ADB cannot.
- The rule is config-derived and belongs in container validation with the other target capability checks.

## Tasks

- [x] Add failing tests for Android `ClickAt.Button=right` and `ClickTemplate.Button=middle`.
- [x] Derive `mouse-button` for non-left button configs.
- [x] Verify Android default/left click still passes.
- [x] Update Flightdeck notes.
- [x] Run verification and commit.
