# ⚠ wails dev fetch transport 把错误信封拍平进 Error.message, 读不到 cause

SUMMARY: dev fetch transport 把 wails 错误信封塞进 Error.message 字符串, e.cause 是 undefined 读不到结构化字段
READ WHEN: 设计"FE 读 wails RPC 抛错的结构化字段 (cause / errors / code)"类方案前; 或撞 toast / 错误显示糊出整坨 wails JSON 信封 (`{"message":...,"cause":...,"kind":"RuntimeError"}`)

---

## 症状

错误 i18n 体系落地后手动 smoke 第一下就崩: 保存一个缺 WindowTarget 的容器, toast **糊出整坨原始 JSON** 而非本地化文案。历史材料在 cold archive `2026-06-02-error-i18n-catalog`;本 trap 不依赖它。

```
操作失败
{"message":"MISSING_WINDOW_TARGET map[]","cause":{"Errors":[{"severity":"error","code":"MISSING_WINDOW_TARGET","graphPath":["main"]}]},"kind":"RuntimeError"}
```

`errorMessage(e)` 应返「主图缺 WindowTarget 节点」, 实际返了整坨 JSON 字符串。

## 根因

spec 的通道 A 设计假设: wails 把 Go error 包成 `{message, cause, kind}`, **`cause` 是抛出对象的可访问属性** → FE 读 `e.cause.Errors[]` 走 i18n, 后端零改动。

**实际**: dev 模式用 default HTTP fetch transport (`frontend/node_modules/@wailsio/runtime/dist/runtime.js:103`):
```js
if (!response.ok) { throw new Error(await response.text()); }
```
整个响应体 (那坨 `{message,cause,kind}` JSON) 被塞进一个**普通 `Error` 的 `.message` 字符串**, `e.cause` **是 undefined** —— 结构化数据被困在 message 字符串里。`normalizeError` 读不到 `cause.Errors`, 回落到 `typeof e === 'string'`/`obj.message` 分支, 把整坨 JSON 当人类可读 message 返出去。

`RuntimeError`(带 `.cause`)那条 doc 注释只在原生 transport 成立; dev fetch transport 根本没走到构造 `RuntimeError`。

## 修法

`normalizeError` 读 `.cause` 前, 先把 `e`(string) 或 `e.message` 里的 JSON 信封 `JSON.parse` 解出来当错误对象 (`frontend/src/lib/invoke.ts`, commit `6575d7b`)。原生 transport 直接给对象时是 no-op → transport-agnostic。覆盖全 app 所有通道 A 错误, 非仅保存路径。

通道 B(worker 事件)走 `app.Emit` 事件通道, `d.Error` 本就是解析好的对象, **不受此 bug 影响** —— 所以通道 B 手动 smoke 跳过, 靠 Go 单测背书。

## 教训

- **头号铁律实证**: 用任何 framework API 的返回/错误 marshal 形态, 必须读实现 (这里是 `@wailsio/runtime` 的 transport 源码), 别假设"wails 会给我结构化 `cause`"。多 reviewer approve 也救不了对着幻觉的 marshal 形态造方案。
- **"实施前置 gate / probe" 不能 defer 到 smoke**: spec/plan 明写了 Task 2「加临时 probe RPC 真跑、肉眼确认 `e.cause.code` 真实字段形态, 再写 normalizeError」, 但实际把它 defer 到手动 smoke 才验 → 基于未验证假设写了 normalizeError + 16 站点, smoke 第一下崩。验证步骤是 blocker 就当 blocker 跑, 别挪到最后。
- 同族坑: [[wails-error-only-rpc-invoke-undefined]] —— 都是 wails RPC 错误/返回形态被 invoke() 抹平后 FE 误判。
