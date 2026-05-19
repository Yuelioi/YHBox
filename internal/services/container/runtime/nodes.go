package runtime

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lxn/win"

	"yhbox/internal/services/container"
	"yhbox/internal/services/container/nodekind"
	_ "yhbox/internal/services/container/nodekind/specs" // ensure registry is populated
	"yhbox/internal/services/execution"
	"yhbox/internal/services/expr"
	"yhbox/internal/services/inputclip"
	clipruntime "yhbox/internal/services/inputclip/runtime"
)

// errThrow 是显式 Throw 节点触发的 sentinel.
// 在 sub-runner 中冒泡到最近的 Try 节点 (Task 7), Try 截获后走 error 出口;
// 没 Try 包就冒泡到主 runner, 当 container:error emit (跟正常 panic 路径合流).
// 用 pointer-receiver Error() 是为了 errors.As(&te) 能精确取回 message.
type errThrow struct {
	message string
}

func (e *errThrow) Error() string {
	return "Throw: " + e.message
}

// execNode 单节点执行入口。返回下游 token 列表（追加进调度队列）。
// 大部分节点产出 1 个或 0 个 token；Parallel/Race 是特例（自跑 sub-runner）。
func (r *ContainerRunner) execNode(ctx context.Context, node *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	// Gatekeep: kind must be registered. Catches typos and stale switch cases.
	spec, ok := nodekind.Get(node.Kind)
	if !ok {
		return nil, fmt.Errorf("execNode: unknown kind %q (not in nodekind registry)", node.Kind)
	}
	// IsPureData kinds should never reach execNode (no exec edge points to them).
	// A switch fall-through would silently return (nil, nil) — explicit reject.
	if spec.IsPureData {
		return nil, fmt.Errorf("execNode: kind %q is pure-data, cannot be executed (validator drift!)", node.Kind)
	}
	// IsVisualOnly = render-only nodes (CommentBox). No-op execution.
	if spec.IsVisualOnly {
		return nil, nil
	}

	switch node.Kind {
	case "Start":
		return r.edges.next(node.ID+".out", tok.LoopStack), nil
	case "Sleep":
		return r.execSleep(ctx, node, tok)
	case "Loop":
		return r.execLoop(ctx, node, tok)
	case "If":
		return r.execIf(ctx, node, tok)
	case "Switch":
		return r.execSwitch(ctx, node, tok)
	case "Cron":
		return r.execCron(ctx, node, tok)
	case "Parallel":
		return r.execParallel(ctx, node, tok)
	case "Race":
		return r.execRace(ctx, node, tok)
	case "Stop":
		return nil, errStopRun
	case "Break":
		return r.execBreak(node, tok)
	case "Continue":
		return r.execContinue(node, tok)
	case "SetVar":
		return r.execSetVar(ctx, node, tok)
	case "IncVar":
		return r.execIncVar(ctx, node, tok)
	case "WaitTemplate":
		return r.execWaitTemplate(ctx, node, tok)
	case "CheckTemplate":
		return r.execCheckTemplate(ctx, node, tok)
	case "ClickTemplate":
		return r.execClickTemplate(ctx, node, tok)
	case "DetectColor":
		return r.execDetectColor(ctx, node, tok)
	case "Subgraph", "CollapsedNode":
		// v4 §9.2 CollapsedNode = isAnonymous Subgraph; same runtime path.
		return r.execSubgraph(ctx, node, tok)
	case "SubgraphInput":
		// FUTURE-WORK (spec §11): SubgraphInput 当前在 graph 里以实节点存在, 但它本质是
		// metadata (子图入口的 marker), 没有任何执行逻辑 — 就是 edges.next 直通. v2 早期
		// 这样做是因为 UI 复用 vue-flow 节点系统简单. 长期看应该把它降级为子图 metadata
		// (Subgraph.EntryNodeID 或类似), 不进 Graph.Nodes. 收益: 1) 用户改不了入口数量 (1
		// 个 enforce); 2) validator 不用再检 SubgraphInput 唯一性; 3) 子图布局更自由
		// (入口不占节点位). 改的时机: 加 ValidationError UI 之后, 顺手清理.
		return r.edges.next(node.ID+".out", tok.LoopStack), nil
	case "SubgraphOutput":
		return r.execSubgraphOutput(ctx, node, tok)
	case "BringGameForeground":
		return r.execBringGameForeground(ctx, node, tok)
	case "MouseCalibration":
		// 声明式节点不参与执行流，走直通
		return nil, nil
	case "WindowTarget":
		// v3 Phase B 声明式节点: hwnd + input/capture backend 解析在 runner 启动期消费, 执行流直通
		return nil, nil
	// CommentBox handled by IsVisualOnly gatekeep above (registry single source of truth).
	case "ClickAt":
		return r.execClickAt(ctx, node, tok)
	case "KeyPress":
		return r.execKeyPress(ctx, node, tok)
	case "MouseMoveRel":
		return r.execMouseMoveRel(ctx, node, tok)
	case "Scroll":
		return r.execScroll(ctx, node, tok)
	case "OnEvent":
		// OnEvent 没 exec-in：listener goroutine 直接 spawn 子 runner 跑 out 后裔。
		// 主图永远不会 dispatch 到 OnEvent 节点；这里防御性返空。
		return nil, nil
	case "Log":
		return r.execLog(ctx, node, tok)
	case "Toast":
		return r.execToast(ctx, node, tok)
	case "PlayClip":
		return r.execPlayClip(ctx, node, tok)
	case "StopwatchStart":
		return r.execStopwatchStart(ctx, node, tok)
	case "StopwatchStop":
		return r.execStopwatchStop(ctx, node, tok)
	case "StopwatchRead":
		return r.execStopwatchRead(ctx, node, tok)
	case "KeyHoldStart":
		return r.execKeyHoldStart(ctx, node, tok)
	case "KeyHoldStop":
		return r.execKeyHoldStop(ctx, node, tok)
	case "MouseHoldStart":
		return r.execMouseHoldStart(ctx, node, tok)
	case "MouseHoldStop":
		return r.execMouseHoldStop(ctx, node, tok)
	case "Throw":
		return r.execThrow(ctx, node, tok)
	case "Try":
		return r.execTry(ctx, node, tok)
	case "DetectColorHSV":
		return r.execDetectColorHSV(ctx, node, tok)
	case "ColorBarTrack":
		return r.execColorBarTrack(ctx, node, tok)
	case "ROIColorScan":
		return r.execROIColorScan(ctx, node, tok)
	case "Screenshot":
		return r.execScreenshot(ctx, node, tok)
	}
	return nil, fmt.Errorf("execNode: kind %q registered but no dispatch case (drift!)", node.Kind)
}

// ---- Control Flow ----

func (r *ContainerRunner) execSleep(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	// v4: try data-in pin first (literal or data edge).
	d := r.pullNumber(n, "durationMs", 0)
	if err := execution.Sleep(ctx, time.Duration(d)*time.Millisecond); err != nil {
		return nil, err
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

func (r *ContainerRunner) execLoop(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	mode := configString(n, "mode")
	if mode == "" {
		mode = "count"
	}

	// 第一次进 Loop：tok.InPin == "in"，建 frame
	// 子图回来：tok.InPin == "loopback"（自定义），iter++
	var frame *LoopFrame
	if tok.InPin == "loopback" && len(tok.LoopStack) > 0 {
		frame = tok.LoopStack[len(tok.LoopStack)-1]
		frame.Iter++
	} else {
		frame = &LoopFrame{LoopNodeID: n.ID, Iter: 0}
		tok.LoopStack = append(tok.LoopStack, frame)
	}

	r.rt.UpdateSys(func(s *SysState) { s.Iter = frame.Iter })

	switch mode {
	case "count":
		// v4: data-in pin "count" (number).
		count := int64(r.pullNumber(n, "count", 1))
		if frame.Iter >= count {
			return r.exitLoop(n, tok)
		}
	case "while":
		// v4: data-in pin "condition" (bool).
		ok, err := r.pullBool(n, "condition")
		if err != nil {
			return nil, err
		}
		if !ok {
			return r.exitLoop(n, tok)
		}
	case "forever":
		// no exit
	default:
		return nil, fmt.Errorf("Loop %s: unknown mode %q", n.ID, mode)
	}

	// 进 body
	body := r.edges.next(n.ID+".body", tok.LoopStack)
	if len(body) == 0 {
		// body 为空 → 立即 loopback
		return []ExecToken{{NodeID: n.ID, InPin: "loopback", LoopStack: tok.LoopStack}}, nil
	}
	return body, nil
}

func (r *ContainerRunner) exitLoop(n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	// 弹 frame
	if len(tok.LoopStack) > 0 {
		tok.LoopStack = tok.LoopStack[:len(tok.LoopStack)-1]
	}
	return r.edges.next(n.ID+".complete", tok.LoopStack), nil
}

func (r *ContainerRunner) execIf(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	// v4: data-in pin "condition" (bool).
	ok, err := r.pullBool(n, "condition")
	if err != nil {
		return nil, err
	}
	pin := "else"
	if ok {
		pin = "then"
	}
	return r.edges.next(n.ID+"."+pin, tok.LoopStack), nil
}

func (r *ContainerRunner) execBreak(n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	if len(tok.LoopStack) == 0 {
		return nil, fmt.Errorf("Break node %s 不在 Loop 内", n.ID)
	}
	loop := tok.LoopStack[len(tok.LoopStack)-1]
	tok.LoopStack = tok.LoopStack[:len(tok.LoopStack)-1]
	return r.edges.next(loop.LoopNodeID+".complete", tok.LoopStack), nil
}

func (r *ContainerRunner) execContinue(n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	if len(tok.LoopStack) == 0 {
		return nil, fmt.Errorf("Continue node %s 不在 Loop 内", n.ID)
	}
	loop := tok.LoopStack[len(tok.LoopStack)-1]
	return []ExecToken{{NodeID: loop.LoopNodeID, InPin: "loopback", LoopStack: tok.LoopStack}}, nil
}

// ---- Parallel / Race ----
//
// 简化实现：sub-runner 在自己的 goroutine 里跑迷你 dispatch（与主 runner 共享
// RuntimeContext / inputBus / matcher）。子分支跑完才返主图 out。
// Parallel 等所有分支结束；Race 第一个 reach terminal 算赢，其它 ctx 取消。
//
// 已知限制：tok.LoopStack 是 []*LoopFrame，分支间共享外层 frame 指针。
// 这意味着：
//   1. 分支内部 push 进 LoopStack 的新 *LoopFrame 是各分支独立的（因为是 append 到 copy 后的切片）。
//   2. 但若分支引用外层 frame 做 Continue/Break，所有分支看到的是同一个 *LoopFrame——
//      实际上这是期望行为（Continue 应跳目标 = 外层 loop）。
//   3. 风险：若多个分支同时 Continue 同一外层 Loop，外层 Loop 会收到多次 loopback
//      token → frame.Iter 多增。当前不挡，使用者应避免在 Parallel 分支里用 Break/Continue
//      跳跃外层 Loop（推荐：分支内的 Loop 自管 Break/Continue，分支末端走 .complete out）。

func (r *ContainerRunner) execParallel(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	// data-in pin "n" (number, default 2)
	nBranches := int(r.pullNumber(n, "n", 2))
	if nBranches <= 0 {
		nBranches = 2
	}
	var wg sync.WaitGroup
	var firstErr atomic.Value // 存第一个 sub-flow error（用 *errBox 包一下保证可比）
	for i := 0; i < nBranches; i++ {
		pin := fmt.Sprintf("branch%d", i)
		seeds := r.edges.next(n.ID+"."+pin, tok.LoopStack)
		wg.Add(1)
		go func(start []ExecToken) {
			defer wg.Done()
			if err := r.runSubFlow(ctx, start); err != nil && !errors.Is(err, context.Canceled) {
				firstErr.CompareAndSwap(nil, &errBox{err: err})
			}
		}(seeds)
	}
	wg.Wait()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if v := firstErr.Load(); v != nil {
		return nil, v.(*errBox).err
	}
	return r.edges.next(n.ID+".complete", tok.LoopStack), nil
}

// errBox 给 atomic.Value 用：CompareAndSwap 要求统一类型，error 接口不能直接比。
type errBox struct{ err error }

func (r *ContainerRunner) execRace(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	// data-in pin "n" (number, default 2)
	nBranches := int(r.pullNumber(n, "n", 2))
	if nBranches <= 0 {
		nBranches = 2
	}
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var winner atomic.Int64
	winner.Store(-1)
	var wg sync.WaitGroup
	nonEmptyBranches := 0

	for i := 0; i < nBranches; i++ {
		pin := fmt.Sprintf("branch%d", i)
		seeds := r.edges.next(n.ID+"."+pin, tok.LoopStack)
		// 跳过空 branch（避免空 sub-flow 立即"获胜"导致用户搞错连线却感觉不到 bug）
		if len(seeds) == 0 {
			continue
		}
		nonEmptyBranches++
		wg.Add(1)
		go func(idx int, start []ExecToken) {
			defer wg.Done()
			_ = r.runSubFlow(childCtx, start)
			if winner.CompareAndSwap(-1, int64(idx)) {
				cancel()
			}
		}(i, seeds)
	}
	if nonEmptyBranches == 0 {
		return nil, fmt.Errorf("Race %s: all branches empty/un-wired", n.ID)
	}
	wg.Wait()
	w := winner.Load()
	r.rt.UpdateSys(func(s *SysState) { s.WinnerIdx = w })
	if ctx.Err() != nil && errors.Is(ctx.Err(), context.Canceled) && w == -1 {
		return nil, ctx.Err()
	}
	return r.edges.next(n.ID+".complete", tok.LoopStack), nil
}

// runSubFlow Parallel/Race 子分支用的迷你 dispatch（与主 dispatch 同语义但分支独立）。
//
// v4 TODO (Determinism contract / spec §10.3): r.currentTick is shared across goroutines.
// If two parallel branches both call GetVar/GetSys during overlapping execNode windows,
// they'll race on the snapshot. Acceptable for now because Phase A pure-data nodes (GetVar/GetSys)
// are not yet wired into Parallel branch logic. When data-flow meets parallel branches
// (Phase B+ when evalDataSource is fully wired into more node kinds), switch r.currentTick to
// a ctx-value or pass snapshot as parameter to make it per-goroutine.
func (r *ContainerRunner) runSubFlow(ctx context.Context, seeds []ExecToken) error {
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
		// v4: capture per-exec-tick snapshot (sequential within this goroutine).
		// NOT goroutine-safe across parallel branches — see TODO above.
		r.currentTick = CaptureSnapshot(r.rt.Vars(), r.rt.Sys())
		out, err := r.execNode(ctx, node, tok)
		r.currentTick = nil
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

// ---- Variables ----

func (r *ContainerRunner) execSetVar(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	name := configString(n, "varName")
	if name == "" {
		return nil, fmt.Errorf("SetVar %s: missing varName", n.ID)
	}
	// v4: value comes from data-in pin (data edge or inline literal), not expr config.
	// Pre-A6: r.configExpr(n, "value"). New: r.pullDataPin.
	val, err := r.pullDataPin(n.ID, "value")
	if err != nil {
		return nil, fmt.Errorf("SetVar %s: pull value: %w", n.ID, err)
	}
	// scope (default "auto"):
	//   - "global" → rt.vars (cross-frame, container-wide).
	//   - "local"  → frame.LocalVars (frame 私有, 子图隔离).
	//   - "auto"   → 当前 frame.LocalVars 已有 name → 写 local; 否则写 rt.vars
	//                (find-or-create-global; 跟 Container.Vars 面板默认一致).
	// 2026-05-19 默认从 "local" 改成 "auto" — 见 evalGetVar 同款 commit msg.
	scope := configString(n, "scope")
	if scope == "" {
		scope = "auto"
	}
	switch scope {
	case "global":
		r.rt.SetVar(name, val)
	case "local":
		r.state.SetLocalVar(name, val)
	case "auto":
		if _, ok := r.state.GetLocalVarHere(name); ok {
			r.state.SetLocalVar(name, val)
		} else {
			r.rt.SetVar(name, val)
		}
	default:
		return nil, fmt.Errorf("SetVar %s: unknown scope %q", n.ID, scope)
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

func (r *ContainerRunner) execIncVar(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	name := configString(n, "varName")
	if name == "" {
		return nil, fmt.Errorf("IncVar %s: missing varName", n.ID)
	}
	// v4: delta comes from data-in pin (data edge or inline literal). Default 1 if unset.
	deltaV, err := r.pullDataPin(n.ID, "delta")
	if err != nil {
		return nil, fmt.Errorf("IncVar %s: pull delta: %w", n.ID, err)
	}
	delta := 1.0
	if deltaV != nil {
		delta, _ = expr.AsNumber(deltaV)
	}
	// scope 同 SetVar (default "auto" — 见 execSetVar 注释).
	scope := configString(n, "scope")
	if scope == "" {
		scope = "auto"
	}
	switch scope {
	case "global":
		if err := r.rt.IncVar(name, delta); err != nil {
			return nil, err
		}
	case "local":
		cur := 0.0
		if v, ok := r.state.GetLocalVarHere(name); ok {
			if f, ok2 := v.(float64); ok2 {
				cur = f
			}
		}
		r.state.SetLocalVar(name, cur+delta)
	case "auto":
		if v, ok := r.state.GetLocalVarHere(name); ok {
			cur := 0.0
			if f, ok2 := v.(float64); ok2 {
				cur = f
			}
			r.state.SetLocalVar(name, cur+delta)
		} else {
			if err := r.rt.IncVar(name, delta); err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("IncVar %s: unknown scope %q", n.ID, scope)
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

// ---- Template ----

func (r *ContainerRunner) execWaitTemplate(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	tmpl := configString(n, "template")
	// v4: timeoutMs/threshold via data-in pin.
	timeout := time.Duration(r.pullNumber(n, "timeoutMs", 5000)) * time.Millisecond
	threshold := r.pullNumber(n, "threshold", 0.85)
	// 默认 250ms 轮询（不是 100ms）：单次 capture+match 已有 5-50ms 开销，
	// 100ms 间隔下 30s 等待期 CPU 占用 30-50%。250ms 间隔 ~12%，反应延迟仍可控。
	pollInterval := 250 * time.Millisecond

	deadline := time.Now().Add(timeout)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		found, point, region, err := r.rt.Matcher.Detect(ctx, tmpl, threshold, nil)
		if err != nil {
			return nil, err
		}
		if found {
			r.rt.UpdateSys(func(s *SysState) {
				s.LastFound = true
				s.LastPoint = point
				s.LastRegion = region
			})
			return r.edges.next(n.ID+".found", tok.LoopStack), nil
		}
		if time.Now().After(deadline) {
			r.rt.UpdateSys(func(s *SysState) { s.LastFound = false })
			return r.edges.next(n.ID+".timeout", tok.LoopStack), nil
		}
		if err := execution.Sleep(ctx, pollInterval); err != nil {
			return nil, err
		}
	}
}

func (r *ContainerRunner) execCheckTemplate(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	tmpl := configString(n, "template")
	// v4: threshold via data-in pin.
	threshold := r.pullNumber(n, "threshold", 0.85)
	found, point, region, err := r.rt.Matcher.Detect(ctx, tmpl, threshold, nil)
	if err != nil {
		return nil, err
	}
	r.rt.UpdateSys(func(s *SysState) {
		s.LastFound = found
		s.LastPoint = point
		s.LastRegion = region
	})
	pin := "no"
	if found {
		pin = "yes"
	}
	return r.edges.next(n.ID+"."+pin, tok.LoopStack), nil
}

func (r *ContainerRunner) execClickTemplate(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	tmpl := configString(n, "template")
	// v4: timeoutMs/threshold via data-in pin.
	timeout := time.Duration(r.pullNumber(n, "timeoutMs", 5000)) * time.Millisecond
	threshold := r.pullNumber(n, "threshold", 0.85)
	button := configString(n, "button")
	if button == "" {
		button = "left"
	}
	deadline := time.Now().Add(timeout)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		found, point, region, err := r.rt.Matcher.Detect(ctx, tmpl, threshold, nil)
		if err != nil {
			return nil, err
		}
		if found {
			r.rt.UpdateSys(func(s *SysState) {
				s.LastFound = true
				s.LastPoint = point
				s.LastRegion = region
			})
			// click 阶段独占输入（InputBus.Lock）；detect 阶段不占
			r.rt.InputBus.Lock()
			err := r.rt.Input.Click(win.HWND(r.rt.Window.HWND), point.X, point.Y, button, 50)
			r.rt.InputBus.Unlock()
			if err != nil {
				return nil, err
			}
			return r.edges.next(n.ID+".done", tok.LoopStack), nil
		}
		if time.Now().After(deadline) {
			r.rt.UpdateSys(func(s *SysState) { s.LastFound = false })
			return r.edges.next(n.ID+".timeout", tok.LoopStack), nil
		}
		if err := execution.Sleep(ctx, 250*time.Millisecond); err != nil {
			return nil, err
		}
	}
}

// ---- Input Primitives ----
//
// 每条 input 节点外套 InputBus.Lock/Unlock 保证键鼠独占（多 worker 并发场景）。

func (r *ContainerRunner) execClickAt(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	// v4: x/y/durationMs via data-in pin.
	x := r.pullNumber(n, "xRatio", 0.5)
	y := r.pullNumber(n, "yRatio", 0.5)
	dur := int(r.pullNumber(n, "durationMs", 50))
	button := configString(n, "button")
	if button == "" {
		button = "left"
	}
	r.rt.InputBus.Lock()
	err := r.rt.Input.Click(win.HWND(r.rt.Window.HWND), x, y, button, dur)
	r.rt.InputBus.Unlock()
	if err != nil {
		return nil, err
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

func (r *ContainerRunner) execKeyPress(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	vk := configString(n, "vk")
	// v4: durationMs via data-in pin.
	dur := int(r.pullNumber(n, "durationMs", 50))
	r.rt.InputBus.Lock()
	err := r.rt.Input.KeyPress(win.HWND(r.rt.Window.HWND), vk, dur)
	r.rt.InputBus.Unlock()
	if err != nil {
		return nil, err
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

func (r *ContainerRunner) execMouseMoveRel(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	// v4: dx/dy/durationMs via data-in pin.
	dx := int(r.pullNumber(n, "dx", 0))
	dy := int(r.pullNumber(n, "dy", 0))
	dur := int(r.pullNumber(n, "durationMs", 200))
	// v2 spec §2.7: scale = target / source. target = 主图 MouseCalibration（启动 snapshot），
	// source = 当前 frame chain 中最近的 Subgraph.RecordingContext.MouseCounts360。
	// 任一为 0 → scale = 1（手动组装 subgraph / 未校准 → 原值回放，跟旧 v1 行为兼容）。
	target := r.state.CalibCounts
	source := r.state.ResolveSourceCalibCounts()
	if target > 0 && source > 0 && target != source {
		scale := float64(target) / float64(source)
		dx = int(roundAwayFromZero(float64(dx) * scale))
		dy = int(roundAwayFromZero(float64(dy) * scale))
	}
	r.rt.InputBus.Lock()
	err := r.rt.Input.MouseMoveRel(win.HWND(r.rt.Window.HWND), dx, dy, dur)
	r.rt.InputBus.Unlock()
	if err != nil {
		return nil, err
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

// roundAwayFromZero 模拟旧 actions runtime 的取整行为（state.go:77-86）：
// 正数 +0.5、负数 -0.5 后 int 截断，结果远离 0。比 math.Round 更接近"四舍五入到整数"。
func roundAwayFromZero(f float64) float64 {
	if f < 0 {
		return f - 0.5
	}
	return f + 0.5
}

func (r *ContainerRunner) execScroll(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	// v4: x/y/delta via data-in pin.
	x := r.pullNumber(n, "xRatio", 0.5)
	y := r.pullNumber(n, "yRatio", 0.5)
	delta := int(r.pullNumber(n, "delta", 3))
	r.rt.InputBus.Lock()
	err := r.rt.Input.Scroll(win.HWND(r.rt.Window.HWND), x, y, delta)
	r.rt.InputBus.Unlock()
	if err != nil {
		return nil, err
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

// ---- Debug ----

func (r *ContainerRunner) execLog(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	level := configString(n, "level")
	if level == "" {
		level = "info"
	}
	// v4: message via data-in pin (any → FormatValue). No v3 expr fallback.
	msgV := r.pullValue(n, "message")
	if r.rt.Emit != nil {
		r.rt.Emit("container:log", map[string]any{
			"level":   level,
			"message": expr.FormatValue(msgV),
		})
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

func (r *ContainerRunner) execToast(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	// v4: title/message via data-in pin (any). No v3 expr fallback.
	titleV := r.pullValue(n, "title")
	msgV := r.pullValue(n, "message")
	color := configString(n, "color")
	if r.rt.Emit != nil {
		r.rt.Emit("container:toast", map[string]any{
			"title":   expr.FormatValue(titleV),
			"message": expr.FormatValue(msgV),
			"color":   color,
		})
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

// ---- DetectColor ----
//
// 钓鱼 / 状态栏 / 像素颜色判断专用。比 CheckTemplate 轻量（不需要模板图），
// 适合"看 ROI 有没有 X 颜色"的场景。
//
// 配置：
//   - region: "x,y,w,h" CSV（客户区比例，全空 = 全屏）
//   - mode:   "hsv" | "rgb"（默认 hsv）
//   - range:  6 元 CSV，hsv: "hMin,hMax,sMin,sMax,vMin,vMax"
//                       rgb: "rMin,rMax,gMin,gMax,bMin,bMax"
//   - minPixels: expr → 命中阈值（默认 5）
//
// 输出：
//   - $sys.lastColor.count / cx / cy
//   - pin: yes（count >= minPixels） / no
func (r *ContainerRunner) execDetectColor(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	region := parseCSV4Float(configString(n, "region"))
	mode := configString(n, "mode")
	if mode == "" {
		mode = "hsv"
	}
	rng := parseCSV6Int(configString(n, "range"))
	// v4: minPixels via data-in pin.
	minPx := int(r.pullNumber(n, "minPixels", 5))

	count, cx, cy, err := r.rt.Color.Detect(ctx, region, mode, rng)
	if err != nil {
		return nil, err
	}
	r.rt.UpdateSys(func(s *SysState) {
		s.LastColorCount = int64(count)
		s.LastColorCenter = expr.Point{X: cx, Y: cy}
	})
	pin := "no"
	if count >= minPx {
		pin = "yes"
	}
	return r.edges.next(n.ID+"."+pin, tok.LoopStack), nil
}

// parseCSV4Float "0.4,0.55,0.2,0.05" → [0.4, 0.55, 0.2, 0.05]。
// 不够 4 个 / 解析失败的位置返 0。全 0 表示 "全屏"。
func parseCSV4Float(s string) [4]float64 {
	var out [4]float64
	if s == "" {
		return out
	}
	parts := strings.Split(s, ",")
	for i := 0; i < 4 && i < len(parts); i++ {
		v, _ := parseFloat(strings.TrimSpace(parts[i]))
		out[i] = v
	}
	return out
}

func parseCSV6Int(s string) [6]int {
	var out [6]int
	if s == "" {
		return out
	}
	parts := strings.Split(s, ",")
	for i := 0; i < 6 && i < len(parts); i++ {
		v, _ := parseInt(strings.TrimSpace(parts[i]))
		out[i] = v
	}
	return out
}

func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func parseInt(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// execSubgraph 进入子图：push frame + 切 dispatch edges/nodesByID 到子图 + 从 SubgraphInput.out 入。
// ⚠️ 已知限制：v1 不支持 Parallel/Race 分支里嵌套子图调用（r.edges / r.nodesByID 全局切换会被覆盖）。
// validator 后续加规则 SUBGRAPH_IN_PARALLEL_BRANCH 防御；本 task 接受这一限制。
func (r *ContainerRunner) execSubgraph(ctx context.Context, node *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	sg, err := ResolveSubgraphCall(r.rt.Container, node)
	if err != nil {
		return nil, err
	}

	// v4: pull each declared InputParam value from the call node's data-in pin BEFORE
	// switching dispatch views (we need parent graph's dataEdges/nodesByID to resolve).
	paramVals := make(map[string]any, len(sg.InputParams))
	for _, p := range sg.InputParams {
		v, perr := r.pullDataPin(node.ID, p.Name)
		if perr != nil {
			return nil, fmt.Errorf("Subgraph %s: pull input %q: %w", node.ID, p.Name, perr)
		}
		if v == nil && p.Default != nil {
			v = toExprValue(p.Default)
		}
		paramVals[p.Name] = v
	}

	// push frame
	r.state.PushFrame(container.SubgraphGraphRef(sg.ID), sg)

	// Populate frame.LocalParams (after push so we write the new frame, not parent).
	maps.Copy(r.state.CurrentFrame.LocalParams, paramVals)

	// 切 dispatch 视图到子图 (含 dataEdges — nested Subgraph 调用需要看子图内部的 data edges).
	r.edges = buildEdgeIndex(sg.Graph)
	r.dataEdges = buildDataEdgeIndex(sg.Graph)
	r.nodesByID = make(map[string]*container.GraphNode)
	for i := range sg.Graph.Nodes {
		n := &sg.Graph.Nodes[i]
		r.nodesByID[n.ID] = n
	}

	// 找 SubgraphInput 节点，从其 out 入子图
	var inputID string
	for _, n := range sg.Graph.Nodes {
		if n.Kind == "SubgraphInput" {
			inputID = n.ID
			break
		}
	}
	if inputID == "" {
		return nil, fmt.Errorf("子图 %s 缺 SubgraphInput 节点", sg.ID)
	}
	return r.edges.next(inputID+".out", tok.LoopStack), nil
}

// execSubgraphOutput 子图退出：pop frame + 切回父图 dispatch 视图 + 找父图调用方 declID 边的下游。
func (r *ContainerRunner) execSubgraphOutput(ctx context.Context, node *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	declID, _ := node.Config["declID"].(string)
	if declID == "" {
		return nil, fmt.Errorf("SubgraphOutput 节点 %s 缺 config.declID", node.ID)
	}
	if r.state.CurrentFrame == nil || r.state.CurrentFrame.Parent == nil {
		return nil, fmt.Errorf("SubgraphOutput 节点 %s 没 parent frame（已在主图）", node.ID)
	}
	parentFrame := r.state.CurrentFrame.Parent
	var parentGraph container.Graph
	if parentFrame.Graph.IsMain() {
		parentGraph = r.rt.Container.Graph
	} else if parentFrame.SubgraphRef != nil {
		parentGraph = parentFrame.SubgraphRef.Graph
	} else {
		return nil, fmt.Errorf("parent frame graph 不可用")
	}
	// 扫父图找 kind=Subgraph 且 config.subgraphId == 当前子图 ID 的节点
	var callNodeID string
	if r.state.CurrentFrame.SubgraphRef != nil {
		targetSGID := r.state.CurrentFrame.SubgraphRef.ID
		for _, n := range parentGraph.Nodes {
			if n.Kind == "Subgraph" {
				if sgid, _ := n.Config["subgraphId"].(string); sgid == targetSGID {
					callNodeID = n.ID
					break
				}
			}
		}
	}
	if callNodeID == "" {
		return nil, fmt.Errorf("找不到当前子图在父图的调用方节点")
	}
	downstreamRef := FindParentDownstreamByDeclID(parentGraph.Edges, callNodeID, declID)
	// pop frame 回父图
	r.state.PopFrame()
	// 切回父图 edges / dataEdges / nodesByID
	r.edges = buildEdgeIndex(parentGraph)
	r.dataEdges = buildDataEdgeIndex(parentGraph)
	r.nodesByID = make(map[string]*container.GraphNode)
	for i := range parentGraph.Nodes {
		n := &parentGraph.Nodes[i]
		r.nodesByID[n.ID] = n
	}
	if downstreamRef == "" {
		return nil, nil // 子图调用完成无下游 → dispatch 自然终止
	}
	parts := strings.SplitN(downstreamRef, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("downstreamRef format invalid: %q", downstreamRef)
	}
	return []ExecToken{{NodeID: parts[0], InPin: parts[1], LoopStack: copyLoops(tok.LoopStack)}}, nil
}

// execBringGameForeground 把游戏窗口置前台；重试逻辑在 bring_foreground.go。
func (r *ContainerRunner) execBringGameForeground(ctx context.Context, node *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	return r.execBringGameForegroundImpl(ctx, node, tok)
}

// ---- PlayClip ----
//
// 解析 clipID + 可选 keepRanges → 构造 ClipPlayer → 阻塞跑完 → 走 .out.
// 期间 InputBus.Lock 独占输入. ctx 取消 → Cancel ClipPlayer 释放 held keys.
func (r *ContainerRunner) execPlayClip(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	clipID := configString(n, "clipID")
	if clipID == "" {
		return nil, fmt.Errorf("PlayClip %s: missing clipID", n.ID)
	}
	if r.rt.ClipResolver == nil {
		return nil, fmt.Errorf("PlayClip %s: no ClipResolver injected", n.ID)
	}
	clip, ok := r.rt.ClipResolver.Resolve(clipID)
	if !ok {
		return nil, fmt.Errorf("PlayClip %s: clip %q not found", n.ID, clipID)
	}
	if r.rt.InputBackend == nil {
		return nil, fmt.Errorf("PlayClip %s: no InputBackend injected", n.ID)
	}

	// keepRanges: config.keepRanges = [{fromUs, toUs}, ...] (JSON 来料).
	var ranges []inputclip.Range
	if raw, ok := n.Config["keepRanges"].([]any); ok {
		for _, item := range raw {
			if m, ok := item.(map[string]any); ok {
				from := uint64(asFloat(m["fromUs"]))
				to := uint64(asFloat(m["toUs"]))
				ranges = append(ranges, inputclip.Range{FromUs: from, ToUs: to})
			}
		}
	}

	startMsg := fmt.Sprintf("PlayClip 开始: events=%d duration=%dms backend=%s mouseCounts360=%d→%d filterMode=%s",
		len(clip.Events), clip.DurationUs/1000, r.rt.InputBackend.Name(),
		clip.Meta.MouseCounts360, r.rt.MouseCounts360, clip.Meta.FilterMode)
	fmt.Println("[PlayClip] " + startMsg)
	if r.rt.Emit != nil {
		r.rt.Emit("container:log", map[string]any{"level": "info", "message": startMsg})
	}

	r.rt.InputBus.Lock()
	defer r.rt.InputBus.Unlock()

	p := clipruntime.NewClipPlayer(clip, ranges, r.rt.InputBackend, r.rt.ClipPolicy, r.rt.MouseCounts360,
		func() (int, int) {
			if r.rt.Window.HWND == 0 {
				return 0, 0 // 未解析窗口 → 不缩放
			}
			return r.rt.Window.ClientW, r.rt.Window.ClientH
		},
	)
	p.Start(ctx)

	select {
	case <-p.Done():
		err := p.Wait()
		doneMsg := fmt.Sprintf("PlayClip 完成: 发送 %d/%d events, err=%v", p.SentCount(), len(clip.Events), err)
		fmt.Println("[PlayClip] " + doneMsg)
		if r.rt.Emit != nil {
			level := "info"
			if err != nil && !errors.Is(err, clipruntime.ErrCancelled) {
				level = "error"
			}
			r.rt.Emit("container:log", map[string]any{"level": level, "message": doneMsg})
		}
		if err != nil && !errors.Is(err, clipruntime.ErrCancelled) {
			return nil, fmt.Errorf("PlayClip %s: %w", n.ID, err)
		}
	case <-ctx.Done():
		p.Cancel()
		<-p.Done()
		return nil, ctx.Err()
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

// asFloat 把 JSON 解出来的 any 转 float64. int / float64 / int64 都兼容.
func asFloat(v any) float64 {
	switch x := v.(type) {
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case float64:
		return x
	case float32:
		return float64(x)
	}
	return 0
}
