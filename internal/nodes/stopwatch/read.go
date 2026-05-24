// internal/nodes/stopwatch/read.go
// StopwatchRead — 读指定 key 的秒表 elapsed (毫秒). 老 runtime:
// stopwatch_nodes.go::execStopwatchRead 把结果推 SysState.LastStopwatchElapsedMs
// (因为老 runtime 没有通用 dataOut 总线). 新框架的 v4 spec 已声明
// DataOut: {elapsedMs: PinNumber} (specs/system.go), 所以这里走数据出口
// — exec exit "Out" 带 Data 字段 elapsedMs (跟 CheckTemplate Found 带
// Point/Conf 同模式).
//
// 不存在 key → 0 (老 runtime 一致).
package stopwatch

import (
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&Read{}) }

type Read struct{}

const (
	swReadInExec       = "in"
	swReadInKey        = "Key"
	swReadOutOut       = "Out"
	swReadDataElapsedMs = "ElapsedMs"
)

func (Read) Spec() node.Spec {
	return node.Spec{
		Kind:        "StopwatchRead",
		Version:     1,
		Category:    "Stopwatch",
		DisplayName: "秒表 读取",
		Description: "读指定 key 的 elapsed (毫秒). running 返 now-start; stopped 返 stoppedAt-start; 不存在 → 0.",
		Inputs: []node.InputSpec{
			{Name: swReadInExec, Type: "Exec"},
			{Name: swReadInKey, Type: "String", Required: true, Default: "default",
				DisplayName: "key",
				Doc:         "跟之前 StopwatchStart 同一个 key",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: swReadOutOut, Type: "Exec", DisplayName: "完成",
				Data: []node.DataField{
					{Name: swReadDataElapsedMs, Type: "Number", Doc: "elapsed 毫秒"},
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

func (Read) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("sw %s = %dms", in.String(swReadInKey), out.Int(swReadDataElapsedMs))
}
