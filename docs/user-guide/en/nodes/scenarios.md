---
title: Node Scenarios
description: Compose nodes for common automation tasks
slug: node-scenarios
order: 33
source_id: yotta-node-scenarios
---

# Node Scenarios

## Poll a condition

```text
Run Started → Repeat → Check → Branch
                         ├─ true  → Break → Next action
                         └─ false → Delay → Continue
```

Put Delay on the unmet path, keep a bound, and distinguish “not yet” status from real failure.

## Click and restore the pointer

```text
Run Started → Read Pointer Position → Move → Click → Move to saved position
```

Use one coordinate space throughout and route failures so restoration is not silently skipped.

## Replay an editable macro

```text
Run Started → Activate Target → Replay Macro → Check result
```

Remove accidental events and excessive waits. Verify client resolution before replay.

## Replay a precise trajectory

```text
Run Started → Activate Target → Apply calibration → Replay Precise Trajectory
```

Use the matching counts-per-360 profile for relative mouse motion. Trim irrelevant start and end
events before saving.

## Choose a path from data

Calculate or compare in data nodes, then feed one boolean into Branch. This keeps Timeline evidence
separate: how the value was produced and why a path was selected.

## Build a safely cancellable long workflow

Prefer multiple cancellable nodes over one opaque action. Repeat, Delay, input playback, scripts, and
target effects receive Run cancellation. Use held-input leases so cancellation can release input.
