package recording

import (
	"sync"
	"testing"
)

// 状态机 + 幂等性单测. 不碰真 OS hook — 只验 Service 的状态转换/幂等/事件广播逻辑.

func newTestService() (*Service, *[]string) {
	s := NewService(NewRecorder(), nil, nil, nil)
	var mu sync.Mutex
	var events []string
	s.SetEmit(func(name string, _ any) {
		mu.Lock()
		events = append(events, name)
		mu.Unlock()
	})
	return s, &events
}

func TestService_InitialStateIdle(t *testing.T) {
	s, _ := newTestService()
	if got := s.GetState().Phase; got != PhaseIdle {
		t.Fatalf("初始 phase = %q, want %q", got, PhaseIdle)
	}
}

func TestService_StopWhenIdle_IsNoOpNoError(t *testing.T) {
	s, events := newTestService()
	// 幂等核心: idle 时 Stop → (nil, nil), 不报 ErrRecorderNotActive, 不碰 recorder.
	payload, err := s.Stop()
	if err != nil {
		t.Fatalf("idle Stop 应无错, got %v", err)
	}
	if payload != nil {
		t.Fatalf("idle Stop 应返 nil payload, got %+v", payload)
	}
	if s.GetState().Phase != PhaseIdle {
		t.Fatalf("idle Stop 后 phase 应仍 idle, got %q", s.GetState().Phase)
	}
	if len(*events) != 0 {
		t.Fatalf("idle Stop 不该 emit 状态事件, got %v", *events)
	}
}

func TestService_StartWhenNotIdle_IsNoOp(t *testing.T) {
	s, _ := newTestService()
	// 模拟已在录 (不走真 hook): 直接置 recording 态.
	s.setState(RecordingState{Phase: PhaseRecording, ContainerID: "cA", TempID: "t1"})
	// Start 撞非 idle → 返当前 tempID, 不报错, 不重启.
	id, err := s.Start(StartArgs{FilterMode: "precise", ContainerID: "cB"})
	if err != nil {
		t.Fatalf("非 idle Start 应无错 (幂等 no-op), got %v", err)
	}
	if id != "t1" {
		t.Fatalf("非 idle Start 应返当前 tempID t1, got %q", id)
	}
	if s.GetState().ContainerID != "cA" {
		t.Fatalf("非 idle Start 不该改容器, got %q", s.GetState().ContainerID)
	}
}

func TestService_GetStateAndEmit(t *testing.T) {
	s, events := newTestService()
	s.setState(RecordingState{Phase: PhaseRecording, ContainerID: "cX", FilterMode: "simple", TempID: "tt"})
	st := s.GetState()
	if st.Phase != PhaseRecording || st.ContainerID != "cX" || st.FilterMode != "simple" || st.TempID != "tt" {
		t.Fatalf("GetState 没反映 setState: %+v", st)
	}
	if len(*events) != 1 || (*events)[0] != "recording:state" {
		t.Fatalf("setState 应 emit 一次 recording:state, got %v", *events)
	}
}
