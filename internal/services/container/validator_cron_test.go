package container

import (
	"testing"
)

func TestValidateCron_EmptyLiteral_NoErr(t *testing.T) {
	// 空 literal = 用户准备连上游 / 还没填 — validator 不报 (dangling pin 别人报)
	errs := validateCronConfig(&GraphNode{Kind: "Cron", Config: map[string]any{}})
	if len(errs) != 0 {
		t.Errorf("空 literal 不应报错, got: %+v", errs)
	}
}

func TestValidateCron_HappyPath_NoErr(t *testing.T) {
	errs := validateCronConfig(&GraphNode{Kind: "Cron", Config: map[string]any{
		"literal": map[string]any{"Expression": "0 */5 * * * *"},
	}})
	if len(errs) != 0 {
		t.Errorf("合法 expr 不应报错, got: %+v", errs)
	}
}

func TestValidateCron_BadExpr_FieldOutOfRange(t *testing.T) {
	errs := validateCronConfig(&GraphNode{Kind: "Cron", Config: map[string]any{
		"literal": map[string]any{"Expression": "60 60 60 60 60 60"},
	}})
	if len(errs) == 0 {
		t.Fatal("字段越界应报错")
	}
	if errs[0].Code != CodeInvalidCronExpr {
		t.Errorf("应是 INVALID_CRON_EXPR, got: %s", errs[0].Code)
	}
	if parseErr, _ := errs[0].Params["parseErr"].(string); parseErr == "" {
		t.Errorf("Params.parseErr 应非空, got: %v", errs[0].Params["parseErr"])
	}
	if expr, _ := errs[0].Params["expr"].(string); expr != "60 60 60 60 60 60" {
		t.Errorf("Params.expr 应是原文, got: %v", errs[0].Params["expr"])
	}
}

func TestValidateCron_SyntaxError(t *testing.T) {
	errs := validateCronConfig(&GraphNode{Kind: "Cron", Config: map[string]any{
		"literal": map[string]any{"Expression": "@@@"},
	}})
	if len(errs) == 0 || errs[0].Code != CodeInvalidCronExpr {
		t.Errorf("语法错应报 INVALID_CRON_EXPR, got: %+v", errs)
	}
}

func TestValidateRegexPattern(t *testing.T) {
	// 非法 literal pattern → SeverityError + INVALID_REGEX_PATTERN
	bad := GraphNode{ID: "r1", Kind: "RegexMatch", Config: map[string]any{
		"literal": map[string]any{"Pattern": "("},
	}}
	errs := checkGraphPerKind([]GraphNode{bad}, []string{"main"}, true)
	if len(errs) != 1 || errs[0].Code != CodeInvalidRegexPattern || errs[0].Severity != SeverityError {
		t.Fatalf("want 1 INVALID_REGEX_PATTERN error, got %+v", errs)
	}

	// 合法 pattern → 无错
	good := GraphNode{ID: "r2", Kind: "RegexExtract", Config: map[string]any{
		"literal": map[string]any{"Pattern": `\d+`},
	}}
	if errs := checkGraphPerKind([]GraphNode{good}, []string{"main"}, true); len(errs) != 0 {
		t.Fatalf("valid pattern should pass, got %+v", errs)
	}

	// 空 pattern → 跳过 (准备连上游/没填, 同 Cron 惯例)
	empty := GraphNode{ID: "r3", Kind: "RegexMatch", Config: map[string]any{}}
	if errs := checkGraphPerKind([]GraphNode{empty}, []string{"main"}, true); len(errs) != 0 {
		t.Fatalf("empty pattern should be skipped, got %+v", errs)
	}
}
