// Package image — 图像节点: Capture(截屏产 Image)/ SaveImage(Image→本地)/ LoadImage(本地→Image)。
// Image 在图里作一等流动值(node.Image 编码字节), 经 exec-data 直连或捕获到变量传递。
package image

import (
	"bytes"
	"encoding/json"
	imagepkg "image"
	"image/jpeg"
	"image/png"

	"yotta/internal/node"
)

func init() { node.Register(&Capture{}) }

const (
	capIn        = "In"
	capDone      = "Done"
	capFail      = "Fail"
	capInROI     = "ROI"
	capInFormat  = "Format"
	capInQuality = "Quality"
	capOutImage  = "Image"
)

type Capture struct{}

func (Capture) Spec() node.Spec {
	return node.Spec{
		Kind:        "Capture",
		Category:    "Image",
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: capIn, Type: node.TypeExec},
			{Name: capInROI, Type: "Geometry", Schema: node.GeometrySchema()},
			{Name: capInFormat, Type: "String", Default: "png",
				Widget: node.WidgetSpec{Kind: "dropdown", Props: node.MarshalProps(node.DropdownProps{
					Options: []node.EnumOption{{Value: "png"}, {Value: "jpeg"}},
				})}},
			{Name: capInQuality, Type: "Integer", Default: json.Number("85"), Advanced: true},
		},
		Outputs: []node.OutputSpec{
			{Name: capDone, Type: node.TypeExec, Data: []node.DataField{
				{Name: capOutImage, Type: "Image"},
			}},
			{Name: capFail, Type: node.TypeExec, Semantic: "error", Data: []node.DataField{
				{Name: "Error", Type: "String"},
				{Name: "Code", Type: "String"},
			}},
		},
	}
}

func (Capture) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	// Geometry 零值 = 全帧; CaptureROI 内统一处理。
	pngData, err := ctx.Capture().CaptureROI(in.Geometry(capInROI))
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "Capture: %v", err)
	}

	format := in.String(capInFormat)
	if format == "" {
		format = "png"
	}
	data := pngData
	if format == "jpeg" {
		jpg, jerr := pngToJPEG(pngData, in.Int(capInQuality))
		if jerr != nil {
			return nil, node.Failf(node.CodeCaptureFailed, jerr, "Capture jpeg encode: %v", jerr)
		}
		data = jpg
	}
	return ctx.Out(capDone).Set(capOutImage, node.Image{Format: format, Data: data}).Fire(), nil
}

// pngToJPEG 解码 PNG 再编 JPEG。CaptureService 只给 PNG, jpeg 选项即在此转码(双编码已知低效)。
func pngToJPEG(pngData []byte, quality int) ([]byte, error) {
	if quality <= 0 || quality > 100 {
		quality = 85
	}
	var img imagepkg.Image
	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
