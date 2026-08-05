# 知识系统收口上下文

## 已确认事实

- Yotta 当前功能与架构相对稳定，短期内不会进行大规模升级；知识应描述现在如何工作，而不是保留旧路线。
- Git 与 Finished Work 已承担历史记录；核心知识不需要重复记录每次故障、迁移过程或阶段验收流水账。
- `docs/` 适合保存需要被开发者直接阅读的稳定产品/架构合同；`flightdeck/knowledge/` 适合保存执行任务时
  可独立应用的项目操作指南。
- 代码、schema、Task、测试和生成合同是最终事实来源；Markdown 负责导航和解释，不复制可机械验证的清单。
- 当前未提交的按键捕获、画布框选和 smoke 修改属于先前任务，知识整理必须保留并与其区分。

## 收口原则

- 删除已被现行代码推翻、只服务旧版本或仅记录一次问题经过的材料；需要保留的结论改写为正向、可复用指南。
- 一个主题只保留一个权威入口，避免 `docs/`、Knowledge、Work 和 `AGENTS.md` 同时维护同一长说明。
- 入口页只做导航；正文按开发任务命名，使文件路径本身足以帮助发现，不引入索引 schema、版本字段或复查账本。
- 所有保留的架构、存储和运行说明必须从当前代码、测试、schema 或正式 Task 入口核对。
- 批量删除前先检查仓库内引用；移动或合并后同步修正所有本地链接。

## 近期问题的入库判断

- Vue Flow 的选中态、外部节点投影和画布手势配置具有跨问题复用价值，应合并进工作流编辑器知识。
- modifier-only key chord 的捕获与 Windows 输入语义可能复用，但只保留稳定合同和测试入口，不记录本次争论过程。
- 单个游戏工作流的“退出按钮”与具体坐标属于用户工作流，不进入仓库核心知识。

## 最终权威入口

- 当前系统事实：`docs/README.md` → architecture/runtime/storage/threat/compatibility/platform/license。
- 可执行任务指南：`flightdeck/knowledge/README.md` → build、frontend UI、workflow editor、nodes、
  automation input/capture、Wails services。
- 领域语言：`CONTEXT.md`；仓库 contract：`AGENTS.md`；当前工作：`flightdeck/deck.md`。
