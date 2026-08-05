# Automation input and capture

## Boundaries and code

- configured Target/profile/provider：`internal/automation/installed/`。
- 平台中立 target/controller contract：`internal/automation/target/`、`controller/`。
- configured target runtime：`internal/targetruntime/`。
- 指针运动：`internal/automation/pointermotion/`。
- Workflow Node Contract/adapter：`internal/nodes/automation_*.go`、`internal/noderuntime/automation_*.go`。
- Windows input/capture：`pkg/input/`、`pkg/capture/`、`pkg/winutil/`；Android/Browser adapter 位于
  `internal/automation/controller/`。

Workflow 只绑定稳定 slot。Settings 的 installation 描述 target kind、adapter kind、profile version 和
adapter-owned profile；运行时才解析 HWND、ADB connection 或 CDP session。当前 target kind 是 desktop
window、Android device 和 Browser CDP，consumer 应使用语义 descriptor，不硬编码 Win32 identity。

## Input semantics

- `sendinput` 写入系统输入流并使用真实前台/光标，适合读取 Raw Input/异步状态的游戏；它受 UIPI、前台、
  secure desktop 和目标保护机制限制。
- `postmessage` 定向投递传统窗口消息，适合兼容的后台窗口；它不等于系统键鼠流，读取 Raw Input、真实
  cursor 或自有轮询状态的目标可以完全忽略。
- backend 必须完整实现所选语义。不能在 sendinput drag 中偷偷调用 PostMessage，也不能在 postmessage
  TypeText 中偷偷把全局 SendInput 当成功。Windows SendInput 以实际注入事件数判断成功。
- coordinate 必须明确 client/screen、pixel/ratio 和 capture resolution；SendInput absolute coordinate 不能
  直接使用 window-client ratio。
- `instant`、`linear`、`bezier` 由共享 pointermotion planner 采样；adapter 只负责把计划投递给目标。

Windows production 以 `requireAdministrator` 运行以匹配桌面自动化完整性；`task dev` 的 `asInvoker` 只是
Wails 开发进程约束，不是发布 fallback。管理员权限仍不能控制 secure desktop、受保护/反作弊进程。

## Chords and held input

- Press Keys 是原子 chord：依次按下 keys，保持 `hold-duration`，再逆序释放。`ALT` 等 modifier-only list 是
  合法 chord。
- 当一个动作需要跨节点保持 Alt/按键或鼠标按钮，例如“按住 Alt 显示游戏鼠标 → 点击退出 → 松开 Alt”，
  使用 Hold Keys → Click Pointer → Release Held Input；不要用多个独立 Press Keys 猜测保持状态。
- Hold Keys/Hold Pointer Button 返回 Run-owned `HeldInput` handle。显式 Release 消费它；cancel、failure、
  teardown 由 Run owner 兜底释放，不能靠无业务意义的 Sleep 延长图生命周期。
- KeyChord 编辑器捕获 modifier-only 值时在 keyup 提交，组合键在普通 keydown 提交；保持与 contract 的 key
  enum 一致。

## Capture and recording

- capture backend 是 configured desktop Target profile 的一部分；GDI/WGC 等能力必须通过 descriptor 和
  host availability 暴露，不在节点/前端散写平台 switch。
- 普通录制保存 Click/Drag/Scroll/Key/Sleep 等可编辑语义；按下期间移动超过阈值才折叠为 Drag，普通自由
  mouse move 不扩张成大量 workflow action。
- 精准录制保存 timestamped absolute move、RawDelta、button/key/wheel 到 immutable InputClip；回放保持稳定
  顺序和相对时间，画布不把每个采样点展开成节点。
- capture、input、recording 和 target session 都必须有 context deadline 与 owner close；cancel/failure 后
  不得留下 hook、held key/button、ADB command 或浏览器 profile。

## Verification

mock 只能证明 request/response 与错误映射。修改平台 adapter 后还需要相应真实旅程：Windows 使用
`task windows:smoke:automation` 串行占用独立桌面；Android/Browser 使用构建指南中的 exact target smoke。
纵向测试必须走 Source → Compiler → Target provider → adapter → Run journal，并检查真实副作用和释放状态。
