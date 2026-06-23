---
version: 3.0
runtime: uv
agents_md: off
---

## House rules

### Project conventions

<!-- Deck-local flightdeck conventions only (e.g. "specs written in Chinese").
     General project conventions — code style, commit/push policy, 中文交流 — live in
     ../CLAUDE.md and outrank deck rules (authority: agent instruction file > ### Rules > defaults). -->

- specs / 知识文件正文用中文；canonical 字段标签与节标题用英文（cockpit / rules / INDEX 结构）。

### Rules

<!-- Behavior rules the AI maintains from natural-language requests (free prose, source + date). -->

- landing 自调用、status 标 done 后自动归档（沿用原 autonomy override 行为；you, 2026-06-23）。
