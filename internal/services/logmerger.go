package services

import (
	"strconv"
	"sync"
	"time"
)

const (
	logMergerFlushInterval = 250 * time.Millisecond
	logMergerIdleTimeout   = 2 * time.Second
)

type dumpSegment struct {
	nodeID, nodeKind, line, lineKey string
	count                           int
	lastUpdate                      time.Time
	dirty                           bool
}

type LogMerger struct {
	mu          sync.Mutex
	segs        map[string]*dumpSegment // key: containerID + "\x00" + nodeID
	emit        func(name string, data any)
	writeFile   func(line string)
	ticker      *time.Ticker
	stop        chan struct{}
	done        chan struct{}
	closeOnce   sync.Once
	closed      bool
	idleTimeout time.Duration
	now         func() time.Time
}

func NewLogMerger(emit func(string, any), writeFile func(string)) *LogMerger {
	m := &LogMerger{
		segs:        map[string]*dumpSegment{},
		emit:        emit,
		writeFile:   writeFile,
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
		idleTimeout: logMergerIdleTimeout,
		now:         time.Now,
	}
	m.ticker = time.NewTicker(logMergerFlushInterval)
	go m.loop()
	return m
}

func segKey(containerID, nodeID string) string { return containerID + "\x00" + nodeID }

func (m *LogMerger) Add(containerID, nodeID, nodeKind, line, lineKey string, isError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return
	}
	k := segKey(containerID, nodeID)
	cur := m.segs[k]
	if isError {
		if cur != nil {
			m.finalizeLocked(k, cur)
		}
		m.writeFile(line)
		m.emitOneLocked(nodeID, nodeKind, lineKey, line, 1, true)
		return
	}
	if cur != nil && cur.lineKey == lineKey {
		cur.count++
		cur.lastUpdate = m.now()
		cur.dirty = true
		return
	}
	if cur != nil {
		m.finalizeLocked(k, cur)
	}
	m.segs[k] = &dumpSegment{nodeID: nodeID, nodeKind: nodeKind, line: line, lineKey: lineKey, count: 1, lastUpdate: m.now(), dirty: true}
}

// finalizeLocked 收尾一个段: 写文件 + emit final 定版 + 删除.
// emit final 是关键 —— 段的前端 emit 只在 tick(dirty) 或这里发生; 走 finalize 而非
// tick 的收尾 (FlushContainer 容器结束 / Add 换 lineKey 收尾旧段) 必须在此 emit,
// 否则短图 (250ms tick 前跑完) 的 dirty 段只写文件、前端面板一条都收不到.
// 前端按 (nodeId,lineKey,!frozen) 幂等更新: 已 emit 过的段再 emit final 只是定版、不重复行.
func (m *LogMerger) finalizeLocked(k string, s *dumpSegment) {
	m.writeFile(renderCount(s.line, s.count))
	m.emitOneLocked(s.nodeID, s.nodeKind, s.lineKey, s.line, s.count, true)
	delete(m.segs, k)
}

func renderCount(line string, count int) string {
	if count > 1 {
		return line + " ×" + strconv.Itoa(count)
	}
	return line
}

func (m *LogMerger) FlushContainer(containerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return
	}
	prefix := containerID + "\x00"
	for k, s := range m.segs {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			m.finalizeLocked(k, s)
		}
	}
}

// Close stops the background loop, finalizes every pending segment, and waits
// until no merger goroutine can write again. It is safe to call repeatedly.
func (m *LogMerger) Close() {
	m.closeOnce.Do(func() {
		m.ticker.Stop()
		close(m.stop)
		<-m.done

		m.mu.Lock()
		for key, segment := range m.segs {
			m.finalizeLocked(key, segment)
		}
		m.closed = true
		m.mu.Unlock()
	})
}

// detachEmit prevents shutdown finalization from calling a presentation
// transport whose GUI loop has already exited. File finalization still runs.
func (m *LogMerger) detachEmit() {
	m.mu.Lock()
	m.emit = nil
	m.mu.Unlock()
}

func (m *LogMerger) loop() {
	defer close(m.done)
	for {
		select {
		case <-m.stop:
			return
		case <-m.ticker.C:
			m.tick()
		}
	}
}

func (m *LogMerger) tick() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	entries := make([]map[string]any, 0, len(m.segs))
	for k, s := range m.segs {
		if now.Sub(s.lastUpdate) >= m.idleTimeout {
			m.finalizeLocked(k, s) // 内部 emit final, 不再 append 进 batch (免重复)
			continue
		}
		if s.dirty {
			entries = append(entries, segEntry(s, false))
			s.dirty = false
		}
	}
	if len(entries) > 0 && m.emit != nil {
		m.emit("container:node-dump-batch", map[string]any{"entries": entries})
	}
}

func (m *LogMerger) emitOneLocked(nodeID, nodeKind, lineKey, line string, count int, final bool) {
	if m.emit == nil {
		return
	}
	m.emit("container:node-dump-batch", map[string]any{"entries": []map[string]any{
		{"nodeId": nodeID, "nodeKind": nodeKind, "lineKey": lineKey, "line": line, "count": count, "final": final},
	}})
}

func segEntry(s *dumpSegment, final bool) map[string]any {
	return map[string]any{"nodeId": s.nodeID, "nodeKind": s.nodeKind, "lineKey": s.lineKey, "line": s.line, "count": s.count, "final": final}
}
