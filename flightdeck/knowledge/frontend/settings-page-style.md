---
kind: checklist
summary: "设置中心统一使用共享壳层、连续内容流、显式保存状态与渐进披露；禁止各主题复制卡片骨架。"
activation: action
read_when: "写/改 SettingsView 或任一设置主题；新增设置项、设置主题、保存状态、危险权限或设置响应式布局前"
recheck_when: "SettingsView 信息架构、Nuxt UI 版本、设置持久化契约或 Settings* 公共组件变化时"
---
# 设置中心设计与实现约定

## 共享结构

- 主题元数据只放在 `frontend/src/settings/registry.ts`，导航、标题与 URL key 不得各自维护副本。
- 顶层使用 `SettingsView` 的共享壳层：主题搜索、深链、上次访问恢复、键盘导航和响应式布局都由壳层负责。
- 子页使用 `SettingsSection`、`SettingsRow`、`SettingsRestartBadge`；保存状态统一由 `SettingsPageHeader` / `SettingsSaveStatus` 展示。
- 页面采用有限宽度的连续内容流和分隔节奏。不要恢复“每个小分区套一张同色卡片”的旧范式，也不要让六个主题各写一套 header。
- 只有启动器编排等真正需要工作台空间的页面使用 `settings-page--wide`。

## 交互与状态

- 自动保存必须有 `saving / saved / error` 可见状态，失败必须可重试。
- Wails error-only/void RPC 用 `invokeVoid`：成功的 `undefined` 不是失败。
- 所有设置写入经 settings store 串行队列；页面不得绕过 store 直接并发 patch。
- 文本输入先改本地草稿，在 change/blur/显式动作时持久化；不要逐键 RPC。
- 重启后生效的设置必须显示 `SettingsRestartBadge`，不能只埋在说明文字中。
- 删除连接、批量覆盖、开放 MCP 执行等高风险动作必须在动作前说明影响并确认。
- AI 连接删除前调用实际引用扫描；安全提示必须如实说明凭据当前保存方式，不得暗示已进入系统凭据库。

## 信息架构

六个主题职责固定为：

1. 常规：界面语言、应用行为、采集与诊断。
2. 快捷键：搜索、状态筛选、分组绑定与集中重置。
3. 输入与校准：录制语义、校准档及全容器同步。
4. 悬浮启动器：显示规则、内容编排与实时预览。
5. AI 连接：provider 连接、默认项、测试、凭据提示与引用保护。
6. MCP 集成：只读能力、执行/写入授权与本地连接信息。

## 响应式与可访问性

- 默认侧栏宽 224px；860px 以下改成横向可滚动主题导航，设置行由双列变单列。
- tab 使用 roving tabindex，并支持方向键、Home、End。
- 图标按钮优先 Nuxt UI `UButton`，必须有 aria-label；原生 button 只用于键盘捕获等组件语义。
- 保存、测试、复制结果使用 `aria-live` 或等价可感知反馈。
- 空态必须同时给出原因和下一步动作，不能只有“暂无数据”。

## 提交前检查

- `pnpm -C frontend format:check`
- `pnpm -C frontend lint`
- `pnpm -C frontend typecheck`
- `pnpm -C frontend i18n:check`
- 交付前仍以仓库根目录 `task check` 为唯一完整门禁。
