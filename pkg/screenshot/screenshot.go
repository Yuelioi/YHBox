// Package screenshot 给 bot 提供异步落盘带标注的 PNG 抓帧能力，给"事后调试"用。
//
// 痛点：bot 跑出来识别有偏差时，事后无法重现哪一帧是怎么 detect 的。
// 解决：bot 关键路径 Submit 一张 (image + 标注框 + 文件名) 进队列，
// 后台 goroutine 异步画框 + 写 PNG 到 bin/captures/<bot>/<date>/。
//
// 设计要点：
//   - 队列 buffered，满则 drop（不阻塞主逻辑——bot 性能优先于完整 dump）
//   - 写盘走独立 goroutine，写入失败 silently drop（不让磁盘满影响 bot）
//   - 同帧多次 Submit 文件名带毫秒级时间戳避免覆盖
//   - 配置走 settings 的 Capture.Debug toggle，关掉时 Submit 是 no-op
//
// 调用例（fish detect 成功后）：
//
//	screenshot.Submit("fish", frame, []screenshot.Box{
//	    {Rect: hitRect, Label: fmt.Sprintf("hook_icon %.2f", conf), Color: color.RGBA{0, 255, 0, 255}},
//	}, fmt.Sprintf("hook_icon_conf_%.2f", conf))

package screenshot

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// Box 是一个标注框：rect + label + 颜色。Submit 时 caller 提供。
type Box struct {
	Rect  image.Rectangle
	Label string // 框上方文字（暂不实现 label 渲染，只画框）
	Color color.RGBA
}

type job struct {
	bot    string
	img    *image.RGBA
	boxes  []Box
	suffix string
	ts     time.Time
}

// Writer 是异步 PNG 落盘器。NewWriter 起一个 goroutine 消费队列。
// 多个 bot 可以共享一个 Writer 实例（队列内含 bot 字段做子目录分隔）。
type Writer struct {
	baseDir   string
	queue     chan job
	wg        sync.WaitGroup
	enabled   atomic.Bool
	closeOnce sync.Once
}

// NewWriter 起 1 个消费 goroutine，队列容量 cap (建议 16-32 之间)。
// baseDir 是 captures/ 根目录；实际写盘走 baseDir/<bot>/<date>/<ts>_<suffix>.png。
func NewWriter(baseDir string, queueCap int) *Writer {
	w := &Writer{
		baseDir: baseDir,
		queue:   make(chan job, queueCap),
	}
	w.enabled.Store(true)
	w.wg.Add(1)
	go w.run()
	return w
}

// SetEnabled 在 settings 改了 Capture.Debug toggle 时调用。disabled 时 Submit 是 no-op。
func (w *Writer) SetEnabled(v bool) { w.enabled.Store(v) }
func (w *Writer) Enabled() bool     { return w.enabled.Load() }

// Submit 入队一张带标注的帧。如果队列满或 Writer 未 enabled，return false（drop）。
// img 必须是 caller 独立分配的副本（Writer 异步用，原 caller 不能再 mutate）；
// 推荐 caller 在 Submit 之前 image.NewRGBA + copy。
func (w *Writer) Submit(bot string, img *image.RGBA, boxes []Box, suffix string) bool {
	if !w.enabled.Load() || w == nil {
		return false
	}
	select {
	case w.queue <- job{bot: bot, img: img, boxes: boxes, suffix: suffix, ts: time.Now()}:
		return true
	default:
		return false // 队列满，drop
	}
}

// Close 关闭队列等 goroutine 退完。多次调用安全（test 里 defer + 显式 Close 不 panic）。
func (w *Writer) Close() {
	w.closeOnce.Do(func() {
		close(w.queue)
		w.wg.Wait()
	})
}

// 包级 default Writer：main.go 启动时 InitDefault 一次，各 bot 用 Submit 不用传引用。
// 未调 InitDefault 时所有 Submit 是 no-op（关掉 dump 时一样）。
var defaultWriter *Writer

// InitDefault 创建包级 Writer 实例。main.go 启动期调一次。enabled=false 时
// 创建但 Submit no-op，用户运行时通过 SetDefaultEnabled 切。
func InitDefault(baseDir string, queueCap int, enabled bool) {
	defaultWriter = NewWriter(baseDir, queueCap)
	defaultWriter.SetEnabled(enabled)
}

// SetDefaultEnabled 给设置 view 切 dump toggle 用。InitDefault 前调是 no-op。
func SetDefaultEnabled(v bool) {
	if defaultWriter != nil {
		defaultWriter.SetEnabled(v)
	}
}

// Submit 走 default Writer。InitDefault 前 / Writer 关闭 / 未 enabled 都是 no-op。
// 注意 img 必须是 caller 独立分配的副本：Writer 异步用，原 caller 不能再 mutate。
func Submit(bot string, img *image.RGBA, boxes []Box, suffix string) bool {
	if defaultWriter == nil {
		return false
	}
	return defaultWriter.Submit(bot, img, boxes, suffix)
}

// CloseDefault 给 app shutdown 用。
func CloseDefault() {
	if defaultWriter != nil {
		defaultWriter.Close()
		defaultWriter = nil
	}
}

func (w *Writer) run() {
	defer w.wg.Done()
	for j := range w.queue {
		w.writeOne(j)
	}
}

func (w *Writer) writeOne(j job) {
	// 在原图上画框（修改 caller 已经放弃的 img 内存，OK）
	for _, b := range j.boxes {
		drawRect(j.img, b.Rect, b.Color)
	}
	// 路径：baseDir/<bot>/<YYYY-MM-DD>/<HH-MM-SS.mmm>_<suffix>.png
	dateDir := j.ts.Format("2006-01-02")
	dir := filepath.Join(w.baseDir, j.bot, dateDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return // 静默 drop，磁盘问题不让影响 bot
	}
	ms := j.ts.Nanosecond() / 1e6
	name := fmt.Sprintf("%s.%03d_%s.png", j.ts.Format("15-04-05"), ms, j.suffix)
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return
	}
	defer f.Close()
	_ = png.Encode(f, j.img)
}

// drawRect 在 img 上画一个 1px 边框矩形（不填充）。简单 inline 实现，避免引入 gocv 之类大依赖。
func drawRect(img *image.RGBA, r image.Rectangle, c color.RGBA) {
	r = r.Intersect(img.Bounds())
	if r.Empty() {
		return
	}
	src := &image.Uniform{C: c}
	// 4 条边各 1 px
	top := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+1)
	bottom := image.Rect(r.Min.X, r.Max.Y-1, r.Max.X, r.Max.Y)
	left := image.Rect(r.Min.X, r.Min.Y, r.Min.X+1, r.Max.Y)
	right := image.Rect(r.Max.X-1, r.Min.Y, r.Max.X, r.Max.Y)
	for _, e := range []image.Rectangle{top, bottom, left, right} {
		draw.Draw(img, e, src, image.Point{}, draw.Src)
	}
}

