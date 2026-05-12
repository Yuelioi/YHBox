package gui

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

// buildSettingsTab 装配「设置」TabPage。
func buildSettingsTab(app *App) TabPage {
	var showLogBox, autoScrollBox, showTimeBox, showTagBox, wrapTextBox *walk.CheckBox

	return TabPage{
		Title:  "设置",
		Layout: VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 12}},
		Children: []Widget{
			GroupBox{
				Title:  "游戏窗口",
				Layout: VBox{},
				Children: []Widget{
					Label{AssignTo: &app.gameStatusLabel, Text: "正在检测..."},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							PushButton{Text: "重新检测", OnClicked: app.detectAndPropagate},
							HSpacer{},
						},
					},
				},
			},
			GroupBox{
				Title:  "日志",
				Layout: VBox{},
				Children: []Widget{
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							CheckBox{
								AssignTo: &showLogBox,
								Text:     "显示日志",
								Checked:  app.settings.UI.Logger.Show,
								OnCheckedChanged: func() {
									app.settings.UI.Logger.Show = showLogBox.Checked()
									app.SaveSettings()
									app.applyLogVisibilityAndShrink()
								},
							},
							HSpacer{},
						},
					},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							CheckBox{
								AssignTo: &autoScrollBox,
								Text:     "自动滚动到底部",
								Checked:  app.settings.UI.Logger.AutoScroll,
								OnCheckedChanged: func() {
									app.settings.UI.Logger.AutoScroll = autoScrollBox.Checked()
									app.SaveSettings()
								},
							},
							HSpacer{},
						},
					},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							CheckBox{
								AssignTo: &showTimeBox,
								Text:     "显示时间",
								Checked:  app.settings.UI.Logger.ShowTime,
								OnCheckedChanged: func() {
									app.settings.UI.Logger.ShowTime = showTimeBox.Checked()
									app.SaveSettings()
									app.rerenderLog() // 改前缀显示要重渲历史行，避免新旧割裂
								},
							},
							HSpacer{},
						},
					},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							CheckBox{
								AssignTo: &showTagBox,
								Text:     "显示状态标签",
								Checked:  app.settings.UI.Logger.ShowTag,
								OnCheckedChanged: func() {
									app.settings.UI.Logger.ShowTag = showTagBox.Checked()
									app.SaveSettings()
									app.rerenderLog()
								},
							},
							HSpacer{},
						},
					},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							CheckBox{
								AssignTo: &wrapTextBox,
								Text:     "自动换行",
								Checked:  app.settings.UI.Logger.WrapText,
								OnCheckedChanged: func() {
									app.settings.UI.Logger.WrapText = wrapTextBox.Checked()
									app.SaveSettings()
									if app.logEdit != nil {
										app.logEdit.SetWrapText(app.settings.UI.Logger.WrapText)
									}
								},
							},
							HSpacer{},
						},
					},
				},
			},
			VSpacer{},
		},
	}
}
