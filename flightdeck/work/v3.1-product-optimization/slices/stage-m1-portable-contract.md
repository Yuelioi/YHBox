# M1 正式可移植合同

## Status

Completed — 2026-07-21

## Deliverable

把开发期 Workflow Source/Bundle 直接替换为首个正式可移植合同：Source 自身拥有 Workflow Resource、
Target Profile Definition、Credential Requirement 与精确 Node Package Dependency；compiler、Blob inventory
和 `.yotta-workflow` 只通过 schema module 的稳定 interface 消费这些事实。

## Starting point

- Source 仍是 `yotta.workflow` / `3.1`，Blob binding 直接携带 BlobRef；没有资源记录、目标定义或 package dependency。
- Bundle 是开发期 `yotta.workflow-bundle` v1，只扫描 node binding 中的 BlobRef。
- 当前没有旧格式用户，允许原地替换 3.1 开发期形状，不增加兼容 reader；本次落地后才开始承担 migration 责任。
- Catalog ImplementationLock 已能把第三方 NodeRef 精确绑定到 package ID 与 manifest digest。

## Steps

1. 用合同测试固定 required collections、closed objects、kind-specific resource shape、稳定排序、悬空引用和敏感本机字段拒绝。
2. 增加 Workflow Resource、Resource Binding、Target Profile Definition、Credential Requirement 与 Node Package Dependency 类型。
3. 在 schema module 内提供 BlobReferences、ResolveResourceBinding 和 dependency/catalog preflight 等深 Interface。
4. compiler 将 Resource Binding 解析为同一 BlobRef value path；Bundle 与应用清理 inventory 复用 schema BlobReferences。
5. 更新 tracked JSON Schema/TypeScript 与所有 Source fixture，运行 schema/compiler/workflowbundle/application 定向门禁。

## Acceptance

- 语义集合必须唯一排序；未知字段、重复 ID、错误 kind shape、悬空 resource/variant/target slot、无效 digest 全部 fail closed。
- Workflow Resource 的 metadata 与 BlobRef 足以在没有 Global Asset 记录时描述图片、Macro 和 InputClip。
- dependency 精确锁定 publisher namespace、packageId、SemVer、manifest digest 和 NodeRef；与 Catalog lock 不符时不可编译。
- Source/Bundle 不出现 exact application path、HWND、设备 serial、credential secret、consent 或 schedule。
- Bundle 精确包含 Source 所引用的全部 Blob，拒绝缺失、额外、重复或身份变化的 entry。

## Verification

- `go test ./internal/workflow/schema ./internal/workflow/compiler ./internal/workflowbundle ./internal/application`
- `task contracts:update` 后 `task contracts:check`
- 按 Git 变更范围运行 `task check`

## Outcome

- Source 直接替换为 required `resources`、`targetProfileDefinitions`、`credentialRequirements`、`dependencies` 集合，
  并移除开发期 `secretRefs`；没有兼容 reader。
- schema module 暴露 `ResolveResourceBinding` 与 `BlobReferences`，统一 compiler、Bundle 与 durable inventory 的资源 seam。
- compiler 将 Resource Binding 解析到原 Blob binding/value path，并拒绝与 Catalog ImplementationLock 不一致或缺失的
  WIT/process Node Package dependency。
- Bundle manifest 精确镜像 dependency，并从 Source 的全部 Workflow Resource、node/call raw Blob binding 生成唯一 Blob 清单；
  inspect 拒绝身份、entry、digest、size 或声明不一致。
- `task check` 通过：20 个受影响 Go 包与 70 个 frontend 测试文件/294 项测试通过；合同、bindings、格式、lint、
  typecheck 与 i18n 均通过。
