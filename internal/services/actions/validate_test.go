package actions

import "testing"

func TestValidate_StepClickRatio(t *testing.T) {
	a := &Action{SchemaVersion: 2, Steps: []Step{{Kind: StepClickLeft, XRatio: 1.5, YRatio: 0.5}}}
	if err := a.Validate(); err == nil {
		t.Error("expected ratio out of range error")
	}
}

func TestValidate_StepKeyNoVk(t *testing.T) {
	a := &Action{SchemaVersion: 2, Steps: []Step{{Kind: StepKey}}}
	if err := a.Validate(); err == nil {
		t.Error("expected vk required")
	}
}

func TestValidate_OK(t *testing.T) {
	a := &Action{
		SchemaVersion: 2,
		Steps: []Step{
			{Kind: StepClickLeft, XRatio: 0.5, YRatio: 0.5, DurationMs: 50},
			{Kind: StepSleep, DurationMs: 1000},
		},
	}
	NormalizeAction(a)
	if err := a.Validate(); err != nil {
		t.Errorf("expected valid: %v", err)
	}
	if a.Steps[0].ID == "" {
		t.Error("step id should be filled")
	}
}

func TestValidate_UnknownStepKind(t *testing.T) {
	a := &Action{SchemaVersion: 2, Steps: []Step{{Kind: StepKind("weird")}}}
	if err := a.Validate(); err == nil {
		t.Error("expected unknown kind error")
	}
}
