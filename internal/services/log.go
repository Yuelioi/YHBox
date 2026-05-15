package services

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	flushDebounce = 80 * time.Millisecond
	maxBatchLines = 128 // 极端情况下立刻 flush，不等 debounce
	ringCapacity  = 500 // GUI snapshot 用，不影响 emit 量
)

// LogSink 实现 io.Writer。zerolog 输出 JSON Lines → 按 \n 切行 →
// trailing-edge debounce + maxBatch 双保险 → flush 时给 emit 加单调 seq → 推 log:lines。
type LogSink struct {
	mu      sync.Mutex
	buf     strings.Builder // 累积未完整行
	ring    []string        // 最近完整行（GUI snapshot 用）
	pending []string        // 自上次 flush 起新增的行
	timer   *time.Timer
	seq     atomic.Uint64
	emit    func(LogLinesEvent) // flush 时调；外部装配（app.Emit）
}

// NewLogSink 创建一个 sink。emit 可为 nil（测试用），nil 时仅维护 ring。
func NewLogSink(emit func(LogLinesEvent)) *LogSink {
	return &LogSink{
		ring: make([]string, 0, ringCapacity),
		emit: emit,
	}
}

// SetEmit 装配 emit 回调（main.go 在 wailsApp 构造完成后调）。
func (s *LogSink) SetEmit(emit func(LogLinesEvent)) {
	s.mu.Lock()
	s.emit = emit
	s.mu.Unlock()
}

// Write 接 zerolog 写来的字节。可能是部分行（没 \n 结尾）。
func (s *LogSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	s.buf.Write(p)
	current := s.buf.String()

	for {
		idx := strings.IndexByte(current, '\n')
		if idx < 0 {
			break
		}
		line := current[:idx]
		current = current[idx+1:]
		s.appendLineLocked(line)
		s.pending = append(s.pending, line)
	}
	s.buf.Reset()
	s.buf.WriteString(current)

	// maxBatch 触发：立刻 flush 不等 debounce
	if len(s.pending) >= maxBatchLines {
		s.flushLocked()
		s.mu.Unlock()
		return len(p), nil
	}

	// 否则 debounce：第一次 pending 时起 timer，已有 timer 不重置
	if s.timer == nil && s.emit != nil {
		s.timer = time.AfterFunc(flushDebounce, s.flushAsync)
	}
	s.mu.Unlock()
	return len(p), nil
}

// flushAsync 由 timer 触发。拿锁后调 flushLocked。
func (s *LogSink) flushAsync() {
	s.mu.Lock()
	s.flushLocked()
	s.mu.Unlock()
}

// flushLocked 在持锁状态下 flush pending 给 emit。
// 必须拷贝 pending 因为 cb 跨 goroutine 读，主路径会复用 underlying array。
func (s *LogSink) flushLocked() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if len(s.pending) == 0 {
		return
	}
	lines := append([]string(nil), s.pending...)
	s.pending = nil
	seq := s.seq.Add(1)
	cb := s.emit

	// 不在锁内调 cb（cb 可能 emit 后回调到这里）
	go func() {
		if cb != nil {
			cb(LogLinesEvent{Seq: seq, Lines: lines})
		}
	}()
}

// Flush 强制立即 flush（Shutdown 时用，避免最后几行日志丢）。
func (s *LogSink) Flush() {
	s.mu.Lock()
	s.flushLocked()
	s.mu.Unlock()
}

// appendLineLocked 持锁加一行到 ring。
func (s *LogSink) appendLineLocked(line string) {
	if len(s.ring) >= ringCapacity {
		copy(s.ring, s.ring[1:])
		s.ring[len(s.ring)-1] = line
	} else {
		s.ring = append(s.ring, line)
	}
}

// Snapshot ring 内全部行（"\n" 连接）。测试 / 调试用。
func (s *LogSink) Snapshot() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Join(s.ring, "\n")
}
