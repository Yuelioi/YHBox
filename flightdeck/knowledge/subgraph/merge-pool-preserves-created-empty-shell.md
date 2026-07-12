# ⚠ 子图 create→update 时 mergePool 会保留先入池的空壳

SUMMARY: 子图先 create 空壳再 update 内容时，changed 事件可能让空壳先进入前端池；mergePool 对同 ID 保留内存对象，普通 refresh 无法带回已写入的节点。
READ WHEN: 新增或修改子图 create→update 流程；折叠/导入/生成子图后后端已有内容但编辑器进入后为空；使用 mergePool 刷新刚创建的子图时。

---

## 症状与根因

“折叠为子图”先调用 `subgraphs.create` 建空子图，再调用 `subgraphs.update` 写入选中节点。create 会触发 changed 同步，空壳可能在 update 完成前进入 `containerEditor` store。

update 后调用普通 `refreshSubgraphStore` 仍不够：它走 `mergePool`，而 `mergePool` 为保护未保存编辑，对同 ID 总是保留内存对象。因此后端快照已经含节点，store 仍保留空 graph，双击进入就显示空白。

## 正确处理

create→update 流程在 update 成功后应按 ID 回读后端完整对象，并用 `replaceSubgraph` 显式覆盖该 ID，再执行常规池刷新。回读同时取得递增后的 `rev`，不能用本地拼装对象或旧 rev 代替，否则后续保存会触发乐观锁冲突。

回归测试要同时断言两层：后端写入快照包含移入节点，且最终 store 中同 ID 子图也包含该节点。只断言 update patch 会漏掉空壳保留问题。
