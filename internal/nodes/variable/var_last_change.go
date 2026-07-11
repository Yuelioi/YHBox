// internal/nodes/variable/var_last_change.go
// VarLastChange — pure-data，返某变量上次 Set/Inc 的 unix-ms 时间戳（没设过返 0，live）。
// 取代旧 $sys.varLastChange.X。watchdog 用 Now - VarLastChange(x) 算 state 多久没变。
package variable

import (
	"fmt"

	"github.com/yottaapp/yotta/internal/node"
)

func init() { node.Register(&VarLastChange{}) }

type VarLastChange struct{}

const (
	vlcInVarName = "VarName"
	vlcOutValue  = "Value"
)

func (VarLastChange) Spec() node.Spec {
	return node.Spec{
		Kind:                "VarLastChange",
		Category:            "Variable",
		RuntimeCapabilities: []node.RuntimeCapability{node.RuntimeCapabilityVars},
		Inputs: []node.InputSpec{
			{Name: vlcInVarName, Type: "String", Required: true, Semantic: "varname",
				Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs:    []node.OutputSpec{{Name: vlcOutValue, Type: "Number"}},
		IsPureData: true,
	}
}

func (VarLastChange) Evaluate(ctx node.Ctx, in node.Inputs) (any, error) {
	name := in.String(vlcInVarName)
	if name == "" {
		return nil, fmt.Errorf("VarLastChange: missing VarName")
	}
	return float64(ctx.Services().Vars.LastChange(name)), nil
}
