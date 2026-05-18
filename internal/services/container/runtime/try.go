package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"yhbox/internal/services/container"
)

// execTry 跑引用的子图, 三路出口:
//   - done    : 子图正常完成 (SubgraphOutput reach 或子图末端 token 耗尽)
//   - timeout : timeoutMs 超时, sub-runner cancel
//   - error   : 子图节点 return error (含 errThrow) → 不冒泡, 走 error 出口
//
// errorMsg 数据写入 $sys.lastTry.errorMsg (timeout + error 两路都填).
//
// 实现要点:
//   - execSubgraph 会全局切换 r.edges / r.nodesByID (v2 历史债), 所以 Try 不能复用它.
//     改为创建一个独立的子 runner, 共享同一 RuntimeContext, 只是 edges/nodesByID 指向
//     子图的图. 这避免了对父图 dispatch state 的破坏.
//   - 超时通过 context.WithTimeout 注入子 runner 的 runSubFlow.
//   - errThrow 从 runSubFlow 冒泡上来, errors.As 精确提取 message.
func (r *ContainerRunner) execTry(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	sgID := configString(n, "subgraphId")
	if sgID == "" {
		return nil, fmt.Errorf("Try %s: missing subgraphId", n.ID)
	}

	// 查子图 — 从 rt.Container.Subgraphs 找 (ContainerRunner 不持有独立 subgraph map)
	var sg *container.Subgraph
	for i := range r.rt.Container.Subgraphs {
		if r.rt.Container.Subgraphs[i].ID == sgID {
			sg = &r.rt.Container.Subgraphs[i]
			break
		}
	}
	if sg == nil {
		return nil, fmt.Errorf("Try %s: unknown subgraph %q", n.ID, sgID)
	}

	timeoutMs := int(r.configFloat(n, "timeoutMs", 0))

	// 建 context — 有 timeout 则 WithTimeout, 否则继承 parent ctx
	childCtx := ctx
	var cancel context.CancelFunc
	if timeoutMs > 0 {
		childCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()
	}

	// 建子 runner: 独立 edges + nodesByID 指向子图, 共享 rt (保持 SysState / vars / InputBus)
	child := r.newSubRunner(sg)

	// 找子图 SubgraphInput 节点, 从其 out 入流
	var inputID string
	for _, node := range sg.Graph.Nodes {
		if node.Kind == "SubgraphInput" {
			inputID = node.ID
			break
		}
	}
	if inputID == "" {
		return nil, fmt.Errorf("Try %s: subgraph %q 缺 SubgraphInput 节点", n.ID, sgID)
	}
	seeds := child.edges.next(inputID+".out", tok.LoopStack)

	// 同步跑子图 — runTrySubFlow 在子 runner 上, 不影响父 runner 的 edges/nodesByID.
	// 用专门的 runTrySubFlow 而不是 runSubFlow, 是因为 SubgraphOutput 节点会
	// 尝试切回父图 frame/edges, 而 Try 子 runner 没有对应的父 frame 上下文.
	// runTrySubFlow 把 SubgraphOutput 当做终点 (正常完成) 处理.
	err := child.runTrySubFlow(childCtx, seeds)

	pin := "done"
	errMsg := ""
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			// context.Canceled 可能来自 WithTimeout 内部; 只在我们自己的 timeout 触发时走 timeout 出口
			if timeoutMs > 0 && childCtx.Err() != nil {
				pin = "timeout"
				errMsg = fmt.Sprintf("timeout after %dms", timeoutMs)
			} else {
				// 父 ctx cancel → 冒泡 (非 Try 的责任)
				return nil, err
			}
		} else if errors.Is(err, errStopRun) {
			// Stop 节点 in subgraph → 视为正常完成 (跟 runSubFlow 逻辑一致)
			pin = "done"
		} else {
			pin = "error"
			var te *errThrow
			if errors.As(err, &te) {
				errMsg = te.message
			} else {
				errMsg = err.Error()
			}
		}
	}

	// 写 SysState — 跟 StopwatchRead / DetectColor 等节点同模式
	r.rt.UpdateSys(func(s *SysState) { s.LastTry.ErrorMsg = errMsg })

	return r.edges.next(n.ID+"."+pin, tok.LoopStack), nil
}

// runTrySubFlow 是 Try 专用的子图 dispatch loop.
// 与 runSubFlow 的区别: SubgraphOutput 节点视为终点 (正常完成, 不尝试切回父图).
// 这是因为 Try 的子 runner 没有对应的父 frame/edges 上下文 (execSubgraphOutput 会
// 尝试 pop frame + 切换 edges → panic / wrong-node-lookup).
func (r *ContainerRunner) runTrySubFlow(ctx context.Context, seeds []ExecToken) error {
	queue := append([]ExecToken{}, seeds...)
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		tok := queue[0]
		queue = queue[1:]
		node, ok := r.nodesByID[tok.NodeID]
		if !ok {
			return fmt.Errorf("subflow: unknown node %q", tok.NodeID)
		}
		// SubgraphOutput 在 Try 上下文中是正常出口, 直接当终点 (不切父图)
		if node.Kind == "SubgraphOutput" {
			return nil
		}
		out, err := r.execNode(ctx, node, tok)
		if err != nil {
			if errors.Is(err, errStopRun) {
				return nil
			}
			return err
		}
		queue = append(queue, out...)
	}
	return nil
}

// newSubRunner 建一个与父 runner 共享 rt 但独立 edges/nodesByID 的子 runner.
// 用于 Try 的子图执行, 避免 execSubgraph 的全局切换问题.
func (r *ContainerRunner) newSubRunner(sg *container.Subgraph) *ContainerRunner {
	child := &ContainerRunner{
		rt:          r.rt,
		edges:       buildEdgeIndex(sg.Graph),
		nodesByID:   make(map[string]*container.GraphNode, len(sg.Graph.Nodes)),
		state:       r.state, // 共享 ExecState (vars / frame)
		stopwatches: r.stopwatches, // 共享 stopwatch map
	}
	for i := range sg.Graph.Nodes {
		n := &sg.Graph.Nodes[i]
		child.nodesByID[n.ID] = n
	}
	return child
}
