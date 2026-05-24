// internal/services/container/nodepkg_bridge.go
//
// Phase 6+ pull-eval partial (atomic #2 — validator nodepkg lookup).
//
// Bridge — validator helpers (pinExists / IsDataOutPin / dataInPinTypeForKind / ...)
// 转换期同时支持新 nodepkg (Spec.Inputs/Outputs PascalCase) 和老 nodekind (lowercase
// DataIn/DataOut/ExecIn/ExecOut). Union 策略: nodepkg 优先短路, miss 再走 nodekind fallback.
// 重叠 kind 上两套 pin name 都过 validator — 让 fishing-v2 redraw 时新老 JSON 同期可加载.
//
// Atomic #5 (拆老 v4) 后, 老 nodekind path 全删, 这个 bridge 文件改成单查 nodepkg.
package container

import (
	"strings"

	nodepkg "yhbox/internal/node"
)

// canonPinType 转 nodepkg pin Type tag (PascalCase, e.g. "Number"/"String"/"Bool"/"Point"/"*")
// 到 nodekind.PinType 字符串 (lowercase, "number"/"string"/"bool"/"point"/"any").
//
// 目的: validator_data_graph PIN_TYPE_MISMATCH 比较 src/dst type 字符串 — 重叠 kind 上一端走
// nodepkg (PascalCase) 另一端走 nodekind (lowercase) 不能撞错. 这个函数把 nodepkg 风格转成
// nodekind 风格让 caller 端比较一致.
//
// Atomic #5 拆老后, nodekind 全删, validator 改成直接吃 nodepkg type tag (PascalCase),
// 这个 canonicalize 也可以删.
func canonPinType(t string) string {
	switch t {
	case "Number":
		return "number"
	case "String":
		return "string"
	case "Bool":
		return "bool"
	case "Point":
		return "point"
	case "*":
		return "any"
	case "Exec":
		return ""
	}
	return strings.ToLower(t)
}

// nodepkgInputType 返 nodepkg Spec.Inputs 上对应 pin 的 Type 字符串 (canonicalized 成 lowercase
// nodekind.PinType 风格 — "number"/"string"/"bool"/"point"/"any"). Exec pin 返 "" (Exec 不算 data type).
//
// 空字符串 = 没找到 (kind 没注册 / pin 不存在 / 是 Exec pin).
// 调用方靠 != "" 判断 "是 data-in pin 且存在".
func nodepkgInputType(kind, pin string) string {
	rn, ok := nodepkg.Get(kind)
	if !ok {
		return ""
	}
	for _, ip := range rn.Spec.Inputs {
		if ip.Name == pin {
			return canonPinType(ip.Type)
		}
	}
	return ""
}

// nodepkgOutputType 同上 outputs.
func nodepkgOutputType(kind, pin string) string {
	rn, ok := nodepkg.Get(kind)
	if !ok {
		return ""
	}
	for _, op := range rn.Spec.Outputs {
		if op.Name == pin {
			return canonPinType(op.Type)
		}
	}
	return ""
}

// nodepkgPinExists 任意 pin (Exec / data, in / out) 是否在 nodepkg Spec 里.
// 注意: nodepkgInputType/nodepkgOutputType 对 Exec pin 返 "" 不可用, 这里直接 iterate.
func nodepkgPinExists(kind, pin string, out bool) bool {
	rn, ok := nodepkg.Get(kind)
	if !ok {
		return false
	}
	if out {
		for _, op := range rn.Spec.Outputs {
			if op.Name == pin {
				return true
			}
		}
		return false
	}
	for _, ip := range rn.Spec.Inputs {
		if ip.Name == pin {
			return true
		}
	}
	return false
}

// nodepkgIsRegistered kind 是否在 nodepkg 注册.
func nodepkgIsRegistered(kind string) bool {
	_, ok := nodepkg.Get(kind)
	return ok
}

// nodepkgExecInPins 新 spec 里 Type="Exec" 的 input pin name 集合.
func nodepkgExecInPins(kind string) []string {
	rn, ok := nodepkg.Get(kind)
	if !ok {
		return nil
	}
	var out []string
	for _, ip := range rn.Spec.Inputs {
		if ip.Type == "Exec" {
			out = append(out, ip.Name)
		}
	}
	return out
}

// nodepkgExecOutPins 新 spec 里 Type="Exec" 的 output pin name 集合.
// nodepkg Spec.Outputs 静态; 动态 exec-out (老 Switch ExecOutFn / Parallel/Race branchN) Phase 6+ 单独 spec.
func nodepkgExecOutPins(kind string) []string {
	rn, ok := nodepkg.Get(kind)
	if !ok {
		return nil
	}
	var out []string
	for _, op := range rn.Spec.Outputs {
		if op.Type == "Exec" {
			out = append(out, op.Name)
		}
	}
	return out
}
