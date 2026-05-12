package fish

import (
	"testing"
	"time"
)

// TestSetState_UpdatesTimestamp setState 改变 state 时刷新 lastStateChangeAt。
func TestSetState_UpdatesTimestamp(t *testing.T) {
	m := &machine{state: IDLE, lastStateChangeAt: time.Now().Add(-time.Hour)}
	old := m.lastStateChangeAt

	m.setState(WAITING)

	if m.state != WAITING {
		t.Errorf("state = %v, want WAITING", m.state)
	}
	if !m.lastStateChangeAt.After(old) {
		t.Errorf("lastStateChangeAt 未刷新: old=%v new=%v", old, m.lastStateChangeAt)
	}
}

// TestSetState_SameStateNoUpdate setState 传相同 state 时不刷新时间戳，
// 避免循环里"自跳回自己"被当成进展。
func TestSetState_SameStateNoUpdate(t *testing.T) {
	old := time.Now().Add(-time.Hour)
	m := &machine{state: IDLE, lastStateChangeAt: old}

	m.setState(IDLE)

	if !m.lastStateChangeAt.Equal(old) {
		t.Errorf("lastStateChangeAt 不该刷新: old=%v new=%v", old, m.lastStateChangeAt)
	}
}

// TestShouldWatchdog_Whitelist FISHING / RECOVERING 不参与，其他全部参与。
func TestShouldWatchdog_Whitelist(t *testing.T) {
	cases := []struct {
		state State
		want  bool
	}{
		{IDLE, true},
		{SETUP, true},
		{WAITING, true},
		{FISHING, false},
		{RESULT, true},
		{RECOVERING, false},
		{SHOPSELL, true},
		{BUYBAIT, true},
		{CHANGEBAIT, true},
	}
	for _, c := range cases {
		m := &machine{state: c.state}
		got := m.shouldWatchdog()
		if got != c.want {
			t.Errorf("state=%v shouldWatchdog=%v want=%v", c.state, got, c.want)
		}
	}
}
