package calibration

// Service 暴露给 wails 前端。无状态——全局 calibration session 管在包级。
type Service struct{}

func NewService() *Service { return &Service{} }

// Start 启动校准（清零累积值，开始监听 raw mouse）。
func (s *Service) Start() error {
	Reset()
	return Start()
}

// Stop 停止校准并返回累积状态。
func (s *Service) Stop() (State, error) { return Stop() }

// Status 当前累积状态（前端 200ms poll 用）。
func (s *Service) Status() State { return Get() }
