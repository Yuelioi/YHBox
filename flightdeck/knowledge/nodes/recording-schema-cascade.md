---
kind: trap
summary: "录制流程一次撞 4 层 schema drift：二号铁律不只指代码也指约定，Spec 是唯一源，重命名必须全扫 callsite"
activation: symptom
read_when: "改节点 Spec 字段名 / pin 名前; 撞 INVALID_PIN / REQUIRED_FIELD_MISSING 撞得莫名其妙; 录制录像跑通但保存/运行炸; 任何 frontend/backend schema mismatch 类问题"
---
# ⚠ 录制流程 schema cascade — 一次 session 撞 4 层 drift
## 教训 (4 条)

1. **二号铁律 "不要兼容" 不只指代码逻辑, 也指 schema/约定**. 如果 Spec 改 PascalCase 但 FE writer / fixture / validator 还小写, 你**默默重新引入了兼容层** — 只不过是隐式的 (key 不匹配 → fallback "" / 缺失). 重命名约定 = 必须扫 ALL grep 命中, 一次性切干净.

2. **Spec 是单一事实来源, 任何 hardcoded 字段/pin 名 都是 schema drift 待发地点**. recording/transform.go 硬码 `"vk"/"durationMs"/"xRatio"/"clipID"` 跟 Spec 字段名 (`VK/DurationMs/XRatio/ClipID`) 错位 —— 这种代码就该走 Spec 反射或常量引用, 不能字面写.

3. **错误码只指向最浅一层, 不是根因**. INVALID_PIN 提示 "pin 名错", 真正修法是 transform.go 的 emit 路径. REQUIRED_FIELD_MISSING 提示 "字段缺", 真正修法是 transform.go 写错了 key. 不读 transform/writer 源码, 只调参数 = 永远修不完.

4. **多层 schema 时, 优先级是 反向(reader)→正向(writer) 全部**:
   - reader: validator / runtime / dispatch
   - writer: FE inspector / capture handler / transform / fixture / test data
   - 修一边不修另一边 = 当时表面修好下一 session 又炸

## 撞了什么 (按发现顺序)

### Layer 1: Win32WindowTarget 节点 config nested vs flat

- BE P2.1 把 `config.match.{title,class,processName,titleMatch}` + `config.runtime.{inputBackend,captureBackend}` 扁平化, runtime 跟 validator 都读顶层 PascalCase `Title/Class/ProcessName/TitleMatch/InputBackend/CaptureBackend`.
- **但 FE NodeInspector + F9 capture handler 还在 nested 老结构写**.
- 用户手 capture → 数据塞 `config.match.title="异环"` → runtime 读 `config.Title` 拿到 "Container 4" (create 时占位串) → `winutil.ResolveWindow(title="Container 4")` 找不到窗口 → 录制启动失败.
- 错误信息 `窗口未找到 (title="Container 4" class="" process="")` 暴露了空 class/process — 应该立刻意识到 Spec 字段全空了, 不是窗口不存在.

### Layer 2: F12 stop-hotkey race (见 separate report)

`2026-05-29-ll-hook-keydown-coalesce.md`. 跟 schema drift 无关, 但同一次 session 撞.

### Layer 3: Pin 名 Start 特例 `out` 全项目都是 `Done`

- `internal/nodes/control/start.go` const `startOutOut = "out"` (唯一小写特例). 所有其它节点 exec out=`Done`.
- recording/transform.go 跟 useRecording.ts:autoConnect 硬码 `.out`/`.in` (全小写) — 只跟 Start 偶然对得上, 其它 step 节点全错.
- 走 workaround (按 source kind 查 Spec) 还是改 Start 跟所有 callsite — 我先选 workaround, 用户拍 二号铁律 推回去重做. 改 Start `out` → `Done` 一次切, 影响 ~45 文件 (start.go const, service.go Create, runner.go seed, dispatch_v5.go 2 处 entry marker, transform.go, useRecording.ts, 25+ test 文件, fishing-v2 main container.json + 15 subgraph JSON 用 `sgin.out`).
- 也含虚拟 marker — `SubgraphInput` 节点 entry marker pin 名 runtime 硬解析, 必须跟 statement 一致改成 `Done`.

### Layer 4: transform.go config 字段名全小写

- transform.go 写: `"vk"/"durationMs"/"xRatio"/"yRatio"/"button"/"delta"/"clipID"`.
- 节点 Spec: `VK/DurationMs/XRatio/YRatio/Button/Delta/ClipID`.
- Sleep 特例: Spec 字段名 = **`Duration`** (Type=Duration, 不是 `DurationMs`); `in.Duration()` 把 float64 当 ms 解.
- 运行时 `validateRequired` 走 `in.Has("ClipID")` → `present["ClipID"]` (config map 实际 key) → false → REQUIRED_FIELD_MISSING.
- 顺手把 `validator.go:validatePlayClip` + `validator_deps.go` + `NodeInspector.vue` 两处读 `node.config.clipID` 也对齐 PascalCase.

### Layer 5: ClickAt 坐标输入迁移为 Point，录制转换器仍写 XRatio/YRatio

- `ClickAt` 的坐标输入已从两个 Number pin `XRatio/YRatio` 收敛为结构化 `Point` pin。
- `recording/transform.go` 仍生成旧键；事件流里的客户区坐标其实完整，转换测试也只断言旧键，因此录制产物在 Inspector 和 runtime 都回退到 `Point` 默认中心，看起来像“没有录到点击坐标”。
- 修复必须在生成器接缝断言 `config["Point"]` 的实际 X/Y，而不是只测底层 hook 是否采到了 B/C。任何 pin shape 迁移都要把 transform / generator 的回归测试一起切到新 shape。

## 怎么修

按照全部层级一次清:

```
# 找全 callsite (任何 schema rename 必跑)
grep -rn '<old-key>' --include='*.go' --include='*.vue' --include='*.ts' --include='*.json'

# 跑全测试 (Go + FE type-check 双轨)
go test ./... && cd frontend && npx vue-tsc --noEmit
```

旧 subgraph / clip 已经写错的 (用户 09:30 录的精准录制 subgraph `"clipID"` 小写, fix 后 BE 读 `"ClipID"` 大写) — 二号铁律 删了重录, **不写 migration**.

## 为什么会成这样

- 节点框架 PascalCase 化是渐进式 refactor (P2.1 等), 每次只动一部分 callsite.
- recording 子系统改动少, 没被 sweep 网兜到; transform.go 一直跟 Spec drift.
- Start 节点的 pin 名 `out` 是 v1 残留, 一直没动 (全项目 45 处引用, 改起来"麻烦"). 这种"小不一致"会一直滚雪球.

## 怎么防

- 任何 Spec 字段/pin 重命名 PR: 必须列 `grep` 命中数 + 全扫完才能 merge. 不允许"先改 reader 再说 writer 以后改".
- transform / generator 类代码引 const 不引字面值. `pcInClipID` / `kpInVK` 这些 const 是单一来源 — recording 包应当 import 这些 const, 不要硬码 `"ClipID"`.
  - 现状: recording 包跟 nodes/* 包跨层引常量是循环依赖隐患, 留 TODO. 短期靠 grep 兜.
- 节点 Spec 改 PascalCase 时同步加 validator/JSON-loader 层的 schema check, 命中老 key 抛 hard error 而不是 silently fallback "".
