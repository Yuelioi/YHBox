---
status: active
summary: Phase 2 子图全局化 — 全局 subgraph.Store + rev 乐观锁 + 闭包咽喉 ClosureResult + referrer 删除安全 + 匿名 GC + library 整删 + 前端池化 + 一次性迁移脚本
last_updated: 2026-06-12
implements: specs/2026-06-12-data-layout-flatten-subgraph-globalize.md
---

# Plan P2: 子图全局化

## Progress

current: 后端 1-8 全部落码, `go build ./...` 绿; Go 测试修复与前端消费面改造由两个子代理并行收尾中。前端核心数据层(backend.ts / containerEditor store 池化 / useContainerDraft / useEditorSave rev 乐观锁)已亲手完成, bindings 已再生成。

**实现期修正(对 spec 的偏差, 均不损语义)**:
- SubgraphStore/SubgraphService 落在 package container 内(`subgraph_store.go` 重写 + 新 `service_subgraph.go`), 没开新包 — 避免导出 normalize 内部件; "全局 store + RPC 无 containerID"语义不变。
- 闭包缓存未实建(池规模无感), 写代数 `Generation()` 已留作未来缓存键 — spec 措辞是"允许复用缓存"非必须。
- MCP / node-catalog 两个工具校验传 nil 池(引用子图会报 MISSING_SUBGRAPH, 码内注释已标已知限制, 真需要再接池)。
- 前端 dirty 归属具体化为 `touchedByContainer`(本容器编辑会话动过的子图 ID 集) — 保存只写本容器动过的, 不跨容器代保; 池数据全局共享, 视图态仍按容器隔离(复发#5 红线)。
- 录制落盘接口 `ContainerSubgraphSaver.SaveSubgraph(cid, sg)` → `SubgraphSaver.Create(sg)`。

## Tasks — 后端

1. **全局 subgraph.Store** 新包 `internal/services/subgraph`(类型仍用 `container.Subgraph`, 单向 import 无环):
   - `data/subgraphs/<sgid>.json` 一条一文件, 原子写(tmp+rename 同容器 store 惯例); 内存索引 byID。
   - `container/subgraph_store.go` 的 normalizeSubgraph 自愈 + 原子写迁入; `required_globals.go` 迁入并改**只算名字**(`RequiredGlobals []string`)。
   - model(`container/model.go:139`): Subgraph 加 `Rev int64` + `SchemaVersion int`; `SubgraphRequiredGlobal{name,type,default}` 类型删除, 字段改 `[]string`; **`Container.Subgraphs` 字段整删**(store.go:109-126 加载循环、validate.go:197-199 Normalize 循环一并删)。
   - ID 铸造 `sg-<uuid.NewString()>`(service 层, 原 service.go:259 的 [:8] 截断废除)。
2. **依赖咽喉重构** `internal/services/container/dependency/`:
   - `ClosureResult{Subgraphs []container.Subgraph; Assets []string}` 结构体; 正向闭包 `Resolve(rootNodes, getSubgraph)`(BFS, **接口不变量: 每 SubgraphID 至多访问一次**, 注释写明是契约)。
   - 反向 `FindReferrers(targetKind, targetKey)` → `[]Referrer{ContainerID, SubgraphID, NodeID, NodeKind}`(遍历全部容器图 + **全局子图池**做节点级提取后匹配 — 注意: 子图图体不再在容器里, wire_asset.go:44-48 的扫描范围必须加上全局池, 否则模板 referrer 也漏)。
   - **写代数缓存**: subgraph store 任何写(Create/Update/Delete/GC)代数 +1, 闭包/列表缓存键带代数。
3. **运行时** `internal/services/container/runtime/`:
   - `runner.go:111-143` NewContainerRunner: 入参增子图解析(起跑时正向闭包 → bundle), `CompileContainer(c)` 改 `CompileContainer(c, bundle)`(compile.go:64-77 迭代 bundle 而非 c.Subgraphs)。
   - `subgraph_call.go:75-85` CallSubgraph 线性查 `rt.Container.Subgraphs` → 查 bundle。
   - 全仓 grep `\.Subgraphs` 清残留(listener.go / exec_frame.go / 测试 fixture)。
4. **校验** `internal/services/container/validator.go:368-403` + `validator_collapsed.go` + 防环: known 集合改注入的全局查找(validator 可见匿名); 新增"引用闭包的 RequiredGlobals 名字 ⊆ 容器 Vars"检查(允许复用写代数缓存)。
5. **RPC 面** 新 `subgraph.Service`(wails bind 进 main.go), `container/service.go:235-313` 子图 CRUD 迁出:
   - List/Get/Create(label)/Update(sgID, patch, **baseRev** — rev 不符拒绝)/Delete(sgID, **baseRev** + referrer 警告流, 同 asset 删除)/Duplicate(深拷贝+新 uuid+rev=1+新时间戳+RequiredGlobals 原样复制+**IsAnonymous=false**)。
   - 录制落子图改走全局 store: `recording/service.go:350,352` 一带的 SaveSubgraph 调用点。
6. **匿名 GC** `internal/services/subgraph/gc.go`: mark-sweep(IsAnonymous && referrer 0 → 删); 触发: main.go 启动 + 容器 Delete RPC 完成后同步; **全局锁序 Container → Subgraph → Asset → Blob → Schedule**(包级注释钉死)。
7. **library 整删**: `internal/services/container/library/` 全删 + main.go:183 接线删; MissingGlobals 名字 diff 逻辑(copy.go:145-166)以"名字集合差"形态进 subgraph.Service(给前端插入流用, 或纯前端算 — 前端已有容器 Vars, 倾向纯前端, 后端不留 RPC)。
8. **container.Store.Save** mkdir 循环整个删(P1 已只剩 subgraphs); 容器 Delete 不再删子图目录。

## Tasks — 前端

9. **数据层** `stores/containerEditor.ts`: `subgraphsByContainer` → 全局 `subgraphsById: Map`; **editorPath/activeContainerID 仍 per-container**(复发#5 不回退); `lib/backend.ts` 子图 RPC 换新签名(去 containerID, 带 rev)。
10. **加载/保存** `composables/containerEditor/useContainerDraft.ts`(204-254 加载/激活路径) + `useEditorSave.ts`: 加载拉全池; 保存循环改 UpdateSubgraph(sgID, patch, baseRev), 被拒 → "盘上已有更新"对话框(重载/放弃); dirty 按子图。
11. **创建路径三处去 containerID**: `useFolding.ts:46` / `useRecording.ts`(onSubgraphCreated 刷新) / `useSubgraphLifecycle.ts:30`(autoCreate)。
12. **库 UI 池化**: `LibraryView.vue` → 全局子图管理页(具名列表: 名称/标签/被 N 容器使用(去重口径)/创建时间; 详情完整 ID; 删除 referrer 警告; 「复制为新子图」); `LibraryExplorerModal.vue` → 选中即插引用; `NodePalette.vue` library tab 数据源换池; `NodeInspector.vue` 「发布到库」按钮删; `ImportToContainerDialog.vue` → 瘦身成"插入需补全变量"确认步(缺名清单前端现算: 闭包 RequiredGlobals 名字 − 容器 Vars; Type/Default 预填按目标容器现算)。
13. **插入流** `useNodeCreation.ts:248-268` onPickLibrarySubgraph: 检缺名 → 确认补全 → addNode(无 import RPC); `stores/library.ts` 删除或缩成池查询皮层。
14. **杂项**: `useSubgraphToScript.ts:41` 数据源换池; 跨容器剪贴板粘贴未知 ID 走现有缺失校验(确认无新增代码需要); i18n zh/en 增删键(library 导入导出文案删, duplicate/乐观锁对话框/被引用警告增); `pnpm i18n:check`。
15. **测试**: containerEditor store spec / useEditorSave / 受影响 vitest 全改; 新增: rev 拒绝路径、Duplicate、插入缺名补全。

## Tasks — 迁移与验收

16. **一次性迁移脚本** `tmp/migrate-flatten/main.go`(go run, 合并后删): 按 spec 迁移节 0-6 步实现(备份 rename → 读备份写新布局 → 全量重铸+去重(忽略集: 顶层 id+rev+schemaVersion+时间戳) → 引用改写(图节点+脚本静态字面, 动态构造阻断报告) → library 资产幂等并回 → 对账打印(计数 + 旧 ID 精确集合词边界扫描=0))。
17. **跑迁移 + 真机验收**(用户): 跑脚本看对账 → 启动 app → 容器列表/编辑器/子图库正常 → fishing-v2 跑一轮 → 顺手验旧债"删被引用模板 → referrer 警告"。回滚预案: 删新 data/ 改回备份名。**迁移同一时刻翻转读真实 bin/data 的测试 fixture 路径**: `runtime/inspect_phase_test.go` templateNameForGUID 的 `assets/records` → `templates`(P1 期间提前翻会让整批 fish state 测试翻红 — 2026-06-12 实测回归后回退, 代码里已留 NOTE)。

## 设计要点

- 砍点顺序: 后端 1-8 一起到 `go test` 绿(中途 Container.Subgraphs 删除会让全仓大面积编译错, 是预期的迁移波, 不留过渡 shim); 前端 9-15 到 typecheck/test/build 绿; 16-17 收尾。
- 二号铁律: 不留 deprecated RPC、不留双读路径; library 相关 i18n/组件/类型一次删干净。
