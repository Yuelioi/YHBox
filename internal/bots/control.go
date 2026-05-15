package bots

import "sync"

// botControl 是 bot run 期间的 pause/resume/stop 控制句柄。
// 用 sync.Cond 在 WaitUnpause 内阻塞，避免轮询 + 跨 goroutine 唤醒。
// 每轮 bot run 用一个新的（service.Start() 里 NewBotControl()）。
type botControl struct {
	mu      sync.Mutex
	cond    *sync.Cond
	paused  bool
	stopped bool
}

// NewBotControl 返回一个未暂停未停止的 control。
func NewBotControl() *botControl {
	c := &botControl{}
	c.cond = sync.NewCond(&c.mu)
	return c
}

func (c *botControl) IsPaused() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.paused
}

// IsBack 跟 runctl.Control 接口对齐（"是否要求 back/退出"）。
func (c *botControl) IsBack() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stopped
}

func (c *botControl) Pause() {
	c.mu.Lock()
	c.paused = true
	c.mu.Unlock()
}

func (c *botControl) Resume() {
	c.mu.Lock()
	c.paused = false
	c.cond.Broadcast()
	c.mu.Unlock()
}

func (c *botControl) Stop() {
	c.mu.Lock()
	c.stopped = true
	c.cond.Broadcast()
	c.mu.Unlock()
}

// WaitUnpause 阻塞直到 unpause 或 stop。返回 true=继续，false=已停止。
func (c *botControl) WaitUnpause() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for c.paused && !c.stopped {
		c.cond.Wait()
	}
	return !c.stopped
}

// Reset 状态归零。新一轮 bot 复用同一 control 时调。
func (c *botControl) Reset() {
	c.mu.Lock()
	c.paused = false
	c.stopped = false
	c.cond.Broadcast()
	c.mu.Unlock()
}
