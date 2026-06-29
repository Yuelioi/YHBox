# Phase67 - capability vocabulary guard

## Why

`node.Spec.TargetCapabilities` intentionally uses the same string vocabulary as `automation/controller.Capability`, but the two packages are separated to avoid making every node import controller internals. That creates a spelling drift risk.

## Contract

- Every capability declared by a node must be recognized by `controller.CapabilitySet.Has`.
- The guard belongs in node spec consistency tests because it validates node metadata, not runtime behavior.

## Tasks

- [x] Add spec consistency test for target capability vocabulary.
- [x] Run related tests and broad verification.
- [x] Update Flightdeck notes.
