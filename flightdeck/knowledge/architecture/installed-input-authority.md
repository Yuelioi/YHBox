---
kind: note
summary: "Yotta 3.1 输入自动化只接受安装式 exact target：应用摘要、唯一 exact title/class、固定 backend/timeout 与 workflow consent；图和 journal 不得携带 path、PID、HWND、文本或按键。"
activation: action
read_when: "新增或修改 input node、自动化 target、窗口解析、SendInput/PostMessage backend、输入 consent、provider runtime 或 action journal 时"
recheck_when: "installed target identity、窗口唯一性、backend failure contract、held-state cleanup 或 input capability/grant 结构改动后"
---
# Installed input authority is exact and host-owned

Workflow 只能引用安装 slot。安装记录绑定已安装 Application 的 executable path 与 SHA-256，但不会继承其 launch argv；目标窗口只能用非空 exact title/class 约束解析，零匹配和多匹配均失败，不允许 active-window、部分匹配、PID、HWND 或运行时路径作为图数据。

Input provider 在每次 operation 前重验 executable identity 和唯一窗口，固定使用安装时选择的 SendInput 或 PostMessage backend。点击、拖拽、按键和文本是 provider 内原子操作；失败或取消必须尝试释放 held key/button。SendInput 必须验证实际注入计数，PostMessage 必须检查投递结果，不能把 OS 拒绝伪装为节点成功。

Graph/Program/Grant 只持 slot 与 operation；journal 只记录稳定 operation 和安全计数。输入文本、键码、坐标目标身份、path、PID、HWND 与 backend 实现细节不得进入 durable journal。Consent 绑定整个 immutable installation；identity、selector、backend 或 timeout 任一语义变化都撤销旧 consent。
