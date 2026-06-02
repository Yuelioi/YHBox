package services

import (
	"fmt"
	"os"
	"path/filepath"
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
// 同时把每完整行原始 bytes 顺手写入 logs/yotta-YYYYMMDD.log (post-mortem 调试).
type LogSink struct {
	mu      sync.Mutex
	buf     strings.Builder // 累积未完整行
	ring    []string        // 最近完整行（GUI snapshot 用）
	pending []string        // 自上次 flush 起新增的行
	timer   *time.Timer
	seq     atomic.Uint64
	emit    func(LogLinesEvent) // flush 时调；外部装配（app.Emit）
	file    *os.File           // 当日日志文件 (logs/yotta-YYYYMMDD.log), 持续 append
	fileDay string             // file 是哪天的 (YYYYMMDD), 跨天自动 rotate
	fileDir string             // logs 目录路径
}

// NewLogSink 创建一个 sink。emit 可为 nil（测试用），nil 时仅维护 ring。
// logsDir 不空时启用 file 持久化 (按天 rotate). 空时仅 ring buffer.
func NewLogSink(emit func(LogLinesEvent)) *LogSink {
	return &LogSink{
		ring: make([]string, 0, ringCapacity),
		emit: emit,
	}
}

// SetFileWriter 启停 file 持久化. dir == "" 关; 非空开 (mkdir + 按天 rotate).
// 跨调用安全 — 关再开后续日志正常落新文件.
func (s *LogSink) SetFileWriter(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if dir == "" {
		// 关: 关现有文件, 清 dir 状态
		if s.file != nil {
			_ = s.file.Close()
			s.file = nil
		}
		s.fileDir = ""
		s.fileDay = ""
		return
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "LogSink: mkdir %s failed: %v\n", dir, err)
		return
	}
	s.fileDir = dir
	s.openTodayFileLocked()
}

func (s *LogSink) openTodayFileLocked() {
	day := time.Now().Format("20060102")
	if s.file != nil && s.fileDay == day {
		return
	}
	if s.file != nil {
		_ = s.file.Close()
		s.file = nil
	}
	path := filepath.Join(s.fileDir, "yotta-"+day+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "LogSink: open %s failed: %v\n", path, err)
		return
	}
	s.file = f
	s.fileDay = day
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

// appendLineLocked 持锁加一行到 ring + file (如启用).
func (s *LogSink) appendLineLocked(line string) {
	if len(s.ring) >= ringCapacity {
		copy(s.ring, s.ring[1:])
		s.ring[len(s.ring)-1] = line
	} else {
		s.ring = append(s.ring, line)
	}
	if s.fileDir != "" {
		s.openTodayFileLocked()
		if s.file != nil {
			_, _ = s.file.WriteString(line)
			_, _ = s.file.WriteString("\n")
		}
	}
}

// Close 关闭日志文件 (shutdown 时调).
func (s *LogSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file != nil {
		err := s.file.Close()
		s.file = nil
		return err
	}
	return nil
}

// Snapshot ring 内全部行（"\n" 连接）。测试 / 调试用。
func (s *LogSink) Snapshot() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Join(s.ring, "\n")
}
