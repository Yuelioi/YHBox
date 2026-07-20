# Slice 36：Android、Browser 与平台 Adapter 连续性

## Outcome / Question

恢复 Android/ADB 与 Browser CDP 的产品闭环，并用最小 macOS descriptor/compile proof 证明新增平台不需要修改 workflow core 或中央 ProfileDraft。

## Completion criterion

- Android adapter 自己定义 profile schema/version、seal、health、editor descriptor 和 runtime factory。
- ADB input/capture/template/clip 使用统一 manifest/admission/journal 路径。
- Browser exact endpoint/page installation 与 operation narrowing 可运行并可诊断。
- 最小 macOS no-runtime adapter 可以注册/编译；新增它不修改 Source、通用节点、compiler、scheduler 或 central profile union。

## Blocked by

Slices 31、34–35。

## Verification

- G16 Android emulator/device matrix、G17 Chrome/Edge controlled smoke。
- Windows golden journeys 回归不变。
- cross-platform core tests、GUI compile 和 adapter conformance 在 Stage R4 末批量执行并形成 commit。

## Out of scope

- 不承诺本次实现 macOS native automation runtime。
- 不把仅有 controller/client 或 WebView 可见当作平台支持。
- 不允许 unsupported platform 静默 fallback 到 desktop Windows profile。

## Result

Completed。

- Android manifest 声明 playback；InputClip 的 touch、drag、scroll 与可映射单键通过统一 provider session，桌面相对移动和录制 chord 在 Android 上明确 fail closed。
- Settings 通过 Wails RPC 按 exact serial 搜索/虚拟化应用并保留精确包名手工输入；断连、unauthorized、多设备 identity、本地 emulator reconnect 与 4096 项预算都有定向诊断。
- G16 在 `bilibili_api35 / emulator-5580` 走完 Source → Compiler → Admission → installed provider → journal：activate、capture、template click、drag、InputClip playback、stop-app 全部成功；最终用例 22.40s、package 28.777s。
- 首次真实旅程暴露 ADB effect 继承无 deadline 的 Run context 后可永久卡住；现在所有 ADB 子进程由 Adapter 叠加默认 10 秒上限、继承更短上游取消，并以阻塞 runner 回归测试固定，副作用不自动重试。
- G17 在受控 Chrome 与 Edge 都完成 exact page discovery、Type Text、Capture 与 journal；未声明窗口操作 fail closed，endpoint/page 改变旋转 profile generation 和 consent digest。
- macOS no-runtime fixture 能独立注册 descriptor、seal profile/manifest 并 HostAvailable=false；`internal/automation/installed` 与 `internal/appbootstrap` 的 darwin/arm64、linux/amd64 cross-compile 均通过，未修改 Source、通用节点、compiler 或 scheduler。
- Stage R4 批量门禁通过：`task check`（282.4s，44 个前端文件 185 tests）、`task build`（36s）、Chrome/Edge/Android 真实 smoke、Wails/Node contracts 与 production manifest 全绿；Yotta.exe SHA-256 `F7B996866CD82A79493BDF8139274910B8C8E43AD751E9DDEC28677FE740C3D2`。
- controlled browser profile/process 与 Android emulator 已精确清理；`adb devices -l` 为空。
