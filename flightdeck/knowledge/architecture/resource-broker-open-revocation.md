# Resource Broker Open and revocation linearization

## 为什么容易出错

`AuthorizeOpen` 与 `Provider.Open` 都可能阻塞、忽略取消或在终止边界后成功。如果 Broker 只检查一次 closed/revoked 状态，Provider 可以在 RevokeRun/Close 之后产生未登记对象；如果 RevokeRun 和 Close 都能从全局 lease map 抢对象，任一流程都可能提前返回或漏报 cleanup error。

## 唯一正确的所有权

- Open 在任何授权或 provider side effect 前登记 cancellable attempt。
- RevokeRun 先永久标记 run revoked，再取消该 Run 的 attempts；Open/Borrow/Invoke 从此立即返回终止错误。Drop 仍允许用于幂等清理。
- Open 在 AuthorizeOpen 后、Provider.Open 后和 lease 注册前重验终止状态。Provider 已成功但无法注册时，必须关闭该值，并把 cleanup error 归入对应 Run。
- 首次 RevokeRun 创建唯一 revocation state。调用方 context 只控制等待；即使超时，后台清理仍继续，后续调用等待同一个结果。
- 已登记 revocation 的 Run 独占其 leases、objects 与 Open cleanup errors。全局 Close 必须跳过这些资源，只等待 revocation 完成并汇总尚未被调用方观察的结果。
- 没有 revocation owner 的资源才由 Close 接管。Close 取消并等待全部 Open attempt，再等待其启动时仍未完成的 revocation，最后才返回。
- Provider object 只允许 exactly-once Close；active Invoke 必须先收到 lifetime cancellation，cleanup 再等待其退出。

不要用超时返回后停止清理、第二套补偿 goroutine、Close/Revoke 双方竞抢或“下次 Sweep 再处理”的兜底设计。这些路径会让 authority 与资源所有权分叉。
