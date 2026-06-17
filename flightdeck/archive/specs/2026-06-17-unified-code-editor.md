---
status: done
summary: 把 Script/Expr/yt 控制台三个编辑器统一到共享 <CodeEditor> 主体(从 EditorModal 抽出: CodeMirror 视图+工具栏+参考面板+状态栏+折叠/键位); 差异收成 per-mode 配置(extensions builder / 补全源 / hover·参考文档 / commentable·foldable)。EditorModal 重构成壳+确认+<CodeEditor>(保行为等价); YtConsoleModal 改 BaseModal+<CodeEditor mode=yt>+运行/输出。新增格式化 prettier 懒加载(JS-only, Expr N/A)。
last_updated: 2026-06-17
---

# 统一代码编辑器 + 格式化 (CodeEditor 抽取)

## 背景 & 目标

产品里有三个代码编辑器, 风格/功能不该差这么多:
- **Script 节点** (`CodeInput.vue` 行内 + 放大用 `EditorModal.vue`) — JS。
- **表达式 Expr** (`ExprInput.vue` 行内 + 放大用 `EditorModal.vue`) — 表达式 DSL。
- **yt 控制台** (`YtConsoleModal.vue`) — JS。

**读源码定论**: Script 和 Expr **已经共享** `EditorModal`(放大编辑器: 工具栏/参考面板/状态栏/折叠/草稿确认) + `baseEditorExtensions`(主题/高亮/键位/括号), 只在一个 **extensions builder** 上分流(`scriptEditorExtensions` JS / `exprEditorExtensions` 表达式)。**yt 控制台是异类** —— 之前没用 `EditorModal`, 自己拿 `BaseModal` + 裸 `EditorView` 搭, 所以没工具栏/参考面板/状态栏/折叠, 显得简陋。

**目标**: 抽出共享主体 `<CodeEditor>`, 三者(放大态)统一用它, 差异只在 per-mode 配置; 并**新增格式化**。用户验收点: 三个编辑器功能/样式一致, 控制台不再简陋。

## 范围

**做**:
- 抽 `<CodeEditor>` 共享主体 (= 现 `EditorModal` 的编辑器主体: CodeMirror 视图 + 工具栏 + 参考面板 + 状态栏 + 折叠/键位 + size/modal 主题)。
- `EditorModal` 重构成 = 模态壳 + 草稿/确认·取消 + `<CodeEditor>`。**行为等价** (Script/Expr 的放大编辑不变)。
- `YtConsoleModal` 重写成 = `BaseModal` + `<CodeEditor mode=yt>` + **运行按钮 + 输出区** (run/output 控制台独有, 包在外面, 不进 CodeEditor)。
- **格式化** (新): `<CodeEditor>` 工具栏加「格式化」按钮, 仅 JS 模式 (Script + yt 控制台) 显示; prettier 标准版**懒加载**。

**不做 (明确)**:
- ~~`yt.nodes.` 嵌套成员补全~~ — **已补 (2026-06-17, 用户验收后追加)**: 控制台换上下文感知补全源 `ytCompletionSource`(yt.→成员 / yt.nodes.·selected.→数组方法 / n.→NodeHandle); `scriptEditorExtensions` 加 `completionSource` 口。Script/Expr 仍用扁平源。
- Script/Expr 的**行内小框** (`CodeInput`/`ExprInput` 非放大态) 不动 — 本次只统一"放大/控制台"这层主体。
- 表达式格式化 — Expr 单行 DSL, N/A, 不显格式化按钮。

## 架构: `<CodeEditor>` 共享主体

新组件 `frontend/src/components/expressions/CodeEditor.vue` (跟 EditorModal 同目录), 封装编辑器主体, **不含**模态壳 / 确认·取消 / 运行·输出 (那些是宿主的事)。

**Props (= 现 EditorModal 透传给编辑器的那些, 收成一处)**:
- `modelValue: string` + `@update:modelValue`
- `extensions: () => Extension[]` — **per-mode 注入** (语言+补全+hover+签名+lint 全在里头)
- `reference?: RefItem[]` — 参考面板项 (per-mode 的"文档")
- `commentable?: boolean` / `foldable?: boolean` — 工具栏开关 (Script/yt: 开; Expr: 关)
- `snippetLang?` / `langLabel?` / `lintFirst?` / `placeholder?` — 同现 EditorModal
- `formattable?: boolean` — 新: 显不显「格式化」按钮 (JS 模式 true)
- 可能需要 `minHeight` / `modal` 透传给 extensions/主题

**per-mode 配置 (差异收敛于此, 这就是"3 份配置")**:

| 配置 | Script | Expr | yt 控制台 |
|---|---|---|---|
| `extensions` builder | `scriptEditorExtensions` | `exprEditorExtensions` | `scriptEditorExtensions`(JS, 复用) |
| 补全源 | 节点函数 + sugar | expr 函数 + 字面量 | `yt.*` / `n.*` |
| hover / `reference` | 节点 specs | expr 函数 | yt API (来自 `ytConsole/completions.ts` 那份 ENTRIES) |
| `commentable`/`foldable` | 开/开 | 关/关 | 开/开 |
| `formattable` | 是 | 否 | 是 |

> 这些 builder/参考/开关都**已存在** (Script/Expr 在用); 控制台的那套已在 `ytConsole/completions.ts`。统一 = 把它们喂给同一个 `<CodeEditor>`。

## 宿主组件

- **`EditorModal.vue`**: 重构为 `模态壳 + 草稿(确认/取消) + <CodeEditor>`。把现在它内部直接 mount 的 EditorView + 工具栏 + 参考面板 + 状态栏代码搬进 `<CodeEditor>`, EditorModal 只留: 打开/草稿/确认回写/取消丢弃 + 套 `<CodeEditor>`。Script/Expr 经 EditorModal 用, **对它们行为零变化**。
- **`YtConsoleModal.vue`**: 重写为 `BaseModal + <CodeEditor mode=yt> + 运行/输出`。删掉现在那段裸 EditorView。运行按钮(Ctrl+Enter) + 输出区(报告/log/错误) 留在 YtConsoleModal, 包在 `<CodeEditor>` 外。控制台从而拿到跟 Script/Expr 一致的工具栏/参考面板/补全/折叠/状态栏。

## 格式化 (prettier 懒加载)

- `<CodeEditor>` 工具栏加「格式化」按钮 (`formattable` 为真时显), 也绑快捷键 (如 Shift+Alt+F, 须避开 WebView DevTools 组合)。
- 点击 → **动态 import** `prettier/standalone` + `prettier/plugins/babel` + `prettier/plugins/estree` (JS 解析) → `prettier.format(code, { parser: 'babel', plugins })` → 把结果 dispatch 回 CodeMirror 文档 (整段替换, 进撤销栈一步)。
- **懒加载**: 只有点格式化才 import prettier, **不进主包** (主包已偏大, 见 build 警告)。新增 `prettier` 依赖 (pnpm)。
- 格式化失败 (语法错 prettier 抛错) → 不动文档, 提示一行 (复用 lint/toast)。
- Expr/其它非 JS 模式: `formattable=false`, 无按钮。

## 测试 & 验证

- Vue 编辑器组件**没法单测** (项目无 `@vue/test-utils`) → 靠 **typecheck + task build + 真机 smoke** (项目惯例)。
- prettier 格式化逻辑 (输入代码 → 格式化后字符串) 若能抽成纯函数 (`formatJs(code): Promise<string>`) 则加 vitest 单测 (mock 或真调 prettier)。
- 已有的执行器/撤销引擎/glue 24 测**不受影响** (本次不动它们)。
- 真机 smoke 清单见 §验收。

## 风险

- **唯一回归点 = `EditorModal` 重构** (Script/Expr 放大编辑器依赖它)。务必**行为等价**: 抽取前完整读 `EditorModal.vue` (工具栏/参考/状态栏/草稿/键位/插入回调), 抽成 `<CodeEditor>` 时一一对应、不改语义。真机验 Script + Expr 放大编辑仍正常 (编辑/补全/参考面板/确认回写/取消丢弃)。

## 验收

- 三个编辑器 (Script 放大 / Expr 放大 / yt 控制台) **工具栏/参考面板/状态栏/补全/折叠 一致**; 控制台不再简陋。
- **格式化**: 在 Script / 控制台里写乱缩进的 JS → 点「格式化」→ 变整齐 (prettier 风格); 一步 Ctrl+Z 可退; Expr 无此按钮。
- **无回归**: Script/Expr 放大编辑器编辑/补全/参考/确认/取消全照旧。
- typecheck / i18n parity / `task build` 全绿; prettier 懒加载、不进主包 (build chunk 里 prettier 单独 chunk)。
