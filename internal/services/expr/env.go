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
