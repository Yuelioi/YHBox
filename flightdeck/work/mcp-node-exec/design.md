# MCP 节点执行：AI 跑节点闭环造容器

## 0. 定位:这是 "AI 功能" epic 的第③块

| | 子系统 | 方向 | 状态 |
|---|---|---|---|
| ① | 本地 AI 配置 | 我们调 AI | done(cold archive `2026-06-23-local-ai-config`) |
| ② | AI 节点(图里调 LLM) | 我们用 AI | done(`docs/ai-nodes.md`) |
| ③ | **MCP 对外暴露** | **AI 调我们** | ← 本 spec |

**北极星(用户定)**:让外部 AI **自己跑节点(实验/探测)→ 攒出容器(生成自动化)**。这是个闭环:

```
list_nodes(看有哪些积木) → find_window(锁窗口) → run_node(Capture 看截图 / DetectColor 拿真实坐标 / ClickAt 验证点对没)
   → … 反复探测,每步的输出就是造图依据 … → save_container(把实测参数烘进图,落 GUI store)
```

`run_node` 是**实验探针**,`save_container` 是**生成**(已有,复用)。两者在同一个 MCP server 里闭环。`run_node` 的返回值是命门——必须把节点输出原样回吐,AI 才能拿实测值填图。

## 1. 目标 / 非目标

**目标(In)**
- GUI 进程**内置**一个 MCP server,复用 GUI 已接好的执行标准件(InputBus / TemplateMatcher / GameProvider / settings / container.Store / asset store)。
- **通用 `run_node(kind, params, window)`**:跑任意单个**动作节点**,返回其输出(数据字段 + Capture 的图像)。工具面**不堆 bespoke 动作工具**——`list_nodes` 即文档,AI 自描述地选 kind+填参。
- **`find_window` / `list_windows`**:解析/枚举顶层窗口 → 返稳定窗口句柄(HWND)给 AI 后续带着用。
- 现有写图四件套(`list_nodes` / `get_graph_schema` / `validate_container` / `save_container`)**整合进 GUI server**,后端换成 GUI 真实 store——AI 存的图直接出现在界面里。
- **全局 arm 安全开关**(设置页,默认关):AI 要驱动鼠标键盘得用户手动「武装」才生效。

**非目标(Out — YAGNI,未发布无兼容包袱)**
- **整容器执行**(场景 A:AI 说「跑 fishing-v2」)。AI 攒好的图由 **GUI 跑**(▶ 按钮/热键),MCP 不接。等 author 闭环跑顺真有需求再议。
- **有状态会话 / attach-detach**:无状态 + 显式句柄(见 §2)。
- **模板匹配 / 回放工具**(check_template / PlayClip):依赖预存资源,较重;v1 不暴露。基础动作不需要 asset/clip/子图池那一坨。
- **流式 / 进度事件回传 MCP**:AI 同步等 `run_node` 返回即可。
- **每次调用弹确认**:会卡死 AI 循环;安全靠全局 arm 开关。
- 独立 stdio bridge 进程:留作将来 fallback(见 §8),现在不建。

## 2. 关键决策(已与用户拍板)

| 决策 | 选择 | 理由 |
|---|---|---|
| 执行场景 | **细粒度动作**(非跑存图、非写图自测) | AI 在动作级当编排引擎,我们只提供原子动作 |
| 运行架构 | **依附正在跑的 GUI** | 复用全套已接线执行栈 + 天然真实桌面会话;不重接 backend |
| 会话模型 | **无状态 + 显式句柄** | AI 节奏慢(每次调用隔数秒),临时建 backend 开销相比 LLM 往返可忽略;免会话生命周期/超时回收 |
| 工具粒度 | **通用 `run_node`**(非 N 个 bespoke 工具) | 节点目录已自描述(kind/pin/类型/required/默认);工具面自动随目录扩,贴合通用框架哲学 |
| 收割输出 | **复用 held-output 缓存**(`execOutputs`) | ②epic 刚落的产物:节点 fire 时输出字段自动入缓存,跑完直读;零新收割机器 |
| Transport | **GUI 内置 Streamable HTTP** | 同一个 mark3labs/mcp-go 库支持;localhost URL 配进 AI 客户端;无第二进程 |
| 安全 | **全局 arm 开关,默认关** | AI 驱动本机输入有风险(prompt injection/乱点);手动武装是力度合适的闸,不卡循环 |
| 现有 spike | **整合进 GUI server,独立 stdio 进程退役** | 未发布无兼容包袱;authoring 与 execution 收一个 server,save 落同一 store |

## 3. 架构:MCP server 进 GUI 进程

现有 `cmd/yotta-mcp` 是**独立 stdio 进程 + 自带 `container.Store`**,只写图。本 spec 把它**整合进 GUI**:

- GUI 启动时(`main.go`)起一个 **Streamable HTTP MCP server**(goroutine,绑 localhost),挂上 authoring + execution 全部工具。
- server 注入 GUI 的**常驻标准件**(已在 `main.go` 接好):`InputBus`、`TemplateMatcher`、`GameProvider`(`newGameProviderAdapter`)、`clipSvc`、`settings`(取 `ActiveMouseCounts360`)、`container.Store`、asset store、`emitForRuntime`。
- 现有 `cmd/yotta-mcp` 独立进程**退役**(或日后改造成纯 stdio↔HTTP bridge,见 §8);catalog/schema/validate/save 逻辑搬进 server 包复用(`internal/catalog` 本就被两端共享,无需重写)。

**新包归属**:`internal/services/mcp`(server 装配 + tool handlers),被 `main.go` 装配。execution harness(§5)落 `internal/services/container/runtime` 旁或同包(因要碰 `ContainerRunner` 内部)。

## 4. 工具面

| 工具 | 入参 | 返回 | 来源 |
|---|---|---|---|
| `list_nodes` | — | 节点目录(kind/pin/类型/required/默认/category/能力位 + i18n 文案) | 复用 `catalog.BuildWithI18n()` |
| `get_graph_schema` | — | 图 schema 约定 + 校验过的样例 | 复用现有 `schema.go` |
| `validate_container` | container JSON | `[]ValidationError` | 复用现有,store 换 GUI 的 |
| `save_container` | container JSON | `{id, path, warnings}` | 复用现有,store 换 GUI 的(AI 存图立刻进界面) |
| **`find_window`** | `{title?, class?, processName?, titleMatch?}` | `WindowHandle`(hwnd + title/class/process/pid/clientW/clientH) | `winutil.ResolveWindow` |
| **`list_windows`** | — | `[]WindowHandle`(全部顶层可见窗口) | 新增 `winutil.EnumTopWindows`(§7) |
| **`run_node`** | `{kind, params{pin→literal}, window(hwnd)}` | `{ok, firedOutput, data{field→value}, image?, error?}` | 新增 harness(§5) |

`list_nodes` 已含每个节点要哪些参数、什么类型、哪些必填——AI 据此构造 `run_node` 的 `params`。

## 5. `run_node` 机制(零新执行机器,全复用)

把单动作节点包成一个**合成微容器**,经现有 `ContainerRunner` 跑,从 held-output 缓存收割:

1. **构造微容器**:`{nodes:[Start, 目标节点(config.literal=params)], edges:[Start.Done → 目标节点.<execIn>]}`。`<execIn>` = 从节点 spec 读出的 exec 输入 pin 名(`catalog` 里 `Exec:true` 的 input),不硬编码 "In"。容器 `InputBackend`/`CaptureBackend` 默认 `postmessage`/`auto`(后台、不抢前台)。
2. **预置目标窗口**:`rt.SetActiveWindow(wh)`(`runtime_context.go:183`,直接按句柄设)。先 `winutil.WindowMetadata(hwnd)` **重校验句柄还活着**(HWND 可能被 OS 复用),拿到当前 clientW/H 一起塞进 `WindowHandle`。
3. **建 backend**:`NewContainerRunner(rt).Run(ctx)`——`setupRuntime` 见 window 已置 + `Input==nil` → **自动建 input/capture backend**(`runner.go:339` 幂等逃生口,测试 fixture 本就走这条)。`ctx` 带超时(per-call deadline)。
4. **收割**:跑完读 `r.execOutputs["<目标节点ID>.<field>"]`(held-output 缓存,§docs/held-exec-outputs)。需给 `ContainerRunner` 加一个**只读访问器** `ExecOutputs() map[string]any`(小改动)。
   - **`firedOutput`**(走了哪个 exec 出口):**spec 驱动确定**——每个 exec 出口声明自己的 Data 字段集(`DetectColor.Found.Data=[Center,Count]` / `NotFound.Data=[]`),按收割到的字段属于哪个出口的 Data 集反推。无需加 sink 节点。
   - **`image`**:某输出字段类型为 `Image`(`node.Image{Format,Data}`)→ 编码成 MCP 图像结果块(base64 + `image/png`|`image/jpeg` 按 Format);其余字段进 `data` JSON。
   - **`error`**:节点 Run 返错 / 走 Fail 出口 → 复用节点错误模型的 `Code`/`Error`(见 §10)。

**为何这条路成立(已核实,非脑补)**:`SetActiveWindow` 按句柄设窗口(`runtime_context.go:183`);`setupRuntime` 幂等且「预置 window + Input 留 nil 即自动建 backend」是**测试现成先例**(`runner.go:337-381`);收割靠 held-output 缓存正是 ②epic 的产物。

## 6. 可跑节点的闸

`run_node` 只接**动作节点**,按 `catalog` 能力位过滤:

- **放行**:`NeedsWindow==true` 的节点(input 类 ClickAt/KeyPress/MouseMove/Scroll、detect 类 DetectColor、image 类 Capture、window 类等)——它们就是「对窗口做一件事」的原子动作。
- **拒绝**:结构/控制节点(Start/Stop/Loop/Switch/Subgraph/EventTick listener)、`IsPureData==true` 纯数据节点(GetVar/Now/Add… 要数据线喂、脱离图无意义)。传进来 → 返 `UNRUNNABLE_KIND` 错误,附一句指引。
- 闸的判定**数据驱动**(读 catalog 字段),不写死 kind 名单——节点增删自动跟随。具体放行集 plan 落地时按 category 收敛(可能需要一个小白/黑名单兜边界,如 Win32WindowTarget 本身不该 run_node——它的职责被 `find_window` 取代)。

## 7. 窗口工具

- **`find_window`**:复用 `winutil.ResolveWindow(ctx, MatchSpec{Title,Class,ProcessName,TitleMatch}, timeout, interval)`,返首个匹配的顶层可见窗口完整元数据。超时未找到 → `WINDOW_NOT_FOUND`。
- **`list_windows`**:`winutil` **新增** `EnumTopWindows() []WindowHandle`——枚举全部顶层可见窗口返列表(EnumWindows 回调收集,复用 `getWindowText`/`getClassName`/`queryProcessName`/`getClientSize` 现成 helper)。给 AI「先看有哪些窗口可选」。
- **句柄稳定性**:句柄 = `HWND`(uintptr)。**HWND 复用坑**:窗口关闭后 OS 可能把同值分配给新窗口。缓解:`run_node` 执行前 `WindowMetadata(hwnd)` 重查,拿到的 title/class/process 一并回吐;AI 可比对是否还是它要的窗口。无状态设计下不在 server 端存句柄→窗口映射,AI 自负其责(可随时 `find_window` 重拿)。

## 8. Transport:GUI 内置 Streamable HTTP

- GUI 启动时起 `server.NewStreamableHTTPServer(mcpServer)`(mark3labs/mcp-go),绑 `127.0.0.1:<port>`,后台 goroutine `Start`。端口固定或 settings 可配(plan 定)。
- AI 客户端(Claude Desktop / Cline 等)配这个 localhost URL 即连。
- **plan 待验**:mark3labs/mcp-go 的 HTTP/SSE server 确切 API(`NewStreamableHTTPServer` vs `NewSSEServer`、`Start`/`http.Handler` 形态、与 Wails 进程生命周期/优雅关停的衔接)。现有 spike 用 `ServeStdio`,HTTP 形态需实测库版本。
- **stdio bridge(未来 fallback,不建)**:若某客户端只认 stdio,现有 `cmd/yotta-mcp` 二进制可改造成瘦 stdio↔HTTP 代理转发给 GUI。YAGNI,真碰到再加。

## 9. 安全:全局 arm 开关

- settings 加一个 `mcp.armed bool`(默认 `false`)+ 设置页一个显眼开关(类似「武装/解除」)。
- **arm 闸只管会改变世界的工具**:`run_node`(驱动输入)、`save_container`(写 store)。`list_nodes`/`get_graph_schema`/`validate_container`/`find_window`/`list_windows`(只读/查询)**不受闸**——AI 随时能看、能规划、能校验,只是不能动手/落盘。
- 未武装时受闸工具 → 返 `NOT_ARMED` 错误,文案告诉用户去设置页武装。
- **单活动执行者**:`run_node` 与 GUI Worker 共用同一 `InputBus`(OS 输入串行)。但两个整体流程交错点击仍乱——v1 加粗粒度 guard:GUI Worker 正在跑容器时 `run_node` 返 `BUSY`(反之 GUI 跑前不被 MCP 占用由 InputBus 自然兜)。plan 定 guard 落点。
- 日志**不打**截图像素 / 不打窗口敏感标题原文超必要;沿用 ② 的脱敏纪律。

## 10. 错误模型(复用节点错误体系)

`run_node` 返回的 `error` 直接复用节点 Fail 出口的 `Code`/`Error`(见 `docs/error-model.md`):

| 情形 | Code |
|---|---|
| 节点走 Fail 出口(运行时故障,如截屏失败) | 节点自带 ErrCode(`capture_failed`/`launch_failed`…) |
| kind 不在放行集 | `UNRUNNABLE_KIND` |
| 句柄查不到 / 窗口已关 | `WINDOW_GONE` |
| params 缺必填 / 类型非法(validator 拦) | `INVALID_PARAMS`(附 ValidationError) |
| 未武装 | `NOT_ARMED` |
| GUI 正在跑容器 | `BUSY` |
| 超时 | `TIMEOUT` |

`run_node` 跑前先对微容器跑一遍 `ValidateContainer` 兜 `INVALID_PARAMS`(复用现有校验),错误级直接返不执行——**但豁免 `MISSING_WIN32_WINDOW_TARGET`**:微容器是 `{Start→节点}`、无 Win32WindowTarget 节点,任何 `NeedsWindow` 节点都会触发该结构校验错;而窗口是经 hwnd/`SetActiveWindow` **带外**提供的,故这条结构检查对 harness 不适用,须显式跳过(`hasBlockingValidationError` 只拦 `Code != CodeMissingWin32WindowTarget` 的错误级)。其余真参数/类型错仍拦。

## 11. 实现触点清单(给 plan 用)

1. `winutil`:新增 `EnumTopWindows() []WindowHandle`(+ 测试)。
2. `ContainerRunner`:加只读访问器 `ExecOutputs() map[string]any`。
3. `internal/services/mcp/`(新包):server 装配 + tool handlers(authoring 工具从 `cmd/yotta-mcp` 搬入复用)。
4. `run_node` harness:合成微容器 → 预置窗口 → `NewContainerRunner.Run` → 收割(可落 `internal/services/mcp` 或 runtime 旁,因碰 runner 内部)。
5. `internal/services/settings.go`:加 `MCPSettings{Armed bool}`(或并入现有结构)+ `defaultSettings` + `Validate`。
6. `main.go`:启动期装配 MCP server(注入常驻标准件)+ goroutine `Start` + 进程关停时优雅停。
7. 前端:设置页加 MCP arm 开关(+ 端口/URL 显示供用户配客户端);i18n `settingsTab.mcp` + `settingsMCP.*`(zh/en)。
8. `cmd/yotta-mcp`:退役(删独立 main,或改造成 bridge——plan 定)。
9. 测试(见 §12)。

## 12. 测试策略

- **`run_node` harness**(核心,纯 Go 不依赖真窗口):用测试 fixture 预注入 mock window + mock input/capture backend(沿用 runner 测试现成范式),断言:
  - 合成微容器跑通、收割到目标节点输出字段;
  - Capture 节点 → 拿到 `Image`,编码成图像块;
  - DetectColor `Found`/`NotFound` 两分支 → `firedOutput` 反推正确、稀疏字段处理对;
  - 节点走 Fail → `error` 带正确 Code;
  - `UNRUNNABLE_KIND`(传 Loop/GetVar)、`INVALID_PARAMS`(缺必填)、`NOT_ARMED`、`BUSY`。
- **`winutil.EnumTopWindows`**:轻量(枚举当前进程窗口非空 + 字段完整);ResolveWindow 已有测试。
- **arm 闸**:受闸工具未武装返 `NOT_ARMED`、只读工具不受闸。
- **save_container**:落 GUI store 后能被 `Store.Get` 读回(整合后 store 换绑验证)。
- **catalog 闸**:`NeedsWindow`/`IsPureData` 过滤的放行/拒绝集符合预期。
- HTTP server 装配:能起、能列工具、优雅关停(plan 按实测库 API 定断言)。
- 验证基线照 `flightdeck/knowledge/build/build.md`；当前 Go/前端测试应绿，旧预存红记录已过期。

## 13. 风险 / plan 待验

- **mark3labs/mcp-go HTTP API 形态**(§8):库版本里 `NewStreamableHTTPServer`/`NewSSEServer` 确切签名 + Wails 进程衔接——实测,不脑补。
- **HWND 复用**(§7):无状态设计下已用 `WindowMetadata` 重校验缓解,但极端竞态(查后即关)仍可能;接受为已知权衡(AI 重试 `find_window`)。
- **input/capture backend 每调用建/弃**:AI 慢节奏下开销可忽略(§2 决策);若 plan 实测某 backend(WGC)建销成本高到影响体验,再考虑 §非目标 的有状态会话——但不预建。
- **`postmessage` 后台输入对部分窗口无效**(已知:某些游戏只认 sendinput/前台)。`run_node` 可暴露可选 backend 参数(默认 postmessage),AI/用户按目标选。plan 定是否暴露。
- **单活动执行者 guard**(§9)粒度:v1 用 `BUSY` 粗挡;细化(排队/抢占)YAGNI。
- HTTP server 绑 localhost 的**安全面**:本机任意进程都能连。单机桌面工具 + arm 开关默认关 → 接受;不做 token 鉴权(YAGNI,可记低风险)。
- bindings 是 gitignore 产物,真机验证须重启后端。

## 14. 与 ① / ② 的衔接

- **不碰 ①②**:run_node 跑的是已有动作节点,不涉 AI 连接 / AI 节点本身。AI 节点(②)是「图里调 LLM」,与本 spec 的「外部 AI 调我们」正交。
- **save_container 复用**:AI 攒图存的是标准 container JSON,与 GUI 编辑器、② 的 AI 节点完全同构——AI 可以在生成的图里放 ② 的 AI 节点(让脚本自己也调 LLM),两个方向自然组合。
- **held-output 是 ②epic 留下的接力棒**:`run_node` 的输出收割直接吃 `execOutputs` 缓存(`docs/held-exec-outputs.md`),无本 spec 专属机制。
