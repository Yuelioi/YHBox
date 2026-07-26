# N2 — 显式子图接口编辑

## Goal

让子图作者显式管理调用契约：一个执行入口、typed data inputs/outputs，以及命名的 exec/error exits；
内部 boundary 只是这个 canonical interface 的可连接投影。

## Status

Complete

## Scope

- 先收敛 Graph interface 当前 schema、authoring patch、compiler 和 caller 校验，保持唯一 Source/runtime。
- 提供清晰的接口编辑 surface：新增、重命名、排序、绑定/解绑与删除。
- 稳定 ID 与显示名称分离；重命名不改变 caller edge/binding identity。
- 删除或改变已被调用点使用的端口时展示引用影响并 fail closed。
- “自动推导接口”保留为带预览的快捷动作，不再是创建入口/出口的唯一方式。

## Steps

1. 建立 interface identity/display/order 的合同与失败测试，核对现有 schema 是否需要兼容性扩展。
2. 抽出单一 `SubgraphInterface` 编辑模块和持久化命令，覆盖 caller 引用影响。
3. 在子图内部提供接口面板，并让 boundary/call 节点共享同一投影。
4. 覆盖新增、重命名、排序、绑定、删除、保存重开和 compile 的定向测试，再运行 `task check`。

## Verification

- 子图可从内部开放端点手工发布入口与至少一个命名出口；空图明确提示先添加节点，不持久化未绑定伪接口。
- 接口 ID 稳定；仅改显示名称后所有已有调用连线与 binding 仍有效。
- 可显式创建多个 exec/error exits，并在调用节点和内部出口 boundary 上一致显示。
- 被调用点使用的端口不可静默删除；未使用端口可删除且保存重开后仍一致。
- 自动推导先展示将新增、保留、移除的差异，由用户确认后才写 Source。

## Result

- `GraphPort`/`GraphExit` 新增可选 `name`；`id` 继续作为 caller binding/edge 的稳定身份，名称修改不改连线。
- `subgraphInterface` 统一派生开放端点、显式发布、同名消歧、排序、引用影响与自动推导差异；不建立平行 store。
- 右侧接口面板可显式绑定单入口、发布 typed data inputs/outputs 和 exec/error exits，支持改名、排序和安全移除。
- Source 合同把 `entries` 收紧为零或一项；调用节点、内部 boundary 和调用 Inspector 一致展示接口名称。
- 自动推导改为确认前预览，并在将移除的接口仍被调用时 fail closed；不可推导原因使用中文。
- `task check` 通过：Workflow 合同无漂移、21 个受影响 Go 包通过、76 个前端测试文件/310 项测试通过。

## References

- [Subgraph management research](../references/subgraph-management-research.md)
- [N1 definition management](stage-n1-subgraph-definition-management.md)
