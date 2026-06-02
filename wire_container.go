// wire_container.go 适配器：容器节点运行时跟 services / pkg 类型对接。
//
// container/runtime 包定义了 Matcher / Runner 等接口；这里把具体实现
// (pkg/vision) 绑到接口上.
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

	"golang.org/x/sync/singleflight"

	"yotta/internal/hotkey"
	"yotta/internal/services/container"
	"yotta/internal/services/execution"
	"yotta/internal/services/expr"
	"yotta/internal/services/template"
	"yotta/pkg/vision"
)

// roiPaddingPx: variant.BBox → ROI 转换时的 px 冗余.
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

// ---- TemplateMatcher: 接 pkg/vision + template store ----
//
// 容器节点 WaitTemplate / CheckTemplate / ClickTemplate 用:
//   - PickBest(key, frameW, frameH) 选精确分辨率 variant (v2.2 多变体 schema)
//   - 由 caller (matchOnce / detectFired) 传入一帧 (帧缓存在 RuntimeContext, 100ms TTL)
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

	// emit: 投递前端事件 (e.g. App.Emit). 测试可传 nil 跳过.
	emit func(eventName string, payload map[string]any)

	// throttle: key = containerID+":"+eventKey → lastEmitUnixMs.
	// 防止 missing variant 等高频警告刷屏 (30s 窗口内同 key 只 emit 一次).
	throttleMu sync.Mutex
	throttle   map[string]int64

	// scaleToleranceFor: 按 containerID 读模板缩放容差 (WindowTarget.ScaleTolerance).
	// nil → 用 defaultScaleTolerance (测试场景). main.go 注入 container.Service.ScaleToleranceFor.
	scaleToleranceFor func(containerID string) float64
}

func newTemplateMatcherAdapter(dataRoot string, emit func(string, map[string]any), scaleToleranceFor func(string) float64) *templateMatcherAdapter {
	return &templateMatcherAdapter{
		dataRoot:          dataRoot,
		stores:            map[string]*template.Store{},
		emit:              emit,
		throttle:          map[string]int64{},
		scaleToleranceFor: scaleToleranceFor,
	}
}

// defaultScaleTolerance: scaleToleranceFor 未注入时 (测试) 的兜底, 与 container.DefaultScaleTolerance 一致.
const defaultScaleTolerance = 2.0

// longEdgeScale 把 variant 模板缩到 frame 分辨率的比例 (长边比). variant 或 frame 为零尺寸时返 0.
func longEdgeScale(frameW, frameH, varW, varH int) float64 {
	fl := frameW
	if frameH > fl {
		fl = frameH
	}
	vl := varW
	if varH > vl {
		vl = varH
	}
	if vl <= 0 || fl <= 0 {
		return 0
	}
	return float64(fl) / float64(vl)
}

// withinScaleTolerance 判断缩放比是否落在 [1/k, k]. k<1 归一到 1 (仅精确). scale<=0 一律 false.
func withinScaleTolerance(scale, k float64) bool {
	if scale <= 0 {
		return false
	}
	if k < 1.0 {
		k = 1.0
	}
	return scale >= 1.0/k && scale <= k
}

// scaleTolerance 取容器的缩放容差; 未注入 (测试) 回落默认 2.0, 注入值 <1 (异常) 回落 1.0 仅精确.
func (m *templateMatcherAdapter) scaleTolerance(containerID string) float64 {
	if m.scaleToleranceFor == nil {
		return defaultScaleTolerance
	}
	if k := m.scaleToleranceFor(containerID); k >= 1.0 {
		return k
	}
	return 1.0
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

// emitScaleTooFarWarning Detect 找到最近 variant 但缩放比超出容器 ScaleTolerance 时调用.
// 跟「无 variant」分开, 方便真机 smoke 时从日志看清是「没录这分辨率」还是「容差挡了」.
func (m *templateMatcherAdapter) emitScaleTooFarWarning(containerID, key string, frameW, frameH int, variantRes [2]int, scale, tolerance float64) {
	eventKey := fmt.Sprintf("scale-too-far:%s:%dx%d", key, frameW, frameH)
	m.emitThrottled(containerID, eventKey, 30*time.Second, map[string]any{
		"code":        "VARIANT_OUT_OF_SCALE_TOLERANCE",
		"severity":    "warning",
		"message":     fmt.Sprintf("template %q nearest variant %dx%d 对帧 %dx%d 缩放比 %.2f 超出容差 [1/%.1f, %.1f], 判 miss", key, variantRes[0], variantRes[1], frameW, frameH, scale, tolerance, tolerance),
		"key":         key,
		"frameSize":   []int{frameW, frameH},
		"variantSize": []int{variantRes[0], variantRes[1]},
		"scale":       scale,
		"tolerance":   tolerance,
	})
}

// Invalidate 丢弃某容器缓存的 store + 已解码模板, 让下次 Detect 重读磁盘.
// 用户在同一 session 内新存/改/删模板后, templateSvc 经 SetOnChange 调这里 ——
// 否则 matcher 一直拿首次 preload 的旧索引, 新模板匹配不上 (报 MISSING_TEMPLATE_VARIANT).
func (m *templateMatcherAdapter) Invalidate(containerID string) {
	m.storesMu.Lock()
	delete(m.stores, containerID)
	m.storesMu.Unlock()
	prefix := containerID + ":"
	m.loadCache.Range(func(k, _ any) bool {
		if ks, ok := k.(string); ok && strings.HasPrefix(ks, prefix) {
			m.loadCache.Delete(ks)
		}
		return true
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

func (m *templateMatcherAdapter) Detect(_ context.Context, containerID string, frame *image.RGBA, key string, threshold float64, region []float64) (bool, expr.Point, [4]float64, float64, error) {
	if frame == nil {
		return false, expr.Point{}, [4]float64{}, 0, nil
	}
	frameW := frame.Bounds().Dx()
	frameH := frame.Bounds().Dy()

	store, err := m.storeFor(containerID)
	if err != nil {
		return false, expr.Point{}, [4]float64{}, 0, err
	}
	variant, ok := store.PickBest(key, frameW, frameH)
	if !ok {
		m.emitMissingVariantWarning(containerID, key, frameW, frameH)
		return false, expr.Point{}, [4]float64{}, 0, nil
	}
	// 跨分辨率缩放兜底: PickBest 可能返回非精确分辨率的最近档. 按长边比算缩放比,
	// 超出容器 ScaleTolerance ([1/k, k]) 则判太远不缩放, 仍 miss (emit 警告).
	scale := longEdgeScale(frameW, frameH, variant.Resolution[0], variant.Resolution[1])
	k := m.scaleTolerance(containerID)
	if !withinScaleTolerance(scale, k) {
		m.emitScaleTooFarWarning(containerID, key, frameW, frameH, variant.Resolution, scale, k)
		return false, expr.Point{}, [4]float64{}, 0, nil
	}
	tpl, err := m.loadVariantTemplate(containerID, key, variant.Resolution)
	if err != nil {
		return false, expr.Point{}, [4]float64{}, 0, err
	}
	// 精确命中 scale==1.0 不缩 (零开销); 否则按长边比缩到 frame 分辨率再 NCC.
	if scale != 1.0 {
		scaled := vision.ScaleTemplate(tpl, float32(scale))
		if scaled == nil {
			return false, expr.Point{}, [4]float64{}, 0, nil // 缩后 <8px, 当 miss.
		}
		tpl = scaled
	}

	// 多槽分支 (e.g. 商店 bait_product 3×2 grid): variant.Regions 非空 → 各 region 各跑 NCC,
	// 收 conf>=threshold 的 hit, 按 reading order (y 升, 同 y 按 x 升) 取末位 (右下).
	// 取右下而非左上: 多槽 UI 常把已选中/默认项放左上, 末位拿未选中项更稳.
	// caller 显式 region 优先于多槽 (常用于强制单点测试).
	// 传 variant.Resolution (原始录制分辨率) 是有意的: regions 的 pixel bbox 在原分辨率坐标系,
	// 转 ratio 后分辨率无关; tpl 已按长边比缩到 frame 尺度, 与裁自真实 frame 的 ROI 同尺度.
	if len(variant.Regions) > 0 && !(len(region) == 4 && (region[2] > 0 || region[3] > 0)) {
		return m.detectMultiRegion(frame, tpl, variant.Regions, variant.Resolution, threshold)
	}

	// ROI 优先级:
	//   1. caller 传 region (node config 显式 ROI) — 用 caller
	//   2. variant.BBox 非零 — pixel bbox / variant.Resolution 转 ratio + 30px padding
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
		return false, expr.Point{}, [4]float64{}, 0, nil
	}
	x, y, conf := vision.Match(roiGray, roiPxW, roiPxH, tpl, vision.DefaultParallel())
	if x < 0 || conf < float32(threshold) {
		return false, expr.Point{}, [4]float64{}, float64(conf), nil // 报真实匹配度供超时/miss 诊断
	}
	cx := float64(roiPxX+x+tpl.W/2) / float64(frameW)
	cy := float64(roiPxY+y+tpl.H/2) / float64(frameH)
	out := [4]float64{rx, ry, rw, rh}
	return true, expr.Point{X: cx, Y: cy}, out, float64(conf), nil
}

// detectMultiRegion 多槽 ROI 各跑 NCC, 收 conf>=threshold hit, 按 reading-order 末位返回.
// 处理多槽位 UI (e.g. 货架 3×2 grid).
// regions 每条 pixel bbox 在 variantRes 坐标系下, 自动加 30px padding (跟 single-BBox 路径一致).
func (m *templateMatcherAdapter) detectMultiRegion(frame *image.RGBA, tpl *vision.Template, regions [][4]int, variantRes [2]int, threshold float64) (bool, expr.Point, [4]float64, float64, error) {
	if variantRes[0] <= 0 || variantRes[1] <= 0 {
		return false, expr.Point{}, [4]float64{}, 0, nil
	}
	type hit struct {
		cx, cy         float64
		rx, ry, rw, rh float64
	}
	var hits []hit
	bestConf := float32(-1) // 各槽见过的最高匹配度 (供 miss 诊断)
	frameW := frame.Bounds().Dx()
	frameH := frame.Bounds().Dy()
	vw, vh := float64(variantRes[0]), float64(variantRes[1])
	padX := roiPaddingPx / vw
	padY := roiPaddingPx / vh
	for _, r := range regions {
		if r[2] <= r[0] || r[3] <= r[1] {
			continue
		}
		bx, by := float64(r[0]), float64(r[1])
		bw, bh := float64(r[2]-r[0]), float64(r[3]-r[1])
		rx := clamp01(bx/vw - padX)
		ry := clamp01(by/vh - padY)
		rw := clamp01Bound(bw/vw+padX*2, rx)
		rh := clamp01Bound(bh/vh+padY*2, ry)
		roiGray, roiPxX, roiPxY, roiPxW, roiPxH := vision.CropROI(frame, rx, ry, rw, rh)
		if roiPxW <= 0 || roiPxH <= 0 {
			continue
		}
		x, y, conf := vision.Match(roiGray, roiPxW, roiPxH, tpl, vision.DefaultParallel())
		if conf > bestConf {
			bestConf = conf
		}
		if x < 0 || conf < float32(threshold) {
			continue
		}
		hits = append(hits, hit{
			cx: float64(roiPxX+x+tpl.W/2) / float64(frameW),
			cy: float64(roiPxY+y+tpl.H/2) / float64(frameH),
			rx: rx, ry: ry, rw: rw, rh: rh,
		})
	}
	cf := float64(bestConf)
	if cf < 0 {
		cf = 0
	}
	if len(hits) == 0 {
		return false, expr.Point{}, [4]float64{}, cf, nil
	}
	// reading-order 末位: ry 升序, ry 同时 rx 升序, 取最后一个 (右下).
	last := hits[0]
	for _, h := range hits[1:] {
		if h.ry > last.ry || (h.ry == last.ry && h.rx > last.rx) {
			last = h
		}
	}
	return true, expr.Point{X: last.cx, Y: last.cy}, [4]float64{last.rx, last.ry, last.rw, last.rh}, cf, nil
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
		err := b.registry.Register(key, hotkey.HotkeySourceContainer,
			"hotkeys.label.container", map[string]string{"name": c.Name},
			hk, "",
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
