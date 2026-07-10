package io

import (
	"strings"

	"github.com/yottaapp/yotta/internal/node"
)

func init() { node.Register(&FileInfo{}) }

const (
	fileInfoInPath = "Path"
	fileInfoInFile = "File"

	fileInfoOutDone    = "Done"
	fileInfoOutFail    = "Fail"
	fileInfoOutFile    = "File"
	fileInfoOutPath    = "Path"
	fileInfoOutName    = "Name"
	fileInfoOutExt     = "Ext"
	fileInfoOutMIME    = "MIME"
	fileInfoOutSize    = "Size"
	fileInfoOutModTime = "ModTimeMs"
	fileInfoOutIsDir   = "IsDir"
)

type FileInfo struct{}

func (FileInfo) Spec() node.Spec {
	return node.Spec{
		Kind:     "FileInfo",
		Category: "IO",
		Inputs: []node.InputSpec{
			{Name: "In", Type: node.TypeExec},
			{Name: fileInfoInPath, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: fileInfoInFile, Type: "File", Advanced: true, Widget: node.WidgetSpec{Kind: "file"}},
		},
		Outputs: []node.OutputSpec{
			{Name: fileInfoOutDone, Type: node.TypeExec, Data: []node.DataField{
				{Name: fileInfoOutFile, Type: "File"},
				{Name: fileInfoOutPath, Type: "String"},
				{Name: fileInfoOutName, Type: "String"},
				{Name: fileInfoOutExt, Type: "String"},
				{Name: fileInfoOutMIME, Type: "String"},
				{Name: fileInfoOutSize, Type: "Integer"},
				{Name: fileInfoOutModTime, Type: "Integer"},
				{Name: fileInfoOutIsDir, Type: "Bool"},
			}},
			{Name: fileInfoOutFail, Type: node.TypeExec, Semantic: "error", Data: []node.DataField{
				{Name: "Error", Type: "String"},
				{Name: "Code", Type: "String"},
			}},
		},
	}
}

func (FileInfo) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	path := fileInfoInputPath(in)
	if path == "" {
		return nil, node.Failf(node.CodeNotFound, nil, "FileInfo: 路径为空")
	}
	file, err := node.FileFromPath(resolveIOPath(path))
	if err != nil {
		return nil, node.Failf(node.CodeNotFound, err, "FileInfo: 找不到文件 %s", path)
	}
	return ctx.Out(fileInfoOutDone).
		Set(fileInfoOutFile, file).
		Set(fileInfoOutPath, file.Path).
		Set(fileInfoOutName, file.Name).
		Set(fileInfoOutExt, file.Ext).
		Set(fileInfoOutMIME, file.MIME).
		Set(fileInfoOutSize, int(file.Size)).
		Set(fileInfoOutModTime, file.ModTimeMs).
		Set(fileInfoOutIsDir, file.IsDir).
		Fire(), nil
}

func fileInfoInputPath(in node.Inputs) string {
	if file, ok := in.File(fileInfoInFile); ok && strings.TrimSpace(file.Path) != "" {
		return strings.TrimSpace(file.Path)
	}
	return strings.TrimSpace(in.String(fileInfoInPath))
}
