package gui

import "sync"

// guiControl 是 GUI 端 runctl.Control 实现。
// 用 sync.Cond 在 WaitUnpause 内阻塞，避免轮询 + 跨 goroutine 唤醒。
type guiControl struct {
	mu      sync.Mutex
	cond    *sync.Cond
	paused  bool
	stopped bool
}

// NewGUIControl 返回一个未暂停未停止的 Control。
// 每轮 bot run 用一个新的（或 Reset 旧的）。
func NewGUIControl() *guiControl {
	c := &guiControl{}
	c.cond = sync.NewCond(&c.mu)
	return c
}

func (c *guiControl) IsPaused() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.paused
}

func (c *guiControl) IsBack() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stopped
}

func (c *guiControl) Pause() {
	c.mu.Lock()
	c.paused = true
	c.mu.Unlock()
}

func (c *guiControl) Resume() {
	c.mu.Lock()
	c.paused = false
	c.cond.Broadcast()
	c.mu.Unlock()
}

func (c *guiControl) Stop() {
	c.mu.Lock()
	c.stopped = true
	c.cond.Broadcast()
	c.mu.Unlock()
}

func (c *guiControl) WaitUnpause() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for c.paused && !c.stopped {
		c.cond.Wait()
	}
	return !c.stopped
}

// Reset 把 control 状态归零。下次开始新 bot 跑前调用。
func (c *guiControl) Reset() {
	c.mu.Lock()
	c.paused = false
	c.stopped = false
	c.cond.Broadcast()
	c.mu.Unlock()
}
