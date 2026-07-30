# Network targets

`internal/httpegress` 把 Settings 中的 Network Target 配置成一个 provider。配置包含 slot、HTTP/HTTPS base URL、响应大小和请求超时；这些值是用户可编辑的运行参数，不是 capability scope、授权范围或安全策略。Node Contract 选择逻辑 slot，并为当前 `GET` operation 提供相对路径和 query。

composition root 把 Network Target 放入不可变 target snapshot。每次 Run 通过 `internal/targetruntime` 按 slot 直接打开 provider、调用 operation 并关闭对象。Network Target 不进入 Host Profile、Policy、Consent、Run Grant、Credential Binding 或 TTL；修改配置只会让后续 Run 使用新的快照，已经运行的 Run 继续持有旧 provider 的生命周期引用。

provider 使用标准 Go `http.Client`。HTTP 和 HTTPS、本机地址、私网地址、远程地址、环境代理以及标准 redirect 都可使用；运行时不执行 DNS/IP 分类、私网阻止、redirect authority pinning 或额外 network admission。base URL 可以包含 path、query 和 user info，节点提供的相对路径按标准 URL resolution 解析，再合并 query。

响应大小和请求超时完全取自目标配置。响应当前按节点数据合同返回 status code、UTF-8 body 与 content type；response body 始终关闭。以后增加认证、请求体、自定义 header、binary、stream 或 WebSocket 时，为它们增加明确的 target operation 和数据合同即可，不需要新增 capability、consent 或 grant。
