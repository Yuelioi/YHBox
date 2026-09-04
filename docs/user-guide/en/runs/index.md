---
title: Runs and Debugging
description: Understand status, Timeline, cancellation, and breakpoints
slug: runs-and-debugging
order: 75
source_id: yotta-runs-debugging
---

# Runs and Debugging

Each start creates an independent Run pinned to the saved workflow revision and target snapshot taken
at start. Later edits affect the next Run.

Statuses are queued, running, succeeded, failed, cancelled, and interrupted. Yotta does not
automatically replay interrupted external effects after restart.

Timeline records node attempts, adapter actions, observed statuses, errors, and produced durable
values. Start diagnosis at the first failed entry. Logs provide supporting context but do not replace
Timeline evidence.

Cancel propagates to delay, loops, input playback, scripts, and adapters, and releases held-input
leases. Debug uses the same real runtime and targets while adding breakpoints, pause, continue, step,
and snapshots of current input/state/value data.
