---
kind: note
summary: "模板与 clip 统一为稳定 GUID + Blob Reference；Blob Store 3.1 独占对象与 GC 生命周期。"
activation: action
read_when: "改模板/clip 存储、模板匹配、资产 RPC、Blob preview、GC 或 package export 时。"
recheck_when: "改 asset schema、Blob Reference、CommitRecordBlob/CommitVariantBlob、PickVariant、preview adapter 或 import/export 时。"
---
# 资产子系统 — GUID + Blob Store 3.1

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

GC 的 live set 是全部 template variant 与 clip Blob Reference。asset 在生命周期锁内形成完整 snapshot，再交给 Blob Store `Sweep`；Blob Store 只删除合法 object name，不处理上层记录。

Package export 写入 v2 record 和经 size/digest 重验的对象，zip object name 使用 digest hex，不能把 `sha256:` 中的冒号当 Windows 文件名。

旧 `ReadBlobDataURL` RPC 已删除。大对象不得整体 base64 后穿过 Wails；缩略图恢复必须使用 bounded Blob preview adapter，并让 UI/session URL 可撤销且不可持久化。当前前端显示占位图，不建立临时兼容 RPC。

资产 RPC 仍包括 list/get、模板 capture/save/add/remove、metadata、delete/referrers、currentResolution/pickVariant 与 GC。capture/save 的现有 data URL 输入输出属于尚待迁移的 capture transport，不可扩展成通用 Blob API。
