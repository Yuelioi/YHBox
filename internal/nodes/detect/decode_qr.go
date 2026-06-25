// internal/nodes/detect/decode_qr.go
// DecodeQR — 裁 ROI 后纯 Go 解码二维码 (gozxing), 返回首个 QR 内容 + 检出总数 + 定位点。spec §节点3。
package detect

import (
	"yotta/internal/node"
)

func init() { node.Register(&DecodeQR{}) }

type DecodeQR struct{}

const (
	dqInExec      = "In"
	dqInROI       = "ROI"
	dqOutFound    = "Found"
	dqOutNotFound = "NotFound"
	dqDataText    = "Text"
	dqDataCount   = "Count"
	dqDataPoints  = "Points"
)

func (DecodeQR) Spec() node.Spec {
	return node.Spec{
		Kind:        "DecodeQR",
		Category:    "Detect",
		NeedsWindow: true,
		Inputs: append([]node.InputSpec{
			{Name: dqInExec, Type: "Exec"},
			{Name: dqInROI, Type: "Geometry", Schema: node.GeometrySchema()},
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: dqOutFound, Type: "Exec",
				Data: []node.DataField{
					{Name: dqDataText, Type: "String"},
					{Name: dqDataCount, Type: "Number"},
					{Name: dqDataPoints, Type: "JSON"},
				}},
			{Name: dqOutNotFound, Type: "Exec",
				Data: []node.DataField{{Name: dqDataCount, Type: "Number"}}},
		},
	}
}

func (DecodeQR) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	// 服务返回全部成功解码的 QR (adapter 已按左上排序); 取首个进 Found, Count 报总数。
	res, err := ctx.Vision().DecodeQR(in.Geometry(dqInROI))
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "DecodeQR: %v", err)
	}
	if len(res) == 0 {
		// 「没检出」与「检出但解码失败」都走 NotFound (adapter 已合并)。
		return ctx.Out(dqOutNotFound).Set(dqDataCount, 0).Fire(), nil
	}
	first := res[0]
	// 产出捕获由 Spec C config.capture 自动处理, 节点只 Set Data 字段。
	return ctx.Out(dqOutFound).
		Set(dqDataText, first.Text).
		Set(dqDataCount, len(res)).
		Set(dqDataPoints, first.Points).
		Fire(), nil
}
