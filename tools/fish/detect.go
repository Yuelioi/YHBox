package fish

import (
	"fmt"
	"image"
	"os"
	"sort"

	"yhbox/pkg/vision"
)

// Hit 是一次模板匹配的结果。
//   - ClientX, ClientY: 命中的中心位置（客户区像素坐标）
//   - Conf:             置信度 [0,1]，<0 表示未匹配
type Hit struct {
	ClientX, ClientY int
	Conf             float32
}

// 必需 slot 列表 — 缺一个 detector 启动失败
var requiredSlots = []string{
	"hook_icon", "hook_text", "start_fish", "fish_escape", "need_bait", "result",
	"warehouse_full", "shop_bag_tab", "shop_sell_all", "shop_confirm_sell",
	"go_fishing",
	"bait_product", "buy_max", "buy_button", "buy_confirm", "buy_success", "no_currency",
	"change_bait_confirm",
}

// Detector 自动从 assets/templates.toml 加载所有命名模板，按 slot 分组。
// 检测时按当前帧分辨率挑最合适的（vision.PickBest），避免缩放损失 conf。
type Detector struct {
	parallel  int
	templates map[string][]*vision.NamedTemplate
}

func NewDetector() (*Detector, error) {
	// 1. 读 embed templates.toml
	tomlBytes := Assets().TemplatesTOML
	tplsCfg, warns, err := vision.LoadTemplatesConfig(tomlBytes)
	if err != nil {
		return nil, fmt.Errorf("解析 embed templates.toml: %w", err)
	}
	for _, w := range warns {
		fmt.Fprintf(os.Stderr, "[fish] WARN embed templates: %s\n", w)
	}

	// 2. 构造 Detector：每 slot 加载所有候选 NamedTemplate
	d := &Detector{
		parallel:  vision.DefaultParallel(),
		templates: map[string][]*vision.NamedTemplate{},
	}
	for slot, specs := range tplsCfg["fish"] {
		for _, spec := range specs {
			pngBytes, pngErr := vision.LoadPNGForSpec(Assets().Templates, spec)
			if pngErr != nil {
				fmt.Fprintf(os.Stderr, "[fish] WARN 加载 PNG %s (source=%d): %v\n",
					spec.File, spec.Source, pngErr)
				continue
			}
			nt, err := vision.BuildNamedTemplate(spec, pngBytes)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[fish] WARN 构造 NamedTemplate %s: %v\n", spec.File, err)
				continue
			}
			d.templates[slot] = append(d.templates[slot], nt)
		}
	}

	// 3. 必需 slot 检查 — 早炸
	for _, slot := range requiredSlots {
		if len(d.templates[slot]) == 0 {
			return nil, fmt.Errorf("必需 slot %q 无候选模板（检查 assets/templates.toml）", slot)
		}
	}
	return d, nil
}

// slotConfTier 决定每个 slot 用哪档 conf 阈值。
// hook 类是核心信号（误检代价大），用 confHigh；其他用 confNormal。
var slotConfTier = map[string]float32{
	"hook_icon":           confHigh,
	"hook_text":           confHigh,
	"start_fish":          confNormal,
	"fish_escape":         confNormal,
	"need_bait":           confNormal,
	"result":              confNormal,
	"warehouse_full":      confNormal,
	"shop_bag_tab":        confNormal,
	"shop_sell_all":       confNormal,
	"shop_confirm_sell":   confNormal,
	"go_fishing":          confNormal,
	"bait_product":        confNormal,
	"buy_max":             confNormal,
	"buy_button":          confNormal,
	"buy_confirm":         confNormal,
	"buy_success":         confNormal,
	"no_currency":         confNormal,
	"change_bait_confirm": confNormal,
}

// detect 通用检测：从指定 slot 的候选挑最匹配的，跑 ROI 内 NCC。
// 阈值由 slotConfTier 决定。
func (d *Detector) detect(frame *image.RGBA, slot string) (Hit, bool) {
	threshold, ok := slotConfTier[slot]
	if !ok {
		threshold = confNormal
	}
	nt := vision.PickBest(d.templates[slot], frame.Bounds().Dx(), frame.Bounds().Dy())
	if nt == nil {
		return Hit{Conf: -1}, false
	}
	conf, x, y := vision.MatchTextROI(frame, nt, roiPaddingPx, d.parallel)
	hit := Hit{ClientX: x, ClientY: y, Conf: conf}
	return hit, conf >= threshold
}

// PickBarROI 返回当前分辨率对应的耐力条 ROI 像素坐标。
// 必须在 roiFishingBars 中精确匹配 Resolution；无匹配 → log + 返回 0 尺寸。
// 调用方拿到 0 尺寸应跳过本次检测（守卫 if w <= 0 || h <= 0 return）。
func (d *Detector) PickBarROI(clientW, clientH int) (x, y, w, h int) {
	for _, r := range BarROIs {
		if r.Resolution[0] == clientW && r.Resolution[1] == clientH {
			return r.BBox[0], r.BBox[1], r.BBox[2] - r.BBox[0], r.BBox[3] - r.BBox[1]
		}
	}
	fmt.Fprintf(os.Stderr, "[fish] FATAL 缺 %dx%d 耐力条 ROI，跳过本次检测\n", clientW, clientH)
	return 0, 0, 0, 0
}

// --- 各阶段检测方法（slot 名对应 templates.toml 里的 slot 字段）---

func (d *Detector) HookIconDim(frame *image.RGBA) (Hit, bool) {
	return d.detect(frame, "hook_icon")
}

func (d *Detector) HookText(frame *image.RGBA) (Hit, bool) {
	return d.detect(frame, "hook_text")
}

func (d *Detector) StartFish(frame *image.RGBA) (Hit, bool) {
	return d.detect(frame, "start_fish")
}

func (d *Detector) FishEscape(frame *image.RGBA) (Hit, bool) {
	return d.detect(frame, "fish_escape")
}

func (d *Detector) NeedBait(frame *image.RGBA) (Hit, bool) {
	return d.detect(frame, "need_bait")
}

func (d *Detector) Result(frame *image.RGBA) (Hit, bool) {
	return d.detect(frame, "result")
}

func (d *Detector) WarehouseFull(frame *image.RGBA) (Hit, bool) {
	return d.detect(frame, "warehouse_full")
}

func (d *Detector) ShopBagTab(frame *image.RGBA) (Hit, bool) {
	return d.detect(frame, "shop_bag_tab")
}

func (d *Detector) ShopSellAll(frame *image.RGBA) (Hit, bool) {
	return d.detect(frame, "shop_sell_all")
}

func (d *Detector) ShopConfirmSell(frame *image.RGBA) (Hit, bool) {
	return d.detect(frame, "shop_confirm_sell")
}

func (d *Detector) GoFishing(frame *image.RGBA) (Hit, bool) {
	return d.detect(frame, "go_fishing")
}

// BaitInShop 在 bait_product 的 6 个候选槽位里找万能鱼饵，返回**最后一个（reading-order）**命中槽。
//
// 之所以挑「最后」而不是「conf 最高」：商店同时上架两份万能鱼饵——前面那个用金币买、
// 后面那个用代币买。NCC 对两份基本同分（图标一致），用阅读顺序（先按 y 后按 x）的末位
// 来稳定挑到代币那份。
//
// 不走通用 detect()：通用版走 vision.PickBest 只挑 1 个候选模板（按分辨率），这里同分辨率
// 下有 6 条候选（同 PNG，不同 bbox = 不同搜索 ROI），需要全跑一遍。
func (d *Detector) BaitInShop(frame *image.RGBA) (Hit, bool) {
	frameH := frame.Bounds().Dy()

	type slotHit struct {
		Hit
		bbox [4]int
	}
	var matches []slotHit
	bestConfSeen := float32(-1) // 给失败日志用，跨阈值的最大 conf

	for _, nt := range d.templates["bait_product"] {
		// 仅用同分辨率的候选；跨分辨率走 ScaleTemplate 反而不稳
		if nt.BaseH != frameH {
			continue
		}
		conf, x, y := vision.MatchTextROI(frame, nt, roiPaddingPx, d.parallel)
		if conf > bestConfSeen {
			bestConfSeen = conf
		}
		if conf >= confNormal {
			matches = append(matches, slotHit{
				Hit:  Hit{ClientX: x, ClientY: y, Conf: conf},
				bbox: nt.BBox,
			})
		}
	}
	if len(matches) == 0 {
		return Hit{Conf: bestConfSeen}, false
	}

	// 按 reading order 排序：y 升序优先，y 相同时 x 升序。末位 = 视觉上最靠后/右下那个。
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].bbox[1] != matches[j].bbox[1] {
			return matches[i].bbox[1] < matches[j].bbox[1]
		}
		return matches[i].bbox[0] < matches[j].bbox[0]
	})
	return matches[len(matches)-1].Hit, true
}

func (d *Detector) BuyMax(frame *image.RGBA) (Hit, bool) {
	return d.detect(frame, "buy_max")
}

func (d *Detector) BuyButton(frame *image.RGBA) (Hit, bool) {
	return d.detect(frame, "buy_button")
}

func (d *Detector) BuyConfirm(frame *image.RGBA) (Hit, bool) {
	return d.detect(frame, "buy_confirm")
}

func (d *Detector) BuySuccess(frame *image.RGBA) (Hit, bool) {
	return d.detect(frame, "buy_success")
}

// NoCurrency 点击「购买」后若货币不足，屏幕中上会弹出「货币不足无法购买」横幅。
// 检出 → buyBaitFlow 中止 → 先 SHOPSELL 把鱼卖了换钱 → 再回 BUYBAIT。
func (d *Detector) NoCurrency(frame *image.RGBA) (Hit, bool) {
	return d.detect(frame, "no_currency")
}

func (d *Detector) ChangeBaitConfirm(frame *image.RGBA) (Hit, bool) {
	return d.detect(frame, "change_bait_confirm")
}

// FishingBarDirect 在已抠出的耐力条 ROI 上跑颜色分析。
func (d *Detector) FishingBarDirect(roiImg *image.RGBA) *BarResult {
	if roiImg == nil {
		return &BarResult{TargetX: -1, CursorX: -1}
	}
	result := analyzeBar(roiImg)
	if result == nil {
		return &BarResult{TargetX: -1, CursorX: -1}
	}
	return result
}
