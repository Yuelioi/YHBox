# Slice 17：资产库规模化管理与安全清理

## Outcome / Question

让模板和 clips 在大规模库中仍可搜索、分页、批量维护，并用完整引用根保护 immutable Blob 数据。

## Completion criterion

- 后端 QueryAssets 提供稳定分页、搜索、kind/category/tag 筛选、排序和 total。
- UI 支持当前页全选、跨页选择、批量改分类/标签与批量删除，返回逐项结果并保留失败选择。
- 模板 variant 可从列表/详情新增、重拍、删除；最后一档仍要求显式删除整个资产。
- 建立 Blob root inventory，覆盖资产 metadata、全部 Workflow Source、持久 Program 与历史 Run durable values；Node Package payload 使用独立 generation store，不属于共享 Blob Store。
- cleanup 先 preview，再显式确认；token 覆盖 live refs 与 object inventory，引用或对象变化时拒绝 stale preview。
- 成功操作原地更新，不使用成功 toast；分页和筛选状态在操作后保持可理解。

## Blocked by

Slice 16 已明确 bundle 对 BlobRef 的拥有/搬运语义。

## Verification

- asset service 查询/筛选/分页、批量 metadata/delete、GC live root/stale preview adversarial 测试。
- Program、Run、Application durable Blob inventory 与 Blob retention 测试。
- frontend 跨页选择、批量、variant、cleanup 聚合测试。
- Stage 8 唯一完整 `task check` 已通过：Go tests/coverage/vet/staticcheck、RPC contract、frontend format/lint/typecheck/i18n/147 tests/production build/bundle budget。

## Out of scope

云资产同步、在线市场、自动修改工作流中的 BlobRef、按推测引用强删 Blob。

## Result

Completed。资源库改为后端 QueryAssets 分页与稳定筛选排序；UI 支持跨页选择、逐项批量结果、失败项保留、模板分辨率档新增/重拍/删除。Blob cleanup 通过 preview token 在确认时重新核对 asset、Workflow Source、Program、Run 与 Blob object inventory；工作流 bundle 导入使用 retention，避免 Source 发布前的对象竞态。
