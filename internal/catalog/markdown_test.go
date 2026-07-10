package catalog

import (
	"strings"
	"testing"

	_ "github.com/yottaapp/yotta/internal/nodes/all"
)

func TestMarkdown_RendersKnownNode(t *testing.T) {
	md := Markdown(BuildWithI18n())
	if md == "" {
		t.Fatal("Markdown 返回空")
	}
	if !strings.Contains(md, "### DetectColor") {
		t.Error("缺 DetectColor 节点标题")
	}
	// 出口携带数据列应把 Center(Point) 渲染出来 (正是此前要 grep 源码才知道的)。
	if !strings.Contains(md, "Center(Point)") {
		t.Error("缺 DetectColor.Yes 的 Center(Point) 渲染")
	}
}
