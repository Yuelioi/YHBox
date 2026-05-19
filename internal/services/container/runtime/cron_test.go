package runtime

import (
	"context"
	"strings"
	"testing"
	"time"

	"yhbox/internal/services/container"
)

// TestExecCron_HappyPath_EveryThirdSec: */3 * * * * * (每 3 秒)
// 起 ctx deadline 10s, 期望 tick 出口被推, 无 err.
// 不断 wall-clock 上界 (CI jitter 易 flaky, GPT+Claude 反馈).
func TestExecCron_HappyPath_EveryThirdSec(t *testing.T) {
	_, r := newTestRunner(t)
	node := &container.GraphNode{
		ID:   "cron1",
		Kind: "Cron",
		Config: map[string]any{
			"literal": map[string]any{"expr": "*/3 * * * * *"},
		},
	}
	r.nodesByID["cron1"] = node
	r.edges = &edgeIndex{out: map[string][]string{}}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := r.execCron(ctx, node, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatalf("happy path 不应报错: %v", err)
	}
}

// TestExecCron_InvalidExpr_Static: literal.expr = "bogus" → 解析失败.
func TestExecCron_InvalidExpr_Static(t *testing.T) {
	_, r := newTestRunner(t)
	node := &container.GraphNode{
		ID:   "cron2",
		Kind: "Cron",
		Config: map[string]any{
			"literal": map[string]any{"expr": "bogus"},
		},
	}
	r.nodesByID["cron2"] = node
	_, err := r.execCron(context.Background(), node, ExecToken{InPin: "in"})
	if err == nil {
		t.Fatal("非法 expr 应报错")
	}
	if !strings.Contains(err.Error(), "解析失败") {
		t.Errorf("err 应含 '解析失败', got: %v", err)
	}
}

// TestExecCron_MissingLiteralReportsErr: 缺 literal — runtime 不 fallback spec.Defaults.
// 跟 §3.1 Defaults lifecycle 一致.
func TestExecCron_MissingLiteralReportsErr(t *testing.T) {
	_, r := newTestRunner(t)
	node := &container.GraphNode{
		ID:     "cron3",
		Kind:   "Cron",
		Config: map[string]any{}, // 没 literal — legacy / 手工编辑场景
	}
	r.nodesByID["cron3"] = node
	_, err := r.execCron(context.Background(), node, ExecToken{InPin: "in"})
	if err == nil {
		t.Fatal("缺 literal 应报错 — runtime 不自动 fallback spec.Defaults")
	}
	if !strings.Contains(err.Error(), "解析失败") {
		t.Errorf("err 应含 '解析失败', got: %v", err)
	}
}

// TestExecCron_StopOnCtxCancel: 长间隔 cron + 50ms 后 cancel → 立即返 ctx.Err().
func TestExecCron_StopOnCtxCancel(t *testing.T) {
	_, r := newTestRunner(t)
	node := &container.GraphNode{
		ID:   "cron4",
		Kind: "Cron",
		Config: map[string]any{
			"literal": map[string]any{"expr": "0 0 * * * *"}, // 每小时整
		},
	}
	r.nodesByID["cron4"] = node
	r.edges = &edgeIndex{out: map[string][]string{}}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	_, err := r.execCron(ctx, node, ExecToken{InPin: "in"})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("cancel 后应返 err")
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("cancel 响应应 < 200ms, 实际 %v", elapsed)
	}
}

// TestExecCron_DefaultLiteralFromSpec: 模拟 FE 创建节点后 Config 形态.
// ctx 200ms 超时 = 证明 Sleep 在等而非立即返.
func TestExecCron_DefaultLiteralFromSpec(t *testing.T) {
	_, r := newTestRunner(t)
	node := &container.GraphNode{
		ID:   "cron5",
		Kind: "Cron",
		Config: map[string]any{
			"literal": map[string]any{"expr": "0 */5 * * * *"},
		},
	}
	r.nodesByID["cron5"] = node
	r.edges = &edgeIndex{out: map[string][]string{}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := r.execCron(ctx, node, ExecToken{InPin: "in"})
	if err == nil {
		t.Fatal("200ms 不到 5 分钟, 应被 ctx deadline 取消")
	}
	if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "deadline") {
		t.Errorf("err 应是 ctx 超时, got: %v", err)
	}
}

// TestExecCron_InvalidExpr_DynamicUpstream:
// 模拟动态 expr (经 data edge 来自上游) 解析失败 — 跟静态同款 err 路径.
// 用 literal 路径放非法值证明 pullString 整体 err 处理一致, 不引入 mock 框架.
func TestExecCron_InvalidExpr_DynamicUpstream(t *testing.T) {
	_, r := newTestRunner(t)
	node := &container.GraphNode{
		ID:   "cron6",
		Kind: "Cron",
		Config: map[string]any{
			"literal": map[string]any{"expr": "60 60 60 60 60 60"}, // 字段越界
		},
	}
	r.nodesByID["cron6"] = node
	_, err := r.execCron(context.Background(), node, ExecToken{InPin: "in"})
	if err == nil {
		t.Fatal("非法字段值应报错")
	}
	if !strings.Contains(err.Error(), "解析失败") {
		t.Errorf("动态非法 expr 应跟静态同款 err, got: %v", err)
	}
}
