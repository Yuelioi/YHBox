package gui

import (
	"strings"
	"sync"
	"time"
)

// flushDebounce 是 LogSink 触发 onChange 的去抖延迟。
// 高频日志（fish 溜鱼时每秒多行）合并到 ~12fps，避免 SetText 把 UI 拖慢。
const flushDebounce = 80 * time.Millisecond

// LogSink 是实现 io.Writer 的 GUI 日志接收器。
//   - 接住 logger 写来的字节流，按 \n 切行
//   - 保持最近 capacity 行（仅供 Snapshot 调试/测试用）
//   - 节流：多次 Write 之后用 trailing-edge timer 合并触发一次 onNewLines
//     避免高频 walk 调用拖累 UI；onNewLines 只拿到 **本次新增** 的行，
//     由 GUI 增量 append 到 RichEdit（不需要全量重渲）
type LogSink struct {
	mu         sync.Mutex
	buf        strings.Builder // 累积未完整行
	lines      []string        // 最近完整行（ring，仅 Snapshot 用）
	pending    []string        // 自上次 flush 起新增的行
	capacity   int
	onNewLines func(lines []string)
	timer      *time.Timer // 当前排队中的 trailing-edge timer；nil 表示无 pending
}

// NewLogSink 创建一个 sink。capacity 限制 Snapshot 返回的行数（GUI 实际渲染数不受此限）。
// onNewLines 可为 nil（测试用）。每次 flush 调用一次，参数是自上次 flush 起新增的行（不带 \n）。
func NewLogSink(capacity int, onNewLines func(lines []string)) *LogSink {
	return &LogSink{
		lines:      make([]string, 0, capacity),
		capacity:   capacity,
		onNewLines: onNewLines,
	}
}

// Write 接 Logger 写来的字节。可能是部分行（没 \n 结尾）。
// 行追加到 ring 是同步的；onNewLines 通过 trailing-edge debounce 触发。
func (s *LogSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	s.buf.Write(p)
	current := s.buf.String()

	// 切出完整行
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

	// 防抖：如果已有 pending timer 就让它继续等，否则起一个新 timer
	if s.timer == nil && s.onNewLines != nil {
		s.timer = time.AfterFunc(flushDebounce, s.flush)
	}
	s.mu.Unlock()
	return len(p), nil
}

// flush 由 trailing-edge timer 触发，把累积的新行推给 onNewLines。
// delta 必须拷贝：cb 跨 goroutine (mw.Synchronize) 读，主路径 Write 会复用
// s.pending 的 underlying array — 不拷的话将来改 pending 预分配 cap 容易踩坑。
func (s *LogSink) flush() {
	s.mu.Lock()
	delta := append([]string(nil), s.pending...)
	s.pending = nil
	s.timer = nil
	cb := s.onNewLines
	s.mu.Unlock()

	if cb != nil && len(delta) > 0 {
		cb(delta)
	}
}

// appendLineLocked 在加锁前提下追加一行，超过 capacity 时丢弃最老。
func (s *LogSink) appendLineLocked(line string) {
	if len(s.lines) >= s.capacity {
		copy(s.lines, s.lines[1:])
		s.lines[len(s.lines)-1] = line
	} else {
		s.lines = append(s.lines, line)
	}
}

// Snapshot 返回当前 ring 内全部行（\n 连接）。测试 / 初次渲染用。
func (s *LogSink) Snapshot() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Join(s.lines, "\r\n")
}
