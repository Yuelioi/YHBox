package io

import (
	"bytes"
	"encoding/json"
	stdio "io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"

	"yotta/internal/node"
)

func init() {
	node.Register(&ReadTextFile{})
	node.Register(&ReadJsonFile{})
}

const (
	fileReadInPath     = "Path"
	fileReadInFile     = "File"
	fileReadInEncoding = "Encoding"
	fileReadInMaxBytes = "MaxBytes"
	fileReadOutDone    = "Done"
	fileReadOutFail    = "Fail"
	fileReadOutText    = "Text"
	fileReadOutJSON    = "JSON"
	fileReadOutFile    = "File"
	fileReadOutSize    = "Size"
	fileReadOutModTime = "ModTimeMs"

	defaultFileReadMaxBytes = 1 << 20
)

type ReadTextFile struct{}
type ReadJsonFile struct{}

func (ReadTextFile) Spec() node.Spec {
	return fileReadSpec("ReadTextFile", []node.DataField{
		{Name: fileReadOutText, Type: "String"},
		{Name: fileReadOutFile, Type: "File"},
		{Name: fileReadOutSize, Type: "Integer"},
		{Name: fileReadOutModTime, Type: "Integer"},
	})
}

func (ReadJsonFile) Spec() node.Spec {
	return fileReadSpec("ReadJsonFile", []node.DataField{
		{Name: fileReadOutJSON, Type: "JSON"},
		{Name: fileReadOutText, Type: "String"},
		{Name: fileReadOutFile, Type: "File"},
		{Name: fileReadOutSize, Type: "Integer"},
		{Name: fileReadOutModTime, Type: "Integer"},
	})
}

func fileReadSpec(kind string, doneData []node.DataField) node.Spec {
	return node.Spec{
		Kind:     kind,
		Category: "IO",
		Inputs: []node.InputSpec{
			{Name: "In", Type: node.TypeExec},
			{Name: fileReadInPath, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: fileReadInFile, Type: "File", Advanced: true, Widget: node.WidgetSpec{Kind: "file"}},
			{Name: fileReadInEncoding, Type: "String", Default: "auto",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "auto"},
							{Value: "utf-8"},
							{Value: "gbk"},
						}})}},
			{Name: fileReadInMaxBytes, Type: "Integer", Default: json.Number("1048576"), Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: fileReadOutDone, Type: node.TypeExec, Data: doneData},
			{Name: fileReadOutFail, Type: node.TypeExec, Semantic: "error", Data: []node.DataField{
				{Name: "Error", Type: "String"},
				{Name: "Code", Type: "String"},
			}},
		},
	}
}

func (ReadTextFile) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	text, info, err := readTextFileInput(in)
	if err != nil {
		return nil, err
	}
	return ctx.Out(fileReadOutDone).
		Set(fileReadOutText, text).
		Set(fileReadOutFile, info.file).
		Set(fileReadOutSize, int(info.size)).
		Set(fileReadOutModTime, info.modTimeMs).
		Fire(), nil
}

func (ReadJsonFile) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	text, info, err := readTextFileInput(in)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	var value any
	if err := dec.Decode(&value); err != nil {
		return nil, node.Failf(node.CodeError, err, "ReadJsonFile: JSON 解析失败: %v", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != stdio.EOF {
		return nil, node.Failf(node.CodeError, err, "ReadJsonFile: JSON 解析失败: 尾部有多余内容")
	}
	return ctx.Out(fileReadOutDone).
		Set(fileReadOutJSON, value).
		Set(fileReadOutText, text).
		Set(fileReadOutFile, info.file).
		Set(fileReadOutSize, int(info.size)).
		Set(fileReadOutModTime, info.modTimeMs).
		Fire(), nil
}

type readFileInfo struct {
	file      node.File
	size      int64
	modTimeMs int64
}

func readTextFileInput(in node.Inputs) (string, readFileInfo, error) {
	path := readInputPath(in)
	if path == "" {
		return "", readFileInfo{}, node.Failf(node.CodeNotFound, nil, "ReadFile: 路径为空")
	}
	abs := resolveIOPath(path)
	stat, err := os.Stat(abs)
	if err != nil {
		return "", readFileInfo{}, node.Failf(node.CodeNotFound, err, "ReadFile: 找不到文件 %s", path)
	}
	if stat.IsDir() {
		return "", readFileInfo{}, node.Failf(node.CodeError, nil, "ReadFile: 路径是目录 %s", path)
	}
	maxBytes := in.Int(fileReadInMaxBytes)
	if maxBytes <= 0 {
		maxBytes = defaultFileReadMaxBytes
	}
	if stat.Size() > int64(maxBytes) {
		return "", readFileInfo{}, node.Failf(node.CodeError, nil, "ReadFile: 文件超过 %d 字节上限", maxBytes)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", readFileInfo{}, node.Failf(node.CodeNotFound, err, "ReadFile read %s: %v", path, err)
	}
	text, err := decodeText(data, in.String(fileReadInEncoding))
	if err != nil {
		return "", readFileInfo{}, node.Failf(node.CodeError, err, "ReadFile: 解码失败: %v", err)
	}
	file, err := node.FileFromPath(abs)
	if err != nil {
		return "", readFileInfo{}, node.Failf(node.CodeError, err, "ReadFile stat %s: %v", path, err)
	}
	return text, readFileInfo{file: file, size: stat.Size(), modTimeMs: stat.ModTime().UnixMilli()}, nil
}

func readInputPath(in node.Inputs) string {
	if file, ok := in.File(fileReadInFile); ok && strings.TrimSpace(file.Path) != "" {
		return strings.TrimSpace(file.Path)
	}
	return strings.TrimSpace(in.String(fileReadInPath))
}

func resolveIOPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(ioDataRoot(), path)
}

func ioDataRoot() string {
	base := os.Getenv("YOTTA_DATA_DIR")
	if base == "" {
		base = filepath.Join("bin", "data")
	}
	return base
}

func decodeText(data []byte, encoding string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "", "auto":
		if utf8.Valid(data) {
			return string(data), nil
		}
		return simplifiedchinese.GBK.NewDecoder().String(string(data))
	case "utf-8", "utf8":
		if !utf8.Valid(data) {
			return "", bytes.ErrTooLarge
		}
		return string(data), nil
	case "gbk":
		return simplifiedchinese.GBK.NewDecoder().String(string(data))
	default:
		if utf8.Valid(data) {
			return string(data), nil
		}
		return simplifiedchinese.GBK.NewDecoder().String(string(data))
	}
}
