// internal/nodes/detect/click_template.go
// ClickTemplate — 等模板出现 → 在命中位置鼠标点击. 命中或超时后单出口路由.
//
// 100ms 内部轮询 (visionPollInterval) + 可选 SettleMs 命中后稳定延迟 + 50ms click duration.
// 可选 MaxAttempts>1: 点完验证模板消失没, 没消失就重点 (每下间隔 RetryIntervalMs), 点满还在走 Timeout(Matched=true).
// ROI: 限定搜索区; OrderBy/Index: 多命中排序选第 N 个 (默认 score/0 = 最高分, 等价旧行为).
package detect

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"yotta/internal/node"
)

func init() { node.Register(&ClickTemplate{}) }

type ClickTemplate struct{}

const (
	clkInExec            = "In"
	clkInTemplates       = "Templates"
	clkInTimeoutMs       = "TimeoutMs"
	clkInThreshold       = "Threshold"
	clkInROI             = "ROI"
	clkInAnchor          = "Anchor"
	clkInOffsetX         = "OffsetX"
	clkInOffsetY         = "OffsetY"
	clkInButton          = "Button"
	clkInSettleMs        = "SettleMs"
	clkInMaxAttempts     = "MaxAttempts"
	clkInRetryIntervalMs = "RetryIntervalMs"
	clkInOrderBy         = "OrderBy"
	clkInIndex           = "Index"
	clkInKeys            = "Keys"
	clkInClickCount      = "ClickCount"
	clkOutDone           = "Done"
	clkOutTimeout        = "Timeout"
	clkDataPoint         = "Point"
	clkDataConf          = "Conf"
	clkDataMatched       = "Matched" // 命中与否 (bool) Data 字段 — 两出口都带, 供自动捕获 (Spec C)
)

// visionPollInterval: locateOnce 轮询间隔 (acquire 内部用).
const visionPollInterval = 100 * time.Millisecond

func (ClickTemplate) Spec() node.Spec {
	return node.Spec{
		Kind:        "ClickTemplate",
		Category:    "Detect",
		NeedsTarget: true,
		TargetCapabilities: []node.TargetCapability{
			node.TargetCapabilityScreenshot,
			node.TargetCapabilityClick,
		},
		NeedsForeground: true,
		Inputs: append([]node.InputSpec{
			{Name: clkInExec, Type: "Exec"},
			{Name: clkInTemplates, Type: "String", Semantic: "TemplateGUID", Required: true,
				Widget: node.WidgetSpec{Kind: "template-picker"}},
			{Name: clkInTimeoutMs, Type: "Number", Default: json.Number("5000"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: clkInThreshold, Type: "Number", Default: json.Number("0.85"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: clkInROI, Type: "Geometry", Schema: node.GeometrySchema()},
			{Name: clkInAnchor, Type: "String", Default: "center", Advanced: true,
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "topLeft"}, {Value: "topCenter"}, {Value: "topRight"},
							{Value: "midLeft"}, {Value: "center"}, {Value: "midRight"},
							{Value: "botLeft"}, {Value: "botCenter"}, {Value: "botRight"},
						}})}},
			{Name: clkInOffsetX, Type: "Number", Default: json.Number("0"), Advanced: true,
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: clkInOffsetY, Type: "Number", Default: json.Number("0"), Advanced: true,
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: clkInButton, Type: "String", Default: "left",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "left"},
							{Value: "right"},
							{Value: "middle"},
						}})}},
			{Name: clkInSettleMs, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: clkInMaxAttempts, Type: "Number", Default: json.Number("1"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: clkInRetryIntervalMs, Type: "Number", Default: json.Number("500"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: clkInOrderBy, Type: "String", Default: "score", Advanced: true,
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "score"},
							{Value: "horizontal"},
							{Value: "vertical"},
							{Value: "area"},
							{Value: "random"},
						}})}},
			{Name: clkInIndex, Type: "Integer", Default: json.Number("0"), Advanced: true,
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: clkInKeys, Type: "String", Default: "", Advanced: true,
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: clkInClickCount, Type: "Integer", Default: json.Number("1"), Advanced: true,
				Widget: node.WidgetSpec{Kind: "number"}},
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: clkOutDone, Type: "Exec",
				Data: []node.DataField{
					{Name: clkDataPoint, Type: "Point"},
					{Name: clkDataConf, Type: "Number"},
					{Name: clkDataMatched, Type: "Bool"},
				}},
			{Name: clkOutTimeout, Type: "Exec",
				Data: []node.DataField{
					{Name: clkDataConf, Type: "Number", Optional: true},
					{Name: clkDataMatched, Type: "Bool"},
				}},
		},
	}
}

// pickMatch 多命中按 orderBy 排序后取 index。score 不重排 (MatchAll 已 conf 降序)。
func pickMatch(matches []node.TemplateMatch, orderBy string, index int) (node.TemplateMatch, bool) {
	if index < 0 || index >= len(matches) {
		return node.TemplateMatch{}, false
	}
	sorted := make([]node.TemplateMatch, len(matches))
	copy(sorted, matches)
	switch orderBy {
	case "horizontal":
		sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].BBox[0] < sorted[j].BBox[0] })
	case "vertical":
		sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].BBox[1] < sorted[j].BBox[1] })
	case "area":
		sort.SliceStable(sorted, func(i, j int) bool {
			return sorted[i].BBox[2]*sorted[i].BBox[3] > sorted[j].BBox[2]*sorted[j].BBox[3]
		})
	case "random":
		rand.Shuffle(len(sorted), func(i, j int) { sorted[i], sorted[j] = sorted[j], sorted[i] })
	default: // score: 保持 MatchAll 的 conf 降序
	}
	return sorted[index], true
}

// locateOnce 单帧定位选中命中。默认 (score/0) 走单帧 WaitMatch (保留 variant.BBox 快速定位);
// 非默认走 MatchAll + pickMatch。Found=false 表本帧没选到。
func locateOnce(ctx node.Ctx, keys []string, threshold float64, roi node.Geometry, orderBy string, index int) (node.MatchHit, error) {
	if (orderBy == "" || orderBy == "score") && index == 0 {
		return ctx.Vision().WaitMatch(ctx.Context(), keys, threshold, roi, 0)
	}
	matches, err := ctx.Vision().MatchAll(ctx.Context(), keys, threshold, 0, roi)
	if err != nil {
		return node.MatchHit{}, err
	}
	tm, ok := pickMatch(matches, orderBy, index)
	if !ok {
		return node.MatchHit{}, nil
	}
	return node.MatchHit{Found: true, Point: tm.Point, BBox: tm.BBox, Conf: tm.Conf}, nil
}

// acquire 轮询 locateOnce 直到命中或 timeout。timeout<=0 单帧。记录见过的最高 conf 供 Timeout 诊断。
func acquire(ctx node.Ctx, keys []string, threshold float64, roi node.Geometry, orderBy string, index int, timeout time.Duration) (node.MatchHit, error) {
	hit, err := locateOnce(ctx, keys, threshold, roi, orderBy, index)
	if err != nil {
		return node.MatchHit{}, err
	}
	if hit.Found || timeout <= 0 {
		return hit, nil
	}
	deadline := ctx.Now().Add(timeout)
	bestConf := hit.Conf
	for {
		if err := ctx.Context().Err(); err != nil {
			return node.MatchHit{}, err
		}
		if err := waitOrCancel(ctx, visionPollInterval); err != nil {
			return node.MatchHit{}, err
		}
		hit, err = locateOnce(ctx, keys, threshold, roi, orderBy, index)
		if err != nil {
			return node.MatchHit{}, err
		}
		if hit.Conf > bestConf {
			bestConf = hit.Conf
		}
		if hit.Found {
			return hit, nil
		}
		if ctx.Now().After(deadline) {
			return node.MatchHit{Conf: bestConf}, nil
		}
	}
}

func (ClickTemplate) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	keys := in.StringList(clkInTemplates)
	threshold := in.Float64(clkInThreshold)
	roi := in.Geometry(clkInROI)
	anchor := in.String(clkInAnchor)
	offXRaw := in.Float64(clkInOffsetX)
	offYRaw := in.Float64(clkInOffsetY)
	orderBy := in.String(clkInOrderBy)
	index := in.Int(clkInIndex)
	timeout := time.Duration(in.Int(clkInTimeoutMs)) * time.Millisecond
	settle := time.Duration(in.Int(clkInSettleMs)) * time.Millisecond
	maxAttempts := in.Int(clkInMaxAttempts)
	retryInterval := time.Duration(in.Int(clkInRetryIntervalMs)) * time.Millisecond
	btn := in.String(clkInButton)
	if btn == "" {
		btn = "left"
	}
	modKeys := in.String(clkInKeys)
	clickCount := in.Int(clkInClickCount)
	if clickCount < 1 {
		clickCount = 1
	}

	// 解析偏移单位: |v|>1 = 像素 → 需 ClientSize 换算成 ratio; |v|<=1 直接用 ratio。
	offX, offY := offXRaw, offYRaw
	if offXRaw < -1 || offXRaw > 1 || offYRaw < -1 || offYRaw > 1 {
		if w, h, err := ctx.Window().ClientSize(); err == nil {
			offX = node.ResolveScalar(offXRaw, w)
			offY = node.ResolveScalar(offYRaw, h)
		}
	}
	// clickPt 把命中框按锚点+偏移算最终落点 (默认 center/0/0 = BBox 中心 = hit.Point)。
	clickPt := func(hit node.MatchHit) node.Point {
		return anchorPoint(hit.BBox, anchor, offX, offY)
	}

	// 获取: 轮询 locateOnce 到命中或超时 (统一两路径).
	hit, err := acquire(ctx, keys, threshold, roi, orderBy, index, timeout)
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "ClickTemplate wait %s: %v", strings.Join(keys, "+"), err)
	}
	if !hit.Found {
		return ctx.Out(clkOutTimeout).Set(clkDataConf, hit.Conf).Set(clkDataMatched, false).Fire(), nil
	}

	// settle: 等稳 + 重定位一次。
	if settle > 0 {
		if err := waitOrCancel(ctx, settle); err != nil {
			return nil, err
		}
		if h2, err := locateOnce(ctx, keys, threshold, roi, orderBy, index); err == nil && h2.Found {
			hit = h2
		}
	}

	if err := node.ClickWithMods(ctx, clickPt(hit), btn, modKeys, clickCount, 50); err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "ClickTemplate click %s @ (%.3f,%.3f): %v", strings.Join(keys, "+"), clickPt(hit).X, clickPt(hit).Y, err)
	}
	if maxAttempts <= 1 {
		return ctx.Out(clkOutDone).Set(clkDataPoint, clickPt(hit)).Set(clkDataConf, hit.Conf).Set(clkDataMatched, true).Fire(), nil
	}
	clicks := 1
	for {
		if err := waitOrCancel(ctx, retryInterval); err != nil {
			return nil, err
		}
		h2, err := locateOnce(ctx, keys, threshold, roi, orderBy, index)
		if err != nil {
			return nil, node.Failf(node.CodeCaptureFailed, err, "ClickTemplate recheck %s: %v", strings.Join(keys, "+"), err)
		}
		if !h2.Found {
			return ctx.Out(clkOutDone).Set(clkDataPoint, clickPt(hit)).Set(clkDataConf, hit.Conf).Set(clkDataMatched, true).Fire(), nil
		}
		if clicks >= maxAttempts {
			return ctx.Out(clkOutTimeout).Set(clkDataConf, h2.Conf).Set(clkDataMatched, true).Fire(), nil
		}
		hit = h2
		if err := node.ClickWithMods(ctx, clickPt(hit), btn, modKeys, clickCount, 50); err != nil {
			return nil, node.Failf(node.CodeCaptureFailed, err, "ClickTemplate click %s @ (%.3f,%.3f): %v", strings.Join(keys, "+"), clickPt(hit).X, clickPt(hit).Y, err)
		}
		clicks++
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// anchorPoint 按九宫格锚点 + ratio 偏移算落点 (归一化, clamp [0,1])。
// bbox=[x,y(左上),w,h]; offX/offY 已是 ratio (调用方经 ResolveScalar 换算)。
func anchorPoint(bbox [4]float64, anchor string, offX, offY float64) node.Point {
	var fx, fy float64
	switch anchor {
	case "topLeft":
		fx, fy = 0, 0
	case "topCenter":
		fx, fy = 0.5, 0
	case "topRight":
		fx, fy = 1, 0
	case "midLeft":
		fx, fy = 0, 0.5
	case "midRight":
		fx, fy = 1, 0.5
	case "botLeft":
		fx, fy = 0, 1
	case "botCenter":
		fx, fy = 0.5, 1
	case "botRight":
		fx, fy = 1, 1
	default: // center
		fx, fy = 0.5, 0.5
	}
	return node.Point{
		X: clamp01(bbox[0] + bbox[2]*fx + offX),
		Y: clamp01(bbox[1] + bbox[3]*fy + offY),
	}
}

func (ClickTemplate) Validate(in node.Inputs) []node.ValidationError {
	var errs []node.ValidationError
	btn := in.String(clkInButton)
	if btn != "" && btn != "left" && btn != "right" && btn != "middle" {
		errs = append(errs, node.ValidationError{
			Code:    "INVALID_MOUSE_BUTTON",
			Message: fmt.Sprintf("button %q not in left/right/middle", btn),
			Field:   clkInButton,
		})
	}
	if _, ok := node.ParseMods(in.String(clkInKeys)); !ok {
		errs = append(errs, node.ValidationError{
			Code:    "INVALID_MODIFIER_KEY",
			Message: "Keys 含非法修饰键 (仅 ctrl/shift/alt/win)",
			Field:   clkInKeys,
		})
	}
	if in.Int(clkInClickCount) < 1 {
		errs = append(errs, node.ValidationError{
			Code:    "INVALID_CLICK_COUNT",
			Message: "ClickCount 必须 >= 1",
			Field:   clkInClickCount,
		})
	}
	return errs
}

func (ClickTemplate) Dependencies(in node.Inputs) []node.Dependency {
	return templateDeps(in.StringList(clkInTemplates))
}
