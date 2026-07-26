package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	flushDebounce  = 80 * time.Millisecond
	fileFlushDelay = 250 * time.Millisecond
	maxBatchLines  = 128 // 极端情况下立刻 flush，不等 debounce
	ringCapacity   = 500 // GUI snapshot 用，不影响 emit 量
	// A blocked presentation callback may accumulate one queued delivery. Keep
	// its newest lines bounded; the ring snapshot remains the recovery source.
	maxQueuedDeliveryLines = 4 * ringCapacity
	fileBufferSize         = 64 * 1024
)

// LogSink 实现 io.Writer。zerolog 输出 JSON Lines → 按 \n 切行 →
// trailing-edge debounce + maxBatch 双保险 → flush 时给 emit 加单调 seq → 推 log:batch。
// 同时把每完整行原始 bytes 顺手写入 logs/yotta-YYYYMMDD.log (post-mortem 调试).
type LogSink struct {
	mu               sync.Mutex
	buf              strings.Builder // 累积未完整行
	ring             []LogEntry      // 最近完整 entry（GUI snapshot 用）
	ringStart        int             // ring 满后最旧 entry 的索引
	pending          []LogEntry      // 自上次 flush 起新增的 entry
	timer            *time.Timer
	seq              atomic.Uint64
	emit             func(LogBatchEvent) // flush 时调；外部装配（app.Emit）
	streamEnabled    bool
	emitGeneration   uint64
	deliveries       []logDelivery // FIFO，保证 seq 按生成顺序交付
	delivering       bool
	lastDeliveryDone <-chan struct{}
	file             *os.File // 当日日志文件 (logs/yotta-YYYYMMDD.log), 持续 append
	fileBuf          *bufio.Writer
	fileTimer        *time.Timer
	fileDay          string // file 是哪天的 (YYYYMMDD), 跨天自动 rotate
	fileDir          string // logs 目录路径
	closed           bool
	closeOnce        sync.Once
	closeErr         error
}

type logDelivery struct {
	event          LogBatchEvent
	emit           func(LogBatchEvent)
	emitGeneration uint64
	droppedLines   uint64
	done           chan struct{}
}

// NewLogSink 创建一个 sink。emit 可为 nil（测试用），nil 时仅维护 ring。
// logsDir 不空时启用 file 持久化 (按天 rotate). 空时仅 ring buffer.
func NewLogSink(emit func(LogBatchEvent)) *LogSink {
	return &LogSink{
		ring:          make([]LogEntry, 0, ringCapacity),
		emit:          emit,
		streamEnabled: true,
	}
}

// SetStreamEnabled controls presentation capture. Disabling it drops pending
// and queued UI work immediately while leaving file persistence independent.
func (s *LogSink) SetStreamEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.streamEnabled == enabled {
		return
	}
	s.streamEnabled = enabled
	if enabled {
		return
	}
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	s.pending = nil
	s.ring = s.ring[:0]
	s.ringStart = 0
	s.emitGeneration++
	for i := range s.deliveries {
		s.deliveries[i].emit = nil
	}
}

// SetFileWriter 启停 file 持久化. dir == "" 关; 非空开 (mkdir + 按天 rotate).
// 跨调用安全 — 关再开后续日志正常落新文件.
func (s *LogSink) SetFileWriter(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}

	if dir == "" {
		// 关: 关现有文件, 清 dir 状态
		if s.file != nil {
			if s.fileBuf != nil {
				_ = s.fileBuf.Flush()
			}
			_ = s.file.Close()
			s.file = nil
			s.fileBuf = nil
		}
		s.fileDir = ""
		s.fileDay = ""
		if s.fileTimer != nil {
			s.fileTimer.Stop()
			s.fileTimer = nil
		}
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
		if s.fileBuf != nil {
			_ = s.fileBuf.Flush()
		}
		_ = s.file.Close()
		s.file = nil
		s.fileBuf = nil
	}
	path := filepath.Join(s.fileDir, "yotta-"+day+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "LogSink: open %s failed: %v\n", path, err)
		return
	}
	s.file = f
	s.fileBuf = bufio.NewWriterSize(f, fileBufferSize)
	s.fileDay = day
}

// SetEmit 装配 emit 回调（main.go 在 wailsApp 构造完成后调）。
func (s *LogSink) SetEmit(emit func(LogBatchEvent)) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	wasDetached := s.emit == nil
	s.emit = emit
	s.emitGeneration++
	if emit != nil && wasDetached && len(s.pending) == 0 {
		s.pending = s.ringEntriesLocked()
	}
	if emit != nil && s.streamEnabled && len(s.pending) > 0 && s.timer == nil {
		s.timer = time.AfterFunc(flushDebounce, s.flushAsync)
	}
	s.mu.Unlock()
}

// Write 接 zerolog 写来的字节。可能是部分行（没 \n 结尾）。
func (s *LogSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return 0, os.ErrClosed
	}
	if !s.streamEnabled && s.fileDir == "" {
		s.mu.Unlock()
		return len(p), nil
	}
	s.buf.Write(p)
	current := s.buf.String()
	urgentFileFlush := false

	for {
		idx := strings.IndexByte(current, '\n')
		if idx < 0 {
			break
		}
		line := current[:idx]
		current = current[idx+1:]
		s.appendFileLineLocked(line)
		if s.streamEnabled {
			entry := parseSystemLogEntry(line)
			s.appendEntryLocked(entry)
			urgentFileFlush = urgentFileFlush || entry.Level == "warn" || entry.Level == "error" || entry.Level == "fatal"
		} else if s.fileDir != "" {
			// File-only mode still flushes warnings/failures promptly.
			entry := parseSystemLogEntry(line)
			urgentFileFlush = urgentFileFlush || entry.Level == "warn" || entry.Level == "error" || entry.Level == "fatal"
		}
	}
	s.buf.Reset()
	s.buf.WriteString(current)
	if urgentFileFlush {
		s.flushFileLocked()
	}

	// maxBatch 触发：立刻 flush 不等 debounce
	if s.streamEnabled && len(s.pending) >= maxBatchLines {
		s.flushLocked()
		s.mu.Unlock()
		return len(p), nil
	}

	// 否则 debounce：第一次 pending 时起 timer，已有 timer 不重置
	if s.streamEnabled && s.timer == nil && s.emit != nil && len(s.pending) > 0 {
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
	if s.emit == nil {
		return
	}
	entries := append([]LogEntry(nil), s.pending...)
	s.pending = nil
	if count := len(s.deliveries); count > 0 {
		last := &s.deliveries[count-1]
		if last.emitGeneration == s.emitGeneration {
			s.appendQueuedEntriesLocked(last, entries)
			return
		}
	}
	seq := s.seq.Add(1)
	cb := s.emit

	done := make(chan struct{})
	delivery := logDelivery{
		event:          LogBatchEvent{Seq: seq},
		emit:           cb,
		emitGeneration: s.emitGeneration,
		done:           done,
	}
	s.appendQueuedEntriesLocked(&delivery, entries)
	s.lastDeliveryDone = done
	s.deliveries = append(s.deliveries, delivery)
	if !s.delivering {
		s.delivering = true
		go s.deliver()
	}
}

func (s *LogSink) appendQueuedEntriesLocked(delivery *logDelivery, entries []LogEntry) {
	combined := len(delivery.event.Entries) + len(entries)
	if combined <= maxQueuedDeliveryLines {
		delivery.event.Entries = append(delivery.event.Entries, entries...)
		return
	}

	overflow := combined - maxQueuedDeliveryLines
	delivery.droppedLines += uint64(overflow)
	bounded := make([]LogEntry, 0, maxQueuedDeliveryLines)
	if len(entries) >= maxQueuedDeliveryLines {
		bounded = append(bounded, entries[len(entries)-maxQueuedDeliveryLines:]...)
	} else {
		keepExisting := maxQueuedDeliveryLines - len(entries)
		existing := delivery.event.Entries
		bounded = append(bounded, existing[len(existing)-keepExisting:]...)
		bounded = append(bounded, entries...)
	}
	delivery.event.Entries = bounded
}

// deliver serializes callbacks without holding s.mu, preserving sequence order
// while allowing an emitter to write more logs without deadlocking the sink.
func (s *LogSink) deliver() {
	for {
		s.mu.Lock()
		if len(s.deliveries) == 0 {
			s.delivering = false
			s.deliveries = nil
			s.mu.Unlock()
			return
		}
		delivery := s.deliveries[0]
		s.deliveries[0] = logDelivery{}
		s.deliveries = s.deliveries[1:]
		s.mu.Unlock()

		if delivery.emit != nil {
			if delivery.droppedLines > 0 {
				delivery.event.Dropped = delivery.droppedLines
			}
			delivery.emit(delivery.event)
		}
		close(delivery.done)
	}
}

// Flush forces pending lines into the delivery queue without waiting for the
// presentation callback. This remains safe when called from an emit callback.
func (s *LogSink) Flush() {
	s.mu.Lock()
	s.flushLocked()
	s.flushFileLocked()
	s.mu.Unlock()
}

// drain flushes pending lines and waits for queued presentation callbacks.
// It is intentionally package-private: emit callbacks may call Flush or Close,
// but lifecycle code must be the sole caller of this non-reentrant barrier.
func (s *LogSink) drain() {
	s.mu.Lock()
	s.flushLocked()
	done := s.lastDeliveryDone
	s.mu.Unlock()
	if done != nil {
		<-done
	}
}

func (s *LogSink) appendEntryLocked(entry LogEntry) {
	if len(s.ring) >= ringCapacity {
		s.ring[s.ringStart] = entry
		s.ringStart = (s.ringStart + 1) % ringCapacity
	} else {
		s.ring = append(s.ring, entry)
	}
	if s.emit != nil {
		s.pending = append(s.pending, entry)
	}
}

func (s *LogSink) ringEntriesLocked() []LogEntry {
	if len(s.ring) < ringCapacity || s.ringStart == 0 {
		return append([]LogEntry(nil), s.ring...)
	}
	entries := make([]LogEntry, 0, len(s.ring))
	entries = append(entries, s.ring[s.ringStart:]...)
	entries = append(entries, s.ring[:s.ringStart]...)
	return entries
}

func (s *LogSink) appendFileLineLocked(line string) {
	if s.fileDir != "" {
		s.openTodayFileLocked()
		if s.fileBuf != nil {
			_, _ = s.fileBuf.WriteString(line)
			_ = s.fileBuf.WriteByte('\n')
			s.scheduleFileFlushLocked()
		}
	}
}

func (s *LogSink) scheduleFileFlushLocked() {
	if s.fileTimer == nil {
		s.fileTimer = time.AfterFunc(fileFlushDelay, s.flushFileAsync)
	}
}

func (s *LogSink) flushFileAsync() {
	s.mu.Lock()
	s.flushFileLocked()
	s.mu.Unlock()
}

func (s *LogSink) flushFileLocked() {
	if s.fileTimer != nil {
		s.fileTimer.Stop()
		s.fileTimer = nil
	}
	if s.fileBuf != nil {
		_ = s.fileBuf.Flush()
	}
}

func parseSystemLogEntry(line string) LogEntry {
	entry := LogEntry{Time: time.Now().Format(time.RFC3339Nano), Level: "info", Source: "SYS", Message: line}
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		entry.Level = "error"
		return entry
	}
	if value, ok := raw["time"].(string); ok {
		entry.Time = value
	}
	if value, ok := raw["level"].(string); ok {
		entry.Level = value
	}
	if value, ok := raw["tag"].(string); ok {
		entry.Tag = value
		if strings.EqualFold(value, "WORKFLOW") {
			entry.Source = "WF"
		}
	}
	if value, ok := raw["message"].(string); ok {
		entry.Message = value
	}
	if value, ok := raw["graphId"].(string); ok {
		entry.GraphID = value
	}
	if value, ok := raw["nodeId"].(string); ok {
		entry.NodeID = value
	}
	if value, ok := raw["invocationId"].(string); ok {
		entry.InvocationID = value
	}
	if value, ok := raw["attempt"].(float64); ok {
		entry.Attempt = int(value)
	}
	for _, key := range []string{"time", "level", "source", "tag", "message", "graphId", "nodeId", "invocationId", "attempt"} {
		delete(raw, key)
	}
	if len(raw) > 0 {
		entry.Fields = raw
	}
	return entry
}

// Close 关闭日志文件 (shutdown 时调).
func (s *LogSink) Close() error {
	s.closeOnce.Do(func() {
		s.Flush()
		s.mu.Lock()
		s.closed = true
		s.emit = nil
		s.emitGeneration++
		s.fileDir = ""
		s.fileDay = ""
		s.pending = nil
		s.ring = nil
		s.ringStart = 0
		if s.timer != nil {
			s.timer.Stop()
			s.timer = nil
		}
		if s.fileTimer != nil {
			s.fileTimer.Stop()
			s.fileTimer = nil
		}
		if s.file != nil {
			if s.fileBuf != nil {
				_ = s.fileBuf.Flush()
			}
			s.closeErr = s.file.Close()
			s.file = nil
			s.fileBuf = nil
		}
		s.mu.Unlock()
	})
	return s.closeErr
}

// Snapshot ring 内全部行（"\n" 连接）。测试 / 调试用。
func (s *LogSink) Snapshot() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := s.ringEntriesLocked()
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		encoded, _ := json.Marshal(entry)
		lines = append(lines, string(encoded))
	}
	return strings.Join(lines, "\n")
}
