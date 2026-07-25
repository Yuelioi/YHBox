# 3.1 产品创作体验与运行工作台优化 context

## What matters

Yotta 3.1 的唯一产品事实是 Workflow Source、Catalog/Node Contract、Compiler 和统一 runtime。
恢复旧体验只能复用用户心智和已验证交互，不得复制 3.0 Container、registry、localStorage store 或
第二套执行路径。当前已批准的 Stage 全部完成；下一次启动首先应确认用户是否带来新的真机反馈，
而不是自动继续历史清单。

## Decisions

- Selection、execution、debug 与 validation 是独立状态，不复用一种节点高亮表达。
- 复杂节点使用通用 Authoring Projection 加类型级 Editor Adapter；画布节点保持低密度，完整参数在 Inspector。
- Macro 与精准 InputClip 在领域、资源、编辑器和回放节点上分轨，不隐式互转。
- Tab 搜索、画布“添加节点”和连线候选共享 Catalog Projection；Snippet 快捷键必须可见、可校验冲突且只在编辑器上下文生效。
- Catalog Projection 继续作为 Tab、画布“添加节点”和连线候选的共同事实，但不再占用常驻左栏；左侧一级
  工具只承载子图定义、Macro、精准 InputClip、视觉模板和 Snippet 等需要持续管理的对象。
- 离开脏工作流必须提供取消、放弃、保存并退出三路选择，保存失败不得继续导航。
- 单对象短流程使用 Modal 保留列表上下文；长生命周期、多页面任务才使用独立路由。
- Stage 内运行最小定向门禁，Stage 完成后再统一执行 `task check` 和触发的真实宿主 smoke。
- 精准 InputClip 固化录制源 `counts/360`，回放目标从本机自动化目标/活动校准解析目标
  `counts/360` 并自动换算；不得用工作流倍率参数近似补偿。
- 可下载工作流携带精确 NodeRef、Workflow Source 和内容寻址 Blob；本机 target、credential 与校准
  不进入可移植资源，导入后必须经过兼容性预检和本机重绑定。
- 资产库中创建的图片、Macro 和 InputClip 是 Global Asset；工作流编辑中创建的是 Workflow Resource。
  编辑器同时发现当前工作流资源和全局资产，Workflow Resource 只能经用户显式提升为 Global Asset。
- 工作流只分发 Target Profile Definition，其设置 schema、应用发现提示与 `counts/360` 等首次安装默认值参与
  发布身份；首次安装会产生独立本机 Workflow Target Profile，其用户值不参与 Source 身份且升级只补新项、
  不覆盖已有项。Global Target Profile 只能经用户显式操作初始化或重绑定工作流配置。
- 精确路径/digest、HWND、设备序列号、provider authority 和 consent 只属于本机 Target Installation；本机配置
  虽不参与工作流签名，仍必须通过 schema、安全与可用性检查。
- Credential 可以是仅当前工作流使用的 Workflow Credential Profile，也可显式绑定可复用的 Global Credential
  Profile；API key/token 等 secret 始终只存在本机安全存储，不进入 Source、Bundle、发布签名或升级数据。
- 发布制品使用不可变 Workflow Release 身份，本机使用独立 Workflow Installation 身份；同一 Release 可安装多份，
  每份分别拥有 target/credential 配置、用户修改、运行授权、计划与更新状态，但可共享不可变 Source/Blob 字节。
- 安装不授予运行权限；首次手动运行和启用计划分别进行 consent，Release、Node Package 或权限范围变化后不得沿用失配授权。
- 在线平台只有上架/下架，没有 Workflow Release 撤销或远程停用；下架只停止未来下载，不影响已安装制品、
  作者签名、平台已出具的发布证明或离线运行。平台不能静默改写本机 Node Package TrustPolicy。
- 默认核心产品不后台联网检查更新；用户显式进入在线功能后才检查、浏览或下载，可选定期检查也必须由用户
  主动开启且不自动下载/安装。无网络或平台下架不影响本地编辑、运行或计划。
- 在线发布不做图片、Macro、InputClip 或 Source 的自动隐私内容检查，不引入 AI 识图或“扫描通过即安全”承诺；
  发布页只提示风险并列出打包资源，内容隐私由发布者负责。格式、大小、hash 和归档安全检查仍必须执行。
- 当前没有需要保护的旧格式用户数据；Stage M 直接替换开发期 Source/Bundle/资源形状，不增加旧 `.yotta-workflow`
  兼容 reader。从此次正式格式开始建立显式版本与未来迁移机制。
- 子图定义 `Graph` 与调用实例 `GraphCall` 是两个独立对象：画布 Delete 只删除调用；定义删除从子图管理器进入，
  默认在存在引用时阻止并列出调用位置。显式级联删除必须作为原子危险动作，不能复用普通 Delete。
- `Graph.inputs / outputs / entries / exits` 是子图接口的唯一事实；接口面板负责编辑该事实，内部 boundary 与外部
  GraphCall 只做投影。当前保持一个执行入口和多个命名 exec/error 出口，不恢复 legacy subgraph runtime。
- 子图接口身份与显示名称应分离；调用数、引用位置和接口健康度由 Source 派生，不新增平行前端 store。
- 折叠器推导 callable graph entry 时必须遵循 compiler Instruction 的主 `entryInput`；Repeat/ForEach 的
  `break/continue` 与 Retry 的 `retry` 是区域控制输入，不得因未连接而被误判为额外图入口。
- WebView 纵向验收必须等待产品真实状态而非固定 sleep：冷启动 hydration 使用独立较长窗口，普通 UI 转换保持
  短窗口；节点 Delete 前先断言选中，框选发送 Shift，推导接口确认预览，调试完成同时等待 Run 回到可再次启动。
- Run 状态可选类型与初值由 Authoring Projection 的 schema-validated `stateInitial` 提供；前端不得从 control
  或 examples 自行猜默认值。状态声明/compiler 负责 durable 与冻结类型，通用 Read/Write 泛型只绑定该已验证
  具体类型，才能同时支持命名类型和 `list<KeyCode>`；Increment 继续独立要求 numeric。
- 在线试验市场独立放在 `E:\projects\organizations\yottaapp\yotta-registry`，不进入 Yotta 桌面仓库；前端固定采用
  Nuxt 4 + Nuxt UI，后端固定采用 GoFrame v2，并优先复用 `yueli-official/platform/flightdeck/knowledge` 的公司约定。
- Registry API 使用 `/api/v1/<plural>`、camelCase JSON、RFC3339 UTC 时间和带 `traceId` 的统一响应 envelope；
  GoFrame 的 typed Req/Res 与路由是 OpenAPI 事实源。Nuxt 通过同源 BFF/SSR internal API 访问后端。
- Registry 的 Workflow Release 上传采用 staged upload → preflight/checked → 用户确认发布的状态机；公开浏览、搜索、
  下载是匿名只读能力，不为普通 GET 创建 guest session。目录页使用服务端分页与 URL query 保存筛选状态。
- Registry 不自建账号密码体系：发布者和管理员复用公司 OIDC，公开浏览与下载无需登录；固定测试身份只能在
  显式开发配置中启用，生产接口和授权模型始终按 OIDC subject 设计。
- Registry 业务层只依赖 Artifact Storage 接口：开发环境使用本地目录，部署环境使用 S3-compatible 后端，未来可接
  公司 Asset 服务而不改变领域/API。PostgreSQL 只保存制品元数据、digest、大小、storage key、签名和状态；上传采用
  init → upload → finalize/preflight，发布后制品不可覆盖，对外以 Release/Artifact ID 交付而不暴露真实存储路径。
- Registry 分别管理 Workflow Release 与 Node Package Release；前者只能以精确 release/digest 引用后者，不把可执行
  代码塞进 `.yotta-workflow`。客户端下载二者后仍分别导入工作流、确认并安装代码包；首版发现界面以工作流为主，
  节点包界面先服务依赖解析、签名检查和下载。
- Workflow/Node Package 上架采用人工审核：候选制品通过机械预检、发布者确认与管理员批准后，平台才对精确 digest
  签发 publication proof 并公开列出；拒绝保留提交记录，新版本独立审核。已上架版本只能下架，不可覆盖、撤销或远程停用。
- 未来 GitHub PR 投稿是与网页上传并列的 Submission Channel，不建立第二套发布语义；两种入口最终都必须生成相同的
  Publication Candidate，并经过同一预检、人工审核和上架流程。领域/API 需从首版保留 submission source/provenance。
- GitHub 投稿仓库只保存小型注册清单，不承载大型制品或作为运行时数据源。PR 清单锁定 identity、version、artifact
  locator/digest、作者签名、许可证和展示元数据；CI 将精确字节导入 Registry 暂存区并执行同一预检，合并映射为批准动作。
  Registry 保存 repo/PR/commit provenance，最终下载始终来自 Registry Storage，不依赖原 GitHub Release 持续存在。
- 发布身份与签名能力计划由公司 User Center 托管：用户用 OIDC 登录后，请求用户中心对 canonical release digest
  签发 Publisher Attestation；发布私钥不可导出，换设备无需迁移私钥。OIDC token key 与 publisher key 必须隔离，
  作者证明与 Registry 审核后的 Platform Publication Proof 也必须分离。GitHub 账号绑定属于同一用户中心能力。
- 用户中心缺失能力已在 `E:\projects\organizations\yueli-official\platform\flightdeck\work\2026-07-21-publisher-identity-attestation`
  建立独立 Work；Yotta Registry 只消费其未来合同，不在自身数据库复制用户、外部账号绑定或发布私钥。
- Publisher Namespace 是独立稳定身份，由 `User` 或未来的 `Organization` owner 持有，不等于可变用户名。首版开放
  个人 namespace 并预置官方 namespace，但数据库/API 从一开始使用通用 owner kind/ID，为以后组织成员与发布角色留 seam。
- Registry MVP 只闭环公开发现/目录/搜索/详情/版本/依赖/许可证/在线安装/离线包下载，发布者的候选上传、预检、
  确认、提交与状态，以及管理员的审核、拒绝、批准和下架。节点包先用简化详情；评分、评论、收藏、关注、付费、
  推荐算法和排行不进入首版。GitHub 先保留 submission source/provenance seam，不立即接入。
- Publisher attestation 通过 `PublisherAuthority` port 获取：正式 adapter 调用未来 User Center，`localdev` adapter 只为
  固定测试身份签发带独立 issuer/environment 的开发证明。生产配置发现 localdev 必须启动失败，开发证明不得进入正式
  Registry；领域、持久化和 API 只依赖通用 attestation，不固化临时实现。
- 在线安装与离线包共用精确 Installation Plan。在线逐项下载 Workflow/Node Package Release；离线交付把相同已签名
  字节装入 `.yotta-offline-pack`，外层清单只锁定内容与 digest，不产生新信任或携带本机配置。依赖缺失、下架或
  不允许再分发时不得声称可生成完整离线包，导入后仍分别确认工作流与可执行节点包。
- Workflow artifact 验证通过后立即创建本机 Workflow Installation，不等待依赖、target、credential 或 consent 完成；
  用户可随时退出设置并在“我的工作流”继续查看/编辑。Installation lifecycle 与 Readiness Report 分离，后者可同时
  报告缺依赖、缺目标配置、缺凭据和缺授权等 blocker；只有无运行 blocker 时才能运行或启用计划。
- Workflow Target Profile、credential binding、consent、schedule 和其他本机 target 选择属于 Installation 配置；修改
  节点图、连接、工作流内容参数、Workflow Resource 或 Target Profile Definition 必须显式创建新的本地派生 Workflow
  Source，并以 `derivedFrom` 保留原 Release provenance。派生内容不接受原 Release 自动覆盖或自动合并。
- 未派生 Installation 可由用户显式更新当前安装：先旁路下载/验签并计算依赖、target definition、credential slot 与
  权限差异，保留已有本机值并只补新增默认项，确认后原子切换 current Release。权限范围变化会暂停相关计划并要求
  重新授权；保留前一 Release 引用供显式回退。派生工作流只能把新 Release 作为独立 Installation 安装在旁边。
- Node Package 的 Publisher Trust、精确 Package Installation 与 Workflow Execution Consent 是三份独立本机事实。
  首版第三方信任只覆盖 `publisherKey + packageId`；已信任 key 的新版本仍需显式安装，capability scope 变化仍需重新
  授权。官方内置包使用应用自带 trust anchor；本机信任、安装和授权不随工作流分发，也不被平台下架或密钥停用远程改写。
- 持久本机 target/credential 配置只在“工作流设置 → 目标与凭据”中编辑：作者声明逻辑 slot 和 Target Profile
  Definition，节点只引用 slot。工作流可声明多个 target slot；不提供可任意改写 Installation 配置的通用节点。确需
  运行时切换时必须由专用 Node Contract 接收目标值并通过 capability 检查。

## Terms

- **Authoring Projection:** 从 Node Contract 与 Data Type 派生、可严格重开的编辑事实。
- **Macro:** 可逐动作编辑的简易输入自动化资源。
- **InputClip:** 保留原始时序、轨迹和交叠输入的精准录制资源。
- **Stage:** 多个相邻交付项组成的一次产品闭环和集中验收边界。
