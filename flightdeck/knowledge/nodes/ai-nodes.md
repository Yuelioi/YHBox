---
kind: note
summary: "AI 节点怎么在图里调 LLM —— 动态带类型 IO、结构化输出、vision 识图、Image 节点与 Provider 缓存"
activation: action
read_when: "改/排查 AI 节点（图里调 LLM）、结构化类型输出、vision 识图、Image 节点（Capture/SaveImage/LoadImage）、Provider 缓存前; 加 outputData role 动态带类型输出机制时; 撞「AI 输出字段绑不上变量 / 结构化解析 Fail / 图像输入没喂进去 / 改连接后没生效」"
recheck_when: "加结构化输出模式/类型 / 改 ChatStructured 协议映射 / 改 Provider 缓存失效条件 / 加图像节点或改 node.Image 形态 / 改 DynamicPorts outputData 机制 / AI 节点 Spec 增删 pin 时"
---
# AI 节点 — 图里调 LLM（动态带类型 IO + 结构化输出 + vision）
"AI 功能" epic 的第②块：让图作者拖一个 AI 节点、选连接+模型、给提示词与任意个带类型输入、拿任意个带类型输出、识图。基础设施①已落地：本地 AI 配置、`llm.Provider` 双协议与连接池；③ MCP 对外暴露另起。历史设计材料在 cold archive `2026-06-23-local-ai-config`;本知识不依赖它。

## AI 节点（Kind `AI`，Category `AI`，`NeedsWindow:false`）

实现 `internal/nodes/ai/`（`ai.go` Run + `structured.go` + `template.go`）。

- **静态输入**：`Connection`（connectionID，空=运行时取 `ai.default`）、`Model`（combobox，FE 按需拉 `ListModels` 建议+手填）、`System`/`User`（提示词模板，User 必填）、`Mode`（`auto`/`native`/`prompt` 结构化模式，默认 auto）、`Temperature`、`MaxTokens`。
- **动态输入**（`DynamicPorts` 的 input/nameTypeRecords descriptor，`config.Inputs[]` `{Name,Type,Var?,Scope?}`）：提示词用 `{{Name}}` 双花括号插值（字面 `{{` 写 `\{{`；引用名**须在 config.Inputs[] 声明，未声明→硬错**）。`Image` 类型输入不进文本插值，改作多模态块，`{{imageInput}}` 渲染成 `[image]` 标记。
- **动态输出**（`DynamicPorts` 的 outputData/nameTypeRecords descriptor，`config.Outputs[]` `{Name,Type}`，parentOutput=`Done`）：见下。
- **输出**：`Done`（Data = `Text` 模型原文**恒有** + 各声明字段）；`Fail`（`Semantic:"error"`，Data = `Error`/`Code`/**`Text` 原文**——下游接 Fail 仍能消费模型原文）。
- **输入名与输出字段名是两个独立命名空间**（`{{}}` 只查 `config.Inputs[]`）。

## 框架机制：动态带类型输出

与动态输入对称的 config 驱动机制（`internal/node/spec.go`，通用、不写死 AI kind）：

- `Spec.DynamicPorts` 以 `role=outputData`、`shape=nameTypeRecords`、`configKey=Outputs`、`parentOutput=Done` 表达完整机器契约；它进入 Catalog identity，不再靠 bool 暗示另一套 parser。
- config 形态 `config.Outputs[]`，后端 `container.ParseDynamicOutputDecls`（镜像 `ParseDynamicInputDecls`）。
- 可绑字段 config 化：`nodepkg.BindableFieldsForNode(spec, config)` = 静态 `BindableFields(spec)` ∪ `config.Outputs[]` 各 Name（去重）；保留名 `Text` 不可声明。捕获校验 + FE「输出」组用它。
- **运行时零特殊**：`applyCaptures` 本就按 `config.capture[field]` 写出口实际带的字段；AI Run 把各声明字段 `.Set()` 即被捕获。各字段也进 held output 缓存可被下游数据线直连（见 [held-exec-outputs](held-exec-outputs.md)）。

## 结构化输出（`llm.ChatStructured`，三模式）

`Mode` 三态，config 存字符串、默认 `auto`：`auto` Run 时按端点定（Anthropic→native tool-use、OpenAI 官方→native json_schema、OpenAI 兼容→prompt 注入）；用户可显式锁 `native`/`prompt`。

- **OpenAI native**：`response_format=json_schema` strict（`additionalProperties:false` + 全 required）。
- **Anthropic native**：强制 `tool_choice` tool-use，取首个 `tool_use.input`；零 tool_use→error。
- **prompt 注入**：schema 描述进 System + 容错解析（剥围栏/取首个 JSON 对象）。
- native 被端点拒（兼容端点显式选 native）→ `ErrKind=unsupported`（文案引导切 prompt）。
- **解析失败走 `Fail`**，`Fail.Text` 带原文（`ChatStructured` 契约：失败时 `error≠nil` 且 `resp.Text` 仍填原文）。

**类型 → JSON → coerce 映射**（单一真相，`internal/services/llm`）：`String`/`Number`/`Bool` 原样；`Integer` 整值校验取整（非整/超界/JSON string → Fail badRequest，`1.0` 视整）；`JSON`→`map[string]any`；`List`→`[]any`。**节点 Type 是 UI 标签**，运行时值就是 Go 的 `float64/string/bool/map/slice`，无 `node.Number` 中间类型；下游 `coerceToType` 按消费 pin 再转。嵌套/递归 schema YAGNI（先扁平标量 + JSON + List）。

## Image 一等值 + 三个图像节点（Category `Image`）

`node.Image{Format string; Data []byte}`（`internal/node/types.go`，format ∈ png/jpeg；**不可变契约**：进程内按引用流动不深拷，消费方禁改底层字节）。`types.go` 的 `Image` GoType 改 `node.Image`（底层 capture/vision 的 `*image.RGBA` 帧代码不动）；`coerceToType` 加 `Image` 分支。

| 节点 | 输入 | 输出 | NeedsWindow |
|---|---|---|---|
| `Capture` | `ROI`(Geometry,零值=全帧)、`Format`(png/jpeg)、`Quality`(1–100,jpeg) | `Image` | true |
| `SaveImage` | `Image`、`PathTemplate` | `Path` | false |
| `LoadImage` | `Path` | `Image` | false |

`internal/nodes/image/`（**Screenshot 已拆删切净**）。`Capture` PNG 直出 / jpeg 转码（PNG→decode→JPEG，记已知低效）。`SaveImage` 沙箱根 `<dataDir>/images/`，扩展名以 `Image.Format` 为准（覆盖模板），模板支持 `{ts}`/`{date}`/`{uuid}`。`LoadImage` 读 `<dataDir>/` 任意子目录（比 Save 宽，设计意图），仅 PNG/JPEG sniff、上限 10MB。

**vision 接线**：`Capture.Done → AI.In`（exec）+ `Capture.Image → AI 的某 Image 动态输入`（数据线）。图像值经 **held output 缓存任意距离直连**进 AI 的动态 Image 输入，无需变量、无紧邻约束（见 [held-exec-outputs](held-exec-outputs.md)；原 spec 的「紧邻 exec + applyExecDataEdges + warn」已被 held output 取代）。多图按 `config.Inputs[]` 声明序置文本 content 之后。

## Provider 缓存 / 失效 / 池（`AIProviderService` + `ctx.Services().AI`）

运行时层服务（`internal/node/interfaces.go` + `ctx.go`，实现在 `internal/services/container/runtime`）：`Provider(connectionID)` 持 `map[connectionID]{provider, cfgHash}`，**单一机制 = 取用时指纹自愈**——`cfgHash = hash(Protocol+BaseURL+APIKey)`（**只 llm 字段，排除 Label/ID**，改名不 evict），指纹不符则 `llm.New` 重建、旧 `Transport.CloseIdleConnections()` 防泄漏。无事件监听、无单独 evict。图执行中途改 settings：正在 `Chat` 的用旧连接跑完，下次取用才换新。池设 `MaxConnsPerHost`/`IdleConnTimeout`。AI 节点经 `ctx.Services().AI.Provider(id)` 取，不自己 `llm.New`。观测日志打 model/Host/latency/ErrKind，**不打 APIKey**。

## 删连接的节点引用检查

settings 删连接时扫含 `Connection==id` 的 AI 节点 → 确认弹窗列出节点（不静默、不硬拦）；删后引用节点运行时 Fail(notFound)，但用户已知情。
