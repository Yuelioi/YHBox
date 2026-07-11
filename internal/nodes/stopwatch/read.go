// StopwatchRead — 读指定 key 的秒表 elapsed (毫秒).
// 结果走数据出口: exec exit "Out" 带 Data 字段 elapsedMs (跟 CheckTemplate Found 带 Point/Conf 同模式).
// 不存在 key → 0.
package stopwatch

import (
	"github.com/yottaapp/yotta/internal/node"
)

func init() { node.Register(&Read{}) }

type Read struct{}

const (
	swReadInExec        = "In"
	swReadInKey         = "Key"
	swReadOutOut        = "Done"
	swReadDataElapsedMs = "ElapsedMs"
)

func (Read) Spec() node.Spec {
	return node.Spec{
		Kind:                "StopwatchRead",
		Category:            "Stopwatch",
		RuntimeCapabilities: []node.RuntimeCapability{node.RuntimeCapabilityStopwatches},
		Inputs: []node.InputSpec{
			{Name: swReadInExec, Type: "Exec"},
			{Name: swReadInKey, Type: "String", Required: true, Default: "default",
				Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: swReadOutOut, Type: "Exec",
				Data: []node.DataField{
					{Name: swReadDataElapsedMs, Type: "Number"},
				}},
		},
	}
}

func (Read) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	key := in.String(swReadInKey)
	if key == "" {
		return nil, errStopwatchEmptyKey
	}
	elapsed := ctx.Stopwatches().Read(key)
	return ctx.Out(swReadOutOut).Set(swReadDataElapsedMs, elapsed).Fire(), nil
}
