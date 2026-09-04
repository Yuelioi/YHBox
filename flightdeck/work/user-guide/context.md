# Context

## Audience

- 第一次使用 Yotta、希望用可视化工作流完成桌面自动化的用户。
- 已经会录制资源，希望理解节点、调试、快捷键和复用方式的进阶用户。

## Documentation decisions

- 当前真值保存在 Yotta 仓库 `docs/user-guide/`，只使用普通 Markdown、YAML frontmatter、相对链接和本地图片。
- 中文与英文分别位于 `docs/user-guide/zh/` 和 `docs/user-guide/en/`，根 `index.md` 只提供语言选择；两套文档共享 assets，不复制图片。
- 仓库产品首页使用两份文件：中文 `README.md` 为默认入口，英文 `README_EN.md` 独立维护；不在一个文件中混排。
- 页面按用户任务组织，不按 Go/TypeScript 模块组织。
- 节点数量和短期测试数据不写死；节点说明回查当前生成 Catalog 与 Node Contract。
- 截图只用于帮助定位界面，不依赖截图表达关键步骤；界面变化后正文仍应可理解。
- Yotta 当前许可证是 source-available，不称为 OSI open source。

## Screenshot sources

当前采用五张 `4.0.0-alpha.2` 截图：工作流首页、工作流编辑器、设置、键鼠宏和精准录制。用户要求现阶段
不发布截图模板说明，对应截图未纳入文档。截图不包含 API key 或凭据；设置截图仅显示模型标签和协议。
