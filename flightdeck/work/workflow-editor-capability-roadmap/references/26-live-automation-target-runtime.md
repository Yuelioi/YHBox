# Slice 26：自动化目标热更新运行时与窗口选择体验

## Outcome / Question

把“目标定义”与“运行期目标实例”分离：用户安装、捕获、修改或授权目标后，录制、资源创作、工具与 Workflow admission 在同一进程内原子看到同一代配置，不再把重启应用当作正常激活步骤。

同时让桌面目标表达真实用户任务，而不是强迫用户把一个瞬时 HWND 当成长期配置：固定窗口保留 exact title/class，动态或多窗口应用可使用显式 regex 和可预测的窗口选择策略。

## Completion criterion

- composition root 不再把启动时 `AutomationInstallations` / `AuthoringTargets` 冻结到进程退出；Settings 提交后可准备并切换新的目标运行时。
- Host Profile、Policy、Workflow providers 与 authoring resolver 使用同一代 sealed installations；切换期间不存在“录制已看到、Admission 看不到”或反向撕裂。
- 已排队/运行的 Run 保留其获准 provider；空闲时可回收旧 generation，进程退出时不会泄漏 adapter 资源。
- 捕获窗口的一次明确确认可完成应用身份安装、目标绑定和该精确身份的 workflow consent；安全摘要变化仍自动撤销 consent，不绕过 capability、admission 或 arm。
- exact 模式逐字符保留 Win32 title；regex 使用 RE2；多匹配不随机，UI 明确显示并要求选择 `unique` 或受约束的前台/顶层匹配策略。
- 正常安装、修改、授权与撤销无需重启；只有真实的进程级能力变更才允许出现重启提示。
- 同进程完成捕获 → 录制/截图 → Run 的回归测试与真实管理员窗口 smoke，随后执行阶段级 `task check` 和 production build。

## Design

`LiveAutomationRuntime` 是深模块：

- 输入：由 Settings 生成的 application / automation installation drafts。
- 准备：seal、验证并构造候选 installations，不发布半成品。
- 切换：在 Application command boundary 内同时更新 Admission environment 与 provider registry，再切换 authoring resolver。
- 生命周期：运行中的旧 provider 延迟回收；空闲 generation 立即回收；shutdown 统一关闭剩余资源。

Settings 只负责持久化用户意图，不再承担运行时激活协议。Recording、Asset、Tools 只依赖稳定的 target resolver 接口，不各自实现 reload。

## Verification

- 单元：Admitter environment replacement 在同一对象上使用新 profile/policy。
- 单元：Application provider registry 与 environment 原子切换，旧 Run 不失去已获准 provider。
- 单元：AuthoringTargets 替换后同一 resolver handle 立即解析新 slot。
- 集成：同一进程 Settings 新建/授权 target 后，recording validation 和 Workflow admission 均成功；删除或改身份后两者同步失效。
- UI：捕获确认一次完成安装/绑定/授权，保存成功不使用多余 toast，失败才 toast；无正常重启提示。
- 人工：异环和多窗口应用各完成一次 exact/regex 捕获、录制、截图、鼠标/组合键 Run。

### 2026-07-18 real-data diagnosis

真实 `bin/data/workspace-3.1/workflows/7db4c59f-5b3b-42ab-9743-762c9750440a.json` 使用 `Press Keys` 并绑定 `window-target`。Settings 中 application / target consent、slot、raw exact title、class 与 executable digest 均正确，但 Admission 仍返回 `target_unavailable`。

根因不是热更新或授权：`builtinHostProfile` 只把 automation provider 的 `input` resource 投影成通用 `automation/input` capability，漏掉了 `Press Keys` 要求的 `automation/key-input`，也同时漏掉 `Move Pointer Relative` 要求的 `automation/desktop-input`。修复后按 adapter descriptor operations 投影专用 capability；桌面和 Browser 支持的 `press-keys`、桌面支持的 `move-relative` 不再被错误分类为目标不存在。

同进程集成回归已改为 `Press Keys + editor-window`：修复前稳定得到 `admission.target_unavailable`，修复后成功 admission 并创建 queued Run。相关 appbootstrap / admission / application / installed / services 聚合测试通过，新 production exe 已构建。

## Out of scope

绕过 Windows UIPI/secure desktop、规避反作弊、对未确认的未来目标静默授权、把原生 HWND 写进 Workflow Source。
