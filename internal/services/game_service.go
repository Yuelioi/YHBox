package services

// GameService 给前端按需触发游戏窗口检测。
//
// 故意不做自动轮询：4 个触发点足够覆盖正常使用。
//   - main.go 启动时跑一次（hydrate 给前端）
//   - 用户在设置面板点"重新检测"按钮
//   - 前端切到任意 bot route（钓鱼/弹琴/...）的 onMounted
//   - BattleService.Enable 前自查（热键启用先确认窗口在）
// 轮询会浪费 CPU 还会让"游戏没启动"这种状态在前端反复闪动 toast。
type GameService struct {
	app *App
}

func NewGameService(app *App) *GameService { return &GameService{app: app} }

// Detect 触发一次检测，更新 App 缓存并 emit game:status。返回值也给同步调用者用。
func (s *GameService) Detect() GameStatusEvent {
	g := DetectGameWindow()
	evt := g.ToEvent()
	s.app.SetGame(evt)
	s.app.Emit(EventGameStatus, evt)
	return evt
}
