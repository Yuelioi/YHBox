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
}
