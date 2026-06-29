// internal/services/script/compile.go
// Script 源码编译: IIFE 包裹 (让顶层 return 合法, 返回值即 Result) + program 缓存 + 行号修正.
package script

import (
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/dop251/goja"
)

// WrapSource 包 IIFE — 行号整体 +1, 报错侧用 AdjustWrapLine 减回.
func WrapSource(src string) string { return "(function(){\n" + src + "\n})()" }

var progCache = struct {
	sync.RWMutex
	m map[string]*goja.Program
}{m: map[string]*goja.Program{}}

// CompileCached 按原始 src 缓存编译产物 (Program 跨 VM 复用, immutable).
func CompileCached(src string) (*goja.Program, error) {
	progCache.RLock()
	if p, ok := progCache.m[src]; ok {
		progCache.RUnlock()
		return p, nil
	}
	progCache.RUnlock()
	p, err := goja.Compile("script", WrapSource(src), false)
	if err != nil {
		return nil, err
	}
	progCache.Lock()
	progCache.m[src] = p
	progCache.Unlock()
	return p, nil
}

// CheckSyntax 编辑期语法检查 (validator SCRIPT_PARSE_ERROR 用).
func CheckSyntax(src string) error {
	_, err := CompileCached(src)
	return err
}

var wrapLineRe = regexp.MustCompile(`script:(\d+)`)

// AdjustWrapLine 把错误文案里 IIFE 造成的 +1 行号减回用户视角.
func AdjustWrapLine(msg string) string {
	return wrapLineRe.ReplaceAllStringFunc(msg, func(m string) string {
		n, _ := strconv.Atoi(strings.TrimPrefix(m, "script:"))
		if n > 1 {
			n--
		}
		return "script:" + strconv.Itoa(n)
	})
}
