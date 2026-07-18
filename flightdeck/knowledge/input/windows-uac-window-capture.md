---
kind: trap
summary: "Yotta 是桌面自动化优先产品；Windows production manifest 固定 requireAdministrator，桌面子进程继承管理员权限，自启使用交互式最高权限计划任务。"
activation: action
read_when: "修改 Windows manifest、窗口捕获、录制 hook、输入注入、桌面应用启动、自启或权限相关 UI/API 时"
recheck_when: "产品不再以桌面自动化为主，或引入独立 privileged host、UIAccess 签名部署、secure desktop 专用能力时"
---
# Windows 默认管理员权限模型

Yotta 的核心用途是桌面自动化。Windows UIPI / 进程完整性级别会阻止普通权限进程可靠监听或操控管理员目标，典型表现是普通窗口可捕获，而管理员游戏收不到 F9、键鼠 hook 或窗口消息。为避免各节点、捕获、录制和输入后端分别维护权限兜底，Windows production manifest 固定为 `requireAdministrator`。

不要恢复 `asInvoker`、按需 `runas`、提权状态探测、双运行级别或静默降级分支。Yotta 启动的桌面应用按产品约定继承当前管理员 token；这能让目标与自动化宿主保持一致的完整性级别。若未来确实需要启动标准权限子进程，应先重新做产品与安全设计，而不是在单个 adapter 内增加临时 fallback。

HKCU Run 无法表达这一权限契约。Windows 自启使用当前交互用户的登录计划任务，并同时设置 interactive-only 与 highest run level；不要改为 SYSTEM，因为 SYSTEM 任务不在用户交互桌面，Wails UI 不可见。

管理员权限不能绕过 Windows secure desktop，也不保证能操控受保护进程或第三方反作弊目标。默认管理员实例仍捕获失败时，应检查目标保护机制、快捷键、窗口身份读取和 backend 能力，不要继续叠加提权逻辑。
