---
title: Control and State Nodes
description: Entry, branching, loops, Run state, time, and randomness
slug: control-and-state-nodes
order: 34
source_id: yotta-control-state-nodes
---

# Control and State Nodes

## Frequently used control nodes

| Node | Use | Caution |
| --- | --- | --- |
| Run Started | Main Run entry | Emits once per Run |
| Branch | Choose true or false | Calculate the condition upstream |
| Delay | Wait before continuing | Prefer observable conditions when available |
| Repeat | Execute a bounded region | Add interval and exit behavior |
| For Each | Process a typed list | One body activation completes before the next |
| Retry Region | Retry routed failure | Retry transient failure only |
| Strongly Typed Switch | Choose among cases | Clearer than deeply nested branches |

Continue and Break apply to their owning region, not an arbitrary outer loop.

## Run state

Read State, Write State, Add State, State Metadata, and State Last Change operate on typed state
initialized separately for each Run. Use state for counters, flags, and recent values inside one Run;
use an explicit durable destination for data that must survive application restart.

## Time and randomness

Observe Time and Stopwatch nodes measure workflow intervals. Random Integer, Number, Boolean, and
Choice produce values recorded as Run evidence, rather than pretending to be deterministic math.
