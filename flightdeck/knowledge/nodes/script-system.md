---
kind: note
summary: "3.1 Script 只在 one-shot 隔离 worker 中执行 canonical JSON → JSON；没有节点函数、Wails/文件/网络/窗口或 ambient variable authority。"
activation: action
read_when: "修改 Script 节点、scriptengine、ScriptWorker、隔离、输入输出 contract、超时或跨平台 script 支持时"
recheck_when: "script guest/runtime protocol、LPAC/AppContainer confinement、host feature admission 或 Script Node Contract 改变后"
---
# Isolated Script 3.1

当前 Script 节点不是旧 goja 控制台，也不是“节点即函数”。Contract 只有：

- config：`source`、`timeoutMilliseconds`；
- typed input：canonical JSON `input`；
- typed output：canonical JSON `result`；
- signal：`in → completed | failed`；
- host feature：隔离 worker 可用。

Script 在 one-shot worker 中执行，默认零 ambient authority：没有 Wails、DOM、文件、网络、进程、窗口、Node Catalog、Run State、credential 或 capability session。需要外部 effect 时应使用显式节点/GraphCall，而不是给脚本开宿主 API。

Windows production 必须通过 LPAC/AppContainer、atomic Job Object、受限 handle/环境和 bounded protocol。缺少等价 confinement 的平台不安装该 host feature并 fail closed，不能回退为普通子进程。

request/response 使用严格 canonical JSON 和 byte/depth/time/stack budget；worker crash、协议漂移、deadline、guest throw、隔离不可用分别映射 contract-declared error code。取消必须终止整个 worker/job。原始 source/input/output 不进入普通日志或 action summary。

旧 `yt` 控制台、动态输入、`$变量`、节点函数绑定、子图转脚本和在脚本中调用自动化节点均属于 3.0 legacy 行为，不得恢复为兼容层。
