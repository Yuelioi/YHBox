---
title: Workflow Editor
description: Add nodes, connect execution, save, run, and debug
slug: workflow-editor
order: 20
source_id: yotta-workflow-editor
---

# Workflow Editor

![Workflow editor](../../assets/workflow-editor.png)

The editor combines subgraph management, a visual canvas, an inspector, workflow-level actions, and
runtime tools. Read the [complete interface tour](interface-tour.md) before building a large graph.

## Core workflow

1. Add or drag nodes from the catalog.
2. Connect signal ports to define execution order.
3. Connect data ports or enter fixed values in the inspector.
4. Save and Check after each small change.
5. Run against a safe target and inspect Timeline.

The compiler rejects incompatible connections. Saving preserves your draft on failure and can locate
the graph, node, and field that must be repaired.

Normal and Debug Runs use the same runtime and real targets. Debug adds breakpoints, pause, continue,
and step; it is not a side-effect-free simulator.

Subgraphs package reusable multi-node logic. Snippets store one reusable node template, not a small
workflow.
