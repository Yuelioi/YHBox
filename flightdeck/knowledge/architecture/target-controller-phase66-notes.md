# Target Controller Upgrade — Phase 66 Notes

SUMMARY: non-left click button configs now require `mouse-button`, so Android ADB tap does not silently pretend to right/middle click
READ WHEN: 改 ClickAt / ClickTemplate / controller button support / target capability validator 时
RECHECK WHEN: Android 引入真实鼠标按钮 backend、Browser/Win32 click button 行为变化、click controller capability 拆分时

---

## Completed

- `ClickAt` / `ClickTemplate` 的 `Button=right|middle` 会额外派生 `mouse-button` capability。
- Android ADB profile 没有 `mouse-button`，所以右键/中键点击在编辑期报 `UNSUPPORTED_TARGET_CAPABILITY`。
- `Button` 为空或 `left` 保持只要求基础 `click` capability，Android tap 继续可用。

## Boundary

`click` 表示普通点击/tap。`mouse-button` 表示目标能够表达非 left button 或独立 down/up 语义。当前没有新增运行时行为，只把已知不支持的配置提前拦截。

## Verification

- `go test ./internal/services/container -run "TestValidate_AndroidTargetWith(Input_NoMissingWin32WindowTarget|ClickAtRightButton_ReportsUnsupportedTargetCapability|ClickTemplateMiddleButton_ReportsUnsupportedTargetCapability)" -count=1`
