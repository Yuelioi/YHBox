// Package runtime 把 Container 蓝图跑起来（token-based dispatch）。
//
// 节点 config 是 string 表达式 → expr.Parse → AST 缓存到 RuntimeContext。
// 节点之间通过 token 流转（不是 graph traversal，方便 Loop/Break/Continue 直接
// 跳转到目标 token）。
package runtime

import (
	"fmt"
	"sync"

	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
	"yhbox/internal/services/expr"
)

// SysState $sys.* 只读视图。每次 Container run 开局清零。
type SysState struct {
	RunID      int64
	Iter       int64      // 当前最内层 Loop 迭代次数
	LastFound  bool       // 上次 WaitTemplate/CheckTemplate/ClickTemplate 命中
	LastPoint  expr.Point // 命中位置（miss 时 zero）
	LastRegion [4]float64 // 命中 region [r,r,r,r]
	WinnerIdx  int64      // 上次 Race 完成时获胜分支
	// DetectColor 节点输出：命中像素数 + 命中中心客户区比例。
	LastColorCount  int64
	LastColorCenter expr.Point
}

// RuntimeContext 单 Container run 的状态。
//   - vars / sys 通过 mutex 保护（Parallel/Race 子分支并发读写）
//   - inputBus 注入：每个 input 节点 Lock/Unlock 保证 OS input 串行
//   - templateMatcher 注入：Wait/Check/ClickTemplate 用
//   - actionRunner 注入：InvokeAction 节点用
//   - emit 注入：Log/Toast 节点把消息推到前端
type RuntimeContext struct {
	Container *container.Container
	InputBus  *execution.InputBus
	Matcher   TemplateMatcher
	Actions   ActionInvoker
	Input     InputDriver
	Color     ColorDetector
	Emit      func(name string, data any)

	mu     sync.Mutex
	vars   map[string]expr.Value
	params map[string]expr.Value
	sys    SysState
}

func NewRuntimeContext(c *container.Container, bus *execution.InputBus, matcher TemplateMatcher, actions ActionInvoker, input InputDriver, color ColorDetector, emit func(name string, data any)) *RuntimeContext {
	if input == nil {
		input = NoopInputDriver{}
	}
	if color == nil {
		color = NoopColorDetector{}
	}
	rt := &RuntimeContext{
		Container: c,
		InputBus:  bus,
		Matcher:   matcher,
		Actions:   actions,
		Input:     input,
		Color:     color,
		Emit:      emit,
		vars:      make(map[string]expr.Value),
		params:    make(map[string]expr.Value),
	}
	rt.initVars()
	return rt
}

func (rt *RuntimeContext) initVars() {
	for _, v := range rt.Container.Vars {
		rt.vars[v.Name] = v.Default
	}
}

// Vars 拿当前快照（拷贝，调用方修改不影响 runtime）。
func (rt *RuntimeContext) Vars() map[string]expr.Value {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	cp := make(map[string]expr.Value, len(rt.vars))
	for k, v := range rt.vars {
		cp[k] = v
	}
	return cp
}

// SetVar 单字段写。线程安全。
func (rt *RuntimeContext) SetVar(name string, val expr.Value) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.vars[name] = val
}

// IncVar 单字段增量。如果当前不是 number → 报错。
func (rt *RuntimeContext) IncVar(name string, delta float64) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	cur, _ := expr.AsNumber(rt.vars[name])
	rt.vars[name] = cur + delta
	return nil
}

// SetParam 仅 ContainerRunner 启动时设（v1 Container 无 params；spec 留扩展位）。
func (rt *RuntimeContext) SetParam(name string, val expr.Value) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.params[name] = val
}

func (rt *RuntimeContext) Sys() SysState {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.sys
}

func (rt *RuntimeContext) UpdateSys(f func(s *SysState)) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	f(&rt.sys)
}

// Env 把 RuntimeContext 当 expr.Env 用。支持 $vars.X / $params.Y / $sys.Z[.W[.V]]。
func (rt *RuntimeContext) Env() expr.Env { return &envAdapter{rt: rt} }

type envAdapter struct{ rt *RuntimeContext }

func (e *envAdapter) Get(path string) (expr.Value, error) {
	rt := e.rt
	rt.mu.Lock()
	defer rt.mu.Unlock()

	switch {
	case len(path) > 6 && path[:6] == "$vars.":
		key := path[6:]
		v, ok := rt.vars[key]
		if !ok {
			return nil, nil
		}
		return v, nil
	case len(path) > 8 && path[:8] == "$params.":
		key := path[8:]
		v, ok := rt.params[key]
		if !ok {
			return nil, nil
		}
		return v, nil
	case len(path) > 5 && path[:5] == "$sys.":
		return resolveSysPath(rt.sys, path[5:])
	}
	return nil, fmt.Errorf("expr: unknown variable namespace %q", path)
}

func resolveSysPath(s SysState, rest string) (expr.Value, error) {
	switch rest {
	case "runId":
		return float64(s.RunID), nil
	case "iter":
		return float64(s.Iter), nil
	case "winnerIdx":
		return float64(s.WinnerIdx), nil
	case "lastTemplate.found":
		return s.LastFound, nil
	case "lastTemplate.point":
		return s.LastPoint, nil
	case "lastTemplate.point.x":
		return s.LastPoint.X, nil
	case "lastTemplate.point.y":
		return s.LastPoint.Y, nil
	case "lastTemplate.region":
		return s.LastRegion, nil
	case "lastColor.count":
		return float64(s.LastColorCount), nil
	case "lastColor.cx", "lastColor.center.x":
		return s.LastColorCenter.X, nil
	case "lastColor.cy", "lastColor.center.y":
		return s.LastColorCenter.Y, nil
	case "lastColor.center":
		return s.LastColorCenter, nil
	}
	return nil, fmt.Errorf("expr: unknown $sys.%s", rest)
}
