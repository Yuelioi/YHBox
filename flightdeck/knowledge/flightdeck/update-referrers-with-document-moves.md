# Update referrers with Flightdeck document moves

Markdown 链接、源码引用和恢复入口都是单向引用；移动、重命名或删除目标不会自动更新引用者。

每次调整 Flightdeck 文档位置时：

1. 先用 `rg` 搜索旧文件名和相对路径，列出 deck、Work、Knowledge、仓库文档和源码中的全部引用者。
2. 只有在目标内容已经由新路径完整承载后，才删除重复权威来源。
3. 同一变更中更新所有引用者；历史材料需要保留时放在 owning Work 的 `references/` 并由 Work 链接。
4. 修改后再次搜索旧路径，并解析所有相对 Markdown 链接，确认目标存在。
5. 最后从 `deck.md` 按 Focus 恢复一次，验证新会话能逐步找到 Work、Context、可选 Plan 和必要 References。

同类的跨消费者审计见 [storage convention consumer audit](../nodes/storage-convention-consumer-audit-gap.md)。
