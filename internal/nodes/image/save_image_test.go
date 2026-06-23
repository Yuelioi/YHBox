package image

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"yotta/internal/node"
)

func runSaveImage(t *testing.T, img node.Image, tmpl string) node.RunResult {
	t.Helper()
	node.ResetRegistryForTest()
	node.Register(&SaveImage{})
	rn, _ := node.Get("SaveImage")
	return node.RunNode(context.Background(), rn,
		map[string]any{"Image": img}, map[string]any{"PathTemplate": tmpl}, nil, node.StubServices(), false)
}

func TestSaveImage_WritesWithFormatExt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YOTTA_DATA_DIR", dir)

	res := runSaveImage(t, node.Image{Format: "jpeg", Data: []byte("jpegbytes")}, "shot.png")
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("exit=%q err=%v", res.ExitName, res.Error)
	}
	p, _ := res.OutputData["Path"].(string)
	if filepath.Ext(p) != ".jpg" {
		t.Errorf("written ext = %q, want .jpg (Format 权威, 覆盖模板 .png)", filepath.Ext(p))
	}
	if !strings.Contains(p, filepath.Join(dir, "images")) {
		t.Errorf("path %q 不在 <dataDir>/images 下", p)
	}
	b, err := os.ReadFile(p)
	if err != nil || string(b) != "jpegbytes" {
		t.Errorf("file content = %q err=%v", b, err)
	}
}

func TestSaveImage_NoImage_Fails(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&SaveImage{})
	rn, _ := node.Get("SaveImage")
	res := node.RunNode(context.Background(), rn, nil,
		map[string]any{"PathTemplate": "x.png"}, nil, node.StubServices(), false)
	if res.Error == nil {
		t.Fatal("缺 Image 输入应失败")
	}
}

func TestSaveImage_UnsafePath_Fails(t *testing.T) {
	t.Setenv("YOTTA_DATA_DIR", t.TempDir())
	res := runSaveImage(t, node.Image{Format: "png", Data: []byte("x")}, "../escape.png")
	if res.Error == nil {
		t.Fatal("'..' 路径应失败")
	}
}
