---
title: Automation Targets
description: Configure Windows, Android, and browser targets
slug: automation-targets
order: 40
source_id: yotta-automation-targets
---

# Automation Targets

Automation nodes use logical target slots. Portable workflows do not pin another machine's window
handle, device session, or browser connection.

## Windows

Add the executable under Desktop Applications, then create a Windows target. Choose Exact Match for a
fixed title or RE2 Regex for a dynamic one. If title and window class are both present, both must
match. Check current matches before saving.

SendInput is foreground system input; PostMessage targets a specific window message path. Select a
capture backend appropriate for the app.

## Android

Select an authorized ADB serial and exact package name. Re-select the target after recreating an
emulator or changing the device identity.

## Browser

Start Chrome/Edge remote debugging, enter the CDP endpoint, discover pages, and save a page identity.
The target becomes unavailable when the page closes or the endpoint cannot be reached.

Saved target configuration is the user's authorization for direct per-Run use. Configure only apps,
devices, pages, and endpoints you trust.
