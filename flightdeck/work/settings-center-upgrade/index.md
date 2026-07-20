# Settings Center Upgrade

## Goal

升级设置中心、AI 凭据安全、容器与计划工作台，以及独立桌面工具窗体验。

## Status

Finished

## Current

六类设置、AI Credential Manager 迁移、容器和计划工作台、独立工具窗与悬浮启动器第一阶段均已
完成并通过当时完整门禁。旧记录中的 Windows 视觉 smoke 和启动器下一阶段是未来产品反馈范围，
不再作为本 Finished Work 的 Next。

## Next

None.

## Progress

- 设置查询和 AI 连接元数据不再返回密钥，Windows 使用 Credential Manager。
- 容器页改为运行导向工作台，计划页改为触发与健康状态控制台。
- 录制、校准、鼠标定位、截图选择器和启动器统一窗口 chrome 与响应式策略。
- 启动器支持搜索、完整键盘操作、运行反馈和过期入口安全清理。
- change listener 同步刷新 hotkey、settings、containers 与独立 WebView。
- 当时完整 `task check`、632 项前端测试和 production build 通过。

## References

- [Settings durability](../../knowledge/architecture/settings-durability.md) — 设置 owner 与持久化边界。
- [AI credential storage](../../knowledge/security/ai-credentials-windows-credential-manager.md) — 凭据安全约定。
- [Settings UI](../../knowledge/frontend/settings-page-style.md) — 设置页面视觉规则。
- [Standalone windows](../../knowledge/wails/standalone-window-style.md) — 工具窗约定。
