package services

import (
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
	flushDebounce = 80 * time.Millisecond
	maxBatchLines = 128 // 极端情况下立刻 flush，不等 debounce
	ringCapacity  = 500 // GUI snapshot 用，不影响 emit 量
	// A blocked presentation callback may accumulate one queued delivery. Keep
	// its newest lines bounded; the ring snapshot remains the recovery source.
	maxQueuedDeliveryLines = 4 * ringCapacity
)

// LogSink 实现 io.Writer。zerolog 输出 JSON Lines → 按 \n 切行 →
// trailing-edge debounce + maxBatch 双保险 → flush 时给 emit 加单调 seq → 推 log:lines。
// 同时把每完整行原始 bytes 顺手写入 logs/yotta-YYYYMMDD.log (post-mortem 调试).
type LogSink struct {
	mu               sync.Mutex
	buf              strings.Builder // 累积未完整行
	ring             []string        // 最近完整行（GUI snapshot 用）
	pending          []string        // 自上次 flush 起新增的行
	timer            *time.Timer
	seq              atomic.Uint64
	emit             func(LogLinesEvent) // flush 时调；外部装配（app.Emit）
	emitGeneration   uint64
	deliveries       []logDelivery // FIFO，保证 seq 按生成顺序交付
	delivering       bool
	lastDeliveryDone <-chan struct{}
	file             *os.File // 当日日志文件 (logs/yotta-YYYYMMDD.log), 持续 append
	fileDay          string   // file 是哪天的 (YYYYMMDD), 跨天自动 rotate
	fileDir          string   // logs 目录路径
	closed           bool
	closeOnce        sync.Once
	closeErr         error
}

type logDelivery struct {
	event          LogLinesEvent
	emit           func(LogLinesEvent)
	emitGeneration uint64
	droppedLines   uint64
	done           chan struct{}
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
	if s.closed {
		return
	}

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
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.emit = emit
	s.emitGeneration++
	s.mu.Unlock()
}

// Write 接 zerolog 写来的字节。可能是部分行（没 \n 结尾）。
func (s *LogSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return 0, os.ErrClosed
	}
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
	if count := len(s.deliveries); count > 0 {
		last := &s.deliveries[count-1]
		if last.emitGeneration == s.emitGeneration {
			s.appendQueuedLinesLocked(last, lines)
			return
		}
	}
	seq := s.seq.Add(1)
	cb := s.emit

	done := make(chan struct{})
	delivery := logDelivery{
		event:          LogLinesEvent{Seq: seq},
		emit:           cb,
		emitGeneration: s.emitGeneration,
		done:           done,
	}
	s.appendQueuedLinesLocked(&delivery, lines)
	s.lastDeliveryDone = done
	s.deliveries = append(s.deliveries, delivery)
	if !s.delivering {
		s.delivering = true
		go s.deliver()
	}
}

func (s *LogSink) appendQueuedLinesLocked(delivery *logDelivery, lines []string) {
	combined := len(delivery.event.Lines) + len(lines)
	if combined <= maxQueuedDeliveryLines {
		delivery.event.Lines = append(delivery.event.Lines, lines...)
		return
	}

	overflow := combined - maxQueuedDeliveryLines
	delivery.droppedLines += uint64(overflow)
	bounded := make([]string, 0, maxQueuedDeliveryLines)
	if len(lines) >= maxQueuedDeliveryLines {
		bounded = append(bounded, lines[len(lines)-maxQueuedDeliveryLines:]...)
	} else {
		keepExisting := maxQueuedDeliveryLines - len(lines)
		existing := delivery.event.Lines
		bounded = append(bounded, existing[len(existing)-keepExisting:]...)
		bounded = append(bounded, lines...)
	}
	delivery.event.Lines = bounded
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
				warning := fmt.Sprintf(
					`{"level":"warn","event":"log-delivery-overflow","dropped":%d,"message":"presentation log delivery was slower than producers"}`,
					delivery.droppedLines,
				)
				delivery.event.Lines = append([]string{warning}, delivery.event.Lines...)
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

// AppendDumpLine 把一条 opt-in 节点 dump 行只写进文件 (不进 ring / 不 emit log:lines,
// 避免与前端 container:node-dump-batch 双显). 写成 JSON 行, 与现有 file log 保持一致.
func (s *LogSink) AppendDumpLine(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.fileDir == "" {
		return
	}
	s.openTodayFileLocked()
	if s.file == nil {
		return
	}
	obj := map[string]any{
		"level": "info",
		"event": "node-dump",
		"line":  line,
		"time":  time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(obj)
	_, _ = s.file.Write(b)
	_, _ = s.file.WriteString("\n")
}

// AppendActionTrace writes a redacted action trace JSON line to the file log only.
// Raw request/result payloads and OS handles are intentionally not persisted.
func (s *LogSink) AppendActionTrace(data any) {
	line := sanitizeActionTrace(data)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.fileDir == "" {
		return
	}
	s.openTodayFileLocked()
	if s.file == nil {
		return
	}
	b, _ := json.Marshal(line)
	_, _ = s.file.Write(b)
	_, _ = s.file.WriteString("\n")
}

func sanitizeActionTrace(data any) map[string]any {
	raw := map[string]any{}
	if b, err := json.Marshal(data); err == nil {
		_ = json.Unmarshal(b, &raw)
	}
	out := map[string]any{
		"level": "info",
		"event": "action-trace",
		"time":  time.Now().Format(time.RFC3339Nano),
	}
	copyStringField(out, raw, "containerId")
	copyStringField(out, raw, "action")
	copyStringField(out, raw, "backend")
	copyStringField(out, raw, "status")
	copyStringField(out, raw, "error")
	copyStringField(out, raw, "startedAt")
	copyStringField(out, raw, "endedAt")
	copyAnyField(out, raw, "durationMs")
	out["source"] = sanitizeActionTraceSource(raw["source"])
	out["target"] = sanitizeActionTraceTarget(raw["target"])
	out["coordinateStepCount"] = lenAnySlice(raw["coordinateSteps"])
	return out
}

func sanitizeActionTraceSource(raw any) map[string]string {
	m, _ := raw.(map[string]any)
	return map[string]string{
		"containerId": firstString(m, "containerId", "ContainerID"),
		"nodeId":      firstString(m, "nodeId", "NodeID"),
		"nodeKind":    firstString(m, "nodeKind", "NodeKind"),
		"inPin":       firstString(m, "inPin", "InPin"),
	}
}

func sanitizeActionTraceTarget(raw any) map[string]string {
	m, _ := raw.(map[string]any)
	return map[string]string{
		"id":   firstString(m, "id", "ID"),
		"kind": firstString(m, "kind", "Kind"),
	}
}

func copyStringField(out, raw map[string]any, key string) {
	if v, ok := raw[key].(string); ok && v != "" {
		out[key] = v
	}
}

func copyAnyField(out, raw map[string]any, key string) {
	if v, ok := raw[key]; ok {
		out[key] = v
	}
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func lenAnySlice(v any) int {
	switch xs := v.(type) {
	case []any:
		return len(xs)
	default:
		return 0
	}
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
		if s.timer != nil {
			s.timer.Stop()
			s.timer = nil
		}
		if s.file != nil {
			s.closeErr = s.file.Close()
			s.file = nil
		}
		s.mu.Unlock()
	})
	return s.closeErr
}

// Snapshot ring 内全部行（"\n" 连接）。测试 / 调试用。
func (s *LogSink) Snapshot() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Join(s.ring, "\n")
}
