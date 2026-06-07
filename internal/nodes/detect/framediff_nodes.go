// internal/nodes/detect/framediff_nodes.go
// WaitStable / WaitChange — 通用帧间像素差分感知节点 (通用基建).
//
// 共享一个降采样签名核心 (ctx.Vision().GridSignature + pkg/vision.Downsample): 每 poll
// 抓一张新帧 (无缓存, 同 DetectColorHSV) 降采样成 gridSize² RGB 签名, 跨 poll 比对:
//   - WaitStable: diff(prev,cur) 连续 N 帧 ≤ 阈 → 画面稳了 (防动画中误识别).
//   - WaitChange: diff(baseline,cur) ≥ 阈 → 画面变了 (弹窗/加载/卡死反面).
//
// 度量 (Metric) 可选 changed_ratio | mean_diff, 都归一 [0,1], 共用同一 Threshold 语义.
package detect

import (
	"encoding/json"
	"fmt"
	"time"

	"yotta/internal/node"
	"yotta/pkg/vision"
)

func init() {
	node.Register(&WaitStable{})
	node.Register(&WaitChange{})
}

type WaitStable struct{}
type WaitChange struct{}

// 共享 pin / 参数.
const (
	wfIn             = "In"
	wfROI            = "ROI"
	wfGridSize       = "GridSize"
	wfMetric         = "Metric"
	wfCellDelta      = "CellDelta"
	wfPollIntervalMs = "PollIntervalMs"
	wfTimeoutMs      = "TimeoutMs"
	wfDataValue      = "Value"

	metricChangedRatio = "changed_ratio"
	metricMeanDiff     = "mean_diff"

	wfGridMin     = 4
	wfGridMax     = 128
	wfPollClampMs = 10
)

// WaitStable 专属.
const (
	wsStableThreshold = "StableThreshold"
	wsStableFrames    = "StableFrames"
	wsOutStable       = "Stable"
	wsOutTimeout      = "Timeout"
)

// WaitChange 专属.
const (
	wcChangeThreshold = "ChangeThreshold"
	wcOutChanged      = "Changed"
	wcOutTimeout      = "Timeout"
)

// commonInputs 两节点共有的输入 (In/ROI/GridSize/Metric/CellDelta/Poll/Timeout).
func commonInputs() []node.InputSpec {
	return []node.InputSpec{
		{Name: wfIn, Type: "Exec"},
		{Name: wfROI, Type: "Geometry", Schema: node.GeometrySchema()},
		{Name: wfGridSize, Type: "Number", Default: json.Number("32"),
			Widget: node.WidgetSpec{Kind: "number"}},
		{Name: wfMetric, Type: "String", Default: metricChangedRatio,
			Widget: node.WidgetSpec{Kind: "dropdown",
				Props: node.MarshalProps(node.DropdownProps{
					Options: []node.EnumOption{
						{Value: metricChangedRatio},
						{Value: metricMeanDiff},
					}})}},
		{Name: wfCellDelta, Type: "Number", Default: json.Number("12"),
			Widget: node.WidgetSpec{Kind: "number"}},
		{Name: wfPollIntervalMs, Type: "Number", Default: json.Number("100"),
			Widget: node.WidgetSpec{Kind: "number"}},
		{Name: wfTimeoutMs, Type: "Number", Default: json.Number("5000"),
			Widget: node.WidgetSpec{Kind: "number"}},
	}
}

func valueExit(name string) node.OutputSpec {
	return node.OutputSpec{Name: name, Type: "Exec",
		Data: []node.DataField{{Name: wfDataValue, Type: "Number"}}}
}

func (WaitStable) Spec() node.Spec {
	return node.Spec{
		Kind:        "WaitStable",
		Category:    "Detect",
		NeedsWindow: true,
		Inputs: append(commonInputs(),
			node.InputSpec{Name: wsStableThreshold, Type: "Number", Default: json.Number("0.02"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			node.InputSpec{Name: wsStableFrames, Type: "Number", Default: json.Number("3"),
				Widget: node.WidgetSpec{Kind: "number"}},
		),
		Outputs: []node.OutputSpec{valueExit(wsOutStable), valueExit(wsOutTimeout)},
	}
}

func (WaitChange) Spec() node.Spec {
	return node.Spec{
		Kind:        "WaitChange",
		Category:    "Detect",
		NeedsWindow: true,
		Inputs: append(commonInputs(),
			node.InputSpec{Name: wcChangeThreshold, Type: "Number", Default: json.Number("0.05"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
		),
		Outputs: []node.OutputSpec{valueExit(wcOutChanged), valueExit(wcOutTimeout)},
	}
}

// frameDiffParams 两节点共用的解析结果.
type frameDiffParams struct {
	roi       node.Geometry
	grid      int
	metric    string
	cellDelta int
	pollDur   time.Duration
	deadline  time.Time // 零值 = 无限
}

func parseFrameDiffParams(in node.Inputs) (frameDiffParams, error) {
	roi := in.Geometry(wfROI)
	metric := in.String(wfMetric)
	if metric == "" {
		metric = metricChangedRatio
	}
	if metric != metricChangedRatio && metric != metricMeanDiff {
		return frameDiffParams{}, fmt.Errorf("unknown Metric %q (want %s|%s)", metric, metricChangedRatio, metricMeanDiff)
	}
	grid := in.Int(wfGridSize)
	if grid < wfGridMin {
		grid = wfGridMin
	}
	if grid > wfGridMax {
		grid = wfGridMax
	}
	pollMs := in.Int(wfPollIntervalMs)
	if pollMs < wfPollClampMs {
		pollMs = wfPollClampMs
	}
	p := frameDiffParams{
		roi:       roi,
		grid:      grid,
		metric:    metric,
		cellDelta: in.Int(wfCellDelta),
		pollDur:   time.Duration(pollMs) * time.Millisecond,
	}
	if timeoutMs := in.Int(wfTimeoutMs); timeoutMs > 0 {
		p.deadline = time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	}
	return p, nil
}

func frameDiffValue(metric string, a, b []uint8, cellDelta int) float64 {
	if metric == metricMeanDiff {
		return vision.GridMeanDiff(a, b)
	}
	return vision.GridChangedRatio(a, b, cellDelta)
}

// sleepPoll 等 poll 间隔, 期间 ctx cancel 直接返 false (调用方应退出).
func sleepPoll(ctx node.Ctx, d time.Duration) bool {
	select {
	case <-ctx.Context().Done():
		return false
	case <-time.After(d):
		return true
	}
}

func (WaitStable) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	p, err := parseFrameDiffParams(in)
	if err != nil {
		return nil, fmt.Errorf("WaitStable: %w", err)
	}
	stableThr := in.Float64(wsStableThreshold)
	stableFrames := in.Int(wsStableFrames)
	if stableFrames < 1 {
		stableFrames = 1
	}

	prev, err := ctx.Vision().GridSignature(p.roi, p.grid)
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "WaitStable capture: %v", err)
	}
	stableCount := 0
	var lastValue float64
	for {
		if !sleepPoll(ctx, p.pollDur) {
			return nil, ctx.Context().Err()
		}
		cur, cerr := ctx.Vision().GridSignature(p.roi, p.grid)
		if cerr != nil {
			return nil, node.Failf(node.CodeCaptureFailed, cerr, "WaitStable capture: %v", cerr)
		}
		v := frameDiffValue(p.metric, prev, cur, p.cellDelta)
		lastValue = v
		prev = cur
		if v <= stableThr {
			stableCount++
		} else {
			stableCount = 0
		}
		if stableCount >= stableFrames {
			return ctx.Out(wsOutStable).Set(wfDataValue, v).Fire(), nil
		}
		if !p.deadline.IsZero() && time.Now().After(p.deadline) {
			return ctx.Out(wsOutTimeout).Set(wfDataValue, lastValue).Fire(), nil
		}
	}
}

func (WaitChange) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	p, err := parseFrameDiffParams(in)
	if err != nil {
		return nil, fmt.Errorf("WaitChange: %w", err)
	}
	changeThr := in.Float64(wcChangeThreshold)

	baseline, err := ctx.Vision().GridSignature(p.roi, p.grid)
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "WaitChange capture: %v", err)
	}
	var lastValue float64
	for {
		if !sleepPoll(ctx, p.pollDur) {
			return nil, ctx.Context().Err()
		}
		cur, cerr := ctx.Vision().GridSignature(p.roi, p.grid)
		if cerr != nil {
			return nil, node.Failf(node.CodeCaptureFailed, cerr, "WaitChange capture: %v", cerr)
		}
		v := frameDiffValue(p.metric, baseline, cur, p.cellDelta)
		lastValue = v
		if v >= changeThr {
			return ctx.Out(wcOutChanged).Set(wfDataValue, v).Fire(), nil
		}
		if !p.deadline.IsZero() && time.Now().After(p.deadline) {
			return ctx.Out(wcOutTimeout).Set(wfDataValue, lastValue).Fire(), nil
		}
	}
}
