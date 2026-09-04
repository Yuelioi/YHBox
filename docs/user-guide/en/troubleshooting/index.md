---
title: Troubleshooting
description: Resolve save, target, shortcut, input, and shutdown problems
slug: troubleshooting
order: 90
source_id: yotta-troubleshooting
---

# Troubleshooting

## Workflow cannot save

Use Locate Problem to open the graph, node, and field. The draft remains in memory. Repair old nodes,
invalid fields, resource references, or connections, then retry. Record the operation ID shown with an
application error.

## Target unavailable or contract violation

Confirm the node uses a target slot configured on this computer. Check that the app/device/page is
running and that Windows title/class matching is correct. Rebind imported workflows locally.

## Shortcut does not work

Check conflict and registration errors. Launcher `1–9` exists only while visible and enabled. Try a
different chord when the OS, game, security software, or another app reserves it.

## Pointer is offset

Check target client resolution and coordinate unit. Screen, client-pixel, and ratio coordinates are
different. Use the correct counts-per-360 profile for relative trajectories.

## Shutdown takes time

Yotta may stop work, restore a discarded draft, and release runtime, shortcut, and window resources.
The shutdown modal reports the current stage. Do not repeatedly start another instance while it is
closing.
