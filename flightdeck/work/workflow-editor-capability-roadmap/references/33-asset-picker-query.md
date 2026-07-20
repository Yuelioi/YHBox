# Slice 33：Asset Picker Query 与规模化绑定

## Outcome / Question

统一资源库和节点 Inspector 的查询/选择契约，用搜索式 picker 取代全量下拉，使模板、clip、图片在 1000+ 资源下仍可发现并绑定 exact variant。

## Completion criterion

- `AssetPickerQuery` 支持服务器分页、搜索、类型/分类/标签过滤、排序、thumbnail budget 和最近使用。
- picker 返回稳定 asset/variant identity 与 BlobRef；Inspector 不加载全库。
- 资源库与 Inspector 共享 query/cache/invalidation 生命周期。
- asset create/delete/metadata/variant 变化后 picker 同进程一致。
- Blob cleanup 在证明 Source/Run/package 等完整 roots 前 fail closed。

## Blocked by

Slice 29；错误反馈依赖 Slice 30。

## Verification

- 1000 assets、多 variants、分页失效和 stale selection fixture。
- 资源库创建模板后无需刷新即可在 Click/Wait Template picker 中搜索并选择。
- G09 contract/integration 通过；Stage R3 再做人工 UX 和性能验收。

## Out of scope

- 不把普通 select 虚拟化后继续伪装成资产浏览器。
- 不在缺完整 roots 时实现破坏性 Blob GC。
- 不改变 BlobRef 内容寻址语义。

## Result

Completed。

- `QueryAssets` 现在提供服务端分页、搜索、类型/分类/标签过滤、稳定排序、recent GUID 排序、thumbnail budget 与 atomic store revision；无界 query 通过 `ASSET_QUERY_INVALID` 拒绝。
- 资源库与节点 Inspector 复用单一 Pinia query/cache/invalidation 生命周期；template/clip 写入统一发布 revisioned `asset:changed`，请求期间发生 mutation 会至多重试一次并丢弃旧页。
- Click/Wait Template 与 InputClip Inspector 已移除 `clips.list + assets.list + asset × variants` 全量展开；统一搜索式 picker 只读取当前页，并让用户选择 exact variant BlobRef。
- Workflow 仍只持久化 immutable BlobRef；`ResolveBinding` 仅把 exact BlobRef 反查为可变显示元数据。资产被删除时 Inspector 保留 durable binding 并显示 stale，而不是静默清空。
- 缩略图只在当前页预算内下发 BlobRef，再通过 bounded `PreviewBlob` 渲染；最近使用只影响 picker 的本地排序提示，不进入 Workflow Source。
- 1000 templates × 2 variants fixture、分页/thumbnail budget/recent sort、exact variant 反查、删除后 stale、统一 revision event、共享 cache invalidation 均有 Go/Vitest 证据；G09 人工 UX/响应预算仍在 R3。
- Blob cleanup 继续在 GC 生命周期锁内合并 Asset Store 与 Application 提供的 Source/Program/Run durable roots；任一 root reopen/inventory 失败即不执行 sweep，preview token 变化也拒绝提交。
- R1 阶段门禁 `task check` 通过：Go 全仓、vet/staticcheck/coverage、43 个前端测试文件 183 tests、类型/i18n/RPC contract、production build 与 bundle budget 全部通过。

