// wire_container.go 适配器：容器节点运行时跟 services / pkg 类型对接。
//
// container/runtime 包定义了 Matcher / Color / Runner 等接口；这里把具体实现
// (pkg/vision + pkg/capture) 绑到接口上.
// v3 Phase B 删除 InputDriver 适配 — input backend 由 runtime.setupRuntime 直接构造.
package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lxn/win"
	"golang.org/x/sync/singleflight"

	"yhbox/internal/hotkey"
	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
	"yhbox/internal/services/expr"
	"yhbox/internal/services/template"
	"yhbox/pkg/capture"
	"yhbox/pkg/vision"
)

// frameCacheTTL: 同 hwnd 100ms 内复用一帧，避免 IDLE 一轮 3 个 CheckTemplate 各
// 跑一次 capture.Frame (1080p 5-30ms × 3). 100ms 是 fishing v2 主循环 Sleep 下限,
// state machine 单 iter 内的多次 Detect 全部命中缓存; 跨 iter 之间 (Sleep ≥ 100ms)
// 会自然 miss → 重新抓帧.
const frameCacheTTL = 100 * time.Millisecond

// roiPaddingPx: variant.BBox → ROI 转换时的 px 冗余 (1:1 fish bot constants.roiPaddingPx).
// 角色/UI 轻微抖动 icon 偏出 BBox 时, padding 给 NCC search 余量.
const roiPaddingPx = 30.0

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// clamp01Bound 限制 w 不超过剩余空间（x + w <= 1）。
func clamp01Bound(w, x float64) float64 {
	if x+w > 1 {
		return 1 - x
	}
	if w < 0 {
		return 0
	}
	return w
}

type frameCacheEntry struct {
	frame *image.RGBA
	ts    time.Time
}

// captureFrameCached 同 hwnd 100ms TTL 内复用. 不同 hwnd 互不影响 (key 含 hwnd).
func captureFrameCached(mu *sync.Mutex, entries map[uintptr]frameCacheEntry, hwnd uintptr) (*image.RGBA, error) {
	mu.Lock()
	ent, ok := entries[hwnd]
	if ok && time.Since(ent.ts) < frameCacheTTL {
		mu.Unlock()
		return ent.frame, nil
	}
	mu.Unlock()

	frame, err := capture.Frame(win.HWND(hwnd))
	if err != nil {
		return nil, err
	}
	mu.Lock()
	entries[hwnd] = frameCacheEntry{frame: frame, ts: time.Now()}
	mu.Unlock()
	return frame, nil
}

// ---- TemplateMatcher: 接 pkg/vision + template store + capture ----
//
// 容器节点 WaitTemplate / CheckTemplate / ClickTemplate 用:
//   - PickBest(key, frameW, frameH) 选精确分辨率 variant (v2.2 多变体 schema)
//   - 抓当前游戏窗口一帧 (100ms TTL 缓存)
//   - 用 pkg/vision.Match 在 caller 指定 ROI 内做 CCOEFF_NORMED 匹配
//   - 返 found + 命中比例坐标 + 实际 region
//
// 命中坐标转换: vision.Match 返 ROI 内左上角像素 → 转客户区比例.
// 无匹配 variant 时 emit "container:warning" (30s 节流), Detect 返 false.

type templateMatcherAdapter struct {
	dataRoot  string
	storesMu  sync.Mutex
	stores    map[string]*template.Store // containerID → store
	loadCache sync.Map                   // "containerID:key:WxH" → *vision.Template
	loadSF    singleflight.Group

	fcMu      sync.Mutex
	fcEntries map[uintptr]frameCacheEntry // hwnd → 最近一帧

	// emit: 投递前端事件 (e.g. App.Emit). 测试可传 nil 跳过.
	emit func(eventName string, payload map[string]any)

	// throttle: key = containerID+":"+eventKey → lastEmitUnixMs.
	// 防止 missing variant 等高频警告刷屏 (30s 窗口内同 key 只 emit 一次).
	throttleMu sync.Mutex
	throttle   map[string]int64
}

func newTemplateMatcherAdapter(dataRoot string, emit func(string, map[string]any)) *templateMatcherAdapter {
	return &templateMatcherAdapter{
		dataRoot:  dataRoot,
		stores:    map[string]*template.Store{},
		fcEntries: map[uintptr]frameCacheEntry{},
		emit:      emit,
		throttle:  map[string]int64{},
	}
}

// emitThrottled 同 (containerID, eventKey) 在 window 内只 emit 一次.
// payload 自动注入 containerId, 复用同一 map.
func (m *templateMatcherAdapter) emitThrottled(containerID, eventKey string, window time.Duration, payload map[string]any) {
	if m.emit == nil {
		return
	}
	key := containerID + ":" + eventKey
	now := time.Now().UnixMilli()
	m.throttleMu.Lock()
	last := m.throttle[key]
	if now-last < window.Milliseconds() {
		m.throttleMu.Unlock()
		return
	}
	m.throttle[key] = now
	m.throttleMu.Unlock()
	if payload == nil {
		payload = map[string]any{}
	}
	payload["containerId"] = containerID
	m.emit("container:warning", payload)
}

// emitMissingVariantWarning Detect 在 PickBest 失败时调用. 30s 节流避免循环刷屏.
func (m *templateMatcherAdapter) emitMissingVariantWarning(containerID, key string, frameW, frameH int) {
	eventKey := fmt.Sprintf("missing-variant:%s:%dx%d", key, frameW, frameH)
	m.emitThrottled(containerID, eventKey, 30*time.Second, map[string]any{
		"code":      "MISSING_TEMPLATE_VARIANT",
		"severity":  "warning",
		"message":   fmt.Sprintf("template %q has no variant matching frame %dx%d", key, frameW, frameH),
		"key":       key,
		"frameSize": []int{frameW, frameH},
	})
}

func (m *templateMatcherAdapter) storeFor(containerID string) (*template.Store, error) {
	m.storesMu.Lock()
	defer m.storesMu.Unlock()
	if s, ok := m.stores[containerID]; ok {
		return s, nil
	}
	root := filepath.Join(m.dataRoot, "containers", containerID, "templates")
	s, err := template.NewStore(root)
	if err != nil {
		return nil, err
	}
	m.stores[containerID] = s
	return s, nil
}

// loadVariantTemplate 加载指定 (key, resolution) 变体的 PNG.
// 缓存 key = containerID:key:WxH (variant 已带 resolution 区分), singleflight 防雪崩.
// 不再做自动缩放 — Phase 2 删除老 doLoadTemplate, 由 PickBest 直接选精确匹配 variant.
func (m *templateMatcherAdapter) loadVariantTemplate(containerID, key string, resolution [2]int) (*vision.Template, error) {
	cacheKey := fmt.Sprintf("%s:%s:%dx%d", containerID, key, resolution[0], resolution[1])
	if cached, ok := m.loadCache.Load(cacheKey); ok {
		return cached.(*vision.Template), nil
	}
	v, err, _ := m.loadSF.Do(cacheKey, func() (interface{}, error) {
		store, err := m.storeFor(containerID)
		if err != nil {
			return nil, err
		}
		pngData, err := store.ReadVariantPng(key, resolution)
		if err != nil {
			return nil, fmt.Errorf("read variant png %q@%dx%d: %w", key, resolution[0], resolution[1], err)
		}
		tpl, err := vision.LoadPNG(bytes.NewReader(pngData))
		if err != nil {
			return nil, fmt.Errorf("decode variant png %q@%dx%d: %w", key, resolution[0], resolution[1], err)
		}
		m.loadCache.Store(cacheKey, tpl)
		return tpl, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*vision.Template), nil
}

func (m *templateMatcherAdapter) Detect(_ context.Context, containerID string, hwnd uintptr, key string, threshold float64, region []float64) (bool, expr.Point, [4]float64, error) {
	if hwnd == 0 {
		return false, expr.Point{}, [4]float64{}, nil
	}
	frame, err := captureFrameCached(&m.fcMu, m.fcEntries, hwnd)
	if err != nil {
		return false, expr.Point{}, [4]float64{}, fmt.Errorf("capture.Frame: %w", err)
	}
	frameW := frame.Bounds().Dx()
	frameH := frame.Bounds().Dy()

	store, err := m.storeFor(containerID)
	if err != nil {
		return false, expr.Point{}, [4]float64{}, err
	}
	variant, ok := store.PickBest(key, frameW, frameH)
	if !ok {
		m.emitMissingVariantWarning(containerID, key, frameW, frameH)
		return false, expr.Point{}, [4]float64{}, nil
	}
	tpl, err := m.loadVariantTemplate(containerID, key, variant.Resolution)
	if err != nil {
		return false, expr.Point{}, [4]float64{}, err
	}

	// ROI 优先级:
	//   1. caller 传 region (node config 显式 ROI) — 用 caller
	//   2. variant.BBox 非零 — pixel bbox / variant.Resolution 转 ratio + 30px padding
	//      (1:1 复刻 fish bot tools/fish/detect.go::MatchTextROI + constants.roiPaddingPx)
	//   3. 全屏 (1×1) — 1080p 大模板 3-5s/match, 兜底
	rx, ry, rw, rh := 0.0, 0.0, 1.0, 1.0
	switch {
	case len(region) == 4 && (region[2] > 0 || region[3] > 0):
		rx, ry, rw, rh = region[0], region[1], region[2], region[3]
	case variant.Resolution[0] > 0 && variant.Resolution[1] > 0 && variant.BBox[2] > variant.BBox[0] && variant.BBox[3] > variant.BBox[1]:
		vw, vh := float64(variant.Resolution[0]), float64(variant.Resolution[1])
		bx, by := float64(variant.BBox[0]), float64(variant.BBox[1])
		bw, bh := float64(variant.BBox[2]-variant.BBox[0]), float64(variant.BBox[3]-variant.BBox[1])
		padX := roiPaddingPx / vw
		padY := roiPaddingPx / vh
		rx = clamp01(bx/vw - padX)
		ry = clamp01(by/vh - padY)
		rw = clamp01Bound(bw/vw+padX*2, rx)
		rh = clamp01Bound(bh/vh+padY*2, ry)
	}

	roiGray, roiPxX, roiPxY, roiPxW, roiPxH := vision.CropROI(frame, rx, ry, rw, rh)
	if roiPxW <= 0 || roiPxH <= 0 {
		return false, expr.Point{}, [4]float64{}, nil
	}
	x, y, conf := vision.Match(roiGray, roiPxW, roiPxH, tpl, vision.DefaultParallel())
	if x < 0 || conf < float32(threshold) {
		return false, expr.Point{}, [4]float64{}, nil
	}
	cx := float64(roiPxX+x+tpl.W/2) / float64(frameW)
	cy := float64(roiPxY+y+tpl.H/2) / float64(frameH)
	out := [4]float64{rx, ry, rw, rh}
	return true, expr.Point{X: cx, Y: cy}, out, nil
}

// ---- ColorDetector: capture.Frame + HSV/RGB 像素扫描 ----
//
// DetectColor 节点用。抠 ROI（客户区比例 → 像素）→ 遍历像素 → 返命中数 +
// 命中中心客户区比例坐标。HSV 用 pkg/vision.RGBToHSV（H 0-360, S/V 0-255）。

type containerColorAdapter struct {
	fcMu      sync.Mutex
	fcEntries map[uintptr]frameCacheEntry // hwnd → 最近一帧
}

func (c *containerColorAdapter) Detect(_ context.Context, hwnd uintptr, region [4]float64, mode string, rng [6]int) (int, float64, float64, error) {
	if hwnd == 0 {
		return 0, 0, 0, fmt.Errorf("游戏窗口未就绪")
	}
	frame, err := captureFrameCached(&c.fcMu, c.fcEntries, hwnd)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("capture.Frame: %w", err)
	}

	frameW := frame.Bounds().Dx()
	frameH := frame.Bounds().Dy()
	rx, ry, rw, rh := region[0], region[1], region[2], region[3]
	if rw == 0 || rh == 0 {
		rx, ry, rw, rh = 0, 0, 1, 1
	}
	x0 := int(rx * float64(frameW))
	y0 := int(ry * float64(frameH))
	x1 := x0 + int(rw*float64(frameW))
	y1 := y0 + int(rh*float64(frameH))
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > frameW {
		x1 = frameW
	}
	if y1 > frameH {
		y1 = frameH
	}
	if x1 <= x0 || y1 <= y0 {
		return 0, 0, 0, nil
	}

	useHSV := mode != "rgb"
	count := 0
	sumX, sumY := 0, 0
	stride := frame.Stride

	for y := y0; y < y1; y++ {
		off := y * stride
		for x := x0; x < x1; x++ {
			i := off + x*4
			r, gC, b := frame.Pix[i], frame.Pix[i+1], frame.Pix[i+2]
			hit := false
			if useHSV {
				hh, ss, vv := vision.RGBToHSV(r, gC, b)
				hit = hh >= rng[0] && hh <= rng[1] && ss >= rng[2] && ss <= rng[3] && vv >= rng[4] && vv <= rng[5]
			} else {
				hit = int(r) >= rng[0] && int(r) <= rng[1] && int(gC) >= rng[2] && int(gC) <= rng[3] && int(b) >= rng[4] && int(b) <= rng[5]
			}
			if hit {
				count++
				sumX += x
				sumY += y
			}
		}
	}
	if count == 0 {
		return 0, 0, 0, nil
	}
	cxPx := float64(sumX) / float64(count)
	cyPx := float64(sumY) / float64(count)
	return count, cxPx / float64(frameW), cyPx / float64(frameH), nil
}

// ---- Container Runner: ExecutionQueue + Worker → container.Runner ----
//
// container.Service 暴露给前端的 Run / StopAll 走这条；source=manual。

type containerRunnerAdapter struct {
	queue  *execution.ExecutionQueue
	worker *execution.Worker
}

func (a *containerRunnerAdapter) RunOnce(id string) error {
	_, ok := a.queue.Enqueue(execution.QueuedRun{
		Targets: []execution.TargetRef{{Kind: "container", ID: id}},
		OnError: execution.OnErrorStop,
		Source:  execution.SourceManual,
	})
	if !ok {
		return fmt.Errorf("execution queue closed")
	}
	return nil
}

func (a *containerRunnerAdapter) StopAll() error {
	a.queue.CancelAll()
	a.worker.CancelCurrent()
	return nil
}

// ---- Container hotkey binder ----
//
// 把 container.Hotkey（用户在 ContainerEditor 底栏配的字符串）注册到 hotkey
// registry。按下后 enqueue 单 target manual run。CRUD 后 Refresh()：
// unregister 所有旧 entries → 重扫 container store → 注册当前列表。

type containerHotkeyBinder struct {
	store    *container.Store
	registry *hotkey.HotkeyRegistry
	queue    *execution.ExecutionQueue
	mu       sync.Mutex
	bound    map[string]string // containerID → registry key
}

func newContainerHotkeyBinder(store *container.Store, reg *hotkey.HotkeyRegistry, q *execution.ExecutionQueue) *containerHotkeyBinder {
	return &containerHotkeyBinder{store: store, registry: reg, queue: q, bound: map[string]string{}}
}

func (b *containerHotkeyBinder) Refresh() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, key := range b.bound {
		_ = b.registry.Unregister(key)
	}
	b.bound = map[string]string{}
	for _, c := range b.store.List() {
		hk := strings.TrimSpace(c.Hotkey)
		if hk == "" {
			continue
		}
		key := "container." + c.ID
		cid := c.ID
		err := b.registry.Register(key, hotkey.HotkeySourceContainer, "容器 "+c.Name, hk, "",
			func() {
				_, _ = b.queue.Enqueue(execution.QueuedRun{
					Targets: []execution.TargetRef{{Kind: "container", ID: cid}},
					OnError: execution.OnErrorStop,
					Source:  execution.SourceHotkey,
				})
			})
		if err == nil {
			b.bound[c.ID] = key
		}
	}
}
