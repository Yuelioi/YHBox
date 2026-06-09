---
status: active
graduate: true
summary: 模板/clip 从'容器按 namespace.name 拥有'改为全局资产库(每资产稳定 GUID, name 仅标签) + 全局 assets/blobs/<sha256> 内容寻址字节池; 节点按 GUID 引用; 资产独立于图引用存在; 分享/导入坍缩成 GUID+sha 幂等合并
last_updated: 2026-06-09
---

# 统一资产子系统：GUID + 全局内容寻址 blob 池

## 1. 背景与问题

容器(container)持有图片素材(模板)与录制(clip)。子图(subgraph)内嵌在容器里、用容器的素材；子图能单独分享、容器也想能分享。**分享的粒度(子图)比拥有的粒度(容器)小**，而素材身份只是个容器内全局的 `namespace.name` 字符串(`template.ValidateKey` 正则 `^[a-z0-9_]+(\.[a-z0-9_]+)+$`，`internal/services/template/keyvalidation.go`)。由此四个痛点(用户全部勾选)：

1. **key 冲突/误覆盖** — 跨容器导入子图时，同一个 `namespace.name` 两边是不同的图 → 要么被挡、要么静默覆盖(`library/copy.go` 按 `kind:key` 检测冲突，不比内容)。
2. **重复/体积膨胀** — 同一张图被多个子图/容器各存一份；`VariantMeta.SHA256` 存盘时已算(`template/store.go:180`)却没用在去重上。
3. **孤儿/误删** — 多个子图共用一张容器素材时，删一个子图无引用计数 → 孤儿或误删。
4. **容器级分享缺失** — 现在只有 `ExportSubgraph`(`library/service.go`)，无 `ExportContainer`。

根因：**拿不稳定的人类命名当身份**。UE/Unity 的解法是稳定 ID + 内容寻址 + 依赖图 + 引用计数。

### 现状关键事实(已核验 file:line)

- 模板存 `containers/<id>/templates/<key>/`：`_meta.json`(KeyMeta) + N 个 `<WxH>.{png,json}` 变体。**节点引用的"key"不是一张图，而是一个多分辨率变体组**；每变体 = PNG + `BBox`(源帧像素位置) + `Regions`(多槽检测) + `SHA256`(`template/model.go`)。运行时 `PickBest(key, frameW, frameH)` 按帧分辨率挑档(`template/store.go:256`)。
- 节点按 key 字符串引用：`check/click/wait_template.go` 的 `Templates`(StringList, semantic `TemplateKey`)；`play_clip.go` 的 `ClipID`。
- 依赖抽取走 `Dependencer` 接口(`internal/node/interfaces.go`)，BFS 在 `dependency/scanner.go`。
- **clip 已用 string ID(非 namespace.name)+ 全局存** `_clips/` / `library/clips`(`inputclip/model.go`, `wire_inputclip.go`)，离目标最近，无 variant。
- GUID 设施现成：`github.com/google/uuid`(容器/子图/节点 ID 都用它)。

## 2. 决策(已拍板)

| 维度 | 决策 |
|---|---|
| 身份模型 | **B — 稳定 GUID + 底层内容寻址(Unity 模型)**。节点存 GUID，name 仅可变标签。重拍保留 GUID、换底层 blob，所有引用自动跟随。 |
| 存储范围 | **全局 blob 池**。跨容器/子图免拷贝、自动去重。 |
| scope | **模板 + clip 一起**，统一 kind-agnostic 资产模型。 |
| 生命周期 | **资产独立于图引用存在(Unity)**。只显式删除；blob GC 只回收被删资产的孤儿字节。 |
| 迁移 | **零迁移**(本地仅一个容器、计划后续重构 fish)。直接切干净、重拍。**二号铁律：不留兼容/双读/旧路径 fallback。** |

## 3. 数据模型 — 全局资产库 + blob 池

废掉每容器的 `templates/<key>/`。新增两个全局结构：

```
<dataDir>/assets/
  blobs/<sha256>            # 纯字节池：PNG 像素 / clip 事件流。内容寻址，天然去重
  records/<guid>.json       # AssetDatabase：一资产一文件(git/diff 友好、并发安全)
```

**资产记录(模板)** `assets/records/<guid>.json`：
```jsonc
{
  "guid": "a1b2…",            // 稳定身份，一次分配永不变
  "kind": "template",
  "name": "登录按钮",          // 可变显示标签，可重名
  "tags": [], "origin": {},
  "variants": [               // 变体组：元数据在记录里，像素在 blob 池
    {"resolution":[1920,1080], "bbox":[..], "regions":[..], "blob":"<sha256>"},
    {"resolution":[1024,768],  "bbox":[..], "regions":[..], "blob":"<sha256>"}
  ],
  "createdAt": "…"
}
```
**资产记录(clip)** 同构、无 variant：`{guid, kind:"clip", name, tags, blob:"<sha256>", …}`。

`origin`(`{kind:"user"|"imported"|"subgraph", sourceID}`)：来源溯源，沿用现 `TemplateOrigin`，用于 picker 展示/筛选。非身份字段。

**blob 身份 = 存盘文件字节的 sha256**(非感知/像素哈希)。我们自己的截图走单一规范编码器 → 同一次捕获字节恒等 → 去重生效；导入的 bundle blob 逐字拷贝 → 相同文件天然去重。**不做**"同像素不同编码"的感知去重(YAGNI，同 Unity)。全文统一写 `<sha256>`。

**变体组 upsert 语义(闭合 §3↔§6↔§11 的 GC 条件)**：
- `variants[]` **按 resolution 唯一**(一个分辨率一条)。`PutRecord` 合并变体 = 按 resolution 覆盖(同分辨率新条目替换旧条目)。
- **重拍**(同分辨率重新截) = 替换该 resolution 条目，写新 blob sha；旧条目被移除 → 其旧 blob 失去唯一引用 → 下次 GC 回收。
- **仅改 bbox/regions**(像素不变) = 原地更新该变体的元数据，**blob 不变**。
- 加/删某分辨率档 = 显式增删 `variants[]` 条目。
- 这是"重拍保留 GUID、所有引用自动跟随新图"的根，也让 §6 的"旧 blob 无记录引用 → GC"判断成立。

实现层：`internal/services/asset/`(新包) 提供 `Store`，**内部 `sync.RWMutex`**(沿用现 `template/store.go` 的并发模型)：
- blob 池：`PutBlob(bytes) -> sha`(幂等：temp+rename 原子写；已存在/rename 撞目标 → 视为成功，字节恒等无害)、`ReadBlob(sha)`、`HasBlob(sha)`。
- 记录：`GetRecord(guid)`、`ListRecords()`、`DeleteRecord(guid)`、`PutRecordMeta(guid, name, tags)`(只动记录级元数据)。
- **变体级写**：`PutVariant(guid, resolution, blobSha, bbox, regions)` —— **锁内按 resolution upsert 单条变体**，不走"读整记录→改→写回"，从根上避免并发改不同分辨率时的 lost update(ds/gpt 第二轮共识)。`RemoveVariant(guid, resolution)` 同理。整记录 `PutRecord` 仅用于新建/导入。
- `PickVariant(guid, frameW, frameH) -> (variant, blobSha, ok)`：把现 `template/store.go:256` 的 PickBest 算法上移到记录层(精确命中优先，否则长边比最近)。
- 启动期 preload 记录到内存索引(替代现 `template/store.go` 的 `metas/vars` map)；**坏/半写 `<guid>.json` 跳过 + warning，不 fail 启动**(沿用现 preload 容错 `store.go:64-72`)。记录文件写入一律 temp+rename 原子，"存在"即"完整"。

`internal/services/template/` 整包被 `asset` 取代/吸收(模板逻辑 = asset 的 template kind)；`inputclip` 的存储层并入 asset 的 clip kind(回放/录制逻辑保留)。

## 4. 节点引用：`namespace.name` → GUID

节点 pin 存 GUID 字符串。触点(机械为主)：
- `check_template.go` / `click_template.go` / `wait_template.go` 的 `Templates` pin：semantic `TemplateKey`→`TemplateGUID`，pin 名不动、StringList 形态不动。
- `play_clip.go` 的 `ClipID`：本就是 ID，语义对齐成 GUID。
- `detect/template_common.go` `templateDeps`：`Dependency{Kind:"template", Key:guid}`(逻辑不变)。
- `dependency/scanner.go` BFS、`validator_deps.go` 的 `hasTemplate(guid)`/`hasClip(guid)`：**逻辑不变**，Key 即 GUID。
- `validator_template_key.go` 的逐 key 格式校验(`ValidateKey` namespace.name 正则)：**整个删除**。GUID 合法性 = "记录存在"，已由 `validator_deps.go` 的 `hasTemplate(guid)` 覆盖，不需要再做格式正则。`template/keyvalidation.go` 一并删。

## 5. 运行时匹配 — PickBest 上移、缓存按 sha

今天 `templateMatcherAdapter.Detect(containerID, frame, key)` → 按 containerID 拿 per-container `template.Store` → `store.PickBest(key, W, H)` → `loadVariantTemplate(containerID, key, res)`(缓存键 `containerID:key:WxH`)(`wire_container.go`)。改为：
- **不再按 containerID 分 store**(资产全局)。`Detect(frame, guid)` → `asset.PickVariant(guid, W, H)` 拿 blob sha → 解码缓存取 `*vision.Template` → 长边比缩放(per-call) → ROI/多槽 NCC 匹配(`wire_container.go` 现算法逐字保留)。
- **PickVariant = 现 `PickBest` 算法逐字上移到记录层**(精确分辨率命中优先，否则长边比最近 + ScaleTolerance gate，`store.go:256-291` + `wire_container.go:258-282`)。纯内存 map 查 + 小循环(变体数 1-5)，每帧跑、零额外开销，**不重新设计**。
- **解码缓存键改为 `blob:<sha256>` → 未缩放 `*vision.Template`**(替代现 `containerID:key:WxH` 键)。缓存的是**解码后的 template**(非裸字节)，缩放仍在缓存之后 per-call。缓存在 PickVariant **之后**(变体已选定) → 不存在跨分辨率误命中；内容寻址让同像素跨容器/跨资产复用同一份解码。
- `node_services.go` 的 `VisionAdapter.Match/WaitMatch` 透传 GUID 列表，不预处理。
- `containerID` 参数从匹配链上**删除**(二号铁律：不留着不用)。miss(变体缺失/缩放太远) 仍走现 `emitMissingVariantWarning`/`emitScaleTooFarWarning` 路径，返回 miss 不崩。

## 6. 归属与生命周期 — "孤儿/误删"从根上消失

**资产不再被容器或子图拥有；它们是全局库里的东西，图只持 GUID 引用。**

- 删子图 / 删容器 → 只删那段图，**碰不到任何资产** → 不可能孤儿、不可能误删。痛点 3 直接没了。
- 引用计数(复用 `Dependencer` BFS，扫全部容器+子图的 GUID 引用)只服务两件事：
  1. **安全删除警告**：库里删某资产时**同步**全量扫一遍所有容器+子图，返回引用列表 `[{containerID, containerName, subgraphID?, nodeID, nodeKind}]`，让用户确认(同 Unity 删 asset)。当前规模(≈1 容器)开销可忽略；容器数大了再加反向索引(GUID→referrers，图保存时维护)——YAGNI，见 §12。
  2. **blob GC**：资产记录被显式删除、或重拍后旧 blob 无记录引用 → mark-sweep 回收。**按需/空闲触发**(或显式"清理未引用字节"动作)，不在每次删除时跑。

GC 算法：live blob = 所有现存资产记录 `variants[].blob` ∪ clip `blob` 的并集；扫 `blobs/` 删不在并集里的。O(记录数 + blob 数)。

**澄清(claude 第二轮)：blob GC 只回收"记录已被删除"产生的孤儿字节**(显式删资产、或重拍替换变体后的旧 blob)。**一条没有任何图引用、但记录仍在的资产，其 blob 不会被 GC** —— 因为记录还在 live set 里。这是 Unity 式"资产独立于图引用"的直接后果：记录默认常驻、**无自动上界**，靠"清理无引用资产"这个显式动作(§12，本期 backlog)删记录后，blob 才会被下一轮 GC 收走。所以"blob GC 自动省空间"仅适用于已删记录，不要误读成会自动清理积压的无引用资产。

**删除后图里的悬挂 GUID 行为(复用现有机制，非新建)**：用户确认删一个仍被引用的资产 → 那些节点的 `hasTemplate(guid)`/`hasClip(guid)`(`validator_deps.go`，现已存在的存在性校验)失败 → 编辑期节点变红(MISSING)。运行期按 kind 分：
- **template miss**(`CheckTemplate`/`WaitTemplate`/`ClickTemplate`)：`PickVariant(guid)` not-found → 现 `emitMissingVariantWarning` 返回 miss，走"未找到"分支，**节点不崩**。
- **clip miss**(`PlayClip`)：`ctx.Clip().Play` 返错 → `node.Failf(CodePlaybackFailed)` → 路由到 PlayClip **自带的 `Fail` exec 出口**(`play_clip.go:46-47`)。语义不同(失败而非未找到)，但同样不崩。

空串/损坏 GUID 同样被存在性校验兜住，不需要单独格式校验。**删除-与-新增引用竞态**(删除扫描通过后另一处图刚保存引用了该 GUID)：单机桌面应用基本不触发；即便触发，结果也只是退化成上面这条"悬挂 GUID = 安全失败"，不是数据损坏 → **不加锁**(YAGNI)。

## 7. 分享/导入 — 冲突机制整个坍缩

`library/copy.go` 的 `ImportConflict` / `strategy(overwrite/skip/cancel)` / key 冲突检测**大部分删除**：

- **导出**(子图或整容器) = 图(graphs) + 资产闭包(`Dependencer` BFS 收 GUID 记录) + 记录引用的 blob。bundle：`graphs/` + `assets/<guid>.json` + `blobs/<sha256>`。`ExportContainer` 与 `ExportSubgraph` 走同一打包路径，只是 **BFS 闭包根的节点集不同**：
  - **子图导出**：根 = 该子图的节点 GUID 引用集，沿 Subgraph-call 节点递归 callee(现 `ScanSubgraphDependencies`)。
  - **容器导出**：根 = 容器**顶层图**的节点 GUID 引用集；顶层的 Subgraph-call 节点同样递归进所有子图 → 收齐整容器的 template/clip GUID。(容器不再"拥有"资产，闭包入口是它的图，不是它的 templates 目录。)
  - bundle 完整性：record 引用的 blob 缺失 / graph 引用的 GUID 无 record → 导出时**报错中止**(不静默跳过半包)。
- **导入** = 幂等合并：
  - 记录按 GUID 落库：
    - GUID 不存在 → 整条写入。
    - GUID 已存在 → **按 resolution 加性合并变体**(claude 第二轮：原"整条跳过"会丢掉对方新加的分辨率档)：bundle 里本地没有的 resolution 档 → 加进来；**本地已有的 resolution 保留本地、忽略 bundle 同档**(绝不静默覆盖你正在匹配用的变体)。记录级元数据(name/tags)保留本地。同 resolution 不同 blob 的"分叉"不在导入期解决(要换图 = 用户显式重拍)。
  - blob 按 sha 落池：已存在 = 字节相同 → 跳过。**永不重复**。
  - 图塞进目标容器。
  - **导入的提交点 = 图写入成功**(gpt/claude 第二轮：非数据库事务)。blob/record 都是内容/GUID 寻址、per-file 原子写(temp+rename)、可幂等重放；崩在图写入前 → 至多留下孤儿 blob/record(GC 可回收)，**绝不产生悬挂图**。重试安全(已落的 blob/record 去重跳过，图重写)。
  - 保留 `MissingGlobals`(RequiredGlobals diff) 这一真实提示(`library/copy.go:computeMissingGlobals`)。**RequiredGlobals 是子图引用的全局变量名(`container.SubgraphRequiredGlobal.Name`，变量系统)，跟资产 GUID 完全正交**，key→GUID 不碰它，diff 逻辑逐字不变。导入时若 MissingGlobals 非空 → FE 弹"添加变量并继续"(沿用现流程，与资产合并互不干扰)。
  - 导入失败行为：bundle 里 record 引用的 blob 缺失 / graph 引用的 GUID 无 record → **报错中止**，不写半包。
  - 子图/节点 ID：沿用现导入语义(uuid，写盘按 ID，碰撞概率可忽略，不重映射)；资产按 GUID 幂等。本 spec **不改** 图/节点 ID 处理，只改资产层。
- `SubgraphPackage`(`library/store.go`)的 `Templates []string` / `Clips []string` 改成 `Assets []guid`；`library/clips` 单独存储路径并入 asset blob 池 → 净删代码。

边界：两人各自独立截"同一个"登录按钮 → 两个不同 GUID、两条不同 blob(除非像素全等则 blob 去重但记录仍分开)。这是正确 Unity 行为(genuinely 两个资产)；"合并成一个"是手动操作，**本期不做(YAGNI)**。

## 8. 捕获 UX — 自动 GUID + 自由 name

今天逼用户手输 `namespace.name`(还要保唯一，`TemplateCapture.vue`)。改为：截图 → **截后单个命名对话框**填 name(GUID 后台静默分配，用户看不到也不用管) → save 建记录。**仍是一步用户动作**(填 name)，不是先 rename 的两步。`TemplatePicker.vue` 遍历 GUID、显示 `name` + 缩略图，picker 靠缩略图/tags/搜索区分重名。clip 无缩略图 → 靠 name + tags + 时长/录制时间区分。前端 `stores/templates.ts` 的 `map[key]` → `map[guid]`，渲染逻辑几乎不动。

## 9. RPC / 前端改动面

- `template/service.go` 的 `Save/List/Delete/ReadPngDataURL/ListVariants/DeleteVariant/ReadVariantPngDataURL`：去掉 `containerID` 参数(资产全局)，key→guid；`Save` 入参不再要 namespace.name(返回新 GUID)。
- 新增/改 RPC：`asset.list()`(全局)、`asset.rename(guid, name)`、`asset.delete(guid)`(带 usage 警告)、`asset.gcBlobs()`。
- `main.go` 的 `templateSvc` / `templateMatcher` 接线(`main.go:186/207/214`)改成单个全局 `assetSvc` + matcher 共享同一 store；`SetOnChange` 失效广播保留。

## 10. 改动面汇总(file:line)

| 面 | 触点 | 性质 |
|---|---|---|
| 数据模型 | 新 `internal/services/asset/`(blob 池 + 记录 + PickVariant) | 新建 |
| 替换旧存储 | `template/store.go` `model.go` `service.go` 整包；`inputclip` 存储层 | 重写/吸收 |
| 节点 semantic | `check/click/wait_template.go`、`play_clip.go` | 低(标记) |
| 运行时匹配 | `wire_container.go`(Detect/PickBest/loadVariant/缓存键)、`node_services.go`(Match/WaitMatch) | 中(签名+去 containerID) |
| 校验 | `validator_template_key.go`、`validator_deps.go`、`template/keyvalidation.go` | 低(换规则) |
| 依赖抽取 | `template_common.go`、`scanner.go` | 低(Key=guid) |
| 分享/导入 | `library/copy.go`、`library/store.go`、`library/service.go`(+ ExportContainer) | 中-高(坍缩+净删) |
| 前端 | `stores/templates.ts`、`TemplatePicker.vue`、`TemplateCapture.vue` | 中(map 键+捕获流程) |
| 接线 | `main.go`、`wire_inputclip.go` | 中 |

## 11. 非目标 / YAGNI

- 不做跨资产"合并成一个 GUID"。
- 不做资产版本历史(重拍按 §3 变体 upsert 替换该分辨率条目，旧 blob 被 GC；不留 version 链)。要冻结快照 = 截成新资产(新 GUID)。
- 不做内容寻址以外的去重(感知哈希/相似图聚类)。
- 不做迁移脚本(零迁移)。
- clip 的回放/录制算法不动，只改它的存储身份层。

## 12. 待定(实现期定，非阻塞)

- GC 触发时机：显式按钮 vs 启动时 vs 空闲定时。倾向**显式 + 启动时扫一次**。
- blob 文件是否带扩展名(`<sha256>.png` vs 裸 `<sha256>`)：倾向裸 sha + 记录里标 kind/格式。
- `asset/records/` 单文件 per guid(本设计选定) vs 单 registry.json：选 per-file(git/diff/并发友好)。
- 反向索引(GUID→referrers)：现规模不做(全量扫够快)；容器数大了再加、图保存时维护。
- 手动"清理未引用资产"动作：资产记录默认常驻(Unity 式不随图删)，但可给用户一个显式入口列出"无任何图引用的资产"供批量删 → 触发 blob GC。本期可只做 blob GC，资产记录清理留 backlog。
- `PutBlob` 并发：临时文件 + rename(沿用现 `atomicWriteFile`)，同 sha 字节恒等 → last-writer-wins 安全。
- preload：启动期全量载入记录到内存索引；现规模无虞，膨胀后再考虑 lazy。

## 13. 验证(smoke)

落地后真机自检：
1. 截两张模板 → 各得 GUID；同一张图截两次 → blob 池只一份字节(sha 去重)。
2. 节点引用模板 GUID → 运行匹配命中(跨分辨率 PickVariant 正常)。
3. 重拍某模板(同 GUID 换图)→ 所有引用它的节点自动用新图，无需改图。
4. 导出含模板的子图 + 导出整容器 → bundle 带 assets/blobs；导入到另一容器 → GUID/sha 幂等合并、零冲突弹窗(除 MissingGlobals)。
5. 删一个引用了共享模板的子图 → 资产仍在库、其他引用不受影响。
6. 库里删某资产 → 弹"被 N 处引用"警告；确认删 → GC 回收其孤儿 blob。

## 14. 审核回应(3 AI review triage)

三份外部 AI 审核(ds/gpt/claude)，**不懂项目全貌**。按头号铁律：reviewer 只能 vet spec 内部一致性，涉及源码事实的断言已回源码核验。

### 已采纳 → 已修进上文
| 问题 | 改在 |
|---|---|
| 变体 upsert / 重拍换 blob 与 GC 条件未闭合(3 家共识，最强) | §3 变体 upsert 语义 + §6 GC + §11 |
| 解码缓存键 `blob:<sha>` 含义/会否跨分辨率误命中 | §5(缓存在 PickVariant 之后、缓存解码后 template) |
| ExportContainer 闭包根未定义 | §7(容器闭包根=顶层图 BFS) |
| 删资产后悬挂 GUID 的失败行为 | §6(复用现 `validator_deps` 存在性校验，节点变红/runtime miss 不崩) |
| blob=像素还是文件字节、去重粒度 | §3(=文件字节 sha256，只做精确字节去重) |
| 删除引用扫描的同步性/规模 | §6 + §12(同步全扫，规模大再加反向索引) |
| 捕获两步 vs 一步 name | §8(截后单对话框，一步) |
| origin 字段作用 / GUID 合法性兜底 / 并发写 / preload | §3 / §6 / §12 |
| `<sha>`↔`<sha256>` 记法不一致 | 全文统一 `<sha256>` |

### 驳回 → 误解，已回源码核验
- **ds: wait_template 是"等任意符合 name 的模板"，GUID 丢查询能力** — 假。`wait_template.go:70` `Templates`=具体 key 的 StringList，`MatchMode` any/all 决定"这几个里任一/全部命中"，**无 name 模糊查询**。GUID 列表行为完全等价。
- **ds: PickBest 有 sync.Map 结果缓存，PickVariant 每帧重算=性能倒退；fallback 未写** — 假。`store.go:256-291` 纯内存 map 查 + 小循环(变体 1-5)，**无结果缓存**；长边比 fallback + ScaleTolerance gate 在 `wire_container.go:258-282`。无倒退；贵的解码本就缓存(现按 sha)。
- **claude: RequiredGlobals 可能存旧 key 格式 → 留死引用** — 假。`copy.go` computeMissingGlobals 用 `rg.Name`(全局变量名)，跟模板 key 正交，key→GUID 不碰它。

### 已知取舍 → 故意为之(非缺陷)，方案 B + 全局池的既定后果
- **重拍改变历史图运行结果 / bundle 非冻结快照**(gpt#2/3)：这正是**方案 B**(原地更新传播到所有引用)对 A(纯 CAS 不可变)的取舍，用户已知情选定。要冻结 = 截新资产(新 GUID)，见 §11。
- **全局库引入跨容器耦合**(gpt#12)：故意。跨容器分享 + 去重就是目标；牺牲隔离换共享。
- **像素相同但 GUID 不同的两资产长期共存**(ds#IV.4)：故意 YAGNI。picker 靠缩略图/tags 区分；"合并资产"本期不做。
- **资产只增不减**(gpt)：Unity 式"资产独立于引用"的选择；blob GC 回收孤儿字节，资产记录常驻，另给手动清理入口(§12 backlog)。
- **GUID 已存在即跳过可能静默合并不同资产**(gpt#1)：uuid v4 碰撞概率≈0，唯一路径是手工伪造记录文件 → 防它是 YAGNI。接受。
- **clip 与 template 统一**(gpt#13)：本 spec 统一的是**身份 + blob 存储 + bundle/分享**，**不统一领域逻辑**——`kind` 分支保留(variants/PickVariant 仅 template；clip 单 blob、无变体、回放算法不动)。避免"模型被最复杂类型牵着走"。

### 第二轮 review triage

**已采纳 → 已修上文**
| 问题 | 改在 |
|---|---|
| 导入"已存在即跳过"丢对方新分辨率档(claude#2，最尖锐) | §7 改为按 resolution **加性合并**变体 |
| `PutRecord` 整记录读改写 → 并发改不同分辨率 lost update(ds/gpt) | §3 加**变体级 `PutVariant`**(锁内 upsert 单档) |
| 导入非事务、崩溃留半包(gpt#3/claude#3) | §7 **提交点=图写入**，blob/record per-file 原子+幂等可重放，孤儿 GC 回收 |
| blob GC 措辞误导(无引用但未删的记录其 blob 不回收)(claude#4) | §6 GC 澄清块 |
| clip 运行期缺失语义未定义(gpt#9) | §6(clip miss → PlayClip 自带 `Fail` 出口，≠template miss) |
| preload 遇坏 json / 原子写 / 并发 PutBlob | §3 |

**澄清 → 已答(非改设计)**
- 解码缓存按 sha 键，blob 内容不可变 → **缓存条目永不 stale**，删 blob 后旧条目只是死重无害(可选 LRU)，不需主动失效(ds#II)。
- PickVariant 选档后 per-call 缩放：模板小、NCC 本就是主开销，**每帧缩小模板可接受**(claude#1)。
- name **仅显示标签、全代码无 name 查找键**(引用一律 GUID，已核验引用面)；捕获对话框最多给非阻断"name 已存在"提示，不强制唯一(gpt#8/ds)。
- 容器导出含容器级全局变量：`Container.Vars`(`model.go:77`)是 container.json 一部分，随容器图天走，**不需单独 globals/ 目录**(ds#IV，已核验)。

**零迁移代价(显式确认，用户已拍板)** — ds#VIII：升级后旧本地图里 `namespace.name` 的 Templates 值不是合法 GUID → `hasTemplate` false → 节点变红；旧 `_clips/`/`library/clips` 文件新系统不可读。**用户接受**：重构 fish、重截模板、重录 clip。无迁移码、无 compat 读(二号铁律)。

**backlog / YAGNI(本期不做，显式记下，免再被提)**
- 无引用资产**记录** GC(本期只做 blob GC；记录清理给显式入口，留 backlog)。
- 反向索引(GUID→referrers)维护细节、`asset.repair`(record 在 blob 缺)、"批量替换失效 GUID 引用"UI、bundle 防篡改自校验、picker 分页/虚拟滚动、多容器(50-100)性能阈值、单独编辑 bbox/regions 的 UI/RPC(模型支持、本期无入口)、clip blob 的 format/version 字段(现 csbf/json 格式不动)。
- 判据：全是"当前规模(≈1 容器)用不上 / 现功能够用"的未来优化，撞到真痛点再做。
