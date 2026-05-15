// Package runctl 定义 machine.run 跟外部交互的 pause/stop 控制接口。
// 实现可以是 GUI button（生产）或 mock（测试）。
package runctl

// Control 抽象 bot 主循环的 pause/stop 信号。
//
//   - IsPaused / IsBack: 非阻塞查询，machine.run loop 内每次迭代检查
//   - Pause / Resume / Stop: 由外部（GUI）调用，幂等
//   - WaitUnpause: 阻塞直到 Resume 或 Stop；返回 true=继续 false=停止
type Control interface {
	IsPaused() bool
	IsBack() bool
	Pause()
	Resume()
	Stop()
	WaitUnpause() bool
}
