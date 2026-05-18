package runtime

import (
	"fmt"
	"strings"

	"yhbox/internal/services/container"
	"yhbox/internal/services/container/nodekind"
	_ "yhbox/internal/services/container/nodekind/specs" // ensure registry is populated
	"yhbox/internal/services/expr"
)

// dataEdgeIndex maps target pin → source pin for data-flow edges (Kind="data").
// Used by pullDataPin to resolve a node's data-in pin to its upstream source.
type dataEdgeIndex struct {
	// key: "<targetNodeID>.<targetPinName>", value: "<sourceNodeID>.<sourcePinName>"
	bySrc map[string]string
}

// buildDataEdgeIndex filters Graph.Edges for Kind=="data" only.
func buildDataEdgeIndex(g container.Graph) *dataEdgeIndex {
	idx := &dataEdgeIndex{bySrc: map[string]string{}}
	for _, e := range g.Edges {
		if e.Kind != "data" {
			continue
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

// pullDataPin resolves a data-in pin's current value (spec §7.2):
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
// Phase A wires GetVar only; later phases extend the switch for GetSys/GetParam/Expr/pure funcs.
func (r *ContainerRunner) evalDataSource(srcNodeID, srcPin string) (expr.Value, error) {
	n := r.nodesByID[srcNodeID]
	if n == nil {
		return nil, fmt.Errorf("evalDataSource: source node %q not found", srcNodeID)
	}
	// Gatekeep: only IsPureData kinds are valid pull sources.
	// An exec-node data-out (like Race.winnerIdx) should be read from
	// sys snapshot, not pulled — validator should have caught this.
	spec, ok := nodekind.Get(n.Kind)
	if !ok {
		return nil, fmt.Errorf("evalDataSource: unknown kind %q", n.Kind)
	}
	if !spec.IsPureData {
		return nil, fmt.Errorf("evalDataSource: kind %q is not pure-data (pin %q); use sys snapshot for exec-node data-out", n.Kind, srcPin)
	}
	switch n.Kind {
	case "GetVar":
		return r.evalGetVar(n)
	case "GetSys":
		return r.evalGetSys(n)
	case "GetParam":
		return r.evalGetParam(n)
	case "Expr":
		return r.evalExpr(n)
	// v4 §6: 22 pure-function nodes — all dispatched through evalPureFunc.
	case "Add", "Sub", "Mul", "Div", "Mod", "Neg",
		"Lt", "LtEq", "Gt", "GtEq", "Eq", "NotEq",
		"And", "Or", "Not",
		"Concat", "Contains", "Length",
		"ToString", "ToNumber", "ToBool",
		"Select":
		return r.evalPureFunc(n)
	}
	return nil, fmt.Errorf("evalDataSource: IsPureData=true but no eval case for kind %q (registry/dispatch drift!)", n.Kind)
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
