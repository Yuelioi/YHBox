---
slice: "40"
title: Effective Target 继承
status: completed
---

# Slice 40：Effective Target 继承

## Outcome / Question

让 workflow 只选择一次默认自动化目标，保留节点显式 override，并保证 authoring、compile、admission、runtime 看到同一 effective slot。

## Completion criterion

- Source 有版本化 `targetDefaults`；node override > workflow default > missing diagnostic。
- 单一 EffectiveTargetResolver 被 Compiler 与 Authoring Projection 复用；Program 只保存解析后的显式 slot。
- UI 提供 workflow default；Inspector 显示 inherited/overridden/missing，可覆盖和恢复继承。
- 非目标节点不受默认值影响；多个允许 target kind 时不做模糊推断。
- 导入导出、复制、子图、删除 target 与诊断覆盖新字段。

## Verification

- default/override/missing Go/TS parity fixtures。
- 多节点共享 default、单节点 override、删除/更换 default 的编译与运行测试。
- 真机用一个目标连续执行 Click、Keys、Template。

## Out of scope

- 不复制 default 到每个 node config。
- 不引入 last-used ambient target 或绕过 admission。

## Result

Completed。

- Workflow Source/JSON Schema/TS contract 新增 targetDefaults，authoring 命令支持设置与清除。
- Compiler 在编译期解析 node override > workflow default > missing，并将 effective config 写入 Program；显式 override 不受默认值覆盖。
- 编辑器顶层可选择一次默认自动化目标；Inspector 展示继承/覆盖并可恢复继承，录制插入的节点不重复写相同 slot。
- schema/compiler/authoring/EditorSession 回归测试与 Wails/workflow 契约检查通过；WebView 编辑器 smoke 通过。
