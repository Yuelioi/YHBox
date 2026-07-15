// internal/node/errorcodes.go
// 集中错误码注册表 + typed 节点错误. 只有实现 Coded 的错误会被 dispatch 失败路由
// 截获 (走节点/region 的 Fail 出口); 裸 fmt.Errorf (配置错) 照旧冒泡中断.
package node

import "fmt"

// ErrCode 机器可读错误类型码. 字面值 snake_case; 常量 CamelCase.
type ErrCode string

const (
	CodeError          ErrCode = "error"           // 兜底/未分类
	CodeLaunchFailed   ErrCode = "launch_failed"   // target application 起不来
	CodeCaptureFailed  ErrCode = "capture_failed"  // 截屏/视觉计算失败
	CodeWriteFailed    ErrCode = "write_failed"    // 文件写盘/路径失败
	CodeNotFound       ErrCode = "not_found"       // 找不到窗/目标
	CodeWindowInvalid  ErrCode = "window_invalid"  // Window 输入指向的句柄已失效
	CodeTimeout        ErrCode = "timeout"         // 超时
	CodePlaybackFailed ErrCode = "playback_failed" // 回放失败
	CodeSendFailed     ErrCode = "send_failed"     // 输入发送失败
	CodeThrown         ErrCode = "thrown"          // Throw 节点默认码

	// CodeSubgraphNoExit — 多出口子图体跑干没到达任何出口 marker, 出口歧义.
	CodeSubgraphNoExit ErrCode = "subgraph_no_exit"
	// CodeSubgraphRecursion — 子图调用嵌套超上限 (脚本动态调用可绕过图层静态防环).
	CodeSubgraphRecursion ErrCode = "subgraph_recursion"
)

// ErrorCodes 合法码全集 (推荐集, 非强约束: 用户 Throw 自填码 / 插件返非注册码仍合法).
var ErrorCodes = map[ErrCode]struct{}{
	CodeError: {}, CodeLaunchFailed: {}, CodeCaptureFailed: {}, CodeWriteFailed: {},
	CodeNotFound: {}, CodeWindowInvalid: {}, CodeTimeout: {}, CodePlaybackFailed: {}, CodeSendFailed: {}, CodeThrown: {},
	CodeSubgraphNoExit: {}, CodeSubgraphRecursion: {},
}

// Coded — dispatch 失败路由的准入接口. *NodeError / *ThrowError 实现它.
type Coded interface{ ErrCode() ErrCode }

// NodeError 运行时可失败节点返回的 typed 错误. cause 保底层错误链.
type NodeError struct {
	Code    ErrCode
	Message string
	cause   error
}

func (e *NodeError) Error() string    { return e.Message }
func (e *NodeError) Unwrap() error    { return e.cause } // errors.Is/As 可追根因
func (e *NodeError) ErrCode() ErrCode { return e.Code }

// Failf 构造带码、显式 cause(可 nil) 的 NodeError. 运行时可失败节点用它替代 fmt.Errorf.
func Failf(code ErrCode, cause error, format string, args ...any) error {
	return &NodeError{Code: code, Message: fmt.Sprintf(format, args...), cause: cause}
}
