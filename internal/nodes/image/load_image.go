package image

import (
	"net/http"
	"os"
	"path/filepath"

	"yotta/internal/node"
)

func init() { node.Register(&LoadImage{}) }

const (
	liIn       = "In"
	liDone     = "Done"
	liFail     = "Fail"
	liInPath   = "Path"
	liOutImage = "Image"

	maxImageBytes = 10 << 20 // 10MB 上限, 防 OOM / 请求体过大
)

type LoadImage struct{}

func (LoadImage) Spec() node.Spec {
	return node.Spec{
		Kind:     "LoadImage",
		Category: "Image",
		Inputs: []node.InputSpec{
			{Name: liIn, Type: node.TypeExec},
			{Name: liInPath, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: liDone, Type: node.TypeExec, Data: []node.DataField{
				{Name: liOutImage, Type: "Image"},
			}},
			{Name: liFail, Type: node.TypeExec, Semantic: "error", Data: []node.DataField{
				{Name: "Error", Type: "String"},
				{Name: "Code", Type: "String"},
			}},
		},
	}
}

func (LoadImage) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	rel := in.String(liInPath)
	if rel == "" {
		return nil, node.Failf(node.CodeNotFound, nil, "LoadImage: 路径为空")
	}
	// 沙箱: 限 <dataDir> 下(比 SaveImage 宽 —— 可读任意子目录, 因 Load 读用户放进 dataDir
	// 任意位置的图; Save 专责写 images/)。仍拒绝绝对 / 盘符 / '..'。
	if err := checkSafePath(rel); err != nil {
		return nil, node.Failf(node.CodeNotFound, err, "LoadImage: %v", err)
	}
	abs := filepath.Join(dataRoot(), rel)
	info, err := os.Stat(abs)
	if err != nil {
		return nil, node.Failf(node.CodeNotFound, err, "LoadImage: 找不到文件 %s", rel)
	}
	if info.Size() > maxImageBytes {
		return nil, node.Failf(node.CodeError, nil, "LoadImage: 文件超过 10MB 上限")
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, node.Failf(node.CodeNotFound, err, "LoadImage read %s: %v", rel, err)
	}
	format, ok := sniffImageFormat(data)
	if !ok {
		return nil, node.Failf(node.CodeError, nil, "LoadImage: 仅支持 PNG / JPEG")
	}
	return ctx.Out(liDone).Set(liOutImage, node.Image{Format: format, Data: data}).Fire(), nil
}

// sniffImageFormat 按文件头判 png / jpeg(http.DetectContentType 读前 512 字节)。
func sniffImageFormat(data []byte) (string, bool) {
	switch http.DetectContentType(data) {
	case "image/png":
		return "png", true
	case "image/jpeg":
		return "jpeg", true
	}
	return "", false
}
