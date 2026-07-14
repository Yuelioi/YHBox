---
topic: settings-center-upgrade
title: Settings center upgrade
summary: Upgrade the six settings themes into a professional, consistent and reliable settings center.
---
# Settings center upgrade

## State

设置中心与六个主题已经完成结构性升级。共享壳层提供主题搜索、URL 深链、上次访问恢复、键盘导航、响应式侧栏与全局自动保存状态；常规、快捷键、输入与校准、悬浮启动器、AI 连接、MCP 集成均已迁移到统一信息架构。

## Next

由用户进行真实桌面视觉与交互 smoke。重点检查 860px 响应式切换、启动器实时预览、校准档编辑、AI 删除引用确认、MCP 授权前确认；若无问题，本主题可归档。

## Read now

- knowledge/frontend/settings-page-style.md
- knowledge/frontend/ui.md
- knowledge/architecture/settings-durability.md

## Read if

- knowledge/build/build.md — 重新构建或运行完整门禁前
- knowledge/frontend/headless-ui-verify.md — 需要离屏视觉回归时

## Progress

Done:
- 修复 Wails void RPC 成功返回 undefined 被误判为失败的设置保存契约。
- 设置写入改为串行队列；成功后本地 deep merge，失败可重试，并监听 settings:changed。
- 新增共享设置组件与六主题 registry。
- 设置中心支持搜索、深链、上次主题恢复、roving tabindex 和窄窗横向导航。
- 完成常规、快捷键、输入与校准、悬浮启动器、AI 连接、MCP 集成六个主题升级。
- AI 删除前扫描节点引用；MCP 授权前确认；凭据风险如实提示。
- HotkeyCaptureInput 正确声明 disabled/ariaLabel，图标按钮统一使用 Nuxt UI。
- ESLint explicit-any 历史债务由 267 降到 265。

Verified:
- frontend format、oxlint、eslint、vue-tsc、i18n 全绿。
- Vitest 93 files / 616 tests全绿。
- 完整 task check 通过：Go test/vet/staticcheck、Wails 14 services / 116 methods / 100 models、生产构建和 bundle budget 全绿。
- entry gzip 329,366 / 350,000；editor gzip 470,305 / 650,000。

## Open questions

- API key 仍以本机 settings 明文保存；迁移 OS credential store 属于 Wave 8，不在本轮伪装成已解决。
- 本轮没有自动化视觉快照，真实桌面 smoke 仍需用户验收。
