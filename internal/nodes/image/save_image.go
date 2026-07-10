package image

import (
	"os"
	"path/filepath"

	"github.com/yottaapp/yotta/internal/node"
)

func init() { node.Register(&SaveImage{}) }

const (
	siIn      = "In"
	siDone    = "Done"
	siFail    = "Fail"
	siInImage = "Image"
	siInPath  = "PathTemplate"
	siOutPath = "Path"
	siOutFile = "File"
)

type SaveImage struct{}

func (SaveImage) Spec() node.Spec {
	return node.Spec{
		Kind:     "SaveImage",
		Category: "Image",
		Inputs: []node.InputSpec{
			{Name: siIn, Type: node.TypeExec},
			{Name: siInImage, Type: "Image"},
			{Name: siInPath, Type: "String", Default: "{ts}.png", Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: siDone, Type: node.TypeExec, Data: []node.DataField{
				{Name: siOutPath, Type: "String"},
				{Name: siOutFile, Type: "File"},
			}},
			{Name: siFail, Type: node.TypeExec, Semantic: "error", Data: []node.DataField{
				{Name: "Error", Type: "String"},
				{Name: "Code", Type: "String"},
			}},
		},
	}
}

func (SaveImage) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	img, ok := in.Raw(siInImage).(node.Image)
	if !ok {
		return nil, node.Failf(node.CodeWriteFailed, nil, "SaveImage: 缺图像输入(接一个产出 Image 的节点)")
	}
	tmpl := in.String(siInPath)
	if tmpl == "" {
		tmpl = "{ts}.png"
	}
	if err := checkSafePath(tmpl); err != nil {
		return nil, node.Failf(node.CodeWriteFailed, err, "SaveImage: %v", err)
	}
	rel := withFormatExt(expandImageTemplate(tmpl, ctx.Now()), img.Format)

	abs, err := filepath.Abs(filepath.Join(imagesOutputRoot(), rel))
	if err != nil {
		return nil, node.Failf(node.CodeWriteFailed, err, "SaveImage resolve %s: %v", rel, err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, node.Failf(node.CodeWriteFailed, err, "SaveImage mkdir %s: %v", filepath.Dir(abs), err)
	}
	if err := os.WriteFile(abs, img.Data, 0o644); err != nil {
		return nil, node.Failf(node.CodeWriteFailed, err, "SaveImage write %s: %v", abs, err)
	}
	file, err := node.FileFromPath(abs)
	if err != nil {
		return nil, node.Failf(node.CodeWriteFailed, err, "SaveImage stat %s: %v", abs, err)
	}
	return ctx.Out(siDone).Set(siOutPath, abs).Set(siOutFile, file).Fire(), nil
}
