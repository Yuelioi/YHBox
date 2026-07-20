# Slice 19：Browser CDP installation 产品闭环

## Outcome / Question

把已有 Browser CDP controller 从隐藏底层构件提升为通过 exact installation、consent、Catalog 与健康诊断约束的真实产品能力。

## Completion criterion

- Browser Adapter 通过 Stage 5 target-kind registry 注册 descriptor、profile validation、operation set 与 provider driver。
- Profile 固定 loopback discovery endpoint 和 exact page identity；拒绝非 loopback/漂移 websocket 身份与 ambient remote debugging authority。
- Settings 支持发现 page、安装、consent、健康检查和 stale/offline/actionable diagnostics。
- 通用截图、点击、移动、拖拽、滚动、组合键和文本节点按 capability targetKinds 向 browser-cdp 开放；不支持的应用生命周期操作不出现。
- 工作流创建/Inspector 能发现 Browser target，Source 仍只保存 installation slot。
- 使用真实 Chrome 或 Edge 显式调试端口完成截图和输入 smoke 后才声明支持。

## Blocked by

Stage 5 Adapter seam 与现有 browsercdp discovery/client/controller。

## Verification

Adapter conformance、profile/provider/service/Catalog/Settings 定向测试；Stage 9 批量运行 task check、task build、Windows WebView 与真 Browser CDP smoke。

## Out of scope

静默修改用户浏览器启动参数、连接远程非 loopback 浏览器、绕过用户显式 consent、浏览器扩展分发。

## Result

Completed。Browser CDP discovery/client 已下沉到 automation 深模块；生产 TargetTypes 与 runtime registry 共用单一 Adapter 注册源。Profile 固定字面 loopback HTTP origin、page id 与同 authority `/devtools/page/{id}` WebSocket，拒绝 redirect、remote authority、page/WS identity 漂移。

Settings 已提供 endpoint、页面发现、精确安装、consent、健康/离线诊断；工作流新增 Browser 模板，Inspector 继续只保存 installation slot。通用点击、移动、拖拽、滚动、文本、组合键和截图对 browser-cdp 开放，应用生命周期、相对移动和录制回放保持隐藏/拒绝。组合键补齐 CDP key/code/virtual-key/modifiers 语义。

阶段证据：完整 `task check`、production `task build`、正式 WebView smoke 均通过；Chrome 127.0.0.1:9337 与 Edge 127.0.0.1:9338 的真实 smoke 均完成 viewport、文本输入、指针移动和截图。WebView workflow/editor 与 assets PNG 已人工查看。
