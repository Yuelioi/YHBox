---
title: Input Nodes
description: Pointer, keyboard, and recorded-input playback
slug: input-nodes
order: 32
source_id: yotta-input-nodes
---

# Input Nodes

Pointer positions may use ratios, target client pixels, or absolute screen coordinates. Ratios adapt
best when the same layout appears at different resolutions; absolute coordinates are sensitive to
window movement.

For “click and restore,” read the current position, move and click, then move back using the same
coordinate space.

Use Press Keys for an atomic chord. Use held-input nodes only when input must remain pressed across
other nodes, and always connect Release Held Input.

Macros suit editable discrete actions. Precise recordings suit drag paths, continuous movement, and
relative camera turns. See [Input recordings](../resources/index.md).
