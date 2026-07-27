# V4 功能保留基线

本文件描述用户可观察能力。重构时允许隐藏、移动和重组，不允许因为实现文件被删除或页面拆分而静默丢失。

| 区域 | 必须保留的能力 | V4 呈现原则 | 验收 |
| --- | --- | --- | --- |
| Workflow 首页 | 新建、模板、搜索、打开、运行 | 默认常驻 | 空 profile 和已有 profile WebView 旅程 |
| Workflow 组织 | 分类、标签、日期、排序、列、分页 | 管理模式 | 查询与大列表组件测试 |
| Workflow 批量 | 元数据、导出、删除 | 管理模式 | 多选和部分失败反馈测试 |
| Workflow 文件 | 导入、导出、替换 | 管理菜单 | Bundle 往返和真实 Source |
| Source 恢复 | 隔离、查看、修复、删除损坏 Source | 有问题时通知 | 损坏 fixture smoke |
| 编辑器基础 | 画布、添加节点、连线、复制、撤销、重做、保存 | 默认核心 | EditorSession 与 WebView |
| 子图 | 创建、接口、调用、复制、展开、级联删除 | 左侧任务面板 | Source 保存重开和 compile |
| 资源创作 | Macro、InputClip、视觉模板、录制、截图 | 左侧任务面板 | 三类资源创建旅程 |
| Snippet | 保存和插入节点片段 | 左侧任务面板 | 快速添加测试 |
| 状态与类型 | 变量、复合初值、动态端口、显式转换 | Inspector/设置 | compiler/runtime 矩阵 |
| 运行 | 普通运行、取消、状态、日志、时间线 | 运行时出现 | 同一 StartRun 路径 |
| 调试 | 断点、暂停、步进、继续 | 调试模式 | debug WebView 旅程 |
| AI | 生成、评审、应用 patch | 按需辅助 | AI eval 与 patch 测试 |
| Target | AI、HTTP、应用、窗口、Android、自动化目标 | 缺项时就地配置 | 热替换和 readiness |
| Resource 库 | 三类资源浏览、预览、搜索、筛选、分页 | 支持工作区 | 资源 smoke |
| Resource 管理 | 元数据、变体、批量删除、录制/创建 | 管理模式 | Blob 引用保护 |
| Schedule | 手动、定时、启动、热键、顺序运行多个 Workflow | 简单主表单，高级限制折叠 | 保存重开和 daemon |
| 启动器 | 收藏、快捷运行、热键、窗口复用 | 工具入口 | full WebView smoke |
| 设置 | 通用、快捷键、输入校准、启动器、AI、网络、应用、自动化 | 常用/连接/高级分组 | 设置持久化和热更新 |
| 数据 | Source、Blob、Run、Schedule、Settings、恢复信息 | 用户无需理解物理路径 | migration、health、backup |
| CLI/MCP | inspect、compile、run 和现有工作流调用 | 与 GUI 共用 Application | CLI/MCP 集成测试 |

## 删除规则

只有同时满足以下条件才能删除能力或实现：

1. 不在本基线中，或用户明确确认不再需要。
2. 生产代码、CLI、MCP、Schedule、smoke 和真实数据均无调用。
3. 删除后复杂度不会转移到多个调用方。
4. 有独立提交或明确恢复点。
