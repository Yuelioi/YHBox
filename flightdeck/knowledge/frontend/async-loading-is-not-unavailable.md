# 异步加载与 immediate watcher 初始化陷阱

当一个视图依赖多个异步数据源时，必须分别表达每条请求的 pending、success 和 unavailable/error 状态。初始化值为 null 只说明结果尚未到达，不能直接渲染成“窗口未开”“未找到”或“不可用”。

不要用一个 Promise.all 或共享 loading 标志把快速的本地记录和可能超时的外部资源解析绑在一起。快速数据应先独立呈现；慢请求只控制自己的状态区域。否则用户会同时看到错误文案和 skeleton，把中间态理解成最终失败，同时已有内容也被无关请求阻塞。

Vue 的 immediate watcher 在 setup 执行到 watch 注册处时就会同步触发首轮回调。回调在第一个 await 之前访问的任何 const/ref，都必须已经在源码顺序上完成初始化。即使 TypeScript 类型检查和生产构建通过，访问后声明的绑定仍会在运行时触发 temporal dead zone 的 ReferenceError，并中断后续异步加载。

回归测试至少应锁定：

- 真实挂载组件并捕获 app errorHandler，保证 setup 和 immediate watcher 没有运行时异常。
- pending 时显示明确加载反馈，不出现 unavailable 终态文案。
- 慢请求未完成时，已经返回的独立数据仍然可见。
- 异步结束后，关键业务内容确实出现在 DOM，而不只是断言源码包含某段文本。

请求结束后确实失败时再进入 unavailable/error 状态；需要诊断的链路还应保留真实错误原因，不能只用 null 静默折叠所有失败。
