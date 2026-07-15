---
kind: note
summary: "可信桌面应用生命周期与第三方 Process Plugin Host 是两种不同 authority：前者精确触发已安装 GUI 应用但不是 sandbox，后者必须隔离并只暴露 typed operation。"
activation: action
read_when: "修改应用启动/终止节点、Application Settings、process capability、AE/UE adapter、CLI worker、Process Plugin Host 或跨平台进程支持时"
recheck_when: "application provider ABI、Windows process identity API、Process Plugin Host sandbox 或支持平台声明变化时"
---
# 已安装桌面应用不能复用插件进程宿主，插件进程也不能复用桌面启动器

桌面自动化需要与用户会话中的受信 GUI 应用交互。Yotta 3.1 为此安装 immutable application profile：绝对 .exe 路径、可执行文件 SHA-256 和固定 argv。安装、授权与每次调用都重新验证文件 identity；workflow 只能选择逻辑 slot 和 launch/terminate operation，不能提供路径、参数、环境、工作目录、PID、进程名、通配符、URL 或文档。Shell/script host 与 Yotta 自身 executable 被拒绝。

Launch 在当前桌面用户 authority 下直接调用精确 executable，不经 shell、文件关联或 PATH，并使用固定环境 allowlist；PID 只在 provider 内部存在，不进入节点输出或 journal。Terminate 枚举进程并比较 OS file identity，只终止与 sealed executable 相同的实例，公开结果只有 terminated count。该 capability 是 dangerous + ConsentOnce，profile 语义变化会使旧 consent 失效。

这不是 sandbox。受信 GUI 应用仍可通过正常 OS API 使用当前用户的文件、注册表、网络、窗口和凭据。Settings/consent 只控制 workflow 能触发哪一个已安装应用，不能把该应用变成不可信代码。

第三方节点、CLI data processor、AE/UE adapter worker 属于 Process Plugin Host。它们必须绑定 content-addressed package 和 typed operation，在 Windows 使用 LPAC/AppContainer、atomic Job Object、exact inherited handles、resource budgets 与 fail-closed host admission；其他平台没有等价隔离时不安装 provider。两条 authority 不能互相 fallback，也不能保留 generic RunProgram/taskkill 兼容面。
