# Slice 16：Workflow Source 安全导入导出

## Outcome / Question

让 3.1 工作流能够在不同安装之间安全迁移，同时保持唯一 Workflow Source、immutable BlobRef 和 installation/secret 本地边界。

## Completion criterion

- 建立一个深的 workflow bundle 模块，调用者只需 Export/Inspect/Import，小 Interface 隐藏 archive 限额、hash、路径、staging 和原子发布。
- bundle manifest 固定 format/version/source hash/blob set；只包含 canonical Source 与实际引用的 Blob payload。
- 导出显式排除 installations、credentials、secret values、endpoints、主机路径/句柄和 executable payload。
- 导入拒绝 zip-slip、重复/额外条目、压缩炸弹、hash/size/schema/BlobRef 不一致和未知格式。
- 默认导入为新 workflow ID/revision 0；显式覆盖要求 exact revision+sourceHash CAS，并沿用现有 active-run 阻止语义。
- 工作流首页提供检查后导入、CAS 替换、单项导出和跨页选择批量导出；文件选择取消无副作用，结果原地反馈。
- 缺失的本地 installation/secret 继续通过 compiler/admission diagnostics 暴露，不把本机配置写进 bundle。

## Blocked by

commit aaa34711；既有 SourceStore、Blob Store 和工作流列表文件对话框 seam。

## Verification

- `internal/workflowbundle` round-trip、copy identity、replace CAS、额外/穿越条目和 Blob 篡改测试。
- workflow service 生产 composition、单项/批量导出、检查、copy import 与 replace import 测试。
- frontend typecheck、WorkflowsView 聚合测试和 i18n parity/compile 已定向通过。
- 与 Slice 17 一起进入 Stage 8 的唯一完整 `task check`。

## Out of scope

旧 Container zip 的自动导入、Node Package executable 打包、跨机器复制 credentials/installations、云同步。

## Result

Completed。新增 `internal/workflowbundle` 深模块，格式为 `yotta.workflow-bundle` v1 / `.yotta-workflow`；archive 只含 canonical `workflow.json`、严格 manifest 和被 Source 实际引用的 content-addressed Blob。工作流首页已完成检查后导入副本、exact revision+sourceHash 替换、单项导出和不覆盖既有文件的批量导出，取消文件对话框保持无副作用，成功状态不使用 toast。
