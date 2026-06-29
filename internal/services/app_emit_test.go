package services

import "testing"

// shouldMirrorToRootLog: container:* 事件镜像到 rootLog 做 post-mortem,
// 但高频/内部 plumbing (node-enter + node-dump 家族) 不镜像 — 否则每次节点执行刷一行 SYS "runtime event".
func TestShouldMirrorToRootLog(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"container:warning", true},
		{"container:node-error", true},
		{"container:log", true},
		{"container:node-enter", false},
		{"container:node-dump", false},       // 每节点执行一发 → 不能镜像 (用户撞到的刷屏)
		{"container:node-dump-batch", false}, // merger 已把它发前端; 文件另有 AppendDumpLine
		{"container:node-dump-flush", false}, // run 停止内部信号
		{"container:action-trace", false},    // 专用脱敏文件行; generic mirror 会泄漏 raw payload
		{"log:lines", false},                 // 非 container: 前缀
		{"hotkey:changed", false},
		{"", false},
	}
	for _, c := range cases {
		if got := shouldMirrorToRootLog(c.name); got != c.want {
			t.Errorf("shouldMirrorToRootLog(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}
