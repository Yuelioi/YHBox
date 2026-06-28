# Target Controller Upgrade — Phase 67 Notes

SUMMARY: node target capability strings are now guarded against controller vocabulary drift
READ WHEN: 新增或重命名 `node.TargetCapability` / `automation/controller.Capability` / 节点 capability metadata 时
RECHECK WHEN: controller capability model 从 bool set 改成枚举注册表，或 node package 直接依赖 controller capability 类型时

---

## Completed

- `internal/node` spec consistency tests now scan every registered node.
- Each `Spec.TargetCapabilities` entry must be recognized by `controller.CapabilitySet.Has`.
- This keeps the intentionally duplicated string vocabulary from silently drifting.

## Boundary

This is a metadata guard only. It does not validate backend support; Phase64-66 container validator handles backend profile matching.

## Verification

- `go test ./internal/node -run TestSpecConsistency_TargetCapabilitiesKnownByController -count=1`
