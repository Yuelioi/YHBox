---
title: Schedules
description: Trigger workflows manually, daily, periodically, or by hotkey
slug: schedules
order: 80
source_id: yotta-schedules
---

# Schedules

A schedule sequentially submits one or more workflow starts. It does not wait for one Run to finish
before starting the next; put strict sequential work in one workflow.

| Trigger | Behavior |
| --- | --- |
| Manual | Starts only from an explicit Run action |
| Daily | Local `HH:MM` each day |
| Interval | Every N minutes |
| Hotkey | Global registered chord |
| Once | Fires whenever the enabled schedule is registered |

Once is registration-based and may fire again after app startup or a full scheduler reload. Use a
durable completion condition inside the workflow when exact-once behavior matters.

Start timeout covers admission to the queue, not the entire Run. `stop` stops submitting later
targets after a start failure; `continue` records the failure and submits the rest.
