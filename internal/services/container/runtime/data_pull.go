package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	nodepkg "yhbox/internal/node"
	"yhbox/internal/services/container"
	"yhbox/internal/services/expr"
)

// dataEdgeIndex maps target pin → source pin for data-flow edges.
// Used by pullDataPin to resolve a node's data-in pin to its upstream source.
type dataEdgeIndex struct {
	// key: "<targetNodeID>.<targetPinName>", value: "<sourceNodeID>.<sourcePinName>"
	bySrc map[string]string
}

// buildDataEdgeIndex filters Graph.Edges to data-flow edges only.
//
// 边类型从 (from-node.kind, from-pin) 派生: 若 fromPin 在 nodekind.Spec.DataOut 里
// → data 边; 否则 exec 边. 跟 validator 同源.
func buildDataEdgeIndex(g container.Graph) *dataEdgeIndex {
	idx := &dataEdgeIndex{bySrc: map[string]string{}}
	kindByID := make(map[string]string, len(g.Nodes))
	for _, n := range g.Nodes {
		kindByID[n.ID] = n.Kind
	}
	for _, e := range g.Edges {
		fromID, fromPin, found := strings.Cut(e.From, ".")
		if !found {
			continue
		}
		kind, ok := kindByID[fromID]
		if !ok {
			continue
		}
		if !container.IsDataOutPin(kind, fromPin) {
			continue // not a data-out — exec edge
		}
		idx.bySrc[e.To] = e.From
	}
	return idx
}

// Source returns (sourceNodeID, sourcePinName) for the given target, or ("","") if no edge.
func (d *dataEdgeIndex) Source(nodeID, pinName string) (string, string) {
	if d == nil {
		return "", ""
	}
	v, ok := d.bySrc[nodeID+"."+pinName]
	if !ok {
		return "", ""
	}
	src, pin, found := strings.Cut(v, ".")
	if !found {
		return v, ""
	}
	return src, pin
}

// pullDataPin resolves a data-in pin's current value:
//  1. If a data edge connects to this pin, recursively evaluate the source.
//  2. Else if config["literal"][pinName] exists, return it (inline literal).
//  3. Else return nil (caller handles default).
//
// Pure data sources (GetVar / GetSys / GetParam / Expr / pure functions) eval on-demand.
// Exec nodes that expose data-out (e.g. Race.winnerIdx) read from the sys snapshot.
//
// ctx 必传 — tick snapshot 走 ctx (tickCtxKey). dispatch hot path 传 dispatchInRegion
// 已 withTickSnapshot 的 ctx; init/cold path (listener config) 传 context.Background() (无 tick OK).
func (r *ContainerRunner) pullDataPin(ctx context.Context, nodeID, pinName string) (expr.Value, error) {
	// 1. Data edge lookup
	if srcID, srcPin := r.dataEdges.Source(nodeID, pinName); srcID != "" {
		return r.evalDataSource(ctx, srcID, srcPin)
	}
	// 2. Literal lookup
	n := r.nodesByID[nodeID]
	if n == nil {
		return nil, fmt.Errorf("pullDataPin: node %q not found", nodeID)
	}
	if lit, ok := n.Config["literal"].(map[string]any); ok {
		if v, ok := lit[pinName]; ok {
			return toExprValue(v), nil
		}
	}
	// 3. Default = nil (consumer applies its own fallback)
	return nil, nil
}

// evalDataSource dispatches by source node Kind. Only pure-data Kinds are valid sources
// (a data edge from an exec node like Sleep would be a schema error caught by validator).
//
// 全 IsPureData 节点 (Get*/Expr/22 pure-function) 走 framework EvaluatePureData 单一 path.
// buildDataWireFor 对 Expr 特例 (config.Inputs[] dynamic) + 普通节点 spec data-in 都覆盖.
func (r *ContainerRunner) evalDataSource(ctx context.Context, srcNodeID, srcPin string) (expr.Value, error) {
	n := r.nodesByID[srcNodeID]
	if n == nil {
		return nil, fmt.Errorf("evalDataSource: source node %q not found", srcNodeID)
	}
	// Disable Node: pure-data nodes (GetVar/GetSys/GetParam/Expr/pure funcs)
	// don't reach execNode, so the execNode disable gate doesn't cover them. Gate here too.
	// Disabled pure-data node returns nil (untyped — consumer's fallback / pullNumber default fires).
	if n.Disabled {
		return nil, nil
	}
	// Gatekeep: only IsPureData kinds are valid pull sources.
	// An exec-node data-out (like Race.winnerIdx) should be read from
	// sys snapshot, not pulled — validator should have caught this.
	rn, ok := nodepkg.Get(n.Kind)
	if !ok {
		return nil, fmt.Errorf("evalDataSource: unknown kind %q", n.Kind)
	}
	if !rn.Spec.IsPureData {
		return nil, fmt.Errorf("evalDataSource: kind %q is not pure-data (pin %q); use sys snapshot for exec-node data-out", n.Kind, srcPin)
	}
	if rn.Evaluate == nil {
		return nil, fmt.Errorf("evalDataSource: IsPureData=true but kind %q does not implement Evaluator", n.Kind)
	}
	// 传 ctx 给递归 buildDataWireFor + EvaluatePureData, 让 bundle.Snapshot(ctx) 拿到 tick.
	srcDataWire := r.buildDataWireFor(ctx, n, rn)
	srcConfig := r.buildConfigFor(n)
	v, err := nodepkg.EvaluatePureData(ctx, rn, srcDataWire, srcConfig, r.bundle)
	return toExprValue(v), err
}

// pullNumber resolves a numeric data-pin (data edge or inline literal).
// Returns `fallback` if pin is unset / type-incompatible.
//
// ctx — listener config init 时传 context.Background() (无 tick OK, listener config 走 literal
// 不依赖 frozen Vars); dispatch hot path 调用 (无 prod 当前 callsite, future-proof) 传带 tick 的 ctx.
func (r *ContainerRunner) pullNumber(ctx context.Context, n *container.GraphNode, pinName string, fallback float64) float64 {
	v, err := r.pullDataPin(ctx, n.ID, pinName)
	if err != nil || v == nil {
		return fallback
	}
	if f, ok := expr.AsNumber(v); ok {
		return f
	}
	return fallback
}

// toExprValue converts JSON-decoded values (always one of: float64 / bool / string / nil /
// map[string]any for Point) to expr.Value.
func toExprValue(v any) expr.Value {
	switch x := v.(type) {
	case nil:
		return nil
	case float64, bool, string:
		return x
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case map[string]any:
		// Point literal: { "x": ..., "y": ... }. 带 w/h 的是 Rect — 留 raw map,
		// 交给 buildDataWireFor 按声明类型 coerce 成 node.Rect (否则 w/h 会被丢)。
		_, hasW := x["w"]
		_, hasH := x["h"]
		if !hasW && !hasH {
			if xv, hasX := x["x"]; hasX {
				if yv, hasY := x["y"]; hasY {
					return expr.Point{X: asFloat(xv), Y: asFloat(yv)}
				}
			}
		}
		return x
	}
	return v
}

// coerceToType 把 data-pin 解析出的 JSON 值按声明 pin 类型转成节点期望的域类型。
// Rect/Point pin 的 literal (画布 rect-editor / 屏幕拾取 / 录制) 存成 JSON map/array,
// 但 in.Rect()/in.Point() 断言 node.Rect/node.Point → 这里按 ip.Type 物化。
// 其余类型原样返 (Number/String/Bool 由 in.Float64/String/Bool 各自解析)。
func coerceToType(v any, typ string) any {
	switch typ {
	case "Rect":
		if r, ok := asNodeRect(v); ok {
			return r
		}
	case "Point":
		if p, ok := asNodePoint(v); ok {
			return p
		}
	case "Geometry":
		b, _ := json.Marshal(v)
		var g nodepkg.Geometry
		_ = json.Unmarshal(b, &g)
		return g
	}
	return v
}

// asNodeRect 把 node.Rect / map{x,y,w,h} / [x,y,w,h] 转 node.Rect。
func asNodeRect(v any) (nodepkg.Rect, bool) {
	switch t := v.(type) {
	case nodepkg.Rect:
		return t, true
	case map[string]any:
		return nodepkg.Rect{X: asFloat(t["x"]), Y: asFloat(t["y"]), W: asFloat(t["w"]), H: asFloat(t["h"])}, true
	case []any:
		if len(t) >= 4 {
			return nodepkg.Rect{X: asFloat(t[0]), Y: asFloat(t[1]), W: asFloat(t[2]), H: asFloat(t[3])}, true
		}
	}
	return nodepkg.Rect{}, false
}

// asNodePoint 把 node.Point / expr.Point / map{x,y} / [x,y] 转 node.Point。
func asNodePoint(v any) (nodepkg.Point, bool) {
	switch t := v.(type) {
	case nodepkg.Point:
		return t, true
	case expr.Point:
		return nodepkg.Point{X: t.X, Y: t.Y}, true
	case map[string]any:
		if _, ok := t["x"]; ok {
			return nodepkg.Point{X: asFloat(t["x"]), Y: asFloat(t["y"])}, true
		}
	case []any:
		if len(t) >= 2 {
			return nodepkg.Point{X: asFloat(t[0]), Y: asFloat(t[1])}, true
		}
	}
	return nodepkg.Point{}, false
}
