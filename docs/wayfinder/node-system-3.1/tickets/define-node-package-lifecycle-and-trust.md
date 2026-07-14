---
title: 定义 Node Package 生命周期与信任模型
label: wayfinder:grilling
parent: ../map.md
status: open
assignee:
blocked_by:
  - define-node-package-and-plugin-protocol.md
---

# 定义 Node Package 生命周期与信任模型

## Question

Node Package 的发现目录、publisher namespace、签名与本地信任、不可变 artifact lock、原子安装、升级、禁用、卸载、回滚、quarantine、漏洞撤销和崩溃恢复应如何定义，才能让 Windows 上的 Wasm Node 与 Process Node 可安全加载，同时让 Linux/macOS 预览构建复用相同的包身份与状态机？

还需明确 Process sandbox、资源配额、安装期 schema/manifest 验证、Program Snapshot 对包实现的锁定，以及没有 marketplace 时本地包和开发包的受控加载流程。
