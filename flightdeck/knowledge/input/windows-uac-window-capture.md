---
kind: trap
summary: "普通权限 Yotta 的窗口捕获无法跨 UAC 完整性级别可靠监听管理员目标；保持 asInvoker，并在捕获超时后显式按需 runas 重启。"
activation: symptom
read_when: "普通应用窗口可捕获但管理员应用或游戏收不到 F9；修改窗口捕获、录制 hook、管理员运行策略或 manifest 时"
recheck_when: "窗口捕获不再依赖当前进程的 LL keyboard hook，或引入独立提权 capture helper / UIAccess 签名部署时"
---
# Windows UAC 窗口捕获边界

Windows 的 UIPI / 进程完整性级别会让普通权限 Yotta 无法可靠接收管理员目标中的低级键盘事件。典型表现是 QQ 等普通窗口可捕获，而以管理员权限运行的游戏按同一 F9 没有结果。

不要因此把桌面程序 manifest 全局改成 `requireAdministrator`：这会让每次启动都弹 UAC，并把不需要提权的设置、编辑和网络能力一起抬高。保持 `asInvoker`，在捕获超时且 Yotta 未提权时给出明确原因，由用户显式确认后通过 Shell `runas` 启动同一可执行文件；新实例成功启动后旧实例正常退出。

如果 Yotta 已提权仍捕获失败，再检查目标是否为受保护进程、反作弊限制、快捷键配置或窗口身份读取失败，不要继续把所有失败都归因于 UAC。
