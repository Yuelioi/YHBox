// internal/nodes/purefunc/expr.go
// Expr — 求值表达式. IsPureData=true, 走 Evaluator 接口.
// dynamic data-in pins 由 config.Inputs[] 声明; dispatch 层 buildDataWireFor 会按
// config.Inputs[].Name 走 pullDataPin 把值塞进 dataWire, framework merge 进 in.
// Expr.Evaluate 用 in.Keys() 遍历这些 dynamic name 构造 expr.InputEnv.
// AST 缓存 (parseExprCached) 跟 expr 字符串绑, 跨节点共享 immutable.
package purefunc

import (
	"fmt"
	"strings"
	"sync"

	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/services/expr"
)

func init() { node.Register(&Expr{}) }

const (
	exprInExpression = "Expression"
	exprOutResult    = "Result"
)

type Expr struct{}

func (Expr) Spec() node.Spec {
	return node.Spec{
		Kind:     "Expr",
		Category: "PureFunc",
		Inputs: []node.InputSpec{
			{Name: exprInExpression, Type: "String", Default: "", Required: true,
				Widget: node.WidgetSpec{Kind: "expr",
					Props: node.MarshalProps(node.TextareaProps{Rows: 3})}},
		},
		Outputs: []node.OutputSpec{
			{Name: exprOutResult, Type: "*"},
		},
		IsPureData: true,
		// dynamic data-in pins 由 config.Inputs[] 声明 — dispatch/validator 走
		// ParseDynamicInputDecls 解析 (见包注释).
		DynamicInputs: true,
		// rand()/now() 非确定: 挂标记让 per-dispatch eval cache 记忆化,
		// 同一次求值内多路径引用同一 Expr 拿同值 (与 random 节点包同语义).
		IsNonDeterministic: true,
	}
}

// Evaluate — pure-data 求值入口. 走 framework EvaluatePureData 路径.
// dynamic inputs (config.Inputs[].Name) 通过 in.Raw(name) 读, framework 已 merge.
//
// env 隔离: 跳过本节点 Spec 静态 input ("Expression") 跟 config 元数据 ("OutType",
// "Inputs"). 防止用户 expression 误引用 framework config 字段为 identifier
// (框架路径走 in.Keys() 会带上 config 字段, 需显式 skip).
func (Expr) Evaluate(ctx node.Ctx, in node.Inputs) (any, error) {
	src := in.String(exprInExpression)
	if src == "" {
		// Required gate 应已捕获; 这里 defensive (e.g. config 给空串绕过 Has).
		return nil, nil
	}
	env := make(expr.InputEnv)
	for _, k := range in.Keys() {
		if _, skip := exprEnvSkipKeys[k]; skip {
			continue
		}
		env[k] = in.Raw(k)
	}
	ast, err := parseExprCached(src)
	if err != nil {
		return nil, fmt.Errorf("Expr parse %q: %w", src, err)
	}
	return expr.Eval(ast, exprEvalEnv{inputs: env, vars: ctx.Vars()})
}

// exprEvalEnv — Expr 求值环境: bare 名查 inputs map; $名 直读变量 (auto scope).
// 快照语义自动继承 — EvaluatePureData 入口已把 ctx.Vars() wrap 成 tick-frozen view.
type exprEvalEnv struct {
	inputs expr.InputEnv
	vars   node.VarStore
}

func (e exprEvalEnv) Get(path string) (expr.Value, error) {
	if strings.HasPrefix(path, "$") {
		if e.vars == nil {
			return nil, nil
		}
		v, _ := e.vars.GetScoped(path[1:], "auto")
		return v, nil
	}
	return e.inputs.Get(path)
}

// exprEnvSkipKeys — env build 时跳过的 key 集合. 内含本节点 Spec 静态 input 跟 config
// 元数据 — 它们存在 merged map 里但语义上不是 user-facing dynamic input. 用户 expression
// 里写 "OutType" 应该是 undefined identifier (InputEnv.Get → nil), 而不是默契命中 config 值.
var exprEnvSkipKeys = map[string]struct{}{
	exprInExpression: {}, // "Expression" — Spec 静态 input
	"OutType":        {}, // config 元数据 (输出类型 hint)
	"Inputs":         {}, // config 元数据 (dynamic input declarations 列表)
}

// exprCache caches parsed AST keyed by expr string.
// Editor-scale (few hundred Expr nodes per container) — no eviction needed.
var exprCache = struct {
	sync.RWMutex
	m map[string]*expr.Node
}{m: map[string]*expr.Node{}}

func parseExprCached(src string) (*expr.Node, error) {
	exprCache.RLock()
	if ast, ok := exprCache.m[src]; ok {
		exprCache.RUnlock()
		return ast, nil
	}
	exprCache.RUnlock()
	ast, err := expr.Parse(src)
	if err != nil {
		return nil, err
	}
	exprCache.Lock()
	exprCache.m[src] = ast
	exprCache.Unlock()
	return ast, nil
}
