# M2d — Source-native Workflow Resource 编辑

## Journey

接收方导入只携带 Source 与 CAS 字节的工作流后，无需先创建任何 Global Asset，即可预览图片、查看
Macro/InputClip 摘要、编辑 Macro 动作、裁剪 InputClip，并复制资源形成独立身份。内容编辑保留原
Workflow Resource ID，因此所有引用同步更新；复制只新增 Resource，不隐式改绑现有节点。

## Implementation

- `resourceauthoring` Module 从不可信 Workflow Resource 的 BlobRef 读取并校验 carrier，返回 Macro 文档或
  InputClip 摘要/分页事件；内容重写负责编码、CAS 发布和 metadata projection。
- authoring patch 增加完整 Resource replacement，要求 ID 与 kind 不变；一次 replacement 对应一个
  EditorSession undo，现有 Resource Binding 无需重写。
- 资源侧栏显示图片分辨率、Macro 动作/时长、InputClip 事件/时长/录制源 counts/360；三类资源均可显式
  duplicate，Macro 与 InputClip 分别进入现有动作编辑器和精准录制工作台。

## Verification

- Module 测试覆盖 carrier/metadata 一致性、缺 Blob、Macro 重写、InputClip 裁剪与分页、三类 duplicate；
  Go/前端 authoring 测试覆盖 replacement 的 ID/kind 保护、单 undo、共享 binding 与无 Global Asset 记录。
- 最终 `task check` 退出 0：113 个变更文件路由到 router self-test、合同、AI 8/8、Wails 17 服务/
  148 方法、37 个受影响 Go 包和前端 format/lint/typecheck/i18n、82 个测试文件/351 项测试。
- `task build` 退出 0：entry gzip 249619/350000、editor gzip 203779/220000，Windows GUI metadata
  与隔离 RootSet 5 秒启动通过。
- `task webview:smoke` 最终退出 0；诊断并修正 R5 后仍向废弃文件夹写 recovery 的旧夹具，改由 smoke
  工具通过正式 Catalog quarantine API 建立恢复记录。`20260726-010939` 的工作流恢复面、编辑器、
  资源工具、资源库与计划编辑器 PNG 已逐张目检，无空白、错位或明显裁切。

## Status

Finished.
