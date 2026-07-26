# Installed input authority

Workflow 只引用 installation slot。安装记录绑定可信 application identity；Windows 当前使用 executable SHA-256，未来平台使用各自 adapter-owned identity。PID、HWND、前台窗口、运行时路径和 native handle 不能成为 Workflow Source 或 Grant 的身份。

窗口 selector 支持两种显式语义：

- `exact`：标题逐字符保留并与 class/application identity 联合匹配；
- `regex`：用户明确选择并保存 RE2 pattern。

零匹配必须返回可恢复的 target-unavailable；多匹配必须按安装中声明的 deterministic selection policy 消歧或要求用户选择，不能随机取第一个，也不能静默退化为 active window。捕获时得到的 HWND 只是一次确认和临时解析结果。

Input provider 每次 operation 重验应用身份和窗口 selector，使用 installation 固定的 backend。按键、文本、点击、拖拽和 held input 必须验证 OS 投递/注入结果；失败、取消和 Run 结束统一释放 held key/button。held input 由 deep Run-owned HandleRef 持有，不允许节点调用结束即释放，也不靠额外 Sleep 保活。UIPI/secure desktop/反作弊边界不能伪装成成功。

Source/Program 可以包含节点契约允许的 typed 操作参数，例如 KeyCode、文本、Point 和时长；这些数据不能反向扩大 target authority。Grant 只封存 slot、operation、provider 和 scope。Action journal 记录 operation、稳定错误码、计数和脱敏摘要，不记录原始文本、完整键序列、坐标轨迹、path、PID 或 HWND。

profile identity、selector、selection policy、backend 或 capability manifest 的语义变化会生成新 digest 并撤销旧 consent。设置提交后应以原子 installation generation 同时更新 authoring、admission 和 provider；正常修改不得要求重启应用。


## Windows R2 verification

2026-07-18 的 R2 native gate 已证明 exact 尾空格、显式 regex、unique ambiguity、真实 SendInput drag、注入数检查、held cleanup 和窗口操作族。真实 `HTGame.exe / UnrealWindow / 异环··` 通过 versioned profile 与 consent 运行 ESC workflow。以后修改 adapter/native path 时运行 `task windows:smoke:automation`；mock 不能替代这条反馈环。