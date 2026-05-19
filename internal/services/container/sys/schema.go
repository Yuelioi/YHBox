// Package sys owns the single source of truth for $sys runtime state schema.
// Both runtime (`evalGetSys`) and validator (`validateGetSysNodes`) import this
// — eliminates the duplicated SysPathSchema / sysPathSchemaCopy maps that
// drifted easily (review §2 finding).
//
// Leaf package: imports only stdlib. Container / runtime / validator depend on
// this; this package depends on nothing — no cycle.
//
// Sync rule: when runtime.resolveSysPath gets a new field, register it here.
package sys

// PathSchema maps $sys.<path> → frontend PinType (string form).
// Used by:
//   - validator/validator_getsys.go to flag unknown paths (GETSYS_UNKNOWN_PATH)
//   - runtime/getsys.go for editor schema export + path type inference
//   - frontend GetSys Inspector dropdown options (via Wails RPC bridging)
var PathSchema = map[string]string{
	"runId":                    "number",
	"iter":                     "number",
	"winnerIdx":                "number",
	"lastTemplate.found":       "bool",
	"lastTemplate.point":       "point",
	"lastTemplate.point.x":     "number",
	"lastTemplate.point.y":     "number",
	"lastTemplate.region":      "any", // [4]float64 ratio — no array type yet
	"lastColor.count":          "number",
	"lastColor.cx":             "number",
	"lastColor.cy":             "number",
	"lastColor.center":         "point",
	"lastColor.center.x":       "number",
	"lastColor.center.y":       "number",
	"lastStopwatch.elapsedMs":  "number",
	"lastTry.errorMsg":         "string",
	"lastDetect.pixelCount":    "number",
	"lastDetect.pixelRatio":    "number",
	"lastROIScan.clusterCount": "number",
	"lastROIScan.clusters":     "any",
	"lastScreenshot.path":      "string",
	"lastBarTrack.cursorX":    "number",
	"lastBarTrack.targetX":    "number",
	"lastBarTrack.targetW":    "number",
	"lastBarTrack.confidence": "number",
	"lastBarTrack.yellowPx":   "number",
	"lastBarTrack.greenPx":    "number",
	// Live time / var-bookkeeping paths (Fishing v2 watchdog 用).
	// 这两类不进 per-tick snapshot, 总是返 live 值; 见 runtime/getsys.go 特殊路径处理.
	"now_ms": "number", // time.Now().UnixMilli(), 跟 tick snapshot 解耦 (watchdog 用)
	// varLastChange.<name> 是动态 wildcard — runtime/getsys.go 通过 HasPrefix 解析, 不进 schema 表.
	// 验证: validateGetSysNodes 也需补对 wildcard 的支持.
}

// IsLiveWildcardPath: 这些 prefix 是动态展开的, 不进 PathSchema 表枚举.
// 用于 validator + runtime 跳过精确匹配, 用 HasPrefix 验证.
func IsLiveWildcardPath(path string) bool {
	return hasPrefix(path, "varLastChange.")
}

func hasPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}
