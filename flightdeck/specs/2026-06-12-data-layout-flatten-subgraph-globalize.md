---
status: active
summary: 数据层大整理 — 目录平铺(删 assets/library 中间层) + 子图全局化(容器只引用不复制) + 架构耐久性根基
---

# 数据层大整理: 平铺 + 子图全局化

**拍板记录 (2026-06-12)**: 用户选定 ①子图全局化(UE/Blender 正统: 子图是全局资产, 容器引用不复制, 改一处全更新); ②资产元数据按类分目录(templates/ + clips/); ③library 子系统删除; ④跨机分享本期不做, **但必须留口子**(后续肯定做在线/跨机); ⑤外审两轮(ds/gpt/claude, 2026-06-12)折入 — 见「评审纪要」; ⑥**架构耐久性**: 这是架构性设计, 发布后再改就要背迁移包袱 — 趁现在还能破坏式更新, 把"以后不能再动"的根基一次钉死, 参考主流做法。

**动机**: 现状 容器/子图/library(clips templates subgraph)/资产(assets+records) 四套概念交叉, library 把资产+blob 整套复制成平行小宇宙(只对跨机分享有意义, 本机复用纯冗余); `assets/records` 名实不符; 多处死目录。主流对照(UE 单 Content 池+Migrate 按需抽出 / Blender datablock+标记不复制+Append 按需带依赖): 单池 + 引用 + 分享时抽取 是共识。

---

## 目标布局

```
data/
├── containers/<id>/container.json   ← 容器只剩自己(图+vars+设置), 不再有 subgraphs/ 子目录
├── subgraphs/<sgid>.json            ← 全局子图池(含匿名子图, 匿名不进库浏览)
├── templates/<guid>.json            ← 模板元数据(原 assets/records kind=template)
├── clips/<guid>.json                ← 录制片段元数据(原 kind=clip)
├── blobs/<sha256>                   ← 全局二进制池(原 assets/blobs 上提为 data/blobs, 内容寻址不变)
├── schedules/<id>.json              ← 拍平(原 schedules/<id>/schedule.json)
├── snippets.json
└── _backups/
```

消失: `assets/` 中间层、`library/` 整棵树、`containers/<id>/templates/` 死目录、`containers/<id>/subgraphs/`。

## 架构耐久性根基 (趁还能破坏, 一次钉死 — 用户拍板 2026-06-12)

发布后每条都改不动(要背用户数据迁移), 所以现在定死:

1. **ID 终身不变 + 全量完整 uuid**: 子图 ID 迁移时**全部统一重铸**为 `sg-<完整uuid>`(不是撞了才重铸 — 半新半旧的 32bit 老 ID 长期并存才是最大遗留, 跨机合并碰撞风险一直在)。资产 GUID 已是完整 uuid。ID 一旦铸出永不变更, **原则自铸出时起算** — 迁移重铸是旧 ID 体系的一次性整体退役, 不是对原则的例外(外审 ds#27 澄清)。这是未来一切迁移/同步/引用的锚(UE asset GUID 同款原则)。
2. **单调版本号 `rev`**: Subgraph 加 `rev int64`, 每次保存 +1。乐观锁比对 rev(不是 UpdatedAt — 同秒保存/时钟回拨会退化), 主流同款(k8s resourceVersion / CouchDB rev)。UpdatedAt 仅展示用。**rev 的地位钉死: 仅单实例并发控制, 不参与跨机真相判定**(跨机冲突以内容比对为真相, 同步协议建包时另定 — 避免分布式时钟谬误; 外审 gpt 采纳)。容器暂不加(非共享对象), 需要时靠 schemaVersion 演进补。
3. **所有持久化 JSON 顶层带 `schemaVersion: 1`**(子图/模板/clip/schedule; container.json 已有): 现在**只写字段不写任何读取分支**(二号铁律不破)。**读取契约(根规则, 现在钉死)**: `schemaVersion > 当前程序支持版本 → 拒绝加载并报错`, 否则版本号是装饰品(外审 gpt 采纳)。发布后格式演进 = "读旧版→一次性升级写回", 它是开关(UE FArchive 版本 / Blender 文件版本同款)。
4. **类型即目录**: 加新资产类型 = 加新顶层目录, 纯增量, 永不破坏现有布局。**目录与 JSON 内 kind 双源定权威**: kind 权威, 目录只是索引; 加载时两者不一致 → 拒载报错(fail fast, 外审 ds#29 采纳)。
5. **文件自描述**: 每个 JSON 内部自带 ID/kind, 离开目录结构可独立解析。
6. **内容寻址 + 幂等合并**: blob 按 sha256, 元数据按 GUID — 跨机导入天然幂等(同 sha 合一, 同 GUID 加性合并), 重构中不许削弱。
7. **持久化引用只准走稳定 ID**(外审 gpt 采纳, 最易被未来新人破坏的一条): 容器/子图/脚本/调度之间的一切持久化引用只能依赖稳定 ID(SubgraphID/GUID/容器 ID), **禁止依赖文件路径、目录结构、文件名、显示名称** — 与目录平铺目标同向。

---

## Phase 1 — 存储平铺 (对外 RPC/数据语义不变; 改动面=store 路径接线+测试 fixture)

1. **asset.Store 按 kind 分目录**: 写入按 `rec.Kind` 落 `templates/` 或 `clips/`; 加载扫两个目录; 内存索引仍按 GUID 统一(Get/List 接口不变)。`blobs/` 上提为 `data/blobs/`。**gcBlobs: 标记-清扫逻辑不变, 但记录扫描入口跟着改为 templates/+clips/ 两目录**(live set 仍 = 全部记录的 Variants[].Blob + Blob; 实现时别漏改扫描路径)。library 下的 blob 副本目录从来不在 GC 范围(=从没被删过), 随 library 整删消失。
2. **schedule.Store 拍平**: `schedules/<id>.json` 单文件。
3. **死目录清除**: `container/store.go:176` Save 里无条件 mkdir `templates/`(及该循环)删除, `store_test.go:102` 断言删除; `ensureV2DataLayout` 改建新布局目录。
4. **启动防呆闸**(外审 claude 采纳): 检测到旧布局**精确标记**(`data/assets/records/` 或 `data/library/subgraphs/` 存在 — 精确路径防全新安装误拒, ds#24) → 拒绝启动, 日志+弹窗提示"先跑迁移脚本"。否则新版以空目录启动, 数据"看起来全丢"且无报错。几行守卫, 不是兼容分支; 迁移期过后可删。
5. main.go 各 store 根路径接线改为新位置; 持久化 JSON 补 schemaVersion 字段(见耐久性 #3)。

## Phase 2 — 子图全局化 (语义改动)

### 后端

1. **全局 subgraph.Store**: `data/subgraphs/<sgid>.json`, 一条一文件。`Container.Subgraphs` 字段**删除**(二号铁律, 不留 shim); container.Store 不再加载/保存子图; normalizeSubgraph 自愈迁入新 store(自愈不依赖容器语境; RequiredGlobals 的语境见 #2)。Subgraph 模型加 `rev`/`schemaVersion`。
2. **RPC 改签名**: CreateSubgraph/GetSubgraph/UpdateSubgraph/DeleteSubgraph/ListSubgraphs 去掉 containerID 参数。**RequiredGlobals 持久化只存名字**(外审 gpt#3/ds#20 采纳, 替代上版"hint 语境重算"): Type/Default **不落盘** — 否则同一子图在 A/B 两容器各存一次, 文件被编辑语境来回污染(git diff 噪音+同步审计污染); 插入弹窗的 Type/Default 预填改为**即时现算**(用当前目标容器 Vars 匹配名字, 匹配不到降级 "any"/nil), 保存类 RPC 不再需要 hintContainerID。**校验只看名字**(见 #4)。**ID 铸造 = `sg-<完整uuid>`**(耐久性 #1)。**乐观锁**: UpdateSubgraph **和 DeleteSubgraph** 都带基准 `rev`, 盘上 rev 更大则拒绝(stale), 前端提示重载(外审两轮采纳; 实时协同/编辑锁仍不做)。
3. **运行时快照语义保持 + 依赖解析单一咽喉**: 咽喉分两层 — **节点级依赖提取**(单一实现: 图节点 Dependencies + 脚本 assetdeps)是 primitive; 其上两个便利函数: **正向闭包**(输入: 根=容器或子图; 输出 `ClosureResult{Subgraphs, Assets}` 结构体, 未来加维度不改签名; **接口不变量: 每个 SubgraphID 至多访问一次** — visited 集是契约不是实现细节, 防未来改实现导致导出无限递归, 外审 gpt 采纳)和**反向 referrer**(谁引用了 X — 遍历全部根做节点级提取后匹配, 返回引用位置列表 Referrer{容器/子图/节点}, 同今天 scanAssetReferrers, **不经 ClosureResult**)。起跑时正向闭包 → bundle 传给 CompileContainer 预编译, 跑中不回查 store(快照语义同现状); `CallSubgraph` 改查 bundle。运行时 bundle / validator / 删除 referrer / 未来导出 四处共用, 不许各自内联递归。**闭包缓存失效策略**(外审 ds 采纳): 子图池维护一个**写代数**(Create/Update/Delete/GC 任何写 +1), 缓存键带代数, 代数变则重建 — GC 删盘与缓存不同步的问题一并闭环。校验本就非每键触发(保存时 + 点检查按钮, 且检查在 dirty 时禁用), 无高频担忧。
4. **Validator 注入全局查找**: validateMissingSubgraph 的 known 来自全局池(validator 可见匿名子图 — CollapsedNode 校验需要; "UI 不可见"只是库浏览层的事; 匿名体从不由用户在 UI 挑选, 是折叠/录制操作程序化建立的)。静态防环(validateCyclicSubgraphs)保留换全局解析源; 运行时 32 层深度兜底不变。新增: 引用子图(含传递闭包)的 RequiredGlobals **名字** ⊆ 容器 Vars, 缺 → 校验错误(编辑器一键补全)。类型不参与校验; 变量运行时缺失/不符 = 现有变量节点错误语义, 不新增兜底。
5. **删除安全**: DeleteSubgraph 走反向 referrer(内存遍历, 非磁盘解析), 被引用 → 弹「被引用」警告(同模板删除现状)。**引用计数口径**(外审 gpt 采纳): UI 显示"被 N 个容器使用"= 直接+间接、按容器去重; 详情列引用位置(容器/子图/节点)。性能: O(全库节点数)内存遍历, 内测规模(个位数容器×数百节点)无感; 变慢预案 = 咽喉处集中维护反向索引(登记, 不建)。
6. **匿名子图 GC**: 仿 gcBlobs mark-sweep — `IsAnonymous && 扫描时点引用数 0` → 删。无持久引用计数; 触发点: 启动时 + 容器删除完成后同步跑一次(**全局锁序约定(外审 gpt 采纳, 后加模块也遵守): Container → Subgraph → Asset → Blob → Schedule**, 原子执行; GC 幂等, 批量删容器多跑几次无害, 有批量 API 则末尾跑一次)。编辑会话中不跑 — 保证编辑器 undo 复活引用时文件还在(undo 栈不跨重启)。具名子图永不自动删(用户资产); 容器删除不再连带删具名子图。
7. **library 子系统整删**: library store/service/copy.go、Export/Import RPC 全删。MissingGlobals 计算逻辑(copy.go:145-166)迁出复用于"插入子图引用时的缺变量检测"。
8. **复制为新子图**(fork 入口, ≈Blender Make Local): DuplicateSubgraph(sgID) = 深拷贝 + 新 `sg-<uuid>` + `rev=1` + 新 CreatedAt/UpdatedAt + **RequiredGlobals 原样复制不重算**(图相同重算结果也相同, 写死防止后人"优化"触发重算) + **产物一律具名**(IsAnonymous=false — Duplicate 是用户主动 fork 行为, 外审 gpt 采纳)。

### 前端

1. **editorStore**: `subgraphsByContainer` 槽位 → 全局 `subgraphsById`; **视图状态(editorPath/activeContainerID)仍 per-container**(复发#5 教训: 数据可全局, 前台指针/视图态必须按容器隔离, 不回退)。
2. **保存流**: useEditorSave 的 per-subgraph 循环改调无 containerID 的 UpdateSubgraph(带基准 rev); 被乐观锁拒 → 弹"盘上已有更新, 重载?"。dirty 跟踪按子图。
3. **库 UI 改池浏览**: LibraryView → 全局子图管理页(列具名子图: 名称/标签/被 N 容器使用/创建时间, 同名靠这些区分, 详情显示完整 ID; 删除走 referrer 警告; **加「复制为新子图」**); LibraryExplorerModal → 选中即插入引用节点(不再 import 复制); NodeInspector「发布到库」按钮删; ImportToContainerDialog 瘦身为"插入需补全变量"确认步。**同名允许**(ID 才是身份; 重命名辅助不本期做)。
4. **插入流**: onPickLibrarySubgraph → 检缺变量 → 确认补全 → addNode({kind:'Subgraph', config:{SubgraphID}})。无 import RPC。
5. **转脚本数据源**: useSubgraphToScript 的 subgraphsById 改取全局池。
6. **跨容器复制 CollapsedNode/Subgraph 节点**: 剪贴板天然可用(引用即语义)。粘贴出未知 SubgraphID(只在跨 data 目录实例间可能) → 校验报缺失子图, 与现状一致。

### 迁移 (一次性脚本, 跑完即删 — 不进产品代码, 不留常驻兼容分支)

0. **备份先行**: 整个 `data/` rename 为 `data.pre-flatten-2026-06-12`(同 v1 迁移先例); **脚本只从备份读、向新 `data/` 写**(library 包内容也从备份读, 时序无歧义) — 回滚 = 删新目录改回名字。
1. assets/records 按 kind 分发到 templates//clips/, blobs 上提; 各 JSON 补 schemaVersion 字段。
2. schedules 拍平。
3. **全部子图统一重铸 `sg-<完整uuid>`**(耐久性 #1, 外审 gpt 采纳 — 废除上版"字典序保留原 ID"规则): 各容器 subgraphs/*.json + library 包 root/embedded 全量汇入 data/subgraphs/。**去重比对**(忽略集写死, 外审 claude 采纳): 规范化 JSON(键排序)字节比对, **只忽略顶层 id + 元字段(rev/schemaVersion/createdAt/updatedAt)**; 嵌套引用(`config.SubgraphID` 等)按**重铸前原值**参与比对 — 同源副本嵌套 ID 相同故判相等, 分叉过的嵌套不同故判分叉, 两个方向都正确。相同 → 合一(铸一个新 ID); 分叉 → 各铸各的。维护映射 `(容器, 旧ID) → 新ID`, 改写全部引用: 图节点 `config.SubgraphID` + **Script 节点 Code 内字面 ID**。**脚本改写只支持静态可解析字面**(assetdeps 同款提取); 检测到旧 ID 出现在不可静态解析形式(变量/拼接) → **阻断并报告**(列容器/节点)。**"清零"的定义**(外审 ds/claude 采纳): 把动态构造改写为静态字面、或确认废弃并删除该节点 — 注释掉不算(文本扫描会过但运行时还会构造旧 ID)。清零后**整跑重来**(脚本只从备份读、新 data/ 整删重建, 天然幂等, 无断点续跑概念)。动态路径的运行时兜底 = 现有"子图不存在"Coded 错误。
4. library 包内 assets/blobs **幂等并回全局池**(GUID/sha 防重; 原资产若导出后被删, 包内副本是孤本, 必须并回)。library/ 不复制即消失。
5. 死目录不复制即消失(containers/*/templates, library/clips, library/templates)。
6. **自动对账**(外审 claude 采纳, 替代"人工复核"): 脚本打印前后计数(子图/模板/clip/blob/schedule、去重数、重铸数); **旧 ID 残留扫描 = 按映射表里的旧 ID 精确字符串集合 + 词边界匹配**(不是裸 grep 也不是格式正则, 防 `sg-xxX` 漏报和注释/描述误报; 命中列出位置人工过目即可判明, 外审 gpt 采纳), 出现次数必须=0; 启动 app → 全容器校验绿 → 抽跑一个容器。
7. 若用户还有别的机器装过, 先报备再搬。

---

## 跨机分享: 本期不做, 留的口子 (用户拍板 2026-06-12)

不建打包功能, 性质钉死(多数已并入「架构耐久性根基」): 幂等合并性(根基 #6) + 文件自描述(根基 #5) + 完整 uuid(根基 #1)。**未来导入必须校验同 ID 内容一致性**, 不一致走显式冲突决策(保留本地/采用导入/复制新 ID) — 协议建包时定。未来"导出包"= 正向闭包函数 + zip 壳(subgraphs + 资产元数据 + blobs + 包格式版本号), 即 UE Migrate; "在线库"= 同一包格式走网络。

## 不做 (YAGNI)

- 跨机打包壳/在线库本体(口子已留)。
- 子图版本化历史/实时协同/编辑锁(rev 乐观锁已够)。
- 无引用具名子图自动清理(库页人肉删; 堆多了再加"清理无引用"按钮)。
- referrer 反向索引(预案登记; 内存全量扫内测无感, 未量化阈值 — 真到了再说)。
- 同名子图重命名辅助。

## 风险与防呆

- **共享子图改动波及**: 库页+属性面板显示"被 N 容器使用"(即扫即得), 想独立改 → 「复制为新子图」; 不弹编辑确认(Blender 模式)。
- **在跑实例**: 起跑快照+预编译, 编辑不波及跑中容器(同现状)。
- **RequiredGlobals 提示**: 持久化只存名字; Type/Default 预填即时按当前容器现算, 匹配不到降级 "any"/nil — 只影响补全预填不影响校验。子图文件不再被编辑语境污染(无 git diff 噪音)。

## 评审纪要 (2026-06-12, ds/gpt/claude 两轮外审; 原文 tmp/*.txt)

**第一轮采纳**: gcBlobs"来源不变"措辞错误→标注扫描入口变更; 手搬迁移→一次性脚本+备份+确定性规则+对账+回滚(三家); 脚本字面 SubgraphID 改写纳入(gpt); last-save-wins→乐观锁(claude/ds); 失去 fork→「复制为新子图」(gpt); library 包 blob 幂等并回(ds 间接); 匿名 GC 时机闭环(三家); hint API 形态+降级语义; validator 可见匿名 vs UI 隐藏; referrer 性能口径+反向索引预案; Phase 1 表述精确化; 同名策略; 跨机导入冲突协议入口子。
**第一轮驳回**: "容器不存子图列表则引用关系丢失"(图节点 config 即持久化引用, 内存遍历同现行模板 referrer); "循环引用未处理"(已有静态防环+32 层兜底+BFS visited); "类型降级运行时崩"(变量动态类型, Type 仅提示, 校验历来名字语义); "剪贴板跨实例"(单应用单池, 走现有缺失校验); "闭包共用污染导出"(闭包输入是纯图结构, 容器绑定不在子图文件); "缺 Make Local"(已加 Duplicate)。

**第二轮采纳**: 老 8 位 ID 全量重铸完整 uuid, 废字典序规则(gpt#6 + claude#1, 与用户耐久性拍板同向); 乐观锁 UpdatedAt→单调 rev(claude#3, k8s/CouchDB 同款); Delete 也带 rev(gpt#2); ClosureResult 结构体防签名膨胀(gpt#4); 启动防呆闸: 旧布局存在拒启提示迁移(claude#5); 脚本改写"仅静态字面受支持, 动态构造阻断报告"(gpt#1/ds#11), 对账自动化"旧 ID 全池零出现"替代人工复核(claude#2); 咽喉两层化解"正向闭包 vs 反向 referrer 签名不匹配"(ds#13); GC 锁序 容器→子图(ds#15); 引用计数口径=去重容器数+详情列位置(gpt#5); validator 闭包缓存(gpt#7); Duplicate 语义写死: 新 ID/rev=1/新时间戳/RequiredGlobals 原样复制(ds#2/gpt#3); library blob 从备份读时序写明(claude#4); 同名区分加创建时间+详情完整 ID(ds#20); RequiredGlobals"残留语境"澄清为每次保存重算的建议声明(ds#24)。
**第二轮驳回**: "library blob 可能已被 GC 删成孤本"(ds#38 — "不在 GC 范围"=从没被删, 读反了); "CollapsedNode 无法在 UI 选匿名子图"(ds#17 — 匿名体由折叠/录制程序化建立, 从不经 UI 挑选); "启动 GC 删掉导出要用的匿名子图"(ds#6 — GC 只删引用数 0, 被引用即不删); "Duplicate 后引用计数归属"(ds#22 — 计数即扫即得无存储归属); "hint 语境 B 容器运行异常无兜底"(ds#8 — 现有变量节点错误语义即兜底); "Phase 1 标题与 gcBlobs 入口变化矛盾"(ds#37 — 对外语义不变成立, 扫描入口是内部实现); "内存规模未量化"(ds#29 — 接受为非阻塞, 阈值真到了再量)。

**第三轮采纳**: schemaVersion 读取契约(>支持版本拒载, gpt#1 根规则); 耐久性 #7 持久化引用只走稳定 ID 禁路径/名称(gpt#8 根规则); rev 地位钉死=仅单实例并发, 不参与跨机真相(gpt#2); RequiredGlobals 只持久化名字, Type/Default 即时现算不落盘(gpt#3/ds#20 — 消灭语境漂移与 git 噪音, hintContainerID 随之取消); 闭包不变量"每 SubgraphID 至多访问一次"入接口契约(gpt#4); 全局锁序约定 Container→Subgraph→Asset→Blob→Schedule(gpt#5); 对账按旧 ID 精确集合+词边界匹配替代裸 grep(gpt#6); Duplicate 产物一律具名(gpt#7); 去重比对忽略集写死=顶层 id+元字段, 嵌套引用按重铸前原值参与(claude#1); "清零"定义=改静态字面或删节点, 注释不算; 阻断后整跑重来(幂等), 动态路径运行时兜底=子图不存在错误(claude#2/ds#10); 闭包缓存失效=子图池写代数, GC 也 bump(ds#4/8/17); 防呆闸标记改精确路径防全新安装误拒(ds#24); "ID 终身不变"自铸出起算, 重铸=旧体系一次性退役(ds#27); kind 权威目录为索引, 不一致拒载(ds#29)。
**第三轮驳回**: "validator 每敲一键触发"(ds — 校验在保存/点检查时跑, 检查按钮 dirty 禁用, 无每键 BFS); "ClosureResult 对 referrer 缺位置信息"(ds#13 — 反向函数返回 Referrer 位置列表, 不经 ClosureResult, 咽喉两层已写明); "批量删除 N 次全量扫"(ds#32 — 内测无批量删 UI, 单删 O(N) 无感); "折叠未保存时校验行为不确定"(ds#22 — 检查 dirty 禁用 + GC 不在编辑会话跑, 已闭环); "rev 跨副本合并单调性"(ds — rev 已钉死不参与跨机判定, 即此问题的答案); "防呆闸改变启动行为与 Phase 1 标题不符"(ds#37 同类 — 标题口径是 RPC/数据语义)。
