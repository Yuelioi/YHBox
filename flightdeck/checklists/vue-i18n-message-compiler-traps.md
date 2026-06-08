---
status: active
last_updated: 2026-06-04
when_to_read: 写 / 改 zh.ts / en.ts 翻译 (尤其含 `{` `}` `|` `@` `$` / JSON literal / 管道符 / 邮箱样例); 派 subagent 批量写 i18n 前; 改 vite.config 的 unplugin-vue-i18n 配置; UI 显示 `{n}` 字面 placeholder 没替换 / 弹 SyntaxError 整个组件挂掉
applies_to: [frontend, vue, i18n, vue-i18n, vite, unplugin-vue-i18n, message-compiler, subagent-dispatch]
when_to_update: 改 vite.config 的 unplugin-vue-i18n 配置 / 升级 vue-i18n / message-compiler 行为变化时
---

# vue-i18n message 特殊字符转义 — Checklist

写 / 改任何 `zh.ts` / `en.ts` 翻译前**前置**读这份。这坑复发过几次 (见文末复发记录)，blast radius 不是一行而是整个组件子树，且 parity / typecheck 都抓不到，跑起来才炸。

## 动笔前必做 (TL;DR)

- [ ] 文案里任何 `{` `}` `|` `@` `$` 一律用 `{'literal'}` escape，**优先 backtick 串** (`` `a {'||'} b` ``)。
- [ ] zh **和** en 都改 (parity 只对 key 集合，不 compile string)。
- [ ] **派 subagent 批量写 i18n 时，dispatch prompt 必须带上这条转义纪律** —— 否则 agent 不知道这些字符是雷，等于把防线绕过去 (Case 2 就是这么炸的)。
- [ ] 改完跑 `pnpm i18n:check` —— 它的 `[compile]` 段会逐条真编译全部 zh+en message，抓没 escape 的特殊字符 (这是堵死复发的自动门)。

## 教训 (3 条)

1. **`unplugin-vue-i18n` 必须显式 `runtimeOnly: false`** — 不设的话默认 `true`, vue-i18n 被 alias 到 runtime-only build (无 message compiler), plain string + `{name}` 这种翻译会**字面渲染 placeholder** (不替换), 没人报错没人提示, 静默坏一片.

2. **打开 compiler 后, 翻译字符串里的 `{` `}` `|` `@` `$` 都是 vue-i18n 特殊字符**, 必须用 escape syntax `{'literal'}` 包. 漏一个 → 要么字面渲染要么 `SyntaxError: Unterminated closing brace` / `Plural must have messages` 抛出, **整个 Vue 组件 mount 时挂掉** (不止那一行文案出错, 整个 component tree 全黑).

3. **写新翻译键时, 包含 JSON literal / 管道符 / `@xxx` 邮箱样例 都要 escape**. 否则 i18n parity check 不抓 (parity 只对 key, 不 compile string)；现已有 `i18n:check` 的 `[compile]` 段兜底, 但仍以动笔时就 escape 为先.

## 为什么

### 1. `unplugin-vue-i18n` 的 runtimeOnly 默认 true

`@intlify/unplugin-vue-i18n/lib/index.mjs:29`:
```js
const runtimeOnly = isBoolean(options.runtimeOnly) ? options.runtimeOnly : true
```

`lib/index.mjs:220-222`:
```js
const getVueI18nAliasPath = ({ ssr = false, runtimeOnly = false }) => {
  return `${module}/dist/${module}${runtimeOnly ? ".runtime" : ""}.${!ssr ? "esm-bundler.js" : "node.mjs"}`
}
```

当 `runtimeOnly:true`, `import 'vue-i18n'` 解析到 `vue-i18n.runtime.esm-bundler.js`. 这个 build 跟全 build 的区别 (`vue-i18n.mjs` vs `vue-i18n.runtime.mjs:2966-2968`):

```js
// full build 有:
} else {
    registerMessageCompiler(compileToFunction)
}
// runtime-only build 这段被砍, 只剩 jit 分支需要外部手动 register
```

后果: `createI18n({ messages: { zh, en } })` 喂 plain string, 没人调 `compileToFunction` 把 `'{n} 节点'` 编译成 `(ctx) => ctx.named('n') + ' 节点'`. vue-i18n 默认行为 → **当成已编译消息函数处理** → 字符串当 message function 失败 fallback → 字面字符串 (placeholder 不被替换).

**症状**: UI 显示 `{n} 节点` `{name}` `{count} 命令` 字面, 切 locale 也是同样字面.

### 2. vue-i18n 特殊字符 + escape syntax

vue-i18n 消息 DSL:

| 字符 | 语义 | 不 escape 后果 |
|---|---|---|
| `{x}` | named placeholder | 内容非合法 identifier → `Invalid token in placeholder` |
| `\|` | plural variant 分隔符 | `a \| b` 当 2-variant plural, 不调 plural api 时只取第一变体 (`"a"`), 静默丢内容 |
| `\|\|` | empty 中间变体 plural | `Plural must have messages` 直接 throw |
| `@:key` | linked message | `@` 后非合法 key → throw |
| `$` | linked modifier prefix (老语法) | 一般 OK 但跟 `${}` 撞容易混 |

escape syntax (vue-i18n 官方):
```
{'literal text'}
```
内容当 raw 渲染. e.g. `{'{'}` 输出 `{`, `{'||'}` 输出 `||`, `{'@'}` 输出 `@`.

**JS 源里推荐 backtick**, 不用单引号 (避免 `\'` 转义满天飞):
```ts
// ✗ 单引号 (要 escape 内嵌 ')
hint: '{\'{\'}\"x\":0{\'}\'}'
// ✓ backtick (内嵌单引号 + 双引号都不用 escape, vue-i18n 的 {} 不是 JS interpolation)
hint: `{'{'}"x":0{'}'}`
```

注意 backtick 的 `${}` 才是 JS 插值, 单纯 `{...}` 在 backtick 里就是字面字符.

### 3. 错误**杀整个 component**, 不是只丢一行

vue-i18n `t()` 在 component setup 期间被调到非法 key → throw SyntaxError → Vue setup 失败 → **整个 component mount 链断**. 表现:

- Tab toggle 节点库 → 节点库不显示 (NodeExplorerModal 挂)
- 右键画布 / drag-out pin → InlineContextMenu 不弹
- Subgraph 节点内部空白 / "(子图未找到)" — 不是真不存在, 是渲染挂

**调试陷阱**: 控制台只看到 `SyntaxError: Unterminated closing brace (at index-XXX.js:NNNN:NN)`, minified 看不出哪行翻译炸. 跨多个看似不相关的 UI 挂掉, 像是 vue-flow 或 router 出问题, 实际全是同一个 i18n key 在不同 component 被 t() 调到.

## 怎么 apply

### 写翻译时

任何含 `{` `}` `|` `@` `$` 的字符串都 escape, 优先用 backtick:

```ts
// ✗
ROI: { hint: '{"x":0,"y":0,"w":100,"h":100} 像素坐标' }
counts: { hint: '硬件累积 |dx|' }
Or: { description: 'a || b' }

// ✓
ROI: { hint: `{'{'}"x":0,"y":0,"w":100,"h":100{'}'} 像素坐标` }
counts: { hint: `硬件累积 {'|'}dx{'|'}` }
Or: { description: `a {'||'} b` }
```

### 派 subagent 批量写 i18n 时

dispatch prompt **必须**带上「文案含 `{ } | @ $` 一律 backtick + `{'literal'}` escape」这条纪律。Case 2 的复发就是因为派了 6 个 agent 写节点说明、但 prompt 没传这条 → agent 写了裸 `||`。

### vite.config.ts 配 unplugin-vue-i18n

强制 `runtimeOnly:false`. 老配置不写 = 默认 true = 静默坏:

```ts
VueI18nPlugin({
  include: [...],
  jitCompilation: false,
  runtimeOnly: false,  // ← 必须显式. 默认 true 会砍 message compiler.
})
```

### 自检 — 已转正进 `i18n:check`

```bash
pnpm i18n:check   # 第三段 [compile] 用 vue-i18n createI18n 逐条真编译全部 zh+en message
```

`[compile] FAIL` 会直接点出哪个 key 炸 (如 `[zh] node.Expr.description :: Plural must have messages`)。parity / typecheck 都抓不到这类，这段是专门补的门。

> 注意: `i18n:check` 目前是手动脚本，未接 git hook / CI (本仓无现成 hook 基建)。要彻底自动拦需另接 precommit / CI。

### 撞 SyntaxError 时的诊断步骤

`SyntaxError: Unterminated closing brace` / `Plural must have messages` / `Invalid token in placeholder` 出现:

1. **先跑 `pnpm i18n:check`** —— `[compile]` 段一次性列出全部坏点 (不要一个个改一个个测).
2. 或手 grep zh.ts/en.ts 的 `{` `}` `|`: 看哪些字符串没 escape (尤其 JSON literal hint, OR/AND 描述, 管道分隔列表).
3. **EN/ZH 都要修** — parity check 不抓 string 内容只对 key 集合.
4. **修完跑 build + 启 dev**, 验之前挂掉的 component 是否恢复. typecheck 不抓这种 (string 内容运行时才 compile).

---

## 复发记录 (为什么这条从 incident 转成 checklist)

### Case 1 — 2026-05-29 placeholder 字面 + SyntaxError 级联

**第一阶段**: 用户报容器 tab 列表 / 编辑器 / Subgraph footer 都看到 `{n} 节点` `{name}` 字面。根因: `vite.config.ts` 没显式设 `runtimeOnly` → 默认 true → runtime-only build 无 compiler → string 字面渲染。Fix: 加 `runtimeOnly: false`。

**第二阶段**: 打开 compiler 后, 之前藏在 runtime-only fallback 里的非法字符串全炸出来。用户报 3 个看似无关的 bug (导入子图内部空 / Tab 节点库不显示 / drag-out pin 不弹菜单), 控制台 `SyntaxError: Unterminated closing brace`。自写 scanner 跑全 1355 leaves 拿出 12 处坏点 (9 处 JSON literal hint、2 处管道符含 `node.Or.description: 'a || b'`、1 处嵌套)。Fix: 12 处换 backtick + escape (zh+en 共 24 编辑)。3 bug 全是同一个 SyntaxError 把不同 component mount 挂掉, 修 i18n 后一起恢复。

**教训**: i18n compile 错误的 blast radius 是整个组件子树。跨多个看似无关 UI 同时挂掉, 别分头查 vue-flow / router / store, 先 grep 翻译里的特殊字符。

### Case 2 — 2026-06-04 批量重写节点说明再次撞 `||`

重写全部 71 节点 description 时, Expr 写了 `与或非 (&& || !)` / `logic (&& || !)`, 那个 `||` 又把 compiler 炸了 — 用户报「Tab 节点 / 节点库菜单灰色」+ `Plural must have messages`。跟 Case 1 第二阶段同根因, 也正是本文「怎么 apply」早点名的 `Or: a || b` 同款。Fix: `&& || !` → `&& {'||'} !`。

**新增根因 = dispatch gap**: 派 6 个 subagent 并行写说明, prompt 给了风格 / 源码纪律, **但没传 escape 纪律** → agent 不知道 `|` 是雷。这次促成两件事转正: (a) dispatch prompt 必带 escape 纪律 (上面已写进 checklist); (b) 全量 compile scan 接进 `pnpm i18n:check` 的 `[compile]` 段, 不再靠肉眼。

参考: vue-i18n v9 文档 [Message syntax — Special characters](https://vue-i18n.intlify.dev/guide/essentials/syntax.html).
