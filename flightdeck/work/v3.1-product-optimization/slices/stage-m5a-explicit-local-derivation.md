# M5a — 显式本地派生

## Journey

用户不能直接编辑一个已安装的 immutable Workflow Release。工作流列表为 Installation 提供显式
“创建可编辑副本”动作；确认后从 exact Release Source 创建新的本地 Workflow Source，进入现有编辑器。
原 Installation、Release、target/credential binding、run/schedule consent 与计划保持不变。

## Boundary

- `schema.WorkflowSource` 增加可选、严格且 canonical 的 `derivedFrom` Release provenance，至少锁定
  release/source/attestation digest、publisher namespace、原 workflow ID 与 SemVer。
- `workflowinstallation` 深 Module 从 Installation 当前 Release 生成 opaque derived Source candidate；
  调用方不能传入原始 Source 或伪造 provenance。
- `appbootstrap.Runtime` 将 candidate 交给唯一 `Application.PublishImportedSource`/`workflowstore` seam，
  新 Source 使用独立 ID、revision 0 和本机时间戳；失败不改变 Installation 或 Release。
- Wails 只暴露一个 derive command。前端必须先确认这是独立本地副本且不会继承本机配置，再打开已有编辑器；
  不建立第二套 authoring store/runtime。

## Verification

- derived Source 精确保留图、连接、Resource、Target Profile Definition、credential requirement 与 dependency，
  但使用新 workflow ID/name、revision 0 和 exact `derivedFrom`。
- schema 拒绝未知字段、无效 digest/publisher/SemVer、自引用 provenance；普通 Source 继续兼容。
- 多次派生得到不同 Source identity；任一次 Source Store 冲突/失败都不修改原 Installation/configuration。
- 服务、Wails contract、前端确认/跳转、`task check` 与 Windows WebView 对应旅程通过。

## Status

Finished.

## Evidence

- Schema、深 Module、Application/Wails 和列表确认旅程均由定向测试覆盖；派生 Source 保留可移植内容与
  exact Release lineage，不包含 Installation-local configuration 或 authority。
- `task check`：退出 0；Workflow/Wails contract、39 个受影响 Go 包、前端格式/静态检查/i18n 与
  83 个测试文件/355 项测试通过。
- `task webview:smoke`：`20260726-121007` 退出 0；显式派生后进入现有 Source 编辑器，截图已目检。
