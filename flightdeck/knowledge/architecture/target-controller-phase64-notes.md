# Target Controller Upgrade — Phase 64 Notes

SUMMARY: `NeedsTarget` 节点新增具体 target capability contract，并在容器校验期对照 controller profiles
READ WHEN: 新增输入/截图/视觉节点、调整 controller capabilities、改 Android/Browser/Win32 target selection、改 catalog/list_nodes 输出时
RECHECK WHEN: 引入路径敏感 target 校验、支持更多 target kind、给节点增加 config-dependent capability 时

---

## Completed

- `node.Spec` 新增 `TargetCapabilities []TargetCapability`。
- Catalog JSON/Markdown 输出 `targetCapabilities` / `Caps:...`，让 LLM/MCP 能看到节点需要的具体目标能力。
- 输入、截图、视觉节点按真实 runtime 调用声明 capability：
  - `screenshot`: `Capture` 和视觉检测节点。
  - `move` / `click` / `scroll` / `drag` / `mouse-button` / `move-relative` / `key-state` / `text`: 对应输入节点。
- Container validator 新增 `UNSUPPORTED_TARGET_CAPABILITY`。
- Validator 会沿 exec 边反向找最近上游 target selection，并用 `automation/controller` backend profile 校验 capability。
- `Window` 输入已连线的 target-aware 节点跳过该校验，因为该节点走显式 Win32 window override。
- `NeedsTarget` 节点必须声明 `TargetCapabilities`，由 `internal/nodes/all` guard 防漂移。

## Boundary

这次只做图内最近上游 target selection 的静态校验。跨子图调用、分支汇合、多 target 动态切换仍保持保守：无法唯一判定 target 时不报，运行时 controller capability check 继续兜底。

`NeedsTarget` 仍表示“节点依赖 active automation target”。`TargetCapabilities` 表示“这个节点在 active target 上至少需要哪些 controller 能力”。

## Verification

- `go test ./internal/services/container -run TestValidate_AndroidTargetWithMouseMoveRel_ReportsUnsupportedTargetCapability -count=1`
- `go test ./internal/node ./internal/nodes/all ./internal/catalog ./internal/services/container ./internal/services/container/runtime -count=1`
