---
title: Node Basics
description: Understand node categories, ports, errors, and state
slug: nodes
order: 30
source_id: yotta-nodes
---

# Node Basics

Nodes are the smallest workflow operations. Each versioned contract declares ports, configuration,
errors, statuses, target requirements, and runtime behavior.

## Guide structure

| Guide | Catalog categories | Typical work |
| --- | --- | --- |
| [Control and state](control-and-state.md) | event, control, state, time, random | entry, branch, loops, retry, state |
| [Data processing](data-processing.md) | logic, comparison, math, text, collection, JSON, conversion, geometry, data | decisions and transformations |
| [Automation and input](automation-and-input.md) | automation, application | windows, pointer, keyboard, apps, recordings |
| [Files, network, scripts, and AI](integration.md) | io, network, script, ai | external data and models |

Signal ports decide *when* a node executes. Data ports provide typed values. Error ports route
declared failures; status outcomes represent normal conditions that were not satisfied.

Use fixed values for constants and connections for runtime data. Select targets and resources through
the editor instead of copying internal IDs manually. See [common scenarios](scenarios.md) for larger
compositions.
