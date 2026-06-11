---
status: active
when_to_read: 写任何"需要子图完整内容(graph/entry/outputPins)"的前端功能前(转脚本 / 导出 / 分享 / 子图内容分析 / 跨子图遍历); 撞"明明磁盘有子图、editorStore 有、但我从 draft 里读出来是空 / undefined"; 给 Subgraph 节点加右键或面板动作要拿被引用子图时; review 任何 `draft.value.subgraphs` 读取
applies_to: [frontend, vue, subgraph, container-editor, draft, editorStore, useContainerEditorStore, subgraphsFor, data-source, subgraphToScript, useSubgraphToScript, json-dash, consumer-audit]
last_updated: 2026-06-12
resolved_by: <pending-commit> (转脚本两入口改从 editorStore.subgraphsFor(cid) 取; SubgraphLike 放宽转换器入参)
---

# `draft.value.subgraphs` 是永远空的幽灵字段 —— 子图完整内容只在 editorStore

## Signature
- symptom: `子图转脚本右键报「该节点未指定子图」(toast.subgraph_not_set); 面板入口静默无反应; 但子图磁盘上/editorStore 里都在`
- error_type: — (前端取数取到恒空字段, 非 exception)
- where: frontend `Container.subgraphs` 字段 ← 后端 `json:"-"` 不持久化也不随 get 返回; `loadIntoEditor` 只把 `listSubgraphs` 结果灌进 editorStore, 不回填 `draft.value.subgraphs`
- trigger: 任何功能从 `draft.value.subgraphs` 找/遍历子图

## 根因

后端 `Container.subgraphs` 标了 `json:"-"`(backend.ts:155 注释明写"后端不持久化到 container.json…前端通过 listSubgraphs 单独拿;这里声明 optional 仅供 type 完整性")。也就是说:

- `backend.containers.get(cid)` 返回的容器**不含** subgraphs;
- `loadIntoEditor` 里 `draft.value = JSON.parse(JSON.stringify(c))` 之后,子图是另走 `listSubgraphs` RPC 拉的,**只灌进 `editorStore`**(`setActiveContainer` → `subgraphsByContainer[cid]`),**没有**回填 `draft.value.subgraphs`;
- 所以 `draft.value.subgraphs` 从头到尾是 `undefined` —— 它只是个 type-only 占位字段,不是真实数据载体。

子图一键转脚本(2026-06-12 ship)两个入口都踩了这个坑:右键 `convertFromNode` 和面板 `onSubgraphPanelToScript` 都 `draft.value?.subgraphs?.find(...)` → undefined → 右键报「该节点未指定子图」、面板 `if(sg)` 假 → 静默。**与被转子图的内容毫无关系,任何子图、任何入口都必坏。**

## 唯一正确的子图完整数据源

前端拿子图**完整内容(含 graph/entry/outputPins)**只有一处:`editorStore.subgraphsFor(cid)`(= `subgraphsByContainer[cid]`,元素是 `SubgraphSummary`,尽管叫 Summary 但**含完整 graph** —— 见 containerEditor.ts:11 注释"v2 修复 Bug:activeGraph computed 需要完整 graph")。

- 用 `editorStore.subgraphsFor(opts.draft.value.id)`(按容器精确,不受 active 漂移),**不要**用 `subgraphsForCurrentContainer`(跟 active 指针走,多编辑器 keep-alive 下会漂 —— 见 [[2026-06-09-keepalive-singleton-subgraph-store-stale]])。
- 该 slot 含**匿名后备子图**(CollapsedNode 的 isAnonymous 子图),嵌套 callee 查找需要它,所以取全量、别先 `visibleSubgraphs` 过滤。
- `currentSubgraph`(useEditorPath)本身就是从该 slot 取的完整子图对象,面板/当前层级入口直接用它,别再绕一圈去 find。

## 为什么单测没拦住(verification gap)

转换器纯函数 `subgraphToScript.ts` 单测 18 例全绿,但它们**直接构造 sg 喂纯函数**,绕过了 composable 这层"从哪取子图"的接线。bug 全在接线层(`useSubgraphToScript` / view 入口),纯函数测试天然测不到。spec 验收第 3 条"真机走一遍"正是拦这个的 —— 真机债没清就漏了网。**教训:数据源接线 ≠ 纯逻辑,只有真机/组件级测试能验;纯函数全绿不等于功能能跑。**(同类 verification-gap 见 [[2026-05-31-node-timed-input-loses-backend-activate]]、[[2026-06-08-slate-click-up-coords-and-hold-lifecycle]]。)

## 怎么避免(下次)

写任何要子图完整内容的前端功能,第一反应就该是 `editorStore.subgraphsFor(cid)`,看到 `draft.value.subgraphs` 立刻警觉它是空的。根因同源于「子图数据在前端有多个表征、各管一摊」这个老问题(磁盘 / 后端缓存 / editorStore / draft),延伸阅读 [[2026-06-09-import-bypasses-container-store-cache]]。
