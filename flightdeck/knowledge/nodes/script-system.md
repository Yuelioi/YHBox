# Script 系统 — 内嵌 JS 脚本节点与节点即函数绑定

SUMMARY: Script 节点（goja JS）+ 节点即函数自动绑定 —— 调用/返回/错误约定、$变量、糖函数、动态输入、前端编辑器链路
READ WHEN: 写/改 Script 节点或绑定层前; 想知道脚本里能调什么函数、怎么传参取值接错误; 加新节点想确认脚本侧是否自动可见; 改 Expr/Script 动态输入机制前; 撞 SCRIPT_* 校验码或脚本取消/行号问题
RECHECK WHEN: 改绑定调用/返回/错误约定、糖函数集合、ScriptBindable 排除规则、IIFE 包裹/行号修正、code widget 组件链路或 DynamicInputs 机制时

---

一句话: **Script 节点让用户用 JavaScript (goja 引擎) 写一段逻辑, 每个已注册的 Runnable/Evaluator 节点自动是脚本里的同名全局函数** — 加新节点零绑定维护, 脚本侧自动多一个函数 + 补全自动多一项。

## 节点形态 (`internal/nodes/script/script.go`)

- Kind `Script`, Category `Control`, **Runnable**。
- 输入: `In` (Exec) / `Code` (String, widget `code`) / `Window` (可选窗口覆盖输入)。
- 输出: `Done` (Data 字段 `Result` `*` = 脚本 `return` 的值) / `Fail` (error 语义, Error+Code)。
- `NeedsWindow: true`, `NeedsForeground: true` — 含 Script 的容器必须有窗口上下文；sendinput 后端派发前会按可选 Window 输入补前台。
- `DynamicInputs: true` — 动态输入 pin, 见下。
- 返回值消费: `Result` 是 Done 出口 Data 字段，可用 held-output 数据线直连下游；需要命名复用时在 Inspector 输出组把 `Result` 绑到变量，再用 GetVar 读。

## 脚本怎么写 (用户面约定)

- **顶层 `return` 返回结果** (实现: 源码包 IIFE `(function(){…})()`, 行号偏移已在报错侧减回, `internal/services/script/compile.go`)。
- **同步模型**: 无 async/await; 调阻塞节点 (WaitTemplate 等) 就是阻塞等, 和图里跑一样。
- ES2017 级语法 (let/const/箭头/解构/模板字符串)。

## 节点即函数 (绑定层 `internal/services/script/binding.go`)

- **哪些节点可调**: `node.ScriptBindable(rn)` (`internal/node/registry.go`) 是唯一判定源 — Runnable 或 Evaluator, 排除 RegionRunner (Loop/ForEach — 脚本有原生循环; Subgraph/CollapsedNode 不走自动绑定, 子图调用是下述定制函数)、IsGraphMarker、IsVisualOnly、Script 自身 (防递归)。前端补全 RPC `GetScriptBindableKinds` 走同一函数。
- **调用约定**: 单对象参数, 键 = pin 名 (PascalCase); 省略键用 Spec 默认值; 缺必填抛异常。`ClickAt({XRatio: 0.5, YRatio: 0.5})`。
- **返回约定**: exec 节点 → `{exit: "出口名", ...出口Data字段}` (如 `r.exit === Exit.Found`, `r.Point`); PureFunc (Evaluator) → 直接返回求值结果。脚本内置 `Exit.*` 标准出口常量，避免手写 `Done/Found/NotFound/Timeout/True/False` 等字符串；`Exit.Default` 的值是 Switch 保留出口 `"default"`。
- **错误约定**: 节点失败 = JS 异常, 异常对象带 `code` / `message` / `kind` 字段, `try/catch` 可接 (超时重试等); 未接住 → Script 走 `Fail` 出口, **透传原节点错误码**; 脚本自己 `throw` → 码 `thrown`。
- 实现走框架公开入口 `node.RunNode` / `node.EvaluatePureData` (自带 Required/Validate 门 + 正确 Spec 的子 ctx + recover); RunResult.Panic 原样 re-panic, 不许被脚本 catch。
- 数字归一: goja Export 整数是 int64, 绑定层 `NormalizeJS` 递归转 float64 (Inputs coercion 只认 float64/json.Number/string)。

## 脚本调子图 — `Subgraph({SubgraphID, ...入参})` (2026-06-11)

- **绑定形态**: 不是节点自动绑定, 是 `installSubgraphCall` (`binding.go`) 注入的定制函数, 走 runner 注入的 `node.SubgraphCaller` 服务 (ServiceBundle.Subgraphs = ContainerRunner 自身; 非 runner 语境如单节点测试为 nil → 函数不装, 调用得 ReferenceError)。
- **语义**: 同步跑完容器内 SubgraphID 指定的子图才返回, 与 Subgraph 节点同核心 (`runtime/subgraph_call.go::runSubgraphCall` — push frame + 切表 + 跑到出口 marker)。除 `SubgraphID` 外的字段直接 seed 为子图入参 (声明入参缺省补 Default)。
- **返回**: `{exit: "<出口名>"}` — 出口 decl 的 **Name** (人读名, 如 "done"/"failed"; 图层路由用的 decl ID 脚本不可见)。这是用户自定义出口名，不自动映射 `Exit.*`。
- **错误**: 同节点函数约定 (JS 异常带 code, try/catch 可接): 子图不存在 `not_found`; 多出口跑干 `subgraph_no_exit`; 嵌套超 32 层 `subgraph_recursion` (脚本动态递归的运行时兜底, 图层静态防环拦不住它)。容器停止由 Script watchdog Interrupt 兜底打断, 子图内阻塞节点用同一 ctx 原生取消。
- **依赖提取**: `AssetDeps` 按调用形态正则抽 `Subgraph({SubgraphID: "<字面量>"})` → `Dependency{Kind:"subgraph"}` (分享导出闭包 / 删除 referrer 警告); 动态拼接的 ID 扫不到 — 约定 SubgraphID 用字面量。
- **前端**: 补全/参考面板/hover 走 `SUGAR_ITEMS` 单源 (`scriptCompletions.ts`, snippet 占位 SubgraphID); `Subgraph({SubgraphID: ▮})` 值位按 `Semantic:"SubgraphID"` 通用补全容器内可见子图 (CodeInput.pinValues, 显示子图名); 文案 `script.fn.Subgraph`。

## 糖函数 / 常量

标准出口常量：`Exit.Done` / `Exit.Fail` / `Exit.Found` / `Exit.NotFound` / `Exit.Timeout` / `Exit.True` / `Exit.False` / `Exit.Body` / `Exit.Changed` / `Exit.Gone` / `Exit.Out` / `Exit.Stable` / `Exit.Default`。这些常量只是脚本人体工学糖，节点真实出口名仍来自 Spec；`Switch.default` 仍保持小写出口名。

糖函数 (三组, 高频到值得短名字):

`params.get(name)` · `sleep(ms)` (可取消) · `log.debug/info/warn(...)`。

> **变量读写不走糖** (2026-06-11 用户拍板删 `vars.*`): 读用 `$hp` (live getter) 或 `GetVar({VarName})` 节点函数; 写/自增用 `SetVar`/`IncVar` 节点函数。理由: 节点函数已完整覆盖变量操作, `vars.*` 是第三套重叠 API (与 `$hp` + 节点形式), 冗余。VarName/Scope 等 pin 在值位置有补全 (见前端编辑器链路)。变量读的是 **live 值** (Snapshot 不包, 脚本是 exec 语境)。

## $变量引用 (2026-06-11)

脚本里 `$hp` = 读容器变量 `hp` 的 live 值 — Run 起跑时给每个已知变量注入 live accessor getter (`installVarGetters`, goja DefineAccessorProperty; 变量名枚举走可选能力接口 `varNamer.Names()`, 生产 varStoreAdapter/测试 stub 实现, **不进 node.VarStore 主接口**免测试 fake 陪绑)。访问时实时读 (脚本中途 set 后 `$hp` 是新值)。运行中动态新建的变量没有 getter (注入集起跑时固定) — 用 `GetVar({VarName: "…"})`。引用未声明变量 → JS ReferenceError → Fail 出口 (JS 太动态, 不做静态查)。`$hp` 只读; 写变量走 `SetVar`/`IncVar` 节点函数。

## 取消

watchdog goroutine 监听 `ctx.Context().Done()` → `vm.Interrupt()`: 停容器时哪怕脚本在纯 JS 死循环也立即打断, Run 返 `ctx.Err()` (graceful halt, 同 Sleep/KeyPress)。被调的阻塞节点用同一 ctx, 原生取消。

## 编辑期校验 (`internal/services/container/validator_script.go`)

- `SCRIPT_PARSE_ERROR` — goja 编译查语法, 报错带行号 (已修正 IIFE 偏移)。
- `SCRIPT_DUPLICATE_INPUT` — 动态输入重名。
- 运行时语法错 = 裸 error 冒泡中断 (配置错, 同 Expr parse 模式), 不走 Fail。

## 动态输入机制 (Expr/Script 共用)

- `Spec.DynamicInputs: true` 节点的额外输入由 `config.Inputs[]` 声明 (PascalCase `{Name, Type, Var?, Scope?}`), 后端统一走 `container.ParseDynamicInputDecls`; dispatch (`buildDataWireFor`) / validator (`dataInPinTypeForNode` / unknown-literal 跳过) / FE (`adapter.ts parseDynamicInputsCfg` / `pinLiterals.ts`) 全部标志驱动, **无 kind 字符串特判**。
- 输入口 = **连线 data-in pin** (接别的节点输出/字面量); 想用变量直接写 `$名` (见上节), 不必声明输入口。
- 声明编辑 UI: Inspector「输入口」区 (`DynamicInputsEditor.vue`) — 名字合法性/重名标红 + 底部一句话提示 `$名` 用法。
- 连线值在脚本里是**同名只读全局变量** (Expr 里是表达式标识符)。卡片 footer 自动列脚本/表达式里的 `$` 引用 (正则提取)。

## 前端编辑器链路

- widget kind `code` → `PinInput.vue` 分支 → `CodeInput.vue` (CodeMirror 6 + `@codemirror/lang-javascript`, 骨架同 ExprInput) + 右上「放大编辑」按钮 → `EditorModal.vue` (Expr/Script 共用的语言无关壳, draft 语义确认才回写): 分组工具栏 (撤销重做 | 注释·查找替换暗色中文面板 | 片段下拉 if·for·while·try·调用方 `#toolbar-extra` 槽 | 右侧 整理缩进·全部折叠/展开) + 右侧可搜索参考面板 (节点按**画布同款分类分组配色** `nodeGroup.*`+`groupLabelColor` 圆点组头带计数, 行点击展开用法: description/参数表带 i18n label/example; 行尾按钮插入) + 状态栏 (语法状态 ✓/首错**可点击跳行** + 光标行列 + 行·字符数 + 语言标签) + header 全屏切换 (BaseModal `tall`/`contentClass`) + Ctrl+Enter 确认。**节点插入是 snippet 占位** (`Kind({Pin: ${Pin}})`, 非 advanced pin 铺开, Tab 逐格填值), 补全 Tab 上屏同模板; 参考面板与补全下拉共用插入项单源 (`scriptCompletions.ts` 的 `InsertItem.snippet/insert`)。
- **外观/手感单源 `lib/editorTheme.ts`** (三处编辑器共用): VSCode Dark+ 成套主题 (全覆盖 HighlightStyle + chrome: 编辑面/当前行/选区/选中词同款/括号配对/gutter/补全 tooltip/查找面板 —— **chrome 底色已另由配色统一改读 `var(--ui-*)` app token, 不再是写死的 #1e1e1e**; **$变量 例外保持橙色徽标** `.cm-yh-dollar`) + `baseEditorExtensions({modal,minHeight})` 基础件 (自动配对/括号高亮/indentOnInput/Tab 缩进/选中词高亮/多选区 Mod-D/连字关闭; modal 档加行号/当前行/lint gutter — **不挂 `scrollPastEnd`**, 它给几行代码也垫一屏虚拟空白常驻滚动条)。字体 JetBrains Mono Variable 本地打包 (@fontsource-variable, style.css)。
- **signature help + 类型色点** (`lib/editorSignature.ts`, editor-ux-v2 加): 光标落在调用括号内时浮层显示函数签名 + 高亮当前参数 (Script 走 lezer 语法树定位调用上下文, 区别于 Expr 的字符串扫描); 签名/info/hover 浮层里参数类型用色块标 (`.cm-yh-doc-param-type`, required 带 `*`)。与 hover 文档互补: hover=停在词上看, signature help=打字进括号跟参数。
- **pin 值位置补全** (`scriptPinValueContext` in `lib/editorSignature.ts`): 光标落在 `Kind({Pin: ▮})` 的 pin 值位 (lezer 树 Property→ObjectExpression→ArgList→CallExpression 定位), 按 pin 给候选 — **枚举 pin** (dropdown, 读 `widget.props.options`) → 候选值 (GetVar 的 Scope → auto/local/global, 同时在签名/hover/参考面板参数行列出, `HoverDoc.params.options` / `.cm-yh-doc-param-enum`); **varname pin** (`Semantic:"varname"`, 如 GetVar/SetVar/IncVar/VarLastChange 的 VarName) → 容器变量名。串内补裸值、裸位补带引号。通用机制不写死 kind。**仅 Script** (Expr 函数无 pin/widget 概念)。
- **变量直达**: `$名` 补全列容器变量名 (读); `GetVar/SetVar({VarName: "▮"})` 的 VarName 值位也补变量名 (见上); 参考面板「变量」组置顶 (点击插 `$name`); 工具栏右侧「新建变量」按钮复用 NewVarModal, 声明走 Inspector 的 `declare-var` 事件链 (PinInput → NodeInspector → useVarMutations.addVar), 建完顺手插一句 `$name`。
- 补全 = 节点函数 (registry store 的 Spec 推导签名, `frontend/src/lib/scriptCompletions.ts`) + 三组糖 (params.get/sleep/log) + 本节点动态输入名。**快速反馈 lint** (`scriptSyntaxErrors` lezer error 节点按行去重 + `scriptDollarRefs` 未声明 `$名` warning, 纯函数可单测) + **悬停文档** (`lib/editorHover.ts` 通用浮层, 节点/糖/Expr 函数共用) — 权威仍是后端 validator (SCRIPT_PARSE_ERROR), 同 Expr 先例。$变量徽标装饰走语法树 (字符串/注释不命中)。
- 画布内联框排除 code widget (8 行代码只在 Inspector/modal 编辑)。

## 子图一键转脚本 (2026-06-12)

编辑器里子图可转成等价脚本 (Subgraph/CollapsedNode 节点右键「转为脚本」/ 子图编辑属性面板按钮 → 预览 modal → 复制或插入为 Script 节点)。转换器是前端纯函数 `frontend/src/lib/subgraphToScript.ts`。当前支持 `Expr` 内联、`Loop count/forever` → JS `for/while`、`Break/Continue`、以及 exec 分支汇合尾部复制；标准节点出口比较生成 `Exit.*`，但 Subgraph/CollapsedNode 的用户出口名仍生成字符串；纯数据节点按引用点内联, 避免跨 JS block 生成失效 `const`。仍拒转成环、Fail/error 出口接线、disabled 节点、同一 exec 出口 fan-out、多输出纯数据节点、跨分支 data 引用等无法安全等价的结构。新节点注册后只要 ScriptBindable 且 spec 完整, 转换器通常自动可转。历史设计材料在 cold archive `2026-06-12-subgraph-to-script`;本知识不依赖它。

## 加新节点时脚本侧要做什么

**什么都不用做。** 注册即绑定 (ScriptBindable), 补全经 RPC 自动出现; 只要按 add-node checklist 把 i18n label 写了, 补全 detail 自动带中文名。

## 设计取舍记录 (简)

- 语言选 JS/goja 而非 Lua/starlark: 纯 Go + Interrupt 可打断 + 语法与 expr 同体系 + CodeMirror 官方包 + 默认无 IO (沙箱)。
- 绑定走「节点即函数自动绑定」而非手写精选 API (用户 2026-06-10 拍板): 零维护、覆盖面随注册表增长; 没有节点替身或 `$` 捷径的高频项用糖弥补人体工学 (现仅 params.get/sleep/log)。
- **删 `vars.*` 糖 (2026-06-11 用户拍板)**: 变量在脚本里曾有三套写法 — `$hp`(读) / `vars.get/set/inc` / `GetVar/SetVar/IncVar` 节点函数。节点函数已完整覆盖且 VarName/Scope pin 值位有补全, `vars.*` 是冗余的第三套, 删之 (单源 `SUGAR_ITEMS`, 连带清死 i18n `script.fn.vars_*`)。读用 `$hp`/GetVar, 写用 SetVar/IncVar。2026-06-12 复查发现后端 `binding.go` 的 vars 对象当时漏删, 用户拍板补删干净 (含 `scopeArg` 死 helper; 有 ReferenceError 回归测试钉死)。
- 编辑器自动建 GetVar / 恢复 $vars 语法等方案的淘汰理由见变量系统议题 (cockpit 在册)。
