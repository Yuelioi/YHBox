package trace

import "sync"

type MemoryRecorder struct {
	mu      sync.Mutex
	records []ActionRecord
}

func NewMemoryRecorder() *MemoryRecorder {
	return &MemoryRecorder{}
}

func (r *MemoryRecorder) Record(record ActionRecord) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, record)
}

func (r *MemoryRecorder) Records() []ActionRecord {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]ActionRecord, len(r.records))
	copy(out, r.records)
	return out
}

func (r *MemoryRecorder) Clear() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = nil
}
