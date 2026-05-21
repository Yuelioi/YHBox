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
// 容器节点 WaitTemplate / CheckTemplate / ClickTemplate 用：
//   - 从 template store 读 png bytes（key = 模板路径）
//   - 抓当前游戏窗口一帧
//   - 用 pkg/vision.Match 在 ROI / 全屏内做 CCOEFF_NORMED 匹配
//   - 返 found + 命中比例坐标 + 实际 region
//
// 命中坐标转换：vision.Match 返 ROI 内左上角像素 → 转客户区比例。

type templateMatcherAdapter struct {
	dataRoot  string
	storesMu  sync.Mutex
	stores    map[string]*template.Store // containerID → store
	loadCache sync.Map                   // (containerID+":"+key) → *vision.Template

	fcMu      sync.Mutex
	fcEntries map[uintptr]frameCacheEntry // hwnd → 最近一帧
}

func newTemplateMatcherAdapter(dataRoot string) *templateMatcherAdapter {
	return &templateMatcherAdapter{
		dataRoot:  dataRoot,
		stores:    map[string]*template.Store{},
		fcEntries: map[uintptr]frameCacheEntry{},
	}
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

func (m *templateMatcherAdapter) loadTemplate(containerID, key string) (*vision.Template, error) {
	cacheKey := containerID + ":" + key
	if cached, ok := m.loadCache.Load(cacheKey); ok {
		return cached.(*vision.Template), nil
	}
	store, err := m.storeFor(containerID)
	if err != nil {
		return nil, fmt.Errorf("template store for %q: %w", containerID, err)
	}
	data, err := store.ReadPng(key)
	if err != nil {
		return nil, fmt.Errorf("read template %q: %w", key, err)
	}
	tpl, err := vision.LoadPNG(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode template %q: %w", key, err)
	}
	m.loadCache.Store(cacheKey, tpl)
	return tpl, nil
}

func (m *templateMatcherAdapter) Detect(_ context.Context, containerID string, hwnd uintptr, key string, threshold float64, region []float64) (bool, expr.Point, [4]float64, error) {
	if hwnd == 0 {
		return false, expr.Point{}, [4]float64{}, nil
	}
	tpl, err := m.loadTemplate(containerID, key)
	if err != nil {
		return false, expr.Point{}, [4]float64{}, err
	}
	frame, err := captureFrameCached(&m.fcMu, m.fcEntries, hwnd)
	if err != nil {
		return false, expr.Point{}, [4]float64{}, fmt.Errorf("capture.Frame: %w", err)
	}

	// region 优先级: caller 传的 > meta.Regions (multi-slot, 选末位 reading order) >
	// meta.Region (single, 扩 30px padding) > 全屏 (兜底).
	// 全屏匹配开销巨大 (1080p 400×40 模板 3-5s/match), 用 ROI 后 ~10ms.
	store, _ := m.storeFor(containerID)
	if len(region) != 4 || (region[2] == 0 && region[3] == 0) {
		// 没 caller region — 查 meta 看是否 multi-slot
		if store != nil {
			if meta, ok := store.Get(key); ok && len(meta.Regions) > 0 {
				return m.detectMultiRegion(frame, tpl, meta.Regions, threshold)
			}
		}
	}

	rx, ry, rw, rh := 0.0, 0.0, 1.0, 1.0
	if len(region) == 4 && (region[2] > 0 || region[3] > 0) {
		rx, ry, rw, rh = region[0], region[1], region[2], region[3]
	} else if store != nil {
		if meta, ok := store.Get(key); ok && (meta.Region[2] > 0 || meta.Region[3] > 0) {
			mx, my, mw, mh := float64(meta.Region[0]), float64(meta.Region[1]), float64(meta.Region[2]), float64(meta.Region[3])
			// 固定 30 px padding (跟 v1 fish bot tools/fish/constants.roiPaddingPx 一致).
			// 老版本用 mw*0.3 比例 padding, 小 template (e.g. hook_icon 45×54) search area 偏窄
			// (72×81 vs v1 105×114), 角色/UI 轻微抖动 icon 偏出 → NCC miss.
			frameW := float64(frame.Bounds().Dx())
			frameH := float64(frame.Bounds().Dy())
			padX := 30.0 / frameW
			padY := 30.0 / frameH
			rx = clamp01(mx - padX)
			ry = clamp01(my - padY)
			rw = clamp01Bound(mw+padX*2, rx)
			rh = clamp01Bound(mh+padY*2, ry)
		}
	}

	roiGray, roiPxX, roiPxY, roiPxW, roiPxH := vision.CropROI(frame, rx, ry, rw, rh)
	if roiPxW <= 0 || roiPxH <= 0 {
		return false, expr.Point{}, [4]float64{}, nil
	}
	x, y, conf := vision.Match(roiGray, roiPxW, roiPxH, tpl, vision.DefaultParallel())
	if x < 0 || conf < float32(threshold) {
		return false, expr.Point{}, [4]float64{}, nil
	}
	frameW := frame.Bounds().Dx()
	frameH := frame.Bounds().Dy()
	cx := float64(roiPxX+x+tpl.W/2) / float64(frameW)
	cy := float64(roiPxY+y+tpl.H/2) / float64(frameH)
	out := [4]float64{rx, ry, rw, rh}
	return true, expr.Point{X: cx, Y: cy}, out, nil
}

// detectMultiRegion 在多个 ROI 各自跑 NCC, 收 conf >= threshold 的 hit, 按 reading order
// (y 优先 x 次) 选末位返回. 1:1 复刻 v1 fish bot Detector.BaitInShop 思路, 处理多槽位 UI
// (商店货架 grid 等). 每个 region 加 30px padding 跟 single ROI / v1 roiPaddingPx 一致 —
// 不加 padding 时 region==template size 导致 NCC search space=1×1, 模板偏 1 像素就 miss.
func (m *templateMatcherAdapter) detectMultiRegion(frame *image.RGBA, tpl *vision.Template, regions [][4]float32, threshold float64) (bool, expr.Point, [4]float64, error) {
	type hit struct {
		cx, cy float64
		rx, ry float64
		rw, rh float64
	}
	var hits []hit
	frameW := frame.Bounds().Dx()
	frameH := frame.Bounds().Dy()
	padX := 30.0 / float64(frameW)
	padY := 30.0 / float64(frameH)
	for _, r := range regions {
		rx, ry := float64(r[0]), float64(r[1])
		rw, rh := float64(r[2]), float64(r[3])
		rx = clamp01(rx - padX)
		ry = clamp01(ry - padY)
		rw = clamp01Bound(rw+padX*2, rx)
		rh = clamp01Bound(rh+padY*2, ry)
		roiGray, roiPxX, roiPxY, roiPxW, roiPxH := vision.CropROI(frame, rx, ry, rw, rh)
		if roiPxW <= 0 || roiPxH <= 0 {
			continue
		}
		x, y, conf := vision.Match(roiGray, roiPxW, roiPxH, tpl, vision.DefaultParallel())
		if x < 0 || conf < float32(threshold) {
			continue
		}
		hits = append(hits, hit{
			cx: float64(roiPxX+x+tpl.W/2) / float64(frameW),
			cy: float64(roiPxY+y+tpl.H/2) / float64(frameH),
			rx: rx, ry: ry, rw: rw, rh: rh,
		})
	}
	if len(hits) == 0 {
		return false, expr.Point{}, [4]float64{}, nil
	}
	// reading order 末位: y 升序, y 相同时 x 升序, 取最后一个 (右下).
	// bait_product 商店金币款在前 (左上) 代币款在后 (右下), 选末位避点金币款.
	last := hits[0]
	for _, h := range hits[1:] {
		if h.ry > last.ry || (h.ry == last.ry && h.rx > last.rx) {
			last = h
		}
	}
	return true, expr.Point{X: last.cx, Y: last.cy}, [4]float64{last.rx, last.ry, last.rw, last.rh}, nil
}

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
