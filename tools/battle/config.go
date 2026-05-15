// Package battle 提供按 L 进编队 UI、点目标队伍、确认切换、ESC 退出的一次性流程。
// 由 GUI 的全局热键（Ctrl+Shift+1..6）触发，不构成长期 bot —— 没有状态机、没有 tick 循环，
// 单次调用 SwitchTeam 跑完即返回。
//
// 1080p 模板还没补，到时候照 720p 的 4 个 slot 在 templates.toml 加 [[battle.*]]
// 块、teamSlotROIs 加一行即可。
package battle

import "time"

// 跟 fish.constants 同款的 3 档延迟。Battle 不接 GUI cfg，写死即可。
const (
	delayShort = 30 * time.Millisecond  // UI post-settle
	delayMid   = 150 * time.Millisecond // 按键 hold
	delayLong  = 2 * time.Second        // UI 加载

	switchToastTimeout = 3 * time.Second        // 等「切换成功」toast 的总超时
	switchToastPoll    = 300 * time.Millisecond // toast 轮询间隔
	teamClickSettle    = 800 * time.Millisecond // 点队伍后等右侧角色刷新

	confNormal = float32(0.75)
)

// TeamCount 编队 UI 固定 6 个队伍槽位。
const TeamCount = 6

// teamSlotROIs 每个分辨率下 6 个队伍编号块的 bbox（数组按队伍号 1..6 顺序）。
// 720p：team2/team3 实测，其余按 y 步长 60、x=[58,102] 推（44×44 块）。
// 1080p：team1/team4 实测，其余按 y 步长 90、x=[87,153] 推（66×66 块，恰好 1.5× 720p）。
// 加新分辨率追加一行即可。
var teamSlotROIs = map[[2]int][TeamCount][4]int{
	{1280, 720}: {
		{58, 162, 102, 206},
		{58, 222, 102, 266},
		{58, 282, 102, 326},
		{58, 342, 102, 386},
		{58, 402, 102, 446},
		{58, 462, 102, 506},
	},
	{1920, 1080}: {
		{87, 243, 153, 309},
		{87, 333, 153, 399},
		{87, 423, 153, 489},
		{87, 513, 153, 579},
		{87, 603, 153, 669},
		{87, 693, 153, 759},
	},
}

// TeamClickPos 返回 target 队伍编号块的中心点（客户区像素），ok=false 说明本分辨率没配。
// target ∈ [1, TeamCount]。越界直接 panic：调用方是 GUI 热键映射，越界是 bug 不是用户输入。
func TeamClickPos(clientW, clientH, target int) (x, y int, ok bool) {
	if target < 1 || target > TeamCount {
		panic("battle: target out of range")
	}
	rois, ok := teamSlotROIs[[2]int{clientW, clientH}]
	if !ok {
		return 0, 0, false
	}
	r := rois[target-1]
	return (r[0] + r[2]) / 2, (r[1] + r[3]) / 2, true
}
