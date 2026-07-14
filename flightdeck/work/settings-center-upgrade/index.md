---
topic: settings-center-upgrade
title: Settings center upgrade
summary: Upgrade the six settings themes, AI credential security, and adjacent automation workspaces.
---
# Settings center upgrade

## State

设置中心与六个主题已完成结构性升级；随后完成 AI API Key 的系统凭据迁移，并升级容器、计划两个主工作台。设置查询和 AI 连接元数据不再返回密钥，Windows 使用 Credential Manager；容器页改为运行导向工作台，计划页改为触发与健康状态导向的控制台。

本轮产品改动已提交为 `c4d1307e`，Flightdeck 文档已提交为 `d92c11b0`。工作台概览栏左右缺边问题已修复并加入回归断言，待提交。

## Next

由用户进行真实桌面视觉与交互 smoke。重点检查：
- AI 连接旧明文密钥启动迁移、替换、删除与测试连接。
- 容器工作台在常用窗口宽度下的概览、筛选、卡片/列表与运行操作。
- 计划控制台的搜索、启停、目标摘要、编辑器双栏与窄窗布局。
- 设置中心原有 860px 响应式切换及六主题交互。

## Read now

- knowledge/frontend/settings-page-style.md
- knowledge/frontend/ui.md
- knowledge/architecture/settings-durability.md
- knowledge/security/ai-credentials-windows-credential-manager.md

## Read if

- knowledge/build/build.md — 重新构建或运行完整门禁前
- knowledge/frontend/headless-ui-verify.md — 需要离屏视觉回归时

## Progress

Done:
- 设置中心共享壳层、搜索/深链/访问恢复、键盘导航、响应式侧栏和六主题升级。
- 设置写入串行、保存状态反馈、AI 引用保护与 MCP 授权确认。
- Windows AI API Key 改存 Credential Manager；旧 settings 明文执行无损启动迁移。
- Settings RPC 只返回 AI 连接元数据；新增密钥存在状态、写入、删除 RPC，测试连接可使用一次性表单密钥或已保存密钥。
- 容器 Tab 增加工作台标题、运行/节点/分类概览、渐进筛选和更完整的运行卡片。
- 计划 Tab 增加启用/自动触发/目标概览、搜索与状态筛选、运行态列表，以及带行为预览的分区编辑器。
- 工作台概览从仅有上下边框改为完整四边框与圆角，容器和计划共用；新增 CSS 契约回归测试。
- Wails RPC contract 更新为 14 services / 119 methods / 100 models。

Verified:
- 修复后完整 `task check` 通过。
- Go test/vet/staticcheck、全局覆盖率门槛通过。
- 前端 format、oxlint、eslint、vue-tsc、i18n、bindings contract 全绿。
- Vitest 93 files / 617 tests 全绿。
- 生产构建与 bundle budget 通过；entry gzip 330,894 / 350,000，editor gzip 470,299 / 650,000。
- 浏览器视觉通道本轮不可用，未伪报自动化视觉截图。

## Open questions

- Windows 桌面真实 smoke 尚待用户验收。
- Linux/macOS 仍为预览平台，当前 secure store 明确返回 unavailable；后续若承诺完整支持，需要接入各平台原生凭据库。
