# Phase 1:Vision 基础层重构 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: 用 superpowers:subagent-driven-development(推荐)或 superpowers:executing-plans 逐 task 实现。步骤用 `- [ ]` 复选框跟踪。

**Goal:** 把 `VisionService.Match/WaitMatch` 重构为「去 `mode`、加 `roi`、返回带 bbox 的 `MatchHit`」,并删除 Check/Wait/ClickTemplate 的 `MatchMode`(多模板恒 OR),为 Phase 2/3 打底。

**Architecture:** 一次原子签名重构 —— 命中框 bbox(`Matcher.Detect` 现成但被丢弃的 `regionOut`)上传到节点;单模板搜索保留 `variant.BBox` 快速定位(roi 零值→nil region),仅在显式 roi 时走比例搜索区。行为除 MatchMode 删除外不变(零回归)。

**Tech Stack:** Go;`internal/node`(接口/类型)、`internal/services/container/runtime`(visionAdapter)、`internal/nodes/detect`(三个模板节点);前端 node-i18n 生成物。

## Global Constraints

- **不要兼容**:项目未发布,直接改签名,不留 deprecated/shim/版本 if(spec §1)。
- **新增 pin 默认 = 旧行为(零回归)**;唯一有意行为变更 = MatchMode 删除(原 `all` 用法改多节点)。
- **TDD**:每个行为变更先写/改测试再实现。
- **构建只信工具链**:`go build ./...` 仅语法 check;产 exe 走 `task build`(见 `flightdeck/knowledge/build/build.md`)。本计划只需 `go build ./...` + `go test`。
- **验证基线**:见 `knowledge/build/build.md`; 当前 Go/前端测试应绿, 旧 runtime fixture / i18n residue 红记录已过期。
- **前端包管理只用 pnpm**;node-i18n 经 `cd frontend && pnpm gen:node-i18n` 生成,catalog drift 测试守护。

---

### Task 1: 定义 MatchHit 并切换 VisionService 签名(原子重构)

这是一次跨文件签名重构:编译无法分步绿,需一次改完所有调用点。中途用 `go build ./...` 让编译器枚举漏网点。

**Files:**
- Modify: `internal/node/template_match.go`(加 `MatchHit` 类型)
- Modify: `internal/node/interfaces.go:130-138`(`Match`/`WaitMatch` 签名)
- Modify: `internal/services/container/runtime/node_services.go:499-591`(`Match`/`WaitMatch`/`matchOnce`)
- Modify: `internal/nodes/detect/template_common.go:26-48`(`matchOnce`/`settleAfterMatch`)
- Modify: `internal/nodes/detect/check_template.go`、`wait_template.go`、`click_template.go`(去 MatchMode pin + 适配 MatchHit)
- Test: `internal/services/container/runtime/node_services_test.go`(mock/断言)、`internal/nodes/detect/*_test.go`(去 MatchMode 用例)

**Interfaces:**
- Produces:
  - `node.MatchHit{ Found bool; Point Point; BBox [4]float64; Conf float64 }`
  - `VisionService.Match(ctx, keys []string, threshold float64, roi Geometry) (MatchHit, error)`
  - `VisionService.WaitMatch(ctx, keys []string, threshold float64, roi Geometry, timeout time.Duration) (MatchHit, error)`
  - `detect.matchOnce(ctx, keys, threshold) (node.MatchHit, error)`(去 mode;roi 单帧用零值)
  - `detect.settleAfterMatch(ctx, keys, threshold, settle, hit node.MatchHit) (node.MatchHit, error)`

- [ ] **Step 1:写 MatchHit 类型**

`internal/node/template_match.go` 末尾追加:
```go
// MatchHit 单模板匹配结果 (Match/WaitMatch 返回, 值语义)。
// Found=false 时 Point/BBox 为零值, Conf = 轮询期间见过的最高匹配度 (诊断"差多少")。
type MatchHit struct {
	Found bool       `json:"found"`
	Point Point      `json:"point"` // 命中中心 (= BBox 中心)
	BBox  [4]float64 `json:"bbox"`  // [x, y(左上), w, h] 全帧归一化
	Conf  float64    `json:"conf"`  // CCOEFF_NORMED 匹配度
}
```

- [ ] **Step 2:改 VisionService 接口签名**

`internal/node/interfaces.go`,把:
```go
	Match(ctx context.Context, keys []string, threshold float64, mode string) (pt *Point, conf float64, err error)
	WaitMatch(ctx context.Context, keys []string, threshold float64, mode string, timeout time.Duration) (pt *Point, conf float64, err error)
```
改为:
```go
	// Match 单帧多模板 OR 匹配 (按 keys 序取首个命中)。roi 零值 = 用模板 variant.BBox 快速定位;
	// 非零 = 在该比例搜索区内找。返回 MatchHit (Found 区分命中/未命中)。
	Match(ctx context.Context, keys []string, threshold float64, roi Geometry) (MatchHit, error)

	// WaitMatch 阻塞轮询直到命中或 timeout。timeout<=0 视为单帧。语义同 Match。
	WaitMatch(ctx context.Context, keys []string, threshold float64, roi Geometry, timeout time.Duration) (MatchHit, error)
```
(同时删掉两段注释里关于 `mode = "any"|"all"` 的说明。)

- [ ] **Step 3:重写 visionAdapter 的 matchOnce / Match / WaitMatch**

`internal/services/container/runtime/node_services.go`,把 `Match`(499)、`WaitMatch`(506)、`matchOnce`(542) 三个方法整体替换为:
```go
func (a *visionAdapter) Match(ctx context.Context, keys []string, threshold float64, roi node.Geometry) (node.MatchHit, error) {
	if a.rt.Matcher == nil || len(keys) == 0 {
		return node.MatchHit{}, nil
	}
	return a.matchOnce(ctx, keys, threshold, roi)
}

func (a *visionAdapter) WaitMatch(ctx context.Context, keys []string, threshold float64, roi node.Geometry, timeout time.Duration) (node.MatchHit, error) {
	if a.rt.Matcher == nil || len(keys) == 0 {
		return node.MatchHit{}, nil
	}
	if timeout <= 0 {
		return a.matchOnce(ctx, keys, threshold, roi)
	}
	deadline := time.Now().Add(timeout)
	bestConf := 0.0
	for {
		if err := ctx.Err(); err != nil {
			return node.MatchHit{}, err
		}
		hit, err := a.matchOnce(ctx, keys, threshold, roi)
		if err != nil {
			return node.MatchHit{}, err
		}
		if hit.Conf > bestConf {
			bestConf = hit.Conf
		}
		if hit.Found {
			return hit, nil
		}
		if time.Now().After(deadline) {
			return node.MatchHit{Conf: bestConf}, nil
		}
		select {
		case <-ctx.Done():
			return node.MatchHit{}, ctx.Err()
		case <-time.After(visionWaitPollMs * time.Millisecond):
		}
	}
}

// matchOnce 单帧多模板 OR (按 keys 序取首个命中)。roi 零值 → nil region (variant.BBox 快速定位);
// 非零 → 解析成比例搜索区下发。命中带 bbox。
func (a *visionAdapter) matchOnce(ctx context.Context, keys []string, threshold float64, roi node.Geometry) (node.MatchHit, error) {
	var frame *image.RGBA
	if a.rt.Capture != nil {
		h, err := a.rt.ActiveHWND()
		if err != nil {
			return node.MatchHit{}, err
		}
		f, err := a.rt.CaptureFrameCached(h)
		if err != nil {
			return node.MatchHit{}, err
		}
		frame = f
	}
	var region []float64
	if frame != nil && (roi.Pct.W > 0 && roi.Pct.H > 0 || len(roi.Overrides) > 0) {
		fw, fh := frame.Bounds().Dx(), frame.Bounds().Dy()
		rx, ry, rw, rh, _ := ResolveGeometry(roi, fw, fh)
		region = []float64{float64(rx) / float64(fw), float64(ry) / float64(fh), float64(rw) / float64(fw), float64(rh) / float64(fh)}
	}
	tol := a.scaleTolerance()
	bestConf := 0.0
	for _, guid := range keys {
		found, pt, bbox, conf, err := a.rt.Matcher.Detect(ctx, frame, guid, threshold, region, tol)
		if err != nil {
			return node.MatchHit{}, err
		}
		if conf > bestConf {
			bestConf = conf
		}
		if found {
			return node.MatchHit{Found: true, Point: node.Point{X: pt.X, Y: pt.Y}, BBox: bbox, Conf: conf}, nil
		}
	}
	return node.MatchHit{Conf: bestConf}, nil
}
```
> 注:删掉了原 `mode == "all"` 分支(多模板恒 OR);原 `_` 丢弃的第三返回值 `bbox` 现接住带出。

- [ ] **Step 4:更新 template_common 的 matchOnce / settleAfterMatch**

`internal/nodes/detect/template_common.go`,把 `matchOnce`(28) 与 `settleAfterMatch`(37) 替换为:
```go
// matchOnce 单帧查模板此刻在不在 (去 mode, roi 用零值 = 默认快速定位)。
func matchOnce(ctx node.Ctx, keys []string, threshold float64) (node.MatchHit, error) {
	return ctx.Vision().WaitMatch(ctx.Context(), keys, threshold, node.Geometry{}, 0)
}

// settleAfterMatch: 命中后可选稳定延迟 (SettleMs) + 新鲜帧重定位一次。settle<=0 原样返回。
func settleAfterMatch(ctx node.Ctx, keys []string, threshold float64, settle time.Duration, hit node.MatchHit) (node.MatchHit, error) {
	if settle <= 0 {
		return hit, nil
	}
	if err := waitOrCancel(ctx, settle); err != nil {
		return node.MatchHit{}, err
	}
	if hit2, err := matchOnce(ctx, keys, threshold); err == nil && hit2.Found {
		return hit2, nil
	}
	return hit, nil
}
```

- [ ] **Step 5:更新 check_template.go(去 MatchMode + MatchHit)**

删 `ctInMatchMode` 常量;删 Spec 里 MatchMode 那段 InputSpec(`check_template.go:41-44`);Run 改为:
```go
func (CheckTemplate) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	keys := in.StringList(ctInTemplates)
	threshold := in.Float64(ctInThreshold)
	hit, err := ctx.Vision().Match(ctx.Context(), keys, threshold, node.Geometry{})
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "vision match %s: %v", strings.Join(keys, "+"), err)
	}
	if hit.Found {
		return ctx.Out(ctOutFound).Set(ctDataPoint, hit.Point).Set(ctDataConf, hit.Conf).Set(ctDataMatched, true).Fire(), nil
	}
	return ctx.Out(ctOutNotFound).Set(ctDataConf, hit.Conf).Set(ctDataMatched, false).Fire(), nil
}
```

- [ ] **Step 6:更新 wait_template.go(去 MatchMode + MatchHit)**

删 `wtInMatchMode` 常量与 Spec 里 MatchMode InputSpec(`wait_template.go:41-44`);Run 改为:
```go
func (WaitTemplate) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	keys := in.StringList(wtInTemplates)
	threshold := in.Float64(wtInThreshold)
	timeout := time.Duration(in.Int(wtInTimeoutMs)) * time.Millisecond
	settle := time.Duration(in.Int(wtInSettleMs)) * time.Millisecond
	hit, err := ctx.Vision().WaitMatch(ctx.Context(), keys, threshold, node.Geometry{}, timeout)
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "vision wait %s: %v", strings.Join(keys, "+"), err)
	}
	if hit.Found {
		hit, err = settleAfterMatch(ctx, keys, threshold, settle, hit)
		if err != nil {
			return nil, err
		}
		return ctx.Out(wtOutFound).Set(wtDataPoint, hit.Point).Set(wtDataConf, hit.Conf).Set(wtDataMatched, true).Fire(), nil
	}
	return ctx.Out(wtOutTimeout).Set(wtDataConf, hit.Conf).Set(wtDataMatched, false).Fire(), nil
}
```

- [ ] **Step 7:更新 click_template.go(去 MatchMode + MatchHit,行为不变)**

删 `clkInMatchMode` 常量与 Spec 里 MatchMode InputSpec(`click_template.go:47-50`)。Run 与 clickAt 改为按 MatchHit:`mode` 变量删除;`WaitMatch`/`settleAfterMatch`/`matchOnce` 调用去掉 mode 实参;`pt`→`hit`,判定用 `hit.Found`,点击点用 `hit.Point`。具体:
```go
func (ClickTemplate) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	keys := in.StringList(clkInTemplates)
	threshold := in.Float64(clkInThreshold)
	timeout := time.Duration(in.Int(clkInTimeoutMs)) * time.Millisecond
	settle := time.Duration(in.Int(clkInSettleMs)) * time.Millisecond
	maxAttempts := in.Int(clkInMaxAttempts)
	retryInterval := time.Duration(in.Int(clkInRetryIntervalMs)) * time.Millisecond
	btn := in.String(clkInButton)
	if btn == "" {
		btn = "left"
	}
	hit, err := ctx.Vision().WaitMatch(ctx.Context(), keys, threshold, node.Geometry{}, timeout)
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "ClickTemplate wait %s: %v", strings.Join(keys, "+"), err)
	}
	if !hit.Found {
		return ctx.Out(clkOutTimeout).Set(clkDataConf, hit.Conf).Set(clkDataMatched, false).Fire(), nil
	}
	hit, err = settleAfterMatch(ctx, keys, threshold, settle, hit)
	if err != nil {
		return nil, err
	}
	if err := clickAt(ctx, keys, hit.Point, btn); err != nil {
		return nil, err
	}
	if maxAttempts <= 1 {
		return ctx.Out(clkOutDone).Set(clkDataPoint, hit.Point).Set(clkDataConf, hit.Conf).Set(clkDataMatched, true).Fire(), nil
	}
	clicks := 1
	for {
		if err := waitOrCancel(ctx, retryInterval); err != nil {
			return nil, err
		}
		hit2, err := matchOnce(ctx, keys, threshold)
		if err != nil {
			return nil, node.Failf(node.CodeCaptureFailed, err, "ClickTemplate recheck %s: %v", strings.Join(keys, "+"), err)
		}
		if !hit2.Found {
			return ctx.Out(clkOutDone).Set(clkDataPoint, hit.Point).Set(clkDataConf, hit.Conf).Set(clkDataMatched, true).Fire(), nil
		}
		if clicks >= maxAttempts {
			return ctx.Out(clkOutTimeout).Set(clkDataConf, hit2.Conf).Set(clkDataMatched, true).Fire(), nil
		}
		hit = hit2
		if err := clickAt(ctx, keys, hit.Point, btn); err != nil {
			return nil, err
		}
		clicks++
	}
}
```
`clickAt` 签名把 `pt *node.Point` 改成 `pt node.Point`(调用点相应去 `*`/`&`):
```go
func clickAt(ctx node.Ctx, keys []string, pt node.Point, btn string) error {
	if err := ctx.Input().Click(pt.X, pt.Y, btn, 50); err != nil {
		return node.Failf(node.CodeCaptureFailed, err, "ClickTemplate click %s @ (%.3f,%.3f): %v", strings.Join(keys, "+"), pt.X, pt.Y, err)
	}
	return nil
}
```

- [ ] **Step 8:编译,让编译器列出漏网调用点 / mock**

Run: `go build ./...`
Expected: 报错列出剩余未适配处 —— 预期是测试里的 `VisionService` mock / `Match`·`WaitMatch` 调用、以及任何其它 `mode` 实参调用点。逐个改成新签名(mode 删、加 `node.Geometry{}` roi、返回值用 `MatchHit`)。`MatchAll` 不受影响(签名未动)。反复 `go build ./...` 到通过。

- [ ] **Step 9:写/改测试 —— bbox 上传 + 多模板 OR**

在 `node_services_test.go` 加(或改现有 `TestVisionAdapter_WaitMatch_*`):
```go
func TestVisionAdapter_Match_ReturnsBBox(t *testing.T) {
	// mockMatcher.Detect 返回 found=true, bbox=[0.1,0.2,0.3,0.4]
	// 期望 hit.Found==true && hit.BBox==[4]float64{0.1,0.2,0.3,0.4} && hit.Point≈bbox 中心
}

func TestVisionAdapter_Match_MultiTemplate_OR(t *testing.T) {
	// keys=[a,b]; mock: a miss, b hit → hit.Found==true (任一命中即 OR)
}
```
(mock 用现有 `mockMatcher`/`stubMatcher`,其 `Detect` 已返回 `[4]float64` bbox 位;给它能按 guid 返不同结果。)

- [ ] **Step 10:删/改引用 MatchMode 的旧节点测试**

`internal/nodes/detect/*_test.go` 里凡断言 `MatchMode`/`mode="all"` 行为的用例:删除 all-mode 专属用例;多模板用例改断言 OR(任一命中即 Found/Done)。

- [ ] **Step 11:跑测试**

Run: `go test ./internal/node/... ./internal/services/container/runtime/... ./internal/nodes/detect/...`
Expected: PASS。

- [ ] **Step 12:重生成 node-i18n + catalog drift**

Run: `cd frontend && pnpm gen:node-i18n`
然后 Run: `go test ./...`(catalog drift 测试守护 MatchMode pin 已从 3 节点消失)。
Expected: drift 测试 PASS。

- [ ] **Step 13:提交**

```bash
git add internal/node/ internal/services/container/runtime/ internal/nodes/detect/ frontend/
git commit -m "refactor(vision): Match/WaitMatch 返回 MatchHit(带 bbox)、去 mode/MatchMode、加 roi 参数

命中框 bbox 上传 (Matcher.Detect 现有 regionOut 不再丢弃);多模板恒 OR,
删 Check/Wait/ClickTemplate 的 MatchMode pin。为 Phase 2 锚点/order_by 打底。"
```

---

## 后续(本计划之外)

- **Phase 2**:ClickTemplate 全家桶 —— `node.ResolveScalar` helper + Anchor/OffsetX/OffsetY(用 MatchHit.BBox)+ OrderBy/Index(走 MatchAll)+ ROI pin + Keys/ClickCount(抽公共 click helper)。依赖本 Phase 的 MatchHit + roi。
- **Phase 3**:新节点群 —— WaitTemplateGone、Swipe(InputService.Drag→pkg.input.MouseDrag)、InputText(新 pkg.input.TypeText)、StopApp(taskkill)、Scroll 横向(pkg.input WM_MOUSEHWHEEL)、ClickAt 采用公共 click helper。含 InputService/pkg.input 后端新增。

各 Phase 落地后再出对应 plan。

## Self-Review(已过)

- **spec 覆盖**:本 Phase 对应 spec §6 底层(VisionService 去 mode/加 roi/带 bbox)+ §2/§3 的 MatchMode 删除。Phase 2/3 覆盖其余,已在「后续」列明。
- **占位符**:无 TBD;mock 细节走「编译器驱动 + 现有 mockMatcher」,非占位。
- **类型一致**:`MatchHit{Found,Point,BBox,Conf}` 在 adapter / 三节点 / template_common 一致;`matchOnce`/`settleAfterMatch`/`clickAt` 新签名贯穿一致。
