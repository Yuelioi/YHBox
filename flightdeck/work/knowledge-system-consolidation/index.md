# 知识系统收口

## Goal

以当前稳定代码为事实来源，删除 `docs/` 与 `flightdeck/knowledge/` 中过时、重复和一次性复盘材料，建立
一套少量、可验证、按任务自然发现的核心知识；让未来会话能从 `AGENTS.md` 或 `flightdeck/deck.md`
快速找到软件架构、本地数据、工作流运行、节点系统、前端编辑器、自动化与构建验证入口。

## Status

Finished

## Current

知识系统收口已完成。`docs/` 现在只有 8 篇当前系统事实，Knowledge 只有总入口和六类任务指南；旧 ADR、
研究、Wayfinder、重复架构和 bug 时间线已删除。AGENTS、deck、README 与历史 Work 的有效链接均指向新
入口。先前按键捕获与画布框选实现改动完整保留，本 Work 只提炼了其中可复用的稳定契约。

## Next

None

## Progress

- 2026-08-05 建立独立 Work；用户明确授权大幅精简 `docs/` 与 `flightdeck/knowledge/`，并要求从
  `AGENTS.md` 或 `deck.md` 提供简短导航，同时补齐本地数据位置、软件架构和关键代码入口等核心知识。
- 2026-08-05 完成 33 篇 docs、91 篇 Knowledge、仓库引用者与核心代码链审计；锁定 8 篇 docs 与
  7 篇 Knowledge 的目标结构，并记录当前 storage schema、运行链、节点、编辑器和输入契约。
- 2026-08-05 完成正文重写与批量收口；新入口覆盖 architecture/code map、Source → Program → Run、
  `%LOCALAPPDATA%\Yotta\Yotta` 数据地图、节点、UI/编辑器、automation/Wails 与分层验证。
- 2026-08-05 最终验证通过：本地 Markdown link scan 为 0 个死链，旧核心路径引用为 0，
  `git diff --check` 通过；`task check` 退出码 0（Go changed package 通过，frontend 108 files / 474 tests）。
