package runtime

import (
	"encoding/json"
	"fmt"
	"strings"

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

func outputNames(spec *nodepkg.Spec) []string {
	ns := make([]string, len(spec.Outputs))
	for i, p := range spec.Outputs {
		ns[i] = p.Name
	}
	return ns
}

// FormatDumpLine 产出 (line, key):
//
//	line = "Kind(name, id) in{…} out{…} [err=…]" (显示)
//	key  = 仅 in{}/out{}/err 段 (合并比较, 不含 name/id)
func FormatDumpLine(spec *nodepkg.Spec, name, id string, in, out map[string]any, runErr error) (line, key string) {
	inSeg := dumpSegment("in", inputNames(spec), in)
	outSeg := dumpSegment("out", outputNames(spec), out)
	errSeg := ""
	if runErr != nil {
		errSeg = "err=" + dumpValue(runErr.Error())
	}
	segs := make([]string, 0, 3)
	for _, s := range []string{inSeg, outSeg, errSeg} {
		if s != "" {
			segs = append(segs, s)
		}
	}
	body := strings.Join(segs, " ")
	head := spec.Kind
	if name != "" {
		head += "(" + name + ", " + id + ")"
	} else {
		head += "(" + id + ")"
	}
	line = strings.TrimSpace(head + " " + body)
	key = body
	return line, key
}
