---
title: Workflow Management and Sharing
description: Organize, copy, import, export, and migrate workflows
slug: workflow-management
order: 25
source_id: yotta-workflow-management
---

# Workflow Management and Sharing

![Workflow library](../../assets/workflows.png)

Use names for purpose, categories for broad grouping, and tags for cross-cutting filters. Copy a
workflow to create an independent experiment; use subgraphs to reuse multi-node logic inside one
workflow.

Exports contain portable Source and referenced portable resources. They exclude executable paths,
window handles, current device/browser sessions, credentials, and active Runs.

On import, create a copy unless you intentionally replace an exact workflow identity. Replacement
uses revision and Source hash checks instead of last-write-wins. Run Check after import and bind local
application, automation, network, and AI slots.

Input recordings can travel with workflow resources, but relative mouse calibration remains local.
Before deletion, remove launcher or other local references and stop active Runs.
