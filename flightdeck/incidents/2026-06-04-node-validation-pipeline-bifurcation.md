---
status: active
last_updated: 2026-06-04
when_to_read: 写"给节点加校验 / 加 Validate"类 spec 或代码前; 撞"我加了 Validate 但编辑期没报/重复报"
applies_to: [validation, node-framework, validator, spec-design, two-pipeline]
---

# 节点校验有两条管线，别把它们当一条

**坑**: P1-1 §D 立项说"DetectColorHSV 没 Validate，HSV 倒置只能 Run 时报"，于是给节点加了个 `Validate()` 方法。真相是**编辑期 HSV 校验早就有**——只是在另一条管线里。§D 漏核 `validator.go` 直接基于"没有"的幻觉立项，加出来的是重复死码（头号铁律反面教材）。

**两条管线**（写校验前必须先认清要进哪条）:

1. **容器/编辑期校验** = `internal/services/container/validator.go`。
   - `checkGraphPerKind` 里**按 `n.Kind` 写死 switch** 分发到手写的 `validateXxx(n)` 函数。
   - 这是用户在 NodeInspector 看到的红错来源。
   - **它不调节点自己的 `Validate()`**。要加编辑期校验 → 改这里的 switch / 对应 `validateXxx`。

2. **节点级 `Validator` 接口** = 节点实现 `Validate(in node.Inputs) []node.ValidationError`。
   - 只在 `internal/node/engine.go`（`rn.Validate != nil` → Run 前跑）触发，即**运行期**。
   - 注册见 `registry.go`（`impl.(Validator)` → `rn.Validate`）。
   - 编辑器**不**走这条。

**怎么做**:
- 要"编辑期就报" → 进 `validator.go` 的 switch，**别**只加节点 `Validate()`（那只在 runtime 跑）。
- 加之前先 grep `validator.go` 里有没有同 kind 的 `validateXxx` 已经在校验同一字段——别重复。
- 写 spec 现状段时，"某节点有没有 Validate"要**两条都查**，不能只看节点结构体有没有 `Validate` 方法。

相关: [[2026-06-04-geometry-pin-value-pct-shape]]（同次 smoke 挖出的另一个 pin 形状坑）。
