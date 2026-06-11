---
status: active
last_updated: 2026-06-12
when_to_read: before writing / editing / deleting source code (.go / .ts / .vue) — including comments / 删疑似死代码或符号前
applies_to: [code-style, comments, naming, go, typescript, vue, dead-code, delete-symbol, grep]
---

# Code Style Playbook

写/改源码 (`.go` / `.ts` / `.vue`) 时**前置**读这份.

### 注释

**注释规范单独成篇 → [comments.md](comments.md)** (通用可移植版). 写/改任何注释前读它.

一句话: 默认不写; 要写只写代码表达不出的 why / 不变量 / 坑 / workaround; 禁止把 spec 引用 / 阶段码 / reviewer 出处 / 日期 / 历史考古 / TODO 散弹塞进注释 (这些各有去处 —— git / tracker / design doc).

### 删 / 改"疑似死代码" / 符号 / 节点前

跨 Go + 前端**双向** grep, 且用 `git grep '<符号>'` 扫**整个 tracked 仓**, 别只 scoped `internal/` + `frontend/src/`. burn 例: `_capturedAtResolution` 被只 grep 前端误判成死字段 (实际 `runner.go` 在用); 删 `OnEvent` 时 scoped grep 漏了根 `main.go` / `cmd/` / `wire_container.go` (reviewer 才抓到); 删 `vars.*` 脚本糖时只删了前端补全单源, 后端 `binding.go` 的实现体漏了整整一天后才撞见 (2026-06-12 补删) — **删"一个 API"时它的实现/绑定/文档/生成物 (node-i18n.json 这类) 是一串, 全列出来逐个删**. 文件 / spec 的移删另见 [[move-delete-update-referrers]].
