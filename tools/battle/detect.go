package battle

import (
	"fmt"
	"image"
	"os"

	"yhbox" // 触发 yhbox.init() 注入 vision.PNGLoader
	"yhbox/pkg/vision"
)

const templatesConfigName = "templates.toml"

// Hit 一次模板匹配结果（跟 fish.Hit 同样的字段，独立于 fish 包以免循环依赖）。
type Hit struct {
	ClientX, ClientY int
	Conf             float32
}

// Detector 加载 battle.* slot 的模板，按当前帧分辨率挑最匹配的。
// 必需 slot 缺一就启动失败 —— 编译期决定，运行期早炸。
type Detector struct {
	parallel  int
	templates map[string][]*vision.NamedTemplate
}

var requiredSlots = []string{
	"team_tag", "team_enabled", "team_select", "team_switch_success",
}

func NewDetector() (*Detector, error) {
	tomlBytes, err := yhbox.AssetBytes(templatesConfigName)
	if err != nil {
		return nil, fmt.Errorf("读 embed %s: %w", templatesConfigName, err)
	}
	tplsCfg, warns, err := vision.LoadTemplatesConfig(tomlBytes)
	if err != nil {
		return nil, fmt.Errorf("解析 embed %s: %w", templatesConfigName, err)
	}
	for _, w := range warns {
		fmt.Fprintf(os.Stderr, "[battle] WARN embed templates: %s\n", w)
	}
	d := &Detector{
		parallel:  vision.DefaultParallel(),
		templates: map[string][]*vision.NamedTemplate{},
	}
	for slot, specs := range tplsCfg["battle"] {
		for _, spec := range specs {
			pngBytes, err := vision.LoadPNGForSpec(spec)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[battle] WARN 加载 PNG %s: %v\n", spec.File, err)
				continue
			}
			nt, err := vision.BuildNamedTemplate(spec, pngBytes)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[battle] WARN 构造 NamedTemplate %s: %v\n", spec.File, err)
				continue
			}
			d.templates[slot] = append(d.templates[slot], nt)
		}
	}
	for _, slot := range requiredSlots {
		if len(d.templates[slot]) == 0 {
			return nil, fmt.Errorf("必需 slot %q 无候选模板（检查 assets/templates.toml）", slot)
		}
	}
	return d, nil
}

// detect 通用：用 vision.PickBest 挑分辨率、跑 ROI NCC，conf 阈值 confNormal。
func (d *Detector) detect(frame *image.RGBA, slot string) (Hit, bool) {
	nt := vision.PickBest(d.templates[slot], frame.Bounds().Dx(), frame.Bounds().Dy())
	if nt == nil {
		return Hit{Conf: -1}, false
	}
	conf, x, y := vision.MatchTextROI(frame, nt, 30, d.parallel)
	hit := Hit{ClientX: x, ClientY: y, Conf: conf}
	return hit, conf >= confNormal
}

func (d *Detector) TeamTag(f *image.RGBA) (Hit, bool)           { return d.detect(f, "team_tag") }
func (d *Detector) TeamEnabled(f *image.RGBA) (Hit, bool)       { return d.detect(f, "team_enabled") }
func (d *Detector) TeamSelect(f *image.RGBA) (Hit, bool)        { return d.detect(f, "team_select") }
func (d *Detector) TeamSwitchSuccess(f *image.RGBA) (Hit, bool) { return d.detect(f, "team_switch_success") }
