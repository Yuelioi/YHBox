package runtime

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	nodepkg "yotta/internal/node"
)

const (
	dumpMaxString = 64
	dumpMaxObject = 200
)

func dumpValue(v any) (s string) {
	defer func() {
		if r := recover(); r != nil {
			s = "<?>"
		}
	}()
	switch t := v.(type) {
	case nil:
		return "nil"
	case string:
		if len(t) > dumpMaxString {
			return t[:dumpMaxString] + "…"
		}
		return t
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprintf("%v", t)
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return "{…}"
		}
		out := string(b)
		if len(out) > dumpMaxObject {
			return out[:dumpMaxObject] + "…"
		}
		return out
	}
}

func dumpSegment(label string, order []string, vals map[string]any) string {
	if len(vals) == 0 {
		return ""
	}
	parts := make([]string, 0, len(vals))
	for _, name := range order {
		if v, ok := vals[name]; ok {
			parts = append(parts, name+"="+dumpValue(v))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return label + "{" + strings.Join(parts, ", ") + "}"
}

func inputNames(spec *nodepkg.Spec) []string {
	ns := make([]string, len(spec.Inputs))
	for i, p := range spec.Inputs {
		ns[i] = p.Name
	}
	return ns
}

// outputNames 收出口名 + 各出口的 data 字段名。OutputData 是按 data 字段名 (Point/Conf) 键的,
// 不是按 exec 出口名 (Done/Timeout); 只列出口名会让 out{} 段永远空 (匹配不上)。两者都收 → 兼容
// 「输出即数据键」的简单节点 (CheckTemplate hit/x/y) 和「出口带 data 字段」的节点 (ClickTemplate)。
func outputNames(spec *nodepkg.Spec) []string {
	ns := make([]string, 0, len(spec.Outputs))
	seen := map[string]bool{}
	add := func(n string) {
		if n != "" && !seen[n] {
			seen[n] = true
			ns = append(ns, n)
		}
	}
	for _, p := range spec.Outputs {
		add(p.Name)
		for _, d := range p.Data {
			add(d.Name)
		}
	}
	return ns
}

// FormatDumpLine 产出 (line, key):
//
//	line = "Kind(name, id) in{…} →exit out{…} [err=…] [took=…]" (显示)
//	key  = in{}/→exit/out{}/err 段 (合并比较, 不含 name/id, 也不含 took — 每次耗时不同, 含了就永不合并)
//
// exitName = 本次 fire 的出口名 (WaitTemplate 的 Found/Timeout、ClickTemplate 的 Done/Timeout):
// 直接回答「命中还是超时」(err 段 = 失败)。空 = 无单出口路由 (no-op / region body)。
// dur = 节点实际耗时 (WaitTemplate/ClickTemplate 等了多久才命中, 一眼看出"模板 12s 才出现")。
func FormatDumpLine(spec *nodepkg.Spec, name, id string, in, out map[string]any, exitName string, dur time.Duration, runErr error) (line, key string) {
	inSeg := dumpSegment("in", inputNames(spec), in)
	outSeg := dumpSegment("out", outputNames(spec), out)
	exitSeg := ""
	if exitName != "" {
		exitSeg = "→" + exitName
	}
	errSeg := ""
	if runErr != nil {
		errSeg = "err=" + dumpValue(runErr.Error())
	}
	segs := make([]string, 0, 4)
	for _, s := range []string{inSeg, exitSeg, outSeg, errSeg} {
		if s != "" {
			segs = append(segs, s)
		}
	}
	key = strings.Join(segs, " ") // took 不进 key

	head := spec.Kind
	if name != "" {
		head += "(" + name + ", " + id + ")"
	} else {
		head += "(" + id + ")"
	}
	tookSeg := ""
	if dur > 0 {
		if d := dur.Round(time.Millisecond); d == 0 {
			tookSeg = "took=<1ms"
		} else {
			tookSeg = "took=" + d.String()
		}
	}
	parts := []string{head}
	if key != "" {
		parts = append(parts, key)
	}
	if tookSeg != "" {
		parts = append(parts, tookSeg)
	}
	line = strings.Join(parts, " ")
	return line, key
}
