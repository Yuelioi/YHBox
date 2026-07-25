# M2c — Workflow Resource 显式提升

## Journey

Workflow Resource 随当前 Source 移植，但用户也需要把验证成熟的图片、Macro 或 InputClip 明确加入本机素材库。
提升必须是显式复制身份：新增 Global Asset Catalog 记录、复用已存在的 CAS 字节，不改变原 Workflow Resource、
Source 或任何 Resource Binding。

## Implementation

- Workflow Resource authoring Module 增加 promote Interface；先严格校验不可信 resource 和全部 BlobRef，再一次性
  发布新的 Global Asset 记录。
- 每次提升分配独立 Global Asset GUID；重复提升得到两个可独立编辑的素材记录，但内容寻址字节继续去重。
- 资源侧栏仅在当前工作流 scope 的 overflow 菜单提供“提升到本机素材库”，成功后刷新 Global Asset query；
  不发送任何 EditorSession patch，因此不产生 Source dirty/undo。

## Verification

- 三类 resource 提升、Blob 缺失、重复提升、Catalog 记录、CAS identity 和 `asset:changed` revision/GUID
  均有后端定向测试。
- 前端锁定仅 workflow scope 显示提升动作、调用 RPC 后刷新素材库并展示反馈，且不修改 Source。
- `task check` 退出码 0：Wails 合同为 17 services/144 methods/214 models，36 个受影响 Go 包、
  frontend format/lint/typecheck/i18n 与 82 个测试文件/351 项测试通过。
- `task build` 退出码 0：editor gzip 202920/220000；Windows GUI metadata 与 5 秒隔离启动 smoke 通过。

## Status

Finished.
