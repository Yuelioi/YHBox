---
title: Control Flow Nodes
description: Organize entry, branches, delay, repeat, and retry
slug: control-flow-nodes
order: 31
source_id: yotta-control-flow-nodes
---

# Control Flow Nodes

- **Run Started** emits once at Run creation.
- **Branch** continues through exactly one true/false path.
- **Delay** waits for a cancellable duration.
- **Repeat** executes a bounded region, waiting for each body activation to drain.
- **For Each** processes a typed list one item at a time.
- **Retry Region** retries only failures explicitly routed back to that region.
- **Strongly Typed Switch** selects the first matching case or its default path.
- **End Branch** ends normally; **Fail Workflow** records a stable business failure.

A repeating or retrying region needs a bound, a cancellation path, and usually a delay. Do not retry
invalid configuration or missing credentials as if they were transient failures.
