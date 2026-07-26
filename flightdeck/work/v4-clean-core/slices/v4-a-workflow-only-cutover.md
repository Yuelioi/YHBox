# V4-A — Workflow-only cutover

## Goal

让 Workflow 成为列表、编辑、运行、导入和计划的唯一对象，删除桌面产品路径中的 Release/Installation。

## Product behavior

- 本地创建与文件导入都创建可编辑 Workflow。
- Workflow 列表不显示来源类型、Release、Installation 或只读状态。
- 每行只有打开、运行和更多操作；目标缺失时从工作流设置直接配置。
- Schedule 持久引用 Workflow ID。
- 文件导入保留可选来源 metadata，但不存在安装、更新、回退或派生副本状态机。

## Steps

1. 盘点 Workflow service、schedule 和 frontend transport 中 Installation 调用。
2. 将统一列表恢复为 Source-only projection，并删除 Installation 操作 UI。
3. 将 schedule target 与运行入口统一为 Workflow ID。
4. 从 desktop composition 移除 Installation runtime。
5. 删除不可达接口、生成合同和相关测试。

## Current

实现完成。Workflow library、GUI run、CLI、Schedule 与 smoke 均使用 Workflow ID；Release、
Installation、更新/回退、离线安装包与对应 Wails projection 已删除。

## Next

无；本 Slice 已完成。
