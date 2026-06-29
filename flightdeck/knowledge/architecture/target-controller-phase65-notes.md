# Target Controller Upgrade — Phase 65 Notes

SUMMARY: target capability validation now includes config-derived requirements for modifier-key clicks
READ WHEN: 改 ClickAt / ClickTemplate / modifier-key behavior / target capability validator 时
RECHECK WHEN: 新增会按配置切换输入语义的节点，例如右键/中键兼容矩阵、浏览器 selector click、Android app lifecycle actions

---

## Completed

- `ClickAt` / `ClickTemplate` 的静态 capability 保持基础点击路径：
  - `ClickAt`: `move` + `click`
  - `ClickTemplate`: `screenshot` + `click`
- Validator 新增 config-derived capability helper。
- 当节点 `Keys` 非空时，额外要求 `key-state`，因为 `node.ClickWithMods` 会调用 `KeyDown/KeyUp`。
- Android ADB 普通点击仍可通过；Android ADB 带修饰键点击会报 `UNSUPPORTED_TARGET_CAPABILITY`。

## Boundary

这次只覆盖已确认的 modifier-key click。其他配置相关差异仍待后续切片，例如 Android 的非 left button 语义、Browser selector-first action、Android app start/stop 节点等。

## Verification

- `go test ./internal/services/container -run "TestValidate_AndroidTargetWith(Input_NoMissingWin32WindowTarget|ClickAtModifierKeys_ReportsUnsupportedTargetCapability|ClickTemplateModifierKeys_ReportsUnsupportedTargetCapability)" -count=1`
