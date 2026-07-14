package container

// 契约守卫: 前端 error.<CODE> 消息里的每个占位符 {x} 必须是后端该 code 实际发出的 Param key。
//
// 为什么: FE 渲染校验错误只走 t("error."+code, params) (见 frontend/src/lib/invoke.ts), 没有后端
// Message 兜底。占位符若不在 params 里 → vue-i18n 渲染成空白 (用户看到 "期望 , 实际 ", 没法诊断)。
// 这正是 LITERAL_TYPE_MISMATCH 踩过的坑: 消息写 {type}/{got} 而后端发 expected/value。
//
// 做法: 扫本包 Go 源拿 code→内联 Params 键, 扫 zh.ts 拿 code→占位符, 交叉比对。
// 保守: 只对「后端确有内联 map[string]any Params 构造」的 code 较真; 用变量传 Params / 从不构造
// (死码) 的 code 跳过 (无法静态确认, 不误报)。
//
// 这是跨树读 (Go 测试读 frontend/i18n) —— frontend 缺失 (纯后端 CI) 时 Skip, 不算失败。

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	reConstDef    = regexp.MustCompile(`(Code[A-Za-z0-9_]+)\s*=\s*"([A-Z0-9_]+)"`)
	reCodeUse     = regexp.MustCompile(`Code:\s*(Code[A-Za-z0-9_]+|"[A-Z0-9_]+")`)
	reParamsOpen  = regexp.MustCompile(`Params:\s*map\[string\]any\{`)
	reParamKey    = regexp.MustCompile(`"(\w+)":`)
	rePlaceholder = regexp.MustCompile(`\{(\w+)\}`)
	reI18nEntry   = regexp.MustCompile(`(?m)^\s{4}([A-Z][A-Z0-9_]+):\s*'((?:[^'\\]|\\.)*)'`)
	reI18nKey     = regexp.MustCompile(`(?m)^\s{4}([A-Z][A-Z0-9_]+):`)
)

// scanBackendParams 扫本包 (含子目录) 非测试 Go 源, 返回 code(字符串值) → 内联 Params 键集合 (并集)。
func scanBackendParams(t *testing.T) map[string]map[string]bool {
	t.Helper()
	constMap := map[string]string{} // CodeXxx -> "STRING"
	var goFiles []string
	_ = filepath.Walk(".", func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go") {
			goFiles = append(goFiles, p)
		}
		return nil
	})
	for _, f := range goFiles {
		b, _ := os.ReadFile(f)
		for _, m := range reConstDef.FindAllStringSubmatch(string(b), -1) {
			constMap[m[1]] = m[2]
		}
	}
	params := map[string]map[string]bool{}
	for _, f := range goFiles {
		b, _ := os.ReadFile(f)
		lines := strings.Split(string(b), "\n")
		cur := ""
		for i := range lines {
			if m := reCodeUse.FindStringSubmatch(lines[i]); m != nil {
				c := m[1]
				if strings.HasPrefix(c, `"`) {
					cur = strings.Trim(c, `"`)
				} else if s, ok := constMap[c]; ok {
					cur = s
				} else {
					cur = c
				}
			}
			if loc := reParamsOpen.FindStringIndex(lines[i]); loc != nil && cur != "" {
				// 从 map literal 起收 "key": 直到大括号平衡 (或 10 行兜底)。
				depth := 0
				var buf strings.Builder
				for j := i; j < len(lines) && j-i <= 10; j++ {
					seg := lines[j]
					if j == i {
						seg = seg[loc[1]-1:] // 从 '{' 起
					}
					done := false
					for _, ch := range seg {
						if ch == '{' {
							depth++
						} else if ch == '}' {
							depth--
							if depth == 0 {
								done = true
								break
							}
						}
					}
					buf.WriteString(seg)
					if done {
						break
					}
				}
				if params[cur] == nil {
					params[cur] = map[string]bool{}
				}
				for _, km := range reParamKey.FindAllStringSubmatch(buf.String(), -1) {
					params[cur][km[1]] = true
				}
			}
		}
	}
	return params
}

// scanI18nPlaceholders 读 zh.ts error 块, 返回 code → 占位符集合。frontend 缺失则返回 nil。
func scanI18nPlaceholders(t *testing.T, path string) map[string][]string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	src := string(b)
	start := strings.Index(src, "\n  error: {")
	if start < 0 {
		return nil
	}
	end := strings.Index(src[start:], "\n  },")
	if end < 0 {
		return nil
	}
	block := src[start : start+end]
	out := map[string][]string{}
	for _, m := range reI18nEntry.FindAllStringSubmatch(block, -1) {
		seen := map[string]bool{}
		var phs []string
		for _, pm := range rePlaceholder.FindAllStringSubmatch(m[2], -1) {
			if !seen[pm[1]] {
				seen[pm[1]] = true
				phs = append(phs, pm[1])
			}
		}
		out[m[1]] = phs // m[1] = error code
	}
	return out
}

func TestI18nErrorPlaceholdersMatchBackendParams(t *testing.T) {
	params := scanBackendParams(t)
	if len(params) == 0 {
		t.Fatal("没扫到任何后端 Params 构造 — 解析器可能失效")
	}

	checked := 0
	for _, locale := range []string{"zh.ts", "en.ts"} {
		path := filepath.Join("..", "..", "..", "frontend", "src", "i18n", locale)
		i18n := scanI18nPlaceholders(t, path)
		if i18n == nil {
			t.Skipf("frontend i18n %s 不可读 (纯后端环境) — 跳过契约校验", locale)
		}
		for code, phs := range i18n {
			if len(phs) == 0 {
				continue
			}
			have, ok := params[code]
			if !ok {
				// 后端无内联 Params 构造 (死码 / 变量传参) — 静态无法确认, 跳过不误报。
				continue
			}
			checked++
			for _, ph := range phs {
				if !have[ph] {
					t.Errorf("%s error.%s 用了占位符 {%s} 但后端该 code 的 Params 没这个键 (会渲染成空白)。"+
						"后端实发: %v — 对齐 i18n 占位符或后端 Params 键。", locale, code, ph, keysOf(have))
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("没有可校验的 code (i18n 占位符 ∩ 后端 Params) — 解析器可能失效")
	}
	t.Logf("校验了 %d 个有占位符且后端有内联 Params 的 error code", checked)
}

func TestI18nCoversEveryValidatorCode(t *testing.T) {
	b, err := os.ReadFile("validator.go")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{}
	for _, m := range reConstDef.FindAllStringSubmatch(string(b), -1) {
		want[m[2]] = true
	}
	if len(want) == 0 {
		t.Fatal("没扫到 validator error code — 解析器可能失效")
	}

	for _, locale := range []string{"zh.ts", "en.ts"} {
		path := filepath.Join("..", "..", "..", "frontend", "src", "i18n", locale)
		keys := scanI18nErrorKeys(t, path)
		if keys == nil {
			t.Skipf("frontend i18n %s 不可读 (纯后端环境) — 跳过契约校验", locale)
		}
		for code := range want {
			if !keys[code] {
				t.Errorf("%s 缺少 error.%s；ProblemsBar 将退化为裸错误码", locale, code)
			}
		}
	}
}

func scanI18nErrorKeys(t *testing.T, path string) map[string]bool {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	src := string(b)
	start := strings.Index(src, "\n  error: {")
	if start < 0 {
		t.Fatalf("%s 没有 error i18n block", path)
	}
	end := strings.Index(src[start:], "\n  },")
	if end < 0 {
		t.Fatalf("%s 的 error i18n block 未闭合", path)
	}
	keys := map[string]bool{}
	for _, m := range reI18nKey.FindAllStringSubmatch(src[start:start+end], -1) {
		keys[m[1]] = true
	}
	return keys
}

func keysOf(m map[string]bool) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
