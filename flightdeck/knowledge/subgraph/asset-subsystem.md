# 资产子系统 — stable asset identity + immutable BlobRef

模板与 clip 是全局资产；每条记录使用稳定 GUID，显示名可变。记录 schema 精确为 v2，旧版、坏 JSON、kind/文件名不一致全部 fail startup，不跳过、不兼容读取。

```text
<dataDir>/assets/
  blobs/.yotta-blob-store
  blobs/<sha256hex>
  templates/<guid>.json
  clips/<guid>.json
```

记录中的 blob 必须是 `{mediaType,digest,size}` 完整 Blob Reference。`internal/blob` 独占 immutable object、单对象/总量 quota、integrity、range read、staging cleanup 与 Sweep；asset 层不能读取路径、扫描目录或自行删除对象。

`CommitRecordBlob` 与 `CommitVariantBlob` 把 blob seal 和 durable reference commit 放在排斥 GC 的同一生命周期临界区。普通 `PutRecord` 不接受 blob reference，避免“先写引用/后写对象”或 GC snapshot 交叉。`Get`/`List` 返回深拷贝，调用方不能修改 Store 内部 slice/pointer。

模板 variant 按 resolution 唯一；同 resolution 重拍会替换 Blob Reference，不同 resolution 追加。`PickVariant` 精确分辨率优先，否则按长边比选择最近档；scale tolerance 仍由 matcher 判定。解码缓存键是 Blob Reference digest，并在资产变更时整体失效。

GC 的 live set 是全部 template variant、clip 与 Application 枚举的 Workflow Source、Program、Run Blob Reference。asset 在生命周期锁内形成完整 snapshot，再交给 Blob Store `Sweep`；任何 durable root reopen/inventory 失败都会终止清理，preview token 失效也拒绝提交。Blob Store 只删除合法 object name，不处理上层记录。

Package export 写入 v2 record 和经 size/digest 重验的对象，zip object name 使用 digest hex，不能把 `sha256:` 中的冒号当 Windows 文件名。

## 创作与规模边界

- 资源库和节点 Inspector 必须复用同一分页 query、搜索、类型/标签过滤、缩略图预算和 variant identity；不能各维护一套 list/cache 生命周期。
- 节点 Inspector 打开搜索式 picker 并接收 exact BlobRef，不把 `asset × variants` 全量展开为普通 select。1000 个资产是基础验收规模。
- asset ID 用于管理和重命名，BlobRef 用于不可变运行绑定；UI 不得把 GUID、variant index 与 digest 混成一个身份。
- `Asset Store revision + asset:changed` 是 query/cache 的统一失效契约；template、clip、metadata、variant 和 delete mutation 都必须推进 revision。客户端可缓存有限页和 exact BlobRef 反查，但 mutation 跨过请求时必须丢弃旧结果。
- `ResolveBinding` 只提供 BlobRef → mutable presentation 的反查。找不到记录表示 stale library identity，不表示 Workflow binding 消失；Inspector 仍应展示 durable BlobRef 和 unavailable 状态。
- 模板创作的完成旅程是截图 → record/blob 原子提交 → 资源库查询可见 → picker 选择 → Workflow 保存 BlobRef → matcher/input adapter 使用同一对象。
- 录制 clip 的完成旅程还必须覆盖 recorder event canonicalization、codec round-trip、asset reload 与 playback；Store 单测不能替代这条链路。

旧 `ReadBlobDataURL` RPC 已删除。大对象不得整体 base64 后穿过 Wails；当前页 query 只按 thumbnail budget 返回 BlobRef，前端通过 bounded Blob preview adapter 渲染缩略图。preview 数据仅用于 UI session，不可持久化，也不建立临时兼容 RPC。

资产 RPC 仍包括 list/get、模板 capture/save/add/remove、metadata、delete/referrers、currentResolution/pickVariant 与 GC。capture/save 的现有 data URL 输入输出属于尚待迁移的 capture transport，不可扩展成通用 Blob API。

