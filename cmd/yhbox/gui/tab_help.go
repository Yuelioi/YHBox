package gui

import (
	"strings"

	. "github.com/lxn/walk/declarative"
)

// buildHelpTab 静态使用说明页。
func buildHelpTab() TabPage {
	helpText := `使用说明

1. 启动游戏《异环》到任意位置（钓鱼点 / 一咖任务界面 / 钢琴前）
2. 双击 YHBox.exe 启动，UAC 提示点[是]获取管理员权限
3. 点 [开始] 启动；运行中可点 [暂停] / [继续] / [停止]
4. 配置自动保存到 exe 同目录的 settings.json，下次自动恢复
5. 日志会保存在logs里 如果有bug可以提交日志反馈

—————— 自动钓鱼 ——————

· 走到钓鱼点附近（右下角出现 "F" 图标即可）
· [自动出售鱼] 
    勾选：鱼仓满 (1000 条) 时自动开商店一键卖
    未勾选：满仓时弹窗暂停, 如果未勾选, 鱼饵钱不够也会自动暂停
     
· 启动后流程：抛竿 → 等上钩 → 按 F 收线 → 溜鱼 → 收杆 循环

—————— 锤子连点 ——————

有自动钓鱼了, 锤子功能目前比较简陋, 本质就是一个连点器

· 一咖任务，看到锤子图标即可
· [点击间隔]：两次点击之间的等待时间（毫秒）
· 默认 1500ms 即每 1.5 秒点一次

—————— 自动弹琴 ——————

· 在游戏内走到钢琴前打开弹奏界面（看到 1-7 数字键 + Shift/Ctrl 提示）
· 选 MIDI 文件
· [键位模式]：
    36 键 — 全半音（含 Shift 升半音、Ctrl 降半音），适合所有曲目
    21 键 — 只白键，遇到升降半音会自动跳过
· [自动对齐音域] 默认勾选：扫描全曲找最佳整体八度偏移，旋律轮廓不变形
· [只弹主旋律] 同时音只保留最高音，过滤伴奏：
    曲子复杂听感乱时打开 → 单线条干净
    简单单旋律曲不需要打开
· [微调] -2..+2：在自动偏移基础上再加减整体八度
· [进度条] 可拖动跳转
· 游戏键盘范围 3 个八度（MIDI 48-83），超出的音 ±1 八度兜底，仍不行则跳过

—————— 注意事项 ——————

· 仅支持 720p / 1080p 分辨率（钢琴功能不依赖分辨率）
· 切换游戏窗口大小后需在「设置」标签 点 [重新检测] 刷新分辨率`

	return TabPage{
		Title:  "帮助",
		Layout: VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 12}},
		Children: []Widget{
			// 用 TextEdit (native EDIT) 而不是 TextLabel — Label 自带 wrap 计算，每次
			// 拖窗口都会重新测量整段文本，长中文文本拖动会很卡。TextEdit 用原生控件渲染。
			//
			// MinSize 设在这里而不是 TabPage 上：walk.tabWidgetLayoutItem.MinSize() 用
			// page.MinSize()（layout 算出来），不查 MinSizeEffectiveForChild，所以 TabPage
			// 自己的 MinSize 字段被忽略；VBox 算时会读 child 的 geometry.MinSize 作为 floor。
			TextEdit{
				Text:          strings.ReplaceAll(helpText, "\n", "\r\n"),
				ReadOnly:      true,
				VScroll:       true,
				MinSize:       Size{Height: 400},
				StretchFactor: 1,
			},
		},
	}
}
