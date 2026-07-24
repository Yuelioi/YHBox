# 3.1 产品优化计划

## Outcome

在唯一 3.1 Source/compiler/runtime 上持续改善专业创作与运行体验，同时保持每个 Stage 都能由真实用户旅程、定向验证和阶段门禁闭环。

## Current stage

Stage A–L 已完成实现、增量门禁与 Windows 真机验收。用户已把 Source-native 子图系统指定为下一阶段目标，
当前执行 Stage N；Stage M 的 M1 合同保持完成，M2 在 Stage N 闭环后继续。

## Stage N — Source-native 子图系统闭环

范围：不增加第二套 runtime，在现有 Graph/GraphCall/Graph interface/authoring patch seam 上完成定义管理、
显式接口编辑和安全可逆生命周期。

- [x] [N1 — 子图定义管理与对象语义](slices/stage-n1-subgraph-definition-management.md)：可搜索管理器、调用数、
  引用定位、唯一可读名称和无歧义删除入口。
- [x] [N2 — 显式接口编辑](slices/stage-n2-subgraph-interface-editor.md)：单入口、typed data inputs/outputs、
  命名 exec/error exits 的新增、重命名、排序、绑定与删除；自动推导降级为带预览的快捷动作。
- [x] [N3 — 生命周期闭环](slices/stage-n3-subgraph-lifecycle.md)：展开调用、复制调用、复制定义/创建独立副本、
  零引用删除和显式原子级联删除。
- [x] [N4 — 编辑器真实旅程与阶段验收](slices/stage-n4-subgraph-acceptance.md)：折叠、进入、接口编辑、
  调用投影、保存重开、编译、物理删除、production build 和 Windows WebView smoke。

非目标：不把 GraphCall 注册为 Catalog node；不持久化 boundary 虚拟节点；不恢复多执行入口或 3.0 Container/
subgraph store；不把 workflow-local Graph 定义升级成 Global Asset。

阶段门槛：每个生命周期动作通过同一 authoring patch interface 在前端 optimistic projection 和 Go 后端得到一致
结果；受影响调用与边在提交前可见；`task check`、保存重开和真实 WebView 旅程全部通过。

## Stage L — 第三批真机反馈闭环

范围：修复 `FD-19` 至 `FD-28`，保持资源、binding、Workflow Source/metadata 与子图接口的唯一事实，
不以临时前端状态伪造成功。

- [x] L1 — 资源拖拽、录制恢复与统一资源列表（FD-19、FD-20、FD-24）
- [x] L2 — 输入清除语义、模板 binding 与日志 message（FD-21、FD-22、FD-23）
- [x] L3 — Inspector 总开关与工作流设置/更多工具栏（FD-25、FD-26）
- [x] L4 — 新建子图 boundary 布局、非阻塞空态与接口刷新前置条件（FD-27）
- [x] L5 — 精准录制继承活动鼠标校准档案并明确目标覆盖语义（FD-20 真机复测）
- [x] L6 — 精准录制保存、鼠标裁剪、运行期校准与失败解释闭环（FD-28）
- [x] L7 — 任意裁剪边界状态补全与可创建资源元数据（FD-28 真机复测）
- [x] L8 — 录制元数据、工作流设置与 AI 审查按需加载，恢复 editor bundle 预算
- [x] L9 — 简易/精准录制统一待开始、3 秒倒计时与 F10/F11/F12 快捷键（FD-29）
- [x] L10 — 恢复精准回放绝对时间轴并把目标解析收敛到 playback session（FD-30）
- [x] L11 — 展示 InputClip 录制源/本机目标 counts/360，并按两份校准自动换算（FD-31）
- [x] L12 — 撤回临时转向倍率契约，清理开发期摘要并修复派生编译诊断

阶段门槛：每个切片先建立捕获截图症状的定向测试；完成后只运行增量 `task check` 与对应 WebView/
真实宿主旅程，不运行无关 Rust、全仓 coverage 或 production bundle。

## Stage M — 网上工作流分发与本机绑定

范围：建立正式的 Workflow Release/Installation、Workflow Resource、Target Profile Definition、精确 Node Package
依赖、在线下载与离线安装闭环。`.yotta-workflow` 保持 data-only；本机 target、credential、trust、consent、schedule
与用户配置不进入 Release 身份。Registry 只有上架/下架，不具备撤销或远程停用本地制品的能力。

### M1 — 正式可移植合同

- [x] 直接替换开发期 Source/Bundle 形状，定义首个正式 format/version；当前没有旧格式用户，不写兼容 reader。
- [x] Workflow Source 增加 Workflow Resource、Target Profile Definition、credential requirement 与精确 dependency manifest。
- [x] Workflow Resource 为图片、Macro、InputClip 保留稳定 ID、种类、名称、领域 metadata 与 BlobRef；多个节点可共享，
  局部分化必须显式 duplicate。
- [x] dependency manifest 锁定 publisher namespace、packageId、精确 package release digest 与完整 NodeRef 集合。
- [x] 固定 canonical Source、内容寻址 Resource Blob、Bundle manifest identity、严格 validation 与未来 migration seam；
  Workflow Release/attestation 的独立身份在 M3/M5 落地。

验收：相同语义产生确定性 digest；未知字段/版本、重复 ID、悬空 resource/slot、额外 ZIP entry、hash/size/media type
不匹配全部 fail closed；Source 中不存在路径、HWND、设备序列号、secret 或本机 consent。

### M2 — 资源归属与编辑器

- [ ] 资产库录制/截图创建 Global Asset；工作流编辑器内录制/截图创建 Workflow Resource。
- [ ] 编辑器同时显示当前工作流资源与 Global Asset；选中 Global Asset 时创建 metadata snapshot 的 Workflow Resource，
  可共享 CAS 字节，但后续全局修改/删除不影响工作流。
- [ ] Workflow Resource 支持显式提升为 Global Asset，提升不改变原工作流归属；共享资源编辑影响全部引用。
- [ ] 导入后图片、Macro、InputClip 的摘要、时长、录制源 counts/360 和编辑能力只依赖 Source/resource metadata，
  不要求接收方先建立 Global Asset 记录。

验收：导出后删除原 Global Asset 仍可完整编辑/运行；提升、duplicate 和共享更新都有领域与 UI 定向测试。

### M3 — Release、Installation 与本机配置

- [ ] 引入不可变 Workflow Release 与可多实例化的 Workflow Installation；verified workflow artifact 到达后立即创建
  Installation，即使依赖/目标/凭据/授权尚未齐全。
- [ ] Installation lifecycle 与 Readiness Report 分离；readiness 同时返回 dependency、target、credential、consent
  blocker 和可执行修复动作，查看/编辑不被阻止，运行/计划 fail closed。
- [ ] 首次安装从 Target Profile Definition materialize Workflow Target Profile；Global Target Profile 仅显式初始化/重绑，
  不做 live inheritance。精确本机身份只进入 Target Installation。
- [ ] 提供“工作流设置 → 目标与凭据”，持久配置不由通用节点修改；credential secret 只进入本机安全存储。

验收：同一 Release 可创建两份使用不同应用/credential/计划的 Installation；退出未完成设置后可继续，任何 blocker
都不会丢失或被折叠成单一模糊状态。

### M4 — Bundle、离线包与代码安装边界

- [ ] `.yotta-workflow` 只包含 canonical Source、Workflow Resource bytes、manifest 与可选 Publisher Attestation/
  Platform Publication Proof；私有离线文件允许 unsigned，并明确标记 unverified source。
- [ ] 定义 Installation Plan；在线逐项下载 workflow/package artifact，`.yotta-offline-pack` 原样装入同一组已签名字节，
  外层 manifest 只锁定列表/digest，不产生新的信任。
- [ ] Node Package Publisher Trust、精确 Package Installation 与 Workflow Execution Consent 分成独立本机记录；
  第三方 trust scope 为 `publisherKey + packageId`，新 release 仍需显式安装。
- [ ] 离线包无法包含缺失、下架或禁止再分发的精确依赖时拒绝声称“完整离线安装”；导入后代码安装仍独立确认。

验收：workflow 导入不会执行代码；未知 trust root 不能由离线包自举；官方包由内置 trust anchor 验证；平台或网络不可用
不影响已安装 workflow/package 的本地编辑、运行与计划。

### M5 — 编辑、更新与回退

- [ ] 本机 target/credential/consent/schedule 修改不改变 Release；节点图、连接、Workflow Resource 或 Target Profile
  Definition 修改必须显式创建带 `derivedFrom` 的本地派生 Source。
- [ ] 未派生 Installation 支持显式 staged update：下载/验签、diff、补新增默认项、展示 blocker、确认后原子切换。
- [ ] capability scope 变化暂停相关计划并要求重新授权；保留前一 Release 引用供显式 rollback。
- [ ] 派生 workflow 不接受原 Release 覆盖或自动 merge，只能旁装新 Release；任何 Release 都可“安装为新实例”。

验收：更新不会覆盖已有本机值；失败发生在切换前时原 Installation 可继续运行；下架不影响已缓存 rollback。

### M6 — 公开 Site Foundation prerelease（外部仓库）

- [ ] 在 `yueli-official/site-foundation` 提炼 Apache-2.0 的 `@yueli/ui/site/manage/auth` 与最小 Go HTTP/Auth module；
  公共中性主题与公司/Yotta Brand Adapter 分离。
- [ ] 通过独立 conformance app 验证真实 npm tarball、Go module、Nuxt SSR/BFF、OIDC、错误 envelope 与基础可访问性。
- [ ] 发布 `0.1.0-alpha.1` 后再让 Registry 消费；禁止跨仓库相对路径或复制 implementation。

依赖 Work：`E:\projects\organizations\yueli-official\platform\flightdeck\work\2026-07-21-public-web-foundation`。

### M7 — Yotta Registry 后端与制品控制面（兄弟仓库）

- [ ] 创建 `E:\projects\organizations\yottaapp\yotta-registry` 独立 polyrepo：`api/` 使用 GoFrame v2/PostgreSQL，
  `web/` 使用 Nuxt 4/Nuxt UI，并消费 Site Foundation 正式 prerelease。
- [ ] 实现 Publisher Namespace、Workflow/Node Package Release、Submission/Review、Artifact、Attestation、Publication
  Proof、Installation Plan 与 Delisting 模型；published identity 为 namespace + packageId + SemVer + exact digest。
- [ ] Artifact Storage 只暴露 port；local adapter 用于开发，S3-compatible adapter 用于部署。上传采用
  init → upload → finalize/preflight → confirm → review，published bytes 不可覆盖。
- [ ] `PublisherAuthority` 正式 adapter 对接未来 User Center；localdev issuer 只在显式开发环境工作，生产配置 fail closed。
- [ ] 提供匿名目录/搜索/详情/下载与 OIDC 发布/审核 API；GoFrame typed Req/Res 是 OpenAPI 事实源，响应使用公司 envelope。

依赖 Work：`E:\projects\organizations\yueli-official\platform\flightdeck\work\2026-07-21-publisher-identity-attestation`。

### M8 — Registry Web MVP

- [ ] 公开端只做发现首页、服务端分页目录/搜索、Workflow 详情/版本/依赖/许可证、简化 Node Package 详情、在线安装
  与离线包下载入口。
- [ ] 发布端只做 namespace、候选上传、预检、内容确认、提交审核与状态；管理端只做审核队列、批准/拒绝与下架。
- [ ] 普通 GET 匿名且不创建 guest session；发布/管理复用公司 OIDC BFF。页面覆盖 loading/empty/error/forbidden、
  keyboard、mobile、dark 与 reduced-motion。
- [ ] 不实现评分、评论、收藏、关注、付费、推荐算法、排行或自动隐私/AI 识图扫描；发布页只列资源并提示责任。

### M9 — Yotta 在线中心与安装旅程

- [ ] 在线功能使用显式入口；只有用户进入时才查询目录与更新，不做默认后台联网、自动下载或自动安装。
- [ ] 详情页展示作者证明、平台上架证明、license、精确 package 依赖、capability 与资源清单；下载后使用 M3/M4
  的同一 Installation/Readiness 流程，不建立在线专用安装路径。
- [ ] 在线检查只提示 immutable 新版本；更新、旁装、fork、rollback 都使用 M5 语义。Registry 下架只让未来获取失败。
- [ ] GitHub PR 仅预留 `submissionSource/provenance` 字段和 adapter seam；不在 MVP 实现 GitHub 集成。

### M10 — 跨仓验收与发布门槛

- [ ] 两个全新 Yotta workspace 验证：作者发布包含图片、Macro、InputClip/源 counts 的工作流，接收方在线安装与单文件
  离线安装得到相同 Source/resource/package digest。
- [ ] 验证接收方无需作者 Global Asset 即可查看/编辑资源；作者的路径、target installation、credential、secret、consent、
  schedule 与目标 counts 不泄漏；未完成本机配置时只能查看/编辑。
- [ ] 验证缺包、未知 publisher、包签名错误、权限扩大、下架、Registry/Identity/网络不可用、更新失败、rollback 和派生
  workflow 均保持既定 fail-closed/离线可用语义。
- [ ] 各仓执行自己的增量门禁；Yotta 最终运行 `task check` 与 Windows 真机安装/运行 smoke，Registry 运行 Go、Nuxt、
  OpenAPI、PostgreSQL integration 与真实浏览器门禁，Site Foundation 验证发布制品而非 workspace source。

阶段门槛：完整旅程不依赖第二套 Source/runtime、不泄漏本机个人信息、不把平台下架解释为本地撤销；所有发布身份、
依赖、资源与 proof 都锁定精确 digest，所有可执行代码、target、credential 和 consent 在正确 seam 显式处理。

## Stage K — 第二批真机反馈闭环

范围：修复 `FD-11` 至 `FD-18`，继续以 Catalog/Projection、Workflow Source、Compiler Program 和统一
runtime 为唯一事实；让调试中的未连接草稿节点不阻塞可达执行链。

- [x] [K1 — 图标、Switch、工作台与侧栏](slices/stage-k1-real-device-follow-up.md)（FD-11、FD-12、FD-14、FD-16）
- [x] [K2 — Run State、节点选择与草稿执行](slices/stage-k1-real-device-follow-up.md)（FD-13、FD-15、FD-17、FD-18）

阶段门槛：Catalog 图标、动态端口标题、状态类型、运行工作台和悬空草稿分别有定向回归测试；完成后
运行 `task check`，再在 Windows 真实宿主逐条复测八条用户旅程。

## Stage J — 真机反馈修复闭环

范围：只在唯一 3.1 Source/Contract/Compiler/runtime 上修复 `FD-01` 至 `FD-10`，复用现有
Authoring Projection、资源与运行事实，不恢复 3.0 Container 或第二套状态路径。

- [x] [J1 — 录制与模板上下文连续性](slices/stage-j1-recording-template-continuity.md)（FD-01、FD-02）
- [x] [J2 — 运行停滞可解释性](slices/stage-j2-runtime-observability.md)（FD-04）
- [x] [J3 — 节点与选项发现](slices/stage-j3-discovery-and-selection.md)（FD-03、FD-08、FD-10）
- [x] [J4 — 参数密度、行内编辑与 Run State 初值](slices/stage-j4-authoring-density-and-state.md)（FD-05、FD-06、FD-09）
- [x] [J5 — 动态 Switch 稳定分支拓扑](slices/stage-j5-dynamic-switch.md)（FD-07）

非目标：不把真机反馈拆成无关的视觉换肤；不改变许可证、发布范围或旧工作流兼容承诺；不以
前端临时状态伪造运行、资源或动态端口事实。

阶段门槛：每个 J 子项先建立能捕获用户症状的定向测试；J1–J5 全部通过后运行 `task check`，并按
改动触发 Windows WebView/真机 smoke。若某项缺少自动化 seam，必须在对应 Slice 记录人工验收步骤。

## Starting the next stage

只有出现新的真机反馈或用户明确扩展产品范围时才创建下一 Stage：

1. 先复现一条具体用户旅程，记录当前行为、期望行为和可核验差异。
2. 核对 [context](context.md) 中的架构边界，确认修复不会恢复 3.0 Container、第二套 store 或第二套 runtime。
3. 把同一旅程上的相邻问题组成一个可交付 Stage，并在这里写明范围、非目标与验收门槛。
4. 实施中运行最小定向检查；Stage 完成后统一运行 `task check` 和被改动触发的真实宿主 smoke。
5. 验收后重写 `index.md` 的 Current、Next 与 Progress，让下一会话只看到仍然成立的状态。

## Stable constraints

- Selection、execution、debug 和 validation 保持独立状态。
- 复杂节点由 Authoring Projection 与类型级 Editor Adapter 承载，画布只显示高频摘要。
- Macro 与 InputClip 分轨；脏资源退出必须保留取消、放弃、保存并退出三路语义。
- 单对象短流程优先 Modal；长生命周期、多页面任务才使用独立路由。

## Historical evidence

- [Approved plan through Stage G](references/approved-plan-through-stage-g.md)
- [Stage H node menu and template flow](references/13-node-context-menu-and-template-flow.md)
- [Stage I node density](references/14-node-density-and-optional-pins.md)
- [Stage I resource editing](references/15-workflow-resource-edit-and-safe-exit.md)
- [Stage I Tab and Snippet flow](references/16-tab-menu-and-snippet-shortcuts.md)
- [Stage I schedule modal flow](references/17-schedule-modal-flow.md)
