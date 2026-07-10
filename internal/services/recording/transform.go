package recording

// This file converts simple recording streams into container subgraphs.
// Simple mode pairs key and mouse transitions into one-shot actions, preserves
// scrolling, and inserts sleeps for idle gaps. It deliberately drops keyboard
// auto-repeat and cannot exactly reproduce concurrent input; precise mode keeps
// the InputClip event stream instead.

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/internal/services/inputclip"
)

// SleepThresholdUs 相邻 Step 间隔超过此值才插 Sleep 节点. 200ms 是人感知阈值,
// 比这短的间隔回放时合并掉, 既不影响行为也避免节点爆炸.
const SleepThresholdUs uint64 = 200_000

// stepNodeXSpacing GraphNode 横向间距 (px). 节点宽有硬上限 maxWidth 360 (ContainerFlowNode 的 CSS cap;
// ClickAt 实测 ~317), 所以间距只要 > 360 就数学上保证任何节点都不横排重叠 —— 440 给最宽节点也留 80+ 间隙.
// 这是简易录制子图布局的唯一来源 (前端不再二次重排). Y 全 200 同行.
const stepNodeXSpacing float32 = 440

// BuildSimpleSubgraph 把 simple 模式录制的事件流转成线性 Subgraph.
//
// clientW/H 是录制时游戏窗口客户区像素, 用来把 Event.B/C → ClickAt.xRatio/yRatio.
// 任一为 0 退化成 0.5 (中心), 不报错.
//
// 返回的 Subgraph 已含 SubgraphInput / SubgraphOutput / 线性 edges, 跟手搓的 subgraph
// 完全同构, 可直接喂 ContainerStore.SaveSubgraph.
func BuildSimpleSubgraph(events []inputclip.Event, meta inputclip.ClipMeta, clientW, clientH int, label string) container.Subgraph {
	steps := compactToSteps(events, clientW, clientH)
	rec := newRecordingContext(meta)
	return assembleSubgraph(steps, label, rec)
}

// stepNode 内部临时结构 — 表示一个待转 GraphNode 的 Step.
type stepNode struct {
	kind   string
	config map[string]any
}

// compactToSteps 事件流 → 高层 Step 序列. 配对规则见包注释.
func compactToSteps(events []inputclip.Event, clientW, clientH int) []stepNode {
	type pendingKey struct {
		firstDownUs uint64
	}
	type pendingBtn struct {
		downUs uint64
		x, y   int32
	}

	var steps []stepNode
	keys := map[int32]*pendingKey{}
	btns := map[int32]*pendingBtn{}
	var lastStepEndUs uint64
	hasLast := false

	// maybeInsertSleep 在 nextStartUs 之前若距上一 Step 结束超阈值, 先插 Sleep.
	maybeInsertSleep := func(nextStartUs uint64) {
		if !hasLast {
			return
		}
		if nextStartUs <= lastStepEndUs {
			return
		}
		gap := nextStartUs - lastStepEndUs
		if gap < SleepThresholdUs {
			return
		}
		steps = append(steps, stepNode{
			kind: "Sleep",
			config: map[string]any{
				// Sleep Spec 字段就是 Duration (Type=Duration), in.Duration() 把 float64 当 ms 解.
				"Duration": float64(gap) / 1000.0,
			},
		})
	}

	for _, ev := range events {
		switch ev.Type {
		case inputclip.EventTypeKeyDown:
			if _, exists := keys[ev.A]; exists {
				continue // auto-repeat (Windows 长按重发 KeyDown), 忽略
			}
			keys[ev.A] = &pendingKey{firstDownUs: ev.TUs}

		case inputclip.EventTypeKeyUp:
			pk, ok := keys[ev.A]
			if !ok {
				continue // orphan Up
			}
			delete(keys, ev.A)
			vkStr := vkName(uint32(ev.A))
			if vkStr == "" {
				continue // 不支持的键 (媒体键 / OEM 等)
			}
			maybeInsertSleep(pk.firstDownUs)
			dur := ev.TUs - pk.firstDownUs
			steps = append(steps, stepNode{
				kind: "KeyPress",
				config: map[string]any{
					"VK":         vkStr,
					"DurationMs": float64(dur) / 1000.0,
				},
			})
			lastStepEndUs = ev.TUs
			hasLast = true

		case inputclip.EventTypeMouseBtnDown:
			btns[ev.A] = &pendingBtn{downUs: ev.TUs, x: ev.B, y: ev.C}

		case inputclip.EventTypeMouseBtnUp:
			pb, ok := btns[ev.A]
			if !ok {
				continue
			}
			delete(btns, ev.A)
			btn := btnName(ev.A)
			if btn == "" {
				continue
			}
			xRatio, yRatio := toRatios(pb.x, pb.y, clientW, clientH)
			maybeInsertSleep(pb.downUs)
			dur := ev.TUs - pb.downUs
			steps = append(steps, stepNode{
				kind: "ClickAt",
				config: map[string]any{
					"XRatio":     xRatio,
					"YRatio":     yRatio,
					"Button":     btn,
					"DurationMs": float64(dur) / 1000.0,
				},
			})
			lastStepEndUs = ev.TUs
			hasLast = true

		case inputclip.EventTypeScroll:
			xRatio, yRatio := toRatios(ev.B, ev.C, clientW, clientH)
			maybeInsertSleep(ev.TUs)
			steps = append(steps, stepNode{
				kind: "Scroll",
				config: map[string]any{
					"XRatio": xRatio,
					"YRatio": yRatio,
					"Delta":  float64(ev.A),
				},
			})
			lastStepEndUs = ev.TUs
			hasLast = true

		default:
			// MouseMove / RawDelta / unknown — simple 模式不要
		}
	}
	return steps
}

// assembleSubgraph 把 stepNode 切片拼成线性 Subgraph.
// SubgraphInput/Output 不在 Graph.Nodes, 用 sg.Entry / sg.OutputPins[].NodeID metadata.
// 结构: entry(virtual) → step0 → step1 → ... → stepN-1 → output(virtual, declID, name="done")
// steps 为空时仍合法: entry 直连 output.
func assembleSubgraph(steps []stepNode, label string, rec *container.RecordingContext) container.Subgraph {
	now := time.Now().UTC()
	if label == "" {
		label = "录制 " + now.Local().Format("15:04")
	}

	outDeclID := uuid.NewString()
	inID := "entry-" + uuid.NewString()[:8]
	outID := "out-" + uuid.NewString()[:8]

	nodes := make([]container.GraphNode, 0, len(steps))
	edges := make([]container.GraphEdge, 0, len(steps)+1)

	// Pin 名约定: 所有节点 exec in=".In" / out=".Done" (含 Start / SubgraphInput 虚拟 entry).
	// virtual entry / output marker 也走同一套, runtime dispatch_v5.go 读 ".Done" 找 entry 下游边.
	prevID := inID
	for i, s := range steps {
		nid := fmt.Sprintf("n-%s-%d-%s", lowercaseFirst(s.kind), i, uuid.NewString()[:4])
		nodes = append(nodes, container.GraphNode{
			ID:        nid,
			Kind:      s.kind,
			X:         stepNodeXSpacing * float32(i+1),
			Y:         200,
			Config:    s.config,
			CreatedAt: now,
		})
		edges = append(edges, container.GraphEdge{
			From: prevID + ".Done",
			To:   nid + ".In",
		})
		prevID = nid
	}
	// 最后一条 edge 接到 output virtual marker
	edges = append(edges, container.GraphEdge{
		From: prevID + ".Done",
		To:   outID + ".In",
	})

	return container.Subgraph{
		ID:    container.NewSubgraphID(),
		Label: label,
		Graph: container.Graph{
			ID:            uuid.NewString(),
			SchemaVersion: container.GraphSchemaVersion,
			Nodes:         nodes,
			Edges:         edges,
		},
		Entry: container.SubgraphMarker{
			NodeID: inID,
			X:      0,
			Y:      200,
		},
		OutputPins: []container.SubgraphOutputDecl{{
			ID:     outDeclID,
			Name:   "done",
			NodeID: outID,
			X:      stepNodeXSpacing * float32(len(steps)+1),
			Y:      200,
		}},
		RecordingContext: rec,
		CreatedAt:        now,
	}
}

// newRecordingContext 录制 metadata. precise 模式 MouseCounts360 用 PlayClip 内回放校准;
// simple 模式不录 RawDelta 所以校准无效, 但 Resolution / RecordedAt 仍有意义 (溯源).
func newRecordingContext(meta inputclip.ClipMeta) *container.RecordingContext {
	return &container.RecordingContext{
		MouseCounts360: meta.MouseCounts360,
		Resolution:     meta.BaseResolution,
		RecordedAt:     time.Now().UTC().Format(time.RFC3339),
	}
}

// toRatios 客户区像素 → 0-1 比例 (ClickAt / Scroll 用). client 维度为 0 时退化中心.
func toRatios(x, y int32, clientW, clientH int) (float64, float64) {
	xr := 0.5
	yr := 0.5
	if clientW > 0 {
		xr = float64(x) / float64(clientW)
	}
	if clientH > 0 {
		yr = float64(y) / float64(clientH)
	}
	return xr, yr
}

// btnName ev.A (HookMouseBtn 0/1/2) → ClickAt.config.button 字符串.
// 跟 recording event contract 的 HookBtnLeft/Middle/Right 常量对齐.
func btnName(a int32) string {
	switch a {
	case int32(HookBtnLeft):
		return "left"
	case int32(HookBtnMiddle):
		return "middle"
	case int32(HookBtnRight):
		return "right"
	}
	return ""
}

// lowercaseFirst kind 名首字母小写当 node ID 前缀, 仅为可读性 (ID 后段 uuid 保证唯一).
func lowercaseFirst(s string) string {
	if s == "" {
		return s
	}
	b := []byte(s)
	if b[0] >= 'A' && b[0] <= 'Z' {
		b[0] += 32
	}
	return string(b)
}
