// internal/services/container/runtime/cron.go
//
// Cron 节点: 阻塞到 cron 表达式匹配的下次时间点, 然后走 tick 出口.
// 配合 Loop forever 组合 = 容器内部不漂移周期任务.
//
// 设计 spec: debug/docs/superpowers/specs/2026-05-19-cron-node-design.md
package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"

	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
)

// cronParser 支持秒字段的 6-field cron 解析器 (second minute hour dom month dow).
// 包级变量避免每次 execCron 重建.
var cronParser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// execCron 阻塞到 cron 表达式下一次匹配时间, 然后走 tick 出口.
// 强停语义跟 Sleep 一致 — execution.Sleep 处理 ctx 取消.
//
// 表达式来源: (1) data edge 上游 (2) n.Config["literal"]["expr"] (FE materialize spec.Defaults).
// Runtime 不 fallback spec.Defaults (源码 data_pull.go: pullDataPin 路径只到 nil).
// 解析失败 (静态或动态) → 容器终止, 同款 err 路径.
//
// 表达式格式: 6 字段 "sec min hour dom month dow" (e.g. "*/3 * * * * *" = 每 3 秒).
func (r *ContainerRunner) execCron(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	s := r.pullString(n, "expr")
	sched, err := cronParser.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("Cron %s: 表达式解析失败 %q: %w", n.ID, s, err)
	}
	now := time.Now()
	// delay<=0 防御: cron/v3 Next 文档保证返 > now, 但若时钟跳变 / NTP 大幅调整 → 不卡死;
	// 同时防 Loop forever 内重复进入 spin loop — 强制最小 1ms 间隔.
	delay := max(sched.Next(now).Sub(now), time.Millisecond)
	if err := execution.Sleep(ctx, delay); err != nil {
		return nil, err
	}
	return r.edges.next(n.ID+".tick", tok.LoopStack), nil
}
