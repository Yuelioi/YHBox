// dispatch_v5.go — Phase 5.5a 新 dispatch 路径 plumbing.
//
// 通过 node.RunNode / RunNodeAsRegion 派发节点, 取代老 nodes.go 大 switch.
//
// 当前阶段 (Phase 5.5a 已 ship): execNodeViaFramework 处理**非 region** 节点,
// routeResult 把 RunResult 映射到 ExecToken / 事件 emit. 测试覆盖全 5 路径.
//
// 不挂接到 execNode entry — 老 nodes.go switch 仍是 production 路径.
// 单元测试通过手动调 r.execNodeViaFramework 验证.
//
// 待 Phase 5.5b: region runner (Loop / Try / Subgraph) body callback + sub-dispatch.
// 待 Phase 5.5c: cutover — execNode entry 改成走 r.execNodeViaFramework (老 switch 删).
package runtime

import (
	"context"
	"fmt"

	nodepkg "yhbox/internal/node"
	"yhbox/internal/services/container"
)

// execNodeViaFramework dispatch 单节点 via 新 framework. 返下游 token 或 error.
// 不处理 RegionRunner — 那些走 r.execNodeAsRegionViaFramework (Phase 5.5b).
func (r *ContainerRunner) execNodeViaFramework(ctx context.Context, node *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	rn, ok := nodepkg.Get(node.Kind)
	if !ok {
		return nil, fmt.Errorf("execNodeViaFramework: kind %q not registered in node package", node.Kind)
	}
	if rn.RunRegion != nil {
		return nil, fmt.Errorf("execNodeViaFramework: kind %q is RegionRunner, must route via region path (Phase 5.5b)", node.Kind)
	}

	dataWire := r.buildDataWireFor(node, rn)
	config := r.buildConfigFor(node)
	execData := r.buildExecDataFor(tok)

	result := nodepkg.RunNode(ctx, rn, dataWire, config, execData, r.bundle)
	return r.routeResult(node, tok, result)
}

// buildDataWireFor pulls all non-Exec data-in pins from node Spec, evaluating via r.pullDataPin.
// Skip pin → pin not in map → framework Inputs.Has = false → falls back to config / default.
//
// 注意 pin name 大小写: r.pullDataPin 用 Spec.Inputs[].Name 当 pin name 查 r.dataEdges, 而
// edges 里的 To 字段是 JSON 序列化的 (老 JSON 的 lowercase pin name). 所以新框架节点 (PascalCase Spec)
// 跟老 JSON edges 对不上 — Phase 5.6 fishing-v2 redraw 后, 新 JSON pin name 跟 Spec 一致就 work.
func (r *ContainerRunner) buildDataWireFor(node *container.GraphNode, rn *nodepkg.RegisteredNode) map[string]any {
	dw := map[string]any{}
	for _, ip := range rn.Spec.Inputs {
		if ip.Type == "Exec" {
			continue
		}
		v, err := r.pullDataPin(node.ID, ip.Name)
		if err != nil || v == nil {
			continue
		}
		dw[ip.Name] = v
	}
	return dw
}

// buildConfigFor 复制 node.Config 当 framework config map. 扣 "literal" 内部字段
// (pullDataPin 已经消费它做 inline data-edge literal source).
//
// Phase 5.5a 不做 pin name normalization — config key 跟 Spec.Inputs[].Name 严格 case-sensitive 比.
// 老 fishing-v2 JSON 用 lowercase, 新 Spec 用 PascalCase, 不会自动对齐. Phase 5.6 redraw 时按新名写.
func (r *ContainerRunner) buildConfigFor(node *container.GraphNode) map[string]any {
	out := make(map[string]any, len(node.Config))
	for k, v := range node.Config {
		if k == "literal" {
			continue
		}
		out[k] = v
	}
	return out
}

// buildExecDataFor 当前返空 map. Phase 5.5b 加 exec-data carry — 上游节点 RunResult.OutputData
// 的字段在 routeResult 时 stash 到 ExecToken (新加字段 Token.ExecData), 下游 build 时 pull 出.
// 这步要先扩 ExecToken struct + 改 edges.next 携带 data + 改 buildExecDataFor 读. 留后续.
func (r *ContainerRunner) buildExecDataFor(_ ExecToken) map[string]any {
	return map[string]any{}
}

// routeResult turns RunResult into next ExecToken batch + emit lifecycle events.
//
// Priority order:
//  1. Panic   → emit container:node-panic + return error (framework invariant broken)
//  2. Validation → emit container:node-validation + return error (graph 写错, Phase 5.5a 中断 run;
//     后续可改 emit + continue 让前端高亮但不停)
//  3. Error   → return error (Run 返 runtime fail, caller 决定 stop / 冒泡 / 走 Try)
//  4. Display → emit container:node-log (Display() 非空才 emit)
//  5. ExitName 空 → return nil tokens (no-op 节点, queue 继续)
//  6. ExitName 非空 → r.edges.next(nodeID.exitName)
func (r *ContainerRunner) routeResult(node *container.GraphNode, tok ExecToken, result nodepkg.RunResult) ([]ExecToken, error) {
	if result.Panic != nil {
		if r.rt.Emit != nil {
			r.rt.Emit("container:node-panic", map[string]any{
				"containerId": r.rt.Container.ID,
				"nodeId":      node.ID,
				"nodeKind":    node.Kind,
				"panic":       fmt.Sprintf("%v", result.Panic),
				"stack":       result.PanicStack,
			})
		}
		return nil, fmt.Errorf("node %q (%s) panic: %v", node.ID, node.Kind, result.Panic)
	}

	if len(result.Validation) > 0 {
		if r.rt.Emit != nil {
			r.rt.Emit("container:node-validation", map[string]any{
				"containerId": r.rt.Container.ID,
				"nodeId":      node.ID,
				"nodeKind":    node.Kind,
				"errors":      result.Validation,
			})
		}
		return nil, fmt.Errorf("node %q (%s) validation: %s", node.ID, node.Kind, result.Validation[0].Message)
	}

	if result.Error != nil {
		return nil, result.Error
	}

	if result.DisplayText != "" && r.rt.Emit != nil {
		r.rt.Emit("container:node-log", map[string]any{
			"containerId": r.rt.Container.ID,
			"nodeId":      node.ID,
			"nodeKind":    node.Kind,
			"message":     result.DisplayText,
		})
	}

	if result.ExitName == "" {
		return nil, nil
	}
	return r.edges.next(node.ID+"."+result.ExitName, tok.LoopStack), nil
}
