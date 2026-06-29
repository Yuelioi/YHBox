# 资产子系统 — GUID + 全局内容寻址 blob 池

SUMMARY: 模板与 clip 统一成稳定 GUID + 全局内容寻址 blob 池的资产子系统 — 存储/变体/匹配/分享导入全貌
READ WHEN: 改模板/clip 存储 / 节点引用素材 / 运行时模板匹配 / 子图分享导入 / 资产 picker UI 前; 撞"刚导入/重拍的素材查不到"、"(子图未找到)"、节点引用 GUID 失效、变体分辨率挑错档 类问题
RECHECK WHEN: 改 asset 存储布局 / 记录或变体 schema / PickVariant 挑档算法 / blob GC 条件 / import-export 合并语义 / asset RPC 面 / 资产 picker 交互时

---

模板(图片识别素材)与 clip(录制)统一成**全局资产**:每资产一个稳定 GUID,name 只是可变标签,像素/事件字节进全局内容寻址池。节点按 GUID 引用,资产独立于图引用存在。**身份是 GUID,不是人类命名** —— 这是整个设计的根(对标 Unity 的 stable-ID + content-addressing + 引用计数)。

## 存储布局

```
<dataDir>/assets/
  blobs/<sha256>          # 纯字节池: PNG 像素 / clip 事件流。内容寻址, 天然去重
  records/<guid>.json     # 一资产一文件 (git/diff 友好、并发安全)
```

**模板记录**: `{guid, kind:"template", name, description, category, tags, origin, variants[], createdAt}`,其中
`variants[] = [{resolution:[W,H], bbox, regions, blob:<sha256>}, ...]`(`description`/`category` 2026-06-13 加,库管理面就地编辑;`updateMeta(guid,name,description,category,tags)`)。
**clip 记录**同构、无 variant: `{guid, kind:"clip", name, tags, blob:<sha256>}`。clip 的 `label`/`description`/`category`/`tags` 权威值在 **clip blob 的 header JSON**(`inputclip/codec.go`),record 只存 Name/Tags 副本(供 GC/referrers);`category` 2026-06-13 加进 header(二进制 event chunk 不动)。

- **blob 身份 = 存盘文件字节的 sha256**(非感知/像素哈希)。自家截图走单一规范编码器 → 同次捕获字节恒等 → 去重生效;导入 bundle blob 逐字拷贝 → 相同文件天然去重。**不做**同像素不同编码的感知去重(YAGNI)。
- `origin` (`{kind:"user"|"imported"|"subgraph", sourceID}`) 仅溯源/picker 筛选,**非身份字段**。

实现: `internal/services/asset/`(`store.go` 记录+blob+锁, `pick.go` 挑档, `service.go` Wails RPC)。`Store` 内部 `sync.RWMutex`;记录启动期 preload 进内存索引,坏/半写 json 跳过+warning 不 fail。

## 变体模型 (核心不变量)

- **`variants[]` 按 resolution 唯一** —— 一个分辨率一条。
- **分辨率 = 截图那一刻目标窗口客户区像素尺寸**。`Capture`/录制走 `GetClientRect`(`pkg/capture` gdiFrame/wgcFrame 用 `image.Rect(0,0,clientW,clientH)` 定尺寸),所以**截图帧尺寸 == GetClientRect == `ResolveWindow().ClientW/H` == 变体的 resolution**,同一个 win32 调用、零 DPI 漂移。要拿"当前分辨率"用 `ResolveWindow().ClientW/H`,**不必截整张图**。
- **变体级写** `store.PutVariant(guid, res, blobSha, bbox, regions)`: 锁内按 resolution **upsert 单条**(同 res 覆盖、不同 res 追加),不走"读整记录→改→写回",从根避免并发改不同分辨率的 lost update。`RemoveVariant(guid, res)` 同理(service 层守卫: 仅剩 1 档拒删,引导走整删)。
- **重拍** = 同 resolution 换 blob(GUID 不变 → 所有引用自动跟随新图)。要"多分辨率档"= 换游戏窗口分辨率后重拍(不同 res 追加)。

## 运行时匹配

`templateMatcherAdapter.Detect(frame, guid, threshold, region, scaleTolerance)`(`wire_container.go`,**不再按 containerID 分 store**):
1. `store.PickVariant(guid, frameW, frameH)`(`pick.go`): **精确分辨率命中优先,否则长边比对称最近**的那一档(纯内存 map + 小循环,变体 1-5,每帧零额外开销)。
2. 长边比缩放比超出容器 `ScaleTolerance` → 判太远, miss + `emitScaleTooFarWarning`。
3. **解码缓存键 = `blob:<sha256>` → 未缩放 `*vision.Template`**(blob 不可变 → 缓存条目永不 stale, 删 blob 后旧条目死重无害);缓存在 PickVariant **之后**(变体已选定)→ 无跨分辨率误命中,同像素跨容器复用同一份解码。
4. 缩放(per-call)→ ROI / 多槽 NCC。

miss 不崩: template miss → `emitMissingVariantWarning` 走"未找到"分支;clip miss → `PlayClip` 自带 `Fail` exec 出口。

## 生命周期

资产**不被容器/子图拥有**,图只持 GUID 引用 → 删子图/删容器碰不到资产,不可能孤儿/误删。
- **安全删除**: 库里删资产时同步全量扫所有容器+子图引用(`Dependencer` BFS),返回 `referrers` 列表让用户确认"被 N 处引用"(现规模 ≈1 容器开销可忽略;大了再加反向索引)。删了仍被引用的资产 → 那些节点 `hasTemplate(guid)` 失败变红(MISSING),运行期 miss 不崩。
- **blob GC** (`gcBlobs`): live = 所有记录 `variants[].blob` ∪ clip `blob`;扫 `blobs/` 删不在并集里的。**只回收"记录已删/重拍换档"产生的孤儿字节** —— 一条没图引用但**记录仍在**的资产,其 blob 不回收(记录默认常驻、无自动上界,Unity 式)。

## 分享 / 导入 (幂等, 无 conflict/strategy)

GUID/sha 寻址天然幂等 → **冲突/strategy 概念整个删掉了**。
- **导出**(`ExportSubgraph` / `ExportContainer`,同一打包路径,闭包根不同): 图 + 资产闭包(BFS 收 GUID 记录)+ blob。bundle = `graphs/` + `assets/<guid>.json` + `blobs/<sha256>`。record 引用的 blob 缺 / graph 引用的 GUID 无 record → 导出报错中止。
- **导入** (`library.ImportToContainer(libSgID, containerID)`,2 参、单次幂等): 资产 GUID 不存在→整写,已存在→**按 resolution 加性合并变体**(本地缺的档加入、已有的保留本地,绝不静默覆盖在用的变体);blob 按 sha 去重;图直接写入(= 提交点)。返 `{imported, missingGlobals}`。**导入即落盘,无 dry-run**。
- **MissingGlobals**: 子图引用的全局变量(`SubgraphRequiredGlobal.Name`,变量系统),跟资产 GUID **正交**。容器缺这些 var 时:
  - 库管理页 `ImportToContainerDialog`(LibraryView)走"添加变量并继续"提示步。
  - 容器编辑器里直接导入(拖/点选 — NodePalette/useFlowInteraction/useNodeCreation)走 `library.importEnsuringGlobals`(`stores/library.ts`): **自动把缺的 var 补进容器 + toast**(子图要的变量本就必须有,自动补最省事)。
- 两人各截"同一个"按钮 = 两个不同 GUID(像素全等则 blob 去重但记录分开)。"合并成一个资产"不做(YAGNI)。

## RPC 面 (`asset.Service`, 全局无 containerID)

`list` / `get(guid)` / `saveTemplateCapture(dataURL,name,tags,recRes,region)→guid` / `addTemplateVariant(guid,...)→guid` / `removeVariant(guid,w,h)→guid` / `updateMeta(guid,name,tags)` / `delete(guid)→referrers` / `referrers(guid)` / `capture(containerID,nodeID?)` / `currentResolution(containerID)→[w,h]` / `pickVariant(guid,w,h)→{index,exact}` / `readBlobDataURL(sha)` / `gcBlobs()`。

`capture`/`currentResolution` 要目标窗口开着(走 `ResolveWindow`);FE 静默封装(窗口没开返 undefined 不弹 toast,浏览常态)。

## 前端

- `TemplatePicker.vue`(节点选模板,有容器上下文): 钻入式资产浏览 modal —— 网格页(虚拟滚动 + 多选 + 标签筛选) ↔ 详情页(滚轮缩放/拖拽平移 + 分辨率变体切换 + 行内改名/标签 + 上一个/下一个翻页 + 重拍/新增/删档)。详情页**当前分辨率感知**: 进来调 `currentResolution`+`pickVariant` 自动切到运行时真会用的那档、顶部显当前窗口分辨率、按钮按"当前分辨率有无精确档"在「重拍」(覆盖)/「新增」(加档)间切换;窗口没开优雅降级。
- `stores/templates.ts` `map[guid]`;`stores/library.ts` `SubgraphPackage = {root, embedded, assets:string[]}`(无旧 templates/clips)。

## 零迁移 (二号铁律)

升级后旧本地图里 `namespace.name` 的 Templates 值不是合法 GUID → 节点变红;旧 `_clips/` 不可读。**用户接受**: 重截模板、重录 clip。无迁移码、无 compat 读。
