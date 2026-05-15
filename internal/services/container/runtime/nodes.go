package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
	"yhbox/internal/services/expr"
)

// execNode 单节点执行入口。返回下游 token 列表（追加进调度队列）。
// 大部分节点产出 1 个或 0 个 token；Parallel/Race 是特例（自跑 sub-runner）。
func (r *ContainerRunner) execNode(ctx context.Context, node *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	switch node.Kind {
	case "Start":
		return r.edges.next(node.ID+".out", tok.LoopStack), nil
	case "Sleep":
		return r.execSleep(ctx, node, tok)
	case "Loop":
		return r.execLoop(ctx, node, tok)
	case "If":
		return r.execIf(ctx, node, tok)
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
	case "InvokeAction":
		return r.execInvokeAction(ctx, node, tok)
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
	}
	return nil, fmt.Errorf("container: unknown node kind %q", node.Kind)
}

// ---- Control Flow ----

func (r *ContainerRunner) execSleep(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	d := r.configFloat(n, "durationMs", 0)
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
		count := int64(r.configFloat(n, "count", 1))
		if frame.Iter >= count {
			return r.exitLoop(n, tok)
		}
	case "while":
		condV, err := r.configExpr(n, "condition")
		if err != nil {
			return nil, err
		}
		if !expr.AsBool(condV) {
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
	condV, err := r.configExpr(n, "condition")
	if err != nil {
		return nil, err
	}
	pin := "else"
	if expr.AsBool(condV) {
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
	nBranches := int(r.configFloat(n, "n", 2))
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
	nBranches := int(r.configFloat(n, "n", 2))
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

// ---- Variables ----

func (r *ContainerRunner) execSetVar(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	name := configString(n, "varName")
	if name == "" {
		return nil, fmt.Errorf("SetVar %s: missing varName", n.ID)
	}
	val, err := r.configExpr(n, "value")
	if err != nil {
		return nil, err
	}
	r.rt.SetVar(name, val)
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

func (r *ContainerRunner) execIncVar(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	name := configString(n, "varName")
	if name == "" {
		return nil, fmt.Errorf("IncVar %s: missing varName", n.ID)
	}
	delta := r.configFloat(n, "delta", 1)
	if err := r.rt.IncVar(name, delta); err != nil {
		return nil, err
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

// ---- Template ----

func (r *ContainerRunner) execWaitTemplate(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	tmpl := configString(n, "template")
	timeout := time.Duration(r.configFloat(n, "timeoutMs", 5000)) * time.Millisecond
	threshold := r.configFloat(n, "threshold", 0.85)
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
	threshold := r.configFloat(n, "threshold", 0.85)
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
	timeout := time.Duration(r.configFloat(n, "timeoutMs", 5000)) * time.Millisecond
	threshold := r.configFloat(n, "threshold", 0.85)
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
			err := r.rt.Input.Click(ctx, point.X, point.Y, button, 50)
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

// ---- Action ----

func (r *ContainerRunner) execInvokeAction(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	actionID := configString(n, "actionId")
	if actionID == "" {
		return nil, fmt.Errorf("InvokeAction %s: missing actionId", n.ID)
	}
	// params 走 config["params"] map[string]string（每 value 是 expr）
	params := make(map[string]expr.Value)
	if raw, ok := n.Config["params"]; ok {
		if m, ok := raw.(map[string]any); ok {
			for k, v := range m {
				if s, ok := v.(string); ok {
					ast, err := expr.Parse(s)
					if err != nil {
						return nil, fmt.Errorf("InvokeAction %s param %s: %w", n.ID, k, err)
					}
					ev, err := expr.Eval(ast, r.rt.Env())
					if err != nil {
						return nil, err
					}
					params[k] = ev
				} else {
					params[k] = v
				}
			}
		}
	}
	if err := r.rt.Actions.Invoke(ctx, actionID, params); err != nil {
		return nil, err
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

// ---- Input Primitives ----
//
// 每条 input 节点外套 InputBus.Lock/Unlock 保证键鼠独占（多 worker 并发场景）。

func (r *ContainerRunner) execClickAt(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	x := r.configFloat(n, "xRatio", 0.5)
	y := r.configFloat(n, "yRatio", 0.5)
	dur := int(r.configFloat(n, "durationMs", 50))
	button := configString(n, "button")
	if button == "" {
		button = "left"
	}
	r.rt.InputBus.Lock()
	err := r.rt.Input.Click(ctx, x, y, button, dur)
	r.rt.InputBus.Unlock()
	if err != nil {
		return nil, err
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

func (r *ContainerRunner) execKeyPress(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	vk := configString(n, "vk")
	dur := int(r.configFloat(n, "durationMs", 50))
	r.rt.InputBus.Lock()
	err := r.rt.Input.KeyPress(ctx, vk, dur)
	r.rt.InputBus.Unlock()
	if err != nil {
		return nil, err
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

func (r *ContainerRunner) execMouseMoveRel(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	dx := int(r.configFloat(n, "dx", 0))
	dy := int(r.configFloat(n, "dy", 0))
	dur := int(r.configFloat(n, "durationMs", 200))
	r.rt.InputBus.Lock()
	err := r.rt.Input.MouseMoveRel(ctx, dx, dy, dur)
	r.rt.InputBus.Unlock()
	if err != nil {
		return nil, err
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

func (r *ContainerRunner) execScroll(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	x := r.configFloat(n, "xRatio", 0.5)
	y := r.configFloat(n, "yRatio", 0.5)
	delta := int(r.configFloat(n, "delta", 3))
	r.rt.InputBus.Lock()
	err := r.rt.Input.Scroll(ctx, x, y, delta)
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
	msgV, err := r.configExpr(n, "message")
	if err != nil {
		return nil, err
	}
	if r.rt.Emit != nil {
		r.rt.Emit("container:log", map[string]any{
			"level":   level,
			"message": expr.FormatValue(msgV),
		})
	}
	return r.edges.next(n.ID+".out", tok.LoopStack), nil
}

func (r *ContainerRunner) execToast(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	titleV, _ := r.configExpr(n, "title")
	msgV, _ := r.configExpr(n, "message")
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
	minPx := int(r.configFloat(n, "minPixels", 5))

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
