# J3 — 节点与选项发现

## Goal

让高频编辑动作发生在指针附近并能快速检索：Tab 呼出的节点添加器提供清晰的一级/二级分类，节点右键
菜单锚定正确，超过十项或来自运行环境的下拉支持搜索。

## Current

已完成。Quick Add 现在锚定画布指针并提供可 hover/键盘切换的分类与全局搜索；节点右键菜单改用
视口原生 Context Menu；`AdaptiveSelect` 在超过 10 项时自动搜索，超过 40 项时自动虚拟化。

## Next

进入 J4，优化区域参数、节点行内输入和 Run State 初值。

## Verification

- `pnpm -C frontend exec vitest run src/components/common/adaptiveSelect.spec.ts src/views/WorkflowAuthoringFoundations.spec.ts`
- `pnpm -C frontend typecheck`
- `pnpm -C frontend i18n:check`

## Acceptance

- Tab 在画布指针位置打开节点添加器；一级分类可用键盘或 hover 切换，二级节点可搜索并添加。
- 节点右键菜单使用视口级定位，不再按节点局部坐标重复偏移。
- 超过 10 项的列表自动提供搜索；运行时动态枚举可显式始终搜索，大列表启用虚拟化。
- 小型固定枚举继续使用轻量 Select，不增加无意义的搜索层级。
- 定向组件测试、typecheck 与 i18n 检查通过。
