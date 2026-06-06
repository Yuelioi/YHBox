package expr

// Env 是变量解析层。Eval 遇到 $vars.X / $params.Y / $sys.Z 时调 Get 取值。
//
// path 形如 "$vars.foo" / "$params.pos.x" / "$sys.lastTemplate.point.x" ——
// 完整 dotted 路径，含 leading $。实现方负责处理嵌套字段访问。
//
// 返 (nil, nil) 表示"路径不存在"——eval 视情况返 null 或报错。
// 返 (v, err) 表示真错（类型不匹配 / 内部坏）。
type Env interface {
	Get(path string) (Value, error)
}

// MapEnv 测试用：把 map[string]Value 包成 Env。key 必须含 $ 前缀。
type MapEnv map[string]Value

func (m MapEnv) Get(path string) (Value, error) {
	if v, ok := m[path]; ok {
		return v, nil
	}
	return nil, nil
}

// InputEnv (v4) 是 Expr 节点专用 Env: 只识别 bare identifier (e.g. "i", "state"),
// 不暴露 $vars / $params 命名空间. 用作 Expr.Eval(InputEnv(inputsMap)).
//
// 设计意图: 让 Expr 节点表达式跟 v3 expr 隔离 — Expr 输入必须显式声明在 inputs[],
// 想引用变量/入参 → 拖 GetVar/GetParam 节点把值送进 Expr.
// 这样表达式不再隐式依赖 runtime 上下文, 可视化+确定性.
type InputEnv map[string]Value

func (e InputEnv) Get(name string) (Value, error) {
	// $-prefixed paths are NOT supported in Expr node context — return nil (treated as null).
	// nIdent (bare identifier) lookups land here with no prefix.
	if v, ok := e[name]; ok {
		return v, nil
	}
	return nil, nil
}
