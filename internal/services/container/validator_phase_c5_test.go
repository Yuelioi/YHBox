package container

import (
	"testing"
)

func helperRunValidate(node *GraphNode) []ValidationError {
	return validateSwitchConfig(node)
}

func TestValidateSwitch_EmptyCases(t *testing.T) {
	errs := helperRunValidate(&GraphNode{Kind: "Switch", Config: map[string]any{
		"Value": "$vars.x",
	}})
	if len(errs) == 0 {
		t.Fatalf("空 cases 应报错")
	}
	found := false
	for _, e := range errs {
		if e.Code == CodeInvalidSwitchCases {
			found = true
		}
	}
	if !found {
		t.Errorf("应有 INVALID_SWITCH_CASES error code, got %+v", errs)
	}
}

func TestValidateSwitch_DotInCase(t *testing.T) {
	errs := helperRunValidate(&GraphNode{Kind: "Switch", Config: map[string]any{
		"Value":      "$vars.x",
		"Case1Value": "v1.0",
	}})
	if len(errs) == 0 || errs[0].Code != CodeInvalidSwitchCases {
		t.Errorf("case 含 . 应报 INVALID_SWITCH_CASES")
	}
}

func TestValidateSwitch_DuplicateCase(t *testing.T) {
	errs := helperRunValidate(&GraphNode{Kind: "Switch", Config: map[string]any{
		"Value":      "$vars.x",
		"Case1Value": "A",
		"Case2Value": "A",
	}})
	if len(errs) == 0 || errs[0].Code != CodeInvalidSwitchCases {
		t.Errorf("重复 case 应报 INVALID_SWITCH_CASES")
	}
}

func TestValidateSwitch_ReservedDefaultCase(t *testing.T) {
	errs := helperRunValidate(&GraphNode{Kind: "Switch", Config: map[string]any{
		"Value":      "$vars.x",
		"Case1Value": "default",
	}})
	if len(errs) == 0 {
		t.Fatalf("case='default' 应报错")
	}
	found := false
	for _, e := range errs {
		if reason, _ := e.Params["reason"].(string); reason == "reserved_default" {
			found = true
		}
	}
	if !found {
		t.Errorf("Params.reason 应是 reserved_default, got %+v", errs)
	}
}

func TestValidateSwitch_LeadingTrailingWhitespace(t *testing.T) {
	errs := helperRunValidate(&GraphNode{Kind: "Switch", Config: map[string]any{
		"Value":      "$vars.x",
		"Case1Value": "IDLE ",
	}})
	if len(errs) == 0 {
		t.Errorf("尾空格应报错")
	}
}

func TestValidateSwitch_HappyPath_NoErr(t *testing.T) {
	errs := helperRunValidate(&GraphNode{Kind: "Switch", Config: map[string]any{
		"Value":      "$vars.state",
		"Case1Value": "IDLE",
		"Case2Value": "钓鱼",
		"Case3Value": "🎣",
	}})
	if len(errs) != 0 {
		t.Errorf("合法 config 不应报错, got %+v", errs)
	}
}
