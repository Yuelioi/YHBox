// internal/node/snapshot.go
package node

// Snapshot — tick 起始时 capture 的全局状态. Vars/Sys 在同 tick 内对 PureData
// Evaluator 是 frozen view. 调用方 (runtime ContainerRunner) 负责 capture; node 包
// 不知 SysState 内部结构, 用 SysStore interface 承载已 freeze 的 view.
//
// IMPORTANT: snapshot consistency is guaranteed only within one dispatch/tick.
// Caller MUST NOT cache the Snapshot returned by services.Snapshot() across ticks
// — Vars map 和 Sys reference 由 runtime 持有, 跨 tick 会变.
type Snapshot struct {
	Vars map[string]any // global vars frozen at tick start
	Sys  SysStore       // frozen SysStore (runtime 端构造)
}

// snapshotVarStore wraps live VarStore + frozen Snapshot.
// PureData Evaluator 内部看到的 Vars view — global / auto-global-fallback 走 snapshot,
// local / auto-local-first 走 live (frame-private 不冻).
// 写操作全 panic (PureData 不该写 Vars).
type snapshotVarStore struct {
	live VarStore
	snap Snapshot
}

func newSnapshotVarStore(live VarStore, snap Snapshot) VarStore {
	return &snapshotVarStore{live: live, snap: snap}
}

func (s *snapshotVarStore) Get(name string) (any, bool) {
	if v, ok := s.snap.Vars[name]; ok {
		return v, true
	}
	return nil, false
}

func (s *snapshotVarStore) GetScoped(name, scope string) (any, bool) {
	switch scope {
	case "local":
		return s.live.GetScoped(name, "local")
	case "auto":
		if v, ok := s.live.GetScoped(name, "local"); ok {
			return v, true
		}
		if v, ok := s.snap.Vars[name]; ok {
			return v, true
		}
		return nil, false
	default: // "global", ""
		if v, ok := s.snap.Vars[name]; ok {
			return v, true
		}
		return nil, false
	}
}

func (s *snapshotVarStore) Set(name string, value any) {
	panic("snapshot VarStore is read-only: PureData Evaluator must not write Vars")
}

func (s *snapshotVarStore) Inc(name string, delta float64) float64 {
	panic("snapshot VarStore is read-only: PureData Evaluator must not write Vars")
}

func (s *snapshotVarStore) SetScoped(name, scope string, value any) {
	panic("snapshot VarStore is read-only: PureData Evaluator must not write Vars")
}

func (s *snapshotVarStore) IncScoped(name, scope string, delta float64) float64 {
	panic("snapshot VarStore is read-only: PureData Evaluator must not write Vars")
}

// LastChange 透传 live — 时间戳不进快照, PureData 读到的是当前真值.
func (s *snapshotVarStore) LastChange(name string) int64 { return s.live.LastChange(name) }
