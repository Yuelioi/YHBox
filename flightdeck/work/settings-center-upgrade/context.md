# Settings Center Upgrade context

## What matters

设置展示值、非秘密连接元数据和真正 credential 必须分离。独立工具窗属于同一产品系统，但不应
复制设置或业务 owner。历史中未完成的视觉偏好和 Launcher 扩张只能在当前桌面版本复现后重开。

## Decisions

- Secret material 只进入系统安全存储，不进入设置 DTO、日志或 Workflow Source。
- 容器、计划和 Launcher 读取各自正式服务，不保存第二份状态。
- 独立工具窗共享 chrome、状态语言和响应式策略，但保留任务特有布局。

## Terms

- **Connection metadata:** 可安全展示的 Provider、endpoint 和状态，不含 credential value。
- **Launcher:** 面向高频运行入口的桌面命令表面，不是完整容器管理器。
