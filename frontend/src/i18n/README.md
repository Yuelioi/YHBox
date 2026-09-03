# 前端 i18n 维护

`index.ts` 是运行入口；`zh.ts` 与 `en.ts` 只组合完整 locale object。由功能独立维护的文案放在
`locales/<locale>/`。

## 归属

- `locales/<locale>/app.ts`：共享 chrome、编辑器基础文案、类型、日志、通用操作、快捷键和错误。
- `locales/<locale>/node.ts`：完整的顶层 `node` namespace，包括节点标题、说明、端口、字段、outcome 和
  builtin 节点文案。
- `locales/<locale>/workflow.ts`：工作流浏览、编辑、调试、AI 提案和运行时间线。
- `locales/<locale>/schedule.ts`：计划浏览与编辑。
- `locales/<locale>/resources.ts`：资源库、资源选择器和批量 metadata。
- `locales/<locale>/recording.ts`：录制 HUD、录制编辑、Macro、精准轨迹、校准和录制保存。
- `locales/<locale>/settings.ts`：顶层 `settingsTab`、`settingsCenter`、`settingsMCP`、
  `settingsLauncher`、`settingsNetwork`、`settingsApplications`、`settingsAutomation` 和 `settingsAI`。
- `locales/<locale>/tools.ts`：图标选择器、HUD、屏幕选择器与悬浮启动器。
- `locales/<locale>/about.ts`：关于页。
- `<locale>.ts`：只负责把上述模块组合成一个 locale object，不直接保存产品文案。

一个顶层 namespace 只能属于一个模块，不要跨文件拆散。新模块导出可直接 spread 到 locale root 的 object；
中英文必须保持相同的模块 seam 和 key。namespace 足够大，或主要随一个产品区域变化时，才建立功能模块；
不要按每个 View 或每组少量字符串创建文件。

## 修改与验证

中英文 key 一起增删，插值参数保持一致；使用完整可翻译句子，不拼接片段。后端动态 Problem ID 归入
`error`，并继续接受错误 ID 门禁检查。

从仓库根运行标准 `task check`。前端路由会检查 locale parity、消息编译、静态 key 引用、后端错误 ID、
格式、类型和测试。
