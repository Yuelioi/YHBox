# J4 — 参数密度、行内编辑与 Run State 初值

## Goal

恢复参数的可读性与高频输入效率，同时让 Run State 的初始值成为显式、可编辑的 Source 事实。

## Current

已完成。Region 的 X/Y/宽/高改为两列布局并缩小紧凑控件；节点可展示优先级最高的三个轻量行内输入；
Run State 创建时可设置初值，已有状态也可直接编辑初值并通过既有 authoring command 保存。

## Next

进入 J5，为 Switch 建立 Source/Contract/Compiler 一致的动态分支拓扑。

## Verification

- `pnpm -C frontend exec vitest run src/app/editor/authoringSurface.test.ts src/views/WorkflowAuthoringFoundations.spec.ts`
- `pnpm -C frontend typecheck`
- `pnpm -C frontend i18n:check`

## Acceptance

- 四个 Region 数值在常见 Inspector 宽度下可读，不再挤成四个窄列。
- 画布节点最多展示三个轻量高频输入，复合参数仍留在 Inspector。
- 新建和已有 Run State 均能看到并修改初始值；修改走唯一 Source authoring command。
