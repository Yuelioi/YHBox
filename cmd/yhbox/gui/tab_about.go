package gui

import (
	"os/exec"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

// buildAboutTab 关于页：版本 + 作者 + 链接。
// 用 GroupBox 包裹是为了让内容横向撑满页面 —— 直接放 Label 的话，walk 的 staticLayoutItem
// 左对齐时不带 GrowableHorz，会缩成自然宽度，右半页留大片空白 (static.go:306-312)。
func buildAboutTab(version string) TabPage {
	return TabPage{
		Title:  "关于",
		Layout: VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 12}},
		Children: []Widget{
			GroupBox{
				Title: "YHBox " + version,
				// Alignment 显式靠上靠左：不设的话 VBox 默认 AlignHVDefault，boxlayout 走 default
				// 分支 halfExcessShare 居中 (boxlayout.go:497)；Label/LinkLabel 都是 LayoutFlags=0、
				// 自然宽度小于容器，看起来就全居中了。
				Layout: VBox{Spacing: 6, Alignment: AlignHNearVNear},
				Children: []Widget{
					Label{
						Text: "异环工具箱 — 自动钓鱼 / 锤子连点 / 自动弹琴 / 队伍切换",
						Font: Font{Family: "Microsoft YaHei", PointSize: 10},
					},
					Label{Text: "作者：月离"},
					LinkLabel{
						Text: `GitHub: <a id="gh" href="https://github.com/Yuelioi/YHBox">github.com/Yuelioi/YHBox</a>`,
						OnLinkActivated: func(link *walk.LinkLabelLink) {
							openURL(link.URL())
						},
					},
					LinkLabel{
						Text: `B 站: <a id="bili" href="https://space.bilibili.com/4279370">space.bilibili.com/4279370</a>`,
						OnLinkActivated: func(link *walk.LinkLabelLink) {
							openURL(link.URL())
						},
					},
					LinkLabel{
						Text: `QQ 群: <a id="qq" href="https://qm.qq.com/q/4xRKwgo3Ic">1025182488</a>`,
						OnLinkActivated: func(link *walk.LinkLabelLink) {
							openURL(link.URL())
						},
					},
					// HSpacer 自带 GrowableHorz|GreedyHorz，会被 boxLayoutFlags 一路 OR 上去
					// (boxlayout.go:242)，让外层 GroupBox 也有 GrowableHorz —— 这样在 TabPage
					// 的 VBox 里，GroupBox 的 cross-axis (横向) 才会被 boxLayoutItems 拉到 space2
					// 满宽 (boxlayout.go:466)。没这一行 GroupBox 缩成内容自然宽度，宽窗口右半空白。
					HSpacer{},
				},
			},
			GroupBox{
				Title:  "免责声明",
				Layout: VBox{Spacing: 4, Alignment: AlignHNearVNear},
				Children: []Widget{
					Label{Text: "1. 本工具仅供学习交流使用，禁止用于任何商业用途。"},
					Label{Text: "2. 本工具与《异环》游戏官方无任何关联，非官方制作。"},
					Label{Text: "3. 使用本工具可能违反游戏用户协议，由此产生的封号、"},
					Label{Text: "   数据丢失等一切后果由使用者自行承担。"},
					Label{Text: "4. 请在使用前自行评估风险，并遵守当地法律法规。"},
					Label{Text: "5. 若官方明确禁止第三方工具，请立即停止使用本工具。"},
					HSpacer{},
				},
			},
			VSpacer{},
		},
	}
}

// openURL 用默认浏览器打开 URL（Windows）。
func openURL(url string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}
