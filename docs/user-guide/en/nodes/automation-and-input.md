---
title: Automation and Input Nodes
description: Control windows, pointer, keyboard, applications, and recordings
slug: automation-input-nodes
order: 36
source_id: yotta-automation-input-nodes
---

# Automation and Input Nodes

Configure an [automation target](../automation/index.md) before using these nodes.

## Window and application

Activate Target, Get Window State, minimize/maximize/restore, move/resize, and Close Window manage a
desktop target. Launch Application and Terminate Application use a stable desktop-app slot.

## Pointer

Read Pointer Position, Move Pointer, Click Pointer, Drag Pointer, Scroll Pointer, and relative motion
cover direct mouse work. Save and restore a position when an action should not leave the user's cursor
elsewhere.

## Keyboard and held input

Press Keys sends an atomic chord. Hold Keys and Hold Pointer Button return a Run-owned lease that must
be connected to Release Held Input. Runtime teardown also attempts release after failure or cancel.

## Recorded input

Replay Keyboard/Mouse Macro runs editable discrete actions. Replay Precise Trajectory retains original
timing, continuous paths, and relative motion. Verify target resolution and mouse calibration first.
