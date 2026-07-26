# Local Markdown issue tracker

本仓库在没有外部 issue tracker 时使用 `docs/wayfinder/` 下的 Markdown 文件表达 Wayfinder map 和 tickets。

## Wayfinding operations

- Map 使用 `label: wayfinder:map`，每个升级主题一个目录。
- Ticket 位于该主题的 `tickets/`，使用 `label: wayfinder:<type>` 与相对 `parent` 指回 map。
- `status` 取 `open` 或 `closed`；`assignee` 为空表示未认领，写入执行者表示已 claim。
- `blocked_by` 保存 ticket 文件名。列表为空且未认领的 open ticket 属于 frontier。
- 解决 ticket 时，在正文追加 `## Resolution`，把状态改成 `closed`，清空 assignee，并向 map 的 `Decisions so far` 追加带链接的一行摘要。
- 新 ticket 先创建取得稳定文件名，再在第二遍写 `blocked_by`；Fog 中已变清晰的内容迁入 ticket 后必须从 Fog 删除。
