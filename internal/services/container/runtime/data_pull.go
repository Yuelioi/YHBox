package runtime

import (
	"context"
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
// v4 (C1): GraphEdge.Kind 已删除 — 边类型从 (from-node.kind, from-pin) 派生:
// 若 fromPin 在 nodekind.Spec.DataOut 里 → data 边; 否则 exec 边. 跟 validator 同源.
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
func (r *ContainerRunner) pullDataPin(nodeID, pinName string) (expr.Value, error) {
	// 1. Data edge lookup
	if srcID, srcPin := r.dataEdges.Source(nodeID, pinName); srcID != "" {
		return r.evalDataSource(srcID, srcPin)
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
// C5b: Get* (GetVar/GetSys/GetParam) + Expr 全走 framework EvaluatePureData (跟
// resolveDataPinV5 同模式). 22 pure-function 仍走 evalPureFunc (legacy) — capability
// segregation 完成前两者并存, pure-func 跨 commit 单独迁.
func (r *ContainerRunner) evalDataSource(srcNodeID, srcPin string) (expr.Value, error) {
	n := r.nodesByID[srcNodeID]
	if n == nil {
		return nil, fmt.Errorf("evalDataSource: source node %q not found", srcNodeID)
	}
	// Editor v2 C — Disable Node: pure-data nodes (GetVar/GetSys/GetParam/Expr/pure funcs)
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
	switch n.Kind {
	// v4 §6: 22 pure-function nodes — all dispatched through evalPureFunc (legacy path,
	// 仍走 r.pullDataPin 拉输入). Get* + Expr 已迁 framework, fallthrough default.
	case "Add", "Sub", "Mul", "Div", "Mod", "Neg",
		"Lt", "LtEq", "Gt", "GtEq", "Eq", "NotEq",
		"And", "Or", "Not",
		"Concat", "Contains", "Length",
		"ToString", "ToNumber", "ToBool",
		"Select":
		return r.evalPureFunc(n)
	}
	// Default: 走 framework EvaluatePureData. Get*/Expr 都在这条 path (Evaluator-impl).
	// buildDataWireFor 对 Expr 特例 (config.Inputs[] dynamic) + 普通节点 spec data-in 都覆盖.
	if rn.Evaluate == nil {
		return nil, fmt.Errorf("evalDataSource: IsPureData=true but kind %q does not implement Evaluator", n.Kind)
	}
	srcDataWire := r.buildDataWireFor(context.Background(), n, rn)
	srcConfig := r.buildConfigFor(n)
	v, err := nodepkg.EvaluatePureData(context.Background(), rn, srcDataWire, srcConfig, r.bundle)
	return toExprValue(v), err
}

// pullNumber: v4-only data-pin resolution (data edge or inline literal). No v3 fallback.
// Returns `fallback` if pin is unset / type-incompatible.
func (r *ContainerRunner) pullNumber(n *container.GraphNode, pinName string, fallback float64) float64 {
	v, err := r.pullDataPin(n.ID, pinName)
	if err != nil || v == nil {
		return fallback
	}
	if f, ok := expr.AsNumber(v); ok {
		return f
	}
	return fallback
}

// pullBool: v4-only data-pin resolution; expr.AsBool coerces.
func (r *ContainerRunner) pullBool(n *container.GraphNode, pinName string) (bool, error) {
	v, err := r.pullDataPin(n.ID, pinName)
	if err != nil {
		return false, err
	}
	return expr.AsBool(v), nil
}

// pullString: v4-only data-pin resolution; expr.FormatValue stringifies non-string values.
func (r *ContainerRunner) pullString(n *container.GraphNode, pinName string) string {
	v, err := r.pullDataPin(n.ID, pinName)
	if err != nil || v == nil {
		return ""
	}
	return expr.FormatValue(v)
}

// pullValue: v4-only data-pin resolution returning raw expr.Value (nil on miss).
// Used by Log/Toast which forward the raw value to FormatValue/Emit.
func (r *ContainerRunner) pullValue(n *container.GraphNode, pinName string) expr.Value {
	v, err := r.pullDataPin(n.ID, pinName)
	if err != nil {
		return nil
	}
	return v
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
		// Point literal: { "x": ..., "y": ... }
		if xv, hasX := x["x"]; hasX {
			if yv, hasY := x["y"]; hasY {
				return expr.Point{X: asFloat(xv), Y: asFloat(yv)}
			}
		}
		return x
	}
	return v
}
