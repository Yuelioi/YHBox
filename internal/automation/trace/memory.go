package trace

import "sync"

const defaultMemoryCapacity = 2048

type MemoryRecorder struct {
	mu      sync.Mutex
	records []ActionRecord
	start   int
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
	if len(r.records) < defaultMemoryCapacity {
		r.records = append(r.records, record)
		return
	}
	r.records[r.start] = record
	r.start = (r.start + 1) % defaultMemoryCapacity
}

func (r *MemoryRecorder) Records() []ActionRecord {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]ActionRecord, 0, len(r.records))
	if len(r.records) < defaultMemoryCapacity || r.start == 0 {
		out = append(out, r.records...)
		return out
	}
	out = append(out, r.records[r.start:]...)
	out = append(out, r.records[:r.start]...)
	return out
}

func (r *MemoryRecorder) Clear() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = nil
	r.start = 0
}
