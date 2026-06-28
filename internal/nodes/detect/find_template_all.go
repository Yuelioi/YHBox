// internal/nodes/detect/find_template_all.go
// FindTemplateAll — 单帧返回 ROI 内某些模板的全部命中 (NMS 去重)。spec §节点2。
package detect

import (
	"encoding/json"

	"yotta/internal/node"
)

func init() { node.Register(&FindTemplateAll{}) }

type FindTemplateAll struct{}

const (
	ftaInExec        = "In"
	ftaInTemplates   = "Templates"
	ftaInROI         = "ROI"
	ftaInThreshold   = "Threshold"
	ftaInMaxResults  = "MaxResults"
	ftaInMinDistance = "MinDistance"
	ftaOutFound      = "Found"
	ftaOutNotFound   = "NotFound"
	ftaDataMatches   = "Matches"
	ftaDataCount     = "Count"
	ftaDataPrimaryPt = "PrimaryPoint"
	ftaDataPrimaryCf = "PrimaryConf"
)

func (FindTemplateAll) Spec() node.Spec {
	return node.Spec{
		Kind:        "FindTemplateAll",
		Category:    "Detect",
		NeedsTarget: true,
		TargetCapabilities: []node.TargetCapability{
			node.TargetCapabilityScreenshot,
		},
		Inputs: append([]node.InputSpec{
			{Name: ftaInExec, Type: "Exec"},
			{Name: ftaInTemplates, Type: "String", Semantic: "TemplateGUID", Required: true,
				Widget: node.WidgetSpec{Kind: "template-picker"}},
			{Name: ftaInROI, Type: "Geometry", Schema: node.GeometrySchema()},
			{Name: ftaInThreshold, Type: "Number", Default: json.Number("0.85"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: ftaInMaxResults, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: ftaInMinDistance, Type: "Number", Default: json.Number("0"), Advanced: true,
				Widget: node.WidgetSpec{Kind: "number"}},
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: ftaOutFound, Type: "Exec",
				Data: []node.DataField{
					{Name: ftaDataMatches, Type: "JSON"},
					{Name: ftaDataCount, Type: "Number"},
					{Name: ftaDataPrimaryPt, Type: "Point"},
					{Name: ftaDataPrimaryCf, Type: "Number"},
				}},
			{Name: ftaOutNotFound, Type: "Exec",
				Data: []node.DataField{{Name: ftaDataCount, Type: "Number"}}},
		},
	}
}

func (FindTemplateAll) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	guids := in.StringList(ftaInTemplates)
	th := in.Float64(ftaInThreshold)
	if th < 0 {
		th = 0
	} else if th > 1 {
		th = 1
	}
	maxR := in.Int(ftaInMaxResults)
	if maxR < 0 {
		maxR = 0
	}
	minD := in.Int(ftaInMinDistance)
	if minD < 0 {
		minD = 0
	}

	matches, err := ctx.Vision().MatchAll(ctx.Context(), guids, th, minD, in.Geometry(ftaInROI))
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "FindTemplateAll: %v", err)
	}
	count := len(matches) // NMS 后总数, 不受 MaxResults 截断 (语义同 DetectColorBlobs.BlobCount)
	if count == 0 {
		return ctx.Out(ftaOutNotFound).Set(ftaDataCount, 0).Fire(), nil
	}
	out := matches
	if maxR > 0 && len(out) > maxR {
		out = out[:maxR] // 列表截断; Count 仍报总数
	}
	primary := matches[0] // MatchAll 已按 conf 降序 → 最高分; 恒在 out[:maxR] 内 (maxR>=1 或 0=不限)
	// 产出捕获走 Spec C config.capture (节点只 Set Data 字段)。
	return ctx.Out(ftaOutFound).
		Set(ftaDataMatches, out).
		Set(ftaDataCount, count).
		Set(ftaDataPrimaryPt, primary.Point).
		Set(ftaDataPrimaryCf, primary.Conf).
		Fire(), nil
}

func (FindTemplateAll) Dependencies(in node.Inputs) []node.Dependency {
	return templateDeps(in.StringList(ftaInTemplates))
}
