---
kind: note
summary: "历史 3.0 Expr/$变量系统；3.1 已删除，不得作为现行节点或类型逃生口。"
activation: action
read_when: "仅在审查 3.0 Expr 行为、迁移旧用户体验，或评估未来独立 typed expression 设计时"
recheck_when: "加/删/改内置函数或其 arity·签名; 改 expr 语法 (lexer/parser); 改 EXPR_* 校验码; 改 ExprInput 编辑器或函数元数据管道时"
---
# 表达式系统 (Expr)

> 历史知识：现行 3.1 Catalog 没有旧 Expr/$变量执行路径。未来若设计表达式，必须重新经过严格类型、Compiler、sandbox 和显式错误决策，不能复活本文实现。
## 一句话

Expr 是一个 pure-data 节点 (`internal/nodes/purefunc/expr.go`): 用户写一行表达式串, 动态声明若干输入 pin, 求值吐一个 `Result`。语法引擎在 `internal/services/expr` (lexer → parser → AST → eval), 与节点框架解耦。

## 语法速览

- 字面量: 数字 / `"字符串"` / `true` / `false` / `null`。
- 运算: 算术 `+ - * / %`、比较 `== != < <= > >=`、逻辑 `&& || !`、三目 `cond ? a : b`、函数调用 `fn(args)`。
- **`$名字` = 变量引用** (2026-06-11 恢复): `$hp / $max * 100` 直读容器变量, auto scope (local 优先 global 兜底), 快照语义同 GetVar。**单标识符形态** — v3 的 `$vars.X` 点路径不回归 (`.` 处 parse error, 不静默)。
- **裸标识符 = 动态输入引用**: 表达式里写 `hp`, 必须在节点 config 的 `Inputs[]` 里声明同名输入 (输入口 = 连线 data-in pin, 接别的节点输出/字面量; 用变量请直接写 `$名`)。
- AST 按表达式串缓存 (`parseExprCached`), 跨节点共享。

## $ 语法的决策史 (两次反转, 别再脑补第三次)

v4 (2026-05-19) 删 `$vars.X`, 理由三条: 拼错静默 nil / 可见性差 / 类型无校验。**2026-06-11 用户拍板恢复** (`$名字` 新形态), 因为三条理由已被设施补上: ① validator `EXPR_UNKNOWN_VAR` 编辑期红错 (validator_expr.go × `expr.VarRefs`); ② 节点卡片 footer 自动列 `$` 引用小字 (ContainerFlowNode `dollarRefs`, 正则提取, 顶替声明式可见性); ③ VarDecl 类型系统已真。实现: lexer `tkVarRef` / AST `nVarRef` / eval 走 `env.Get("$"+name)` 前缀通道 (bare 与 $ 两命名空间不撞), Expr 节点 `exprEvalEnv` 组合 env (bare→inputs map, $→`ctx.Services().Vars.GetScoped(名, "auto")`, 快照由 EvaluatePureData wrap 自动继承)。Script 侧 `$hp` 是 live getter (见 [script-system.md](script-system.md))。中途曾以"输入声明绑定变量"(路线 A) 落地过一版, **同日被 $ 语法取代并删除**；历史材料在 cold archive `2026-06-11-var-bound-inputs` / `2026-06-11-dollar-var-syntax`。

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

- 语言/补全/lint/悬停文档的 CodeMirror 扩展抽在 `lib/exprEditorExtensions.ts` (纯函数, i18n 经回调注入), 小框和放大编辑共用; 主题与编辑手感 (VSCode Dark+ 成套 + 自动配对/括号/Tab 缩进等基础件) 在 `lib/editorTheme.ts` 的共享层, $变量保持橙色徽标; 内置函数名 token 走 `variableName.function` (黄, 同 Script 节点函数)。
- **signature help** (`lib/editorSignature.ts`, editor-ux-v2 加): 光标落在函数调用括号内时, 浮层显示该函数签名 + 高亮当前参数 (Expr 走字符串扫描找最内层未闭合括号 + argIndex, 跳过双引号串)。与悬停文档区别: hover 是"停在词上才看", signature help 是"打字进括号实时跟着参数走"。
- **类型色点**: 悬停/info/signature 浮层里, 参数行的类型用色块标出 (`renderSignature` / `.cm-yh-doc-param-type` span, required 带 `*`), 跟节点 pin 类型色一致, 一眼分清参数类型。
- 行号: 小框档不挂行号 (省空间), 放大 modal 档挂 `lineNumbers` + 当前行 + lint gutter (`editorTheme.ts` 的 modal 分支; **不挂 `scrollPastEnd`** — 它给三行表达式也垫一屏虚拟空白常驻滚动条)。
- 右上「放大编辑」按钮 → `EditorModal.vue` (Expr/Script 共用壳): 分组工具栏 + 大编辑器 + 右侧可搜索函数参考面板 (签名+说明, 点击插入) + 状态栏 (lint 首错可点击跳转 + 行列/统计/语言标签) + 全屏切换 + Ctrl+Enter 确认; draft 语义, 确认才回写。
- 动态输入名 (config.Inputs[]) 进补全和参考面板 (PinInput 传 `inputNames`)。
