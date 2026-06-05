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
	idleTimeout time.Duration
	now         func() time.Time
}

func NewLogMerger(emit func(string, any), writeFile func(string)) *LogMerger {
	m := &LogMerger{
		segs:        map[string]*dumpSegment{},
		emit:        emit,
		writeFile:   writeFile,
		stop:        make(chan struct{}),
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

func (m *LogMerger) finalizeLocked(k string, s *dumpSegment) {
	m.writeFile(renderCount(s.line, s.count))
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
	prefix := containerID + "\x00"
	for k, s := range m.segs {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			m.finalizeLocked(k, s)
		}
	}
}

func (m *LogMerger) Close() { close(m.stop); m.ticker.Stop() }

func (m *LogMerger) loop() {
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
			entries = append(entries, segEntry(s, true))
			m.finalizeLocked(k, s)
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
