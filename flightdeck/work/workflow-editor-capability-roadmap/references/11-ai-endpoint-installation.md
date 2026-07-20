# Slice 11：可配置 AI API endpoint

## Outcome / Question

AI 连接允许配置 API URL/endpoint，同时保持 endpoint、凭据和协议属于可信主机安装，Workflow Source 只引用稳定 slot，不把网络权限塞进图。

## Completion criterion

- AI installation profile 增加规范化 endpoint/base URL 字段；官方 OpenAI/Anthropic 地址仍可作为明确默认值，UI 可查看和编辑。
- endpoint 参与 profile digest、consent 与 installation lock；地址改变会使旧授权失效，TestProfile 和真实运行使用同一 sealed endpoint。
- provider-native 协议保持显式：OpenAI Responses、Anthropic Messages 或未来单独定义的 compatible provider 不得静默互相回退。
- URL 校验拒绝用户信息、fragment、无 host 和不可信 scheme；默认要求 HTTPS，本地 loopback HTTP 如支持必须显式标识风险。
- API key 继续进入 secure store，不进入 endpoint、settings 明文、Workflow Source、journal 或错误日志。
- 设置页提供 endpoint 输入、恢复官方默认、连接测试与可定位错误；原地显示测试成功，失败才 toast/inline error。
- 网络 capability 与 redirect/proxy/SSRF 边界有明确策略，不能因自定义地址获得环境网络的 ambient authority。

## Blocked by

Stage 3 批量验收；需决定首期只支持 provider-native endpoint，还是新增显式 OpenAI-compatible Chat Completions provider。

## Verification

先做 profile seal/digest、URL validation、consent invalidation、TestProfile endpoint 注入和 secrets redaction 定向测试；Stage 4 完成后统一运行 task check、task build 和官方/本地兼容测试服务 smoke。

## Out of scope

把 endpoint 或 API key 写进工作流、自动探测多种协议、静默回退 Chat Completions、任意代理脚本、自建模型管理平台。

## Result

Completed。AI Model Profile v2 新增规范化 exact endpoint 与 allowLocalHttp，官方 OpenAI Responses/Anthropic Messages 完整 URL 是显式默认；自定义地址拒绝 userinfo/query/fragment/空 host/空 API path，默认只允许 HTTPS，HTTP 仅在用户显式确认且 host 为 localhost/loopback 时接受。endpoint 进入 profile digest、evaluation subject、workflow consent 和 installation lock，设置语义修改会清除旧 evaluation/consent。NewNativeProvider/TestProfile/真实 installation 均从 sealed profile 读取同一 endpoint，协议不探测、不回退；生产客户端禁用环境代理和 redirect。设置页可编辑、恢复官方默认、明确本机 HTTP 风险并原地显示测试结果，API key 仍不进入 settings/graph/journal。AI/services Go 测试、6 个前端测试、1552-key i18n 与 typecheck 通过；官方/本地服务 smoke 留到 Stage 4 门禁。
