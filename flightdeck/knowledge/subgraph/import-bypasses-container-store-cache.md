---
kind: trap
summary: "跨 store 写盘(import 子图)没同步属主 store 内存缓存, 致前端节点 \"(子图未找到)\""
activation: symptom
read_when: "写\"一个服务直接写另一个 store 拥有的磁盘文件\"类逻辑(library import/export / 录制落盘 / 外部工具改盘)前; 撞\"刚导入/刚生成的东西后端 List 查不到但磁盘明明有\" / 前端节点 \"(子图未找到)\" 但文件存在; review 跨 store 写盘是否同步属主内存"
---
# ⚠ 导入子图绕过容器 Store 内存缓存致"子图未找到"
## Signature
- symptom: `前端 Subgraph 节点出口渲染成 "(子图未找到)" (node.Subgraph.fallback_missing); 磁盘上 containers/<id>/subgraphs/<sgID>.json 明明存在`
- error_type: —  (数据/缓存一致性, 非 exception/错误码)
- where: internal/services/container/library/service.go ImportToContainer → 容器 container.Store 内存缓存 byID[*].Subgraphs
- trigger: 子图分享到库 → 在另一个容器里"添加这个子图"(LibraryExplorer import), 同一 app session 内不重启

## 症状/复现

把子图分享到库 → 在别的容器里通过 LibraryExplorer "添加这个子图" → 新容器的 Subgraph 节点显示「(子图未找到)」; 原容器的同名子图节点也跟着显示未找到。重启 app / 点重载后原容器自愈, 新容器里当时加的节点丢了 (从没存进 container.json)。

磁盘实锤: 目标容器 `subgraphs/<sgID>.json` 已写入, 但 `container.json` 里没有引用它的节点 (import 后 FE addNode 进 draft 但未保存)。

## 根因

**一个服务直接写了另一个 store 拥有的磁盘文件, 却没同步那个 store 的内存缓存。**

- `library.Store.ImportToContainer` (copy.go) 用 `os.WriteFile` 把子图直接写进 `containers/<id>/subgraphs/*.json` —— 这是它的设计提交点。
- 但容器子图的**属主**是 `container.Store`, 它只在 `load()` (开 app) / `Reload()` 时才把 `subgraphs/*.json` 读进内存 `byID[id].Subgraphs`。import 这条路绕过了它。
- FE 导入后调 `refreshSubgraphStore()` → `containers.ListSubgraphs(id)`, 拿的是**没刷新的内存旧名单**(缺刚导入的子图) → FE 又 addNode 引用该子图 → 名单里查不到 → `resolveSubgraphCallExecOut` 返回 `(子图未找到)`。

原容器"也坏"是次生: `useContainerEditorStore` 是**全局单例 Pinia store**, `subgraphsForCurrentContainer` 被目标容器的旧名单覆盖; 原容器编辑器若还挂着(keep-alive)读同一份 → 跟着未找到。原容器磁盘本来自洽, 切回/重载即自愈。**这个次生根因已单独成案并修**: 见 [keepalive-singleton-subgraph-store-stale.md](keepalive-singleton-subgraph-store-stale.md) (同症状 "(子图未找到)" 不同根因)。

跟 [storage-convention-consumer-audit-gap.md](../nodes/storage-convention-consumer-audit-gap.md) 同一类坑: 改/写一处共享存储时没把**所有读它的人**(此处是属主 store 的内存缓存)一并对齐。

## 修法

import 写盘后回调属主 store 刷新内存:
- `library/service.go`: 加 `reloadContainer func(id string) error` + `SetContainerReloader`; `ImportToContainer` 写盘成功后调 `s.reloadContainer(containerID)`。
- `main.go`: `librarySvc.SetContainerReloader(func(id string) error { _, err := containerStore.Reload(id); return err })`。
- 回归测试 `TestService_ImportToContainer_SyncsContainerStore` (service_test.go): import 后断言 `containerStore.GetSubgraph(id, sgID)` 可见。已验证修复前失败、修复后通过。

为何用 `Reload` 不用 `SaveSubgraph` 逐个加: `SaveSubgraph` 会按目标容器 `Vars` 重算 `RequiredGlobals`, 会冲掉 bundle 自带的; `Reload` 只是把 import 刚写的原样读回内存, 干净。

**通用教训**: 任何"服务 A 直接写服务 B 拥有的文件"的跨 store 写盘, 必须给一个回调让 B 刷新自己的内存视图 —— 写盘 ≠ store 知道。export 方向是只读磁盘, 不踩此坑。

## Cases
- 2026-06-09 首次 (用户报: 分享子图到库后两个容器都"子图未找到")
