---
status: active
when_to_read: 加/改 Expr 内置函数前; 改表达式语法/parser/求值前; 碰 ExprInput 编辑器或函数补全前; 撞 EXPR_* 校验码不懂含义时
applies_to: [expr, builtins, Expr, ExprInput, exprFunctions, internal/services/expr, internal/nodes/purefunc/expr.go, internal/services/container/validator_expr.go, frontend/src/lib/exprFunctions.ts, frontend/src/components/expressions]
last_updated: 2026-06-11
when_to_update: 加/删/改内置函数或其 arity·签名; 改 expr 语法 (lexer/parser); 改 EXPR_* 校验码; 改 ExprInput 编辑器或函数元数据管道时
---

# 表达式系统 (Expr)

## 一句话

Expr 是一个 pure-data 节点 (`internal/nodes/purefunc/expr.go`): 用户写一行表达式串, 动态声明若干输入 pin, 求值吐一个 `Result`。语法引擎在 `internal/services/expr` (lexer → parser → AST → eval), 与节点框架解耦。

## 语法速览

- 字面量: 数字 / `"字符串"` / `true` / `false` / `null`。
- 运算: 算术 `+ - * / %`、比较 `== != < <= > >=`、逻辑 `&& || !`、三目 `cond ? a : b`、函数调用 `fn(args)`。
- **裸标识符 = 动态输入引用**: 表达式里写 `hp`, 必须在节点 config 的 `Inputs[]` 里声明同名输入 (`$vars.*` 语法 v4 已删, 变量走 GetVar 节点连进来)。
- AST 按表达式串缓存 (`parseExprCached`), 跨节点共享。

## 内置函数 — 单一来源与同步链

**唯一权威: `internal/services/expr/builtins.go` 的 `builtins` 表** — 每项含 `MinArgs/MaxArgs/Sig/impl`。这张表同时驱动:

1. **运行时**: `evalCall` 查表分发, arity gate 集中 (未知名/参数个数错都在这里兜底)。
2. **编辑期校验**: `validator_expr.go` 用 `expr.Builtins()` × `expr.CallRefs(ast)` 出 `EXPR_UNKNOWN_FUNCTION` / `EXPR_FN_ARITY` (节点红错)。
3. **前端补全**: `expr.Functions()` DTO → `NodeService.GetExprFunctions` RPC → FE 启动拉进 `stores/nodeRegistry`, `populateRegistryFromBackend` 再 `setExprFunctions()` 喂 `lib/exprFunctions.ts` 模块级表 → ExprInput 下拉/未知函数即时红错。**FE 没有手写函数表**。

### 加一个新函数 = 两步

1. `builtins.go` 表加一项 (含 `Sig`, 参数名用通用英文)。`builtins_test.go` 的 WANT 表同步 +1 (这是防误删的锁, 不是第二来源)。
2. zh.ts / en.ts 各加 1 条 `expression.fn.<name>.desc` 一行说明 (没配也不炸 — ExprInput 用 `te()` 守门, 下拉只是不显示说明)。

补全下拉 / 拼错红错 / 参数个数校验 / RPC 全自动, 不用碰前端代码。

### 非确定函数注意

`rand()` / `randint()` / `now()` 非确定。**Expr Spec 挂了 `IsNonDeterministic: true`** — per-dispatch eval 缓存只记忆化带此标记的节点 (`runtime/data_pull.go` 的 gate), 保证同一次求值内多路径引用同一 Expr 拿同值 (语义对齐 random 节点包)。加新的非确定函数不需要再动这个标记; 但**删光所有非确定函数时也别摘标记**, 摘了 rand/now 语义回归坑。

## 编辑期校验码 (validator_expr.go)

| 码 | 含义 |
|---|---|
| EXPR_PARSE_ERROR | 表达式语法解析失败 |
| EXPR_UNKNOWN_INPUT | 裸标识符没在 Inputs[] 声明 |
| EXPR_DUPLICATE_INPUT | Inputs[] 重名 |
| EXPR_UNKNOWN_FUNCTION | 函数名不在 builtins 表 (typo) |
| EXPR_FN_ARITY | 参数个数出 [MinArgs, MaxArgs] |

## 前端编辑器 (ExprInput)

Expr 的 Expression 输入 Widget Kind 是 `"expr"` → `PinInput` 分发到 `components/expressions/ExprInput.vue`: 光标处取词补全 (签名+i18n 说明, Tab/Enter 上屏, 光标落括号内) + 即时启发式红错 (括号/引号/裸词/尾运算符/未知函数, 纯函数在 `lib/exprFunctions.ts`)。启发式只是快速反馈, **权威是后端 validator 的节点红错**。画布上的内联编辑 (PinLiteral) 刻意保持裸文本 — 空间小, 不塞下拉。

- 语言/补全/lint 的 CodeMirror 扩展抽在 `lib/exprEditorExtensions.ts` (纯函数, i18n 经回调注入), 小框和放大编辑共用; 补全提示用 VSCode 风格主题 (`lib/editorTheme.ts`)。
- 右上「放大编辑」按钮 → `EditorModal.vue` (Expr/Script 共用壳): 工具栏 (撤销/重做/查找替换) + 大编辑器 + 右侧可搜索函数参考面板 (签名+说明, 点击插入) + 状态栏实时报 lint 首错 + Ctrl+Enter 确认; draft 语义, 确认才回写。
- 动态输入名 (config.Inputs[]) 进补全和参考面板 (PinInput 传 `inputNames`)。
