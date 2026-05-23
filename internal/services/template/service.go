package template

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// Service per-container template RPC 服务. 内部 map[containerID]*Store.
// 启动期 main.go 注入 dataRoot, 每个 container 第一次访问时 lazy create Store.
type Service struct {
	dataRoot string
	capture  CaptureAdapter
	mu       sync.Mutex
	stores   map[string]*Store // containerID → store
}

type CaptureAdapter interface {
	Capture() ([]byte, error) // 返 PNG bytes
}

func NewService(dataRoot string, capture CaptureAdapter) *Service {
	return &Service{
		dataRoot: dataRoot,
		capture:  capture,
		stores:   map[string]*Store{},
	}
}

func (s *Service) storeFor(containerID string) (*Store, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st, ok := s.stores[containerID]; ok {
		return st, nil
	}
	root := filepath.Join(s.dataRoot, "containers", containerID, "templates")
	st, err := NewStore(root)
	if err != nil {
		return nil, err
	}
	s.stores[containerID] = st
	return st, nil
}

// Save 用 RPC 入口. containerID 是必填. dataURL 必须 data:image/png;base64,...
// v2.2 schema: 拆 KeyMeta (跨 variant 共享 name/desc) + VariantMeta (per-resolution PNG+bbox).
// recRes = 录制时的 frame 分辨率 [W,H]; region = ratio [x,y,w,h] within recRes 帧.
// 同 key + 同 recRes 再调 → SaveVariant 覆盖该 variant; 不同 recRes → 加一个 variant.
func (s *Service) Save(containerID, key, dataURL, name, description string, recRes [2]int, region [4]float32) error {
	st, err := s.storeFor(containerID)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		return fmt.Errorf("dataURL must be data:image/png;base64,...")
	}
	pngData, err := base64.StdEncoding.DecodeString(dataURL[len("data:image/png;base64,"):])
	if err != nil {
		return fmt.Errorf("decode dataURL: %w", err)
	}

	// upsert KeyMeta: first save 创建; 后续 save 覆盖 name/desc, 保留 existing tags/origin
	existing, _ := st.GetMeta(key)
	km := KeyMeta{
		Name:        name,
		Description: description,
		Tags:        existing.Tags,
		Origin: TemplateOrigin{
			Kind:     ifStr(existing.Origin.Kind, "user"),
			SourceID: existing.Origin.SourceID,
		},
	}
	if err := st.SaveMeta(key, km); err != nil {
		return err
	}

	// region (ratio in recRes frame) → bbox (pixel in recRes frame)
	bbox := [4]int{
		int(region[0] * float32(recRes[0])),
		int(region[1] * float32(recRes[1])),
		int((region[0] + region[2]) * float32(recRes[0])),
		int((region[1] + region[3]) * float32(recRes[1])),
	}
	_, err = st.SaveVariant(key, pngData, VariantMeta{
		Resolution: recRes,
		BBox:       bbox,
	})
	return err
}

// ifStr 返 a 非空时 a, 否则 fallback. file-local helper.
func ifStr(a, fallback string) string {
	if a == "" {
		return fallback
	}
	return a
}

// List 返 map[key]TemplateMeta. 兼容 shape — 内部从 KeyMeta + first VariantMeta (area DESC, 最大分辨率) 拼.
// VariantCount = ListVariants 长度, FE 用来显示 "1080p+720p" 等徽标.
func (s *Service) List(containerID string) (map[string]TemplateMeta, error) {
	st, err := s.storeFor(containerID)
	if err != nil {
		return nil, err
	}
	out := map[string]TemplateMeta{}
	for key, km := range st.List() {
		vs := st.ListVariants(key)
		if len(vs) == 0 {
			// _meta exists 但 0 variant (半状态). 仍返条目让 UI 显示 broken thumbnail.
			out[key] = TemplateMeta{
				Name:         km.Name,
				Description:  km.Description,
				Tags:         km.Tags,
				Origin:       km.Origin,
				VariantCount: 0,
			}
			continue
		}
		first := vs[0] // ListVariants 已按 area DESC 排序
		// 反推 region (ratio) — FE 渲染 preview overlay 用
		region := [4]float32{}
		if first.Resolution[0] > 0 && first.Resolution[1] > 0 {
			region = [4]float32{
				float32(first.BBox[0]) / float32(first.Resolution[0]),
				float32(first.BBox[1]) / float32(first.Resolution[1]),
				float32(first.Width) / float32(first.Resolution[0]),
				float32(first.Height) / float32(first.Resolution[1]),
			}
		}
		out[key] = TemplateMeta{
			Name:               km.Name,
			Description:        km.Description,
			RecordedResolution: first.Resolution,
			SHA256:             first.SHA256,
			Width:              first.Width,
			Height:             first.Height,
			Region:             region,
			CreatedAt:          first.CreatedAt,
			Tags:               km.Tags,
			Origin:             km.Origin,
			VariantCount:       len(vs),
		}
	}
	return out, nil
}

func (s *Service) Delete(containerID, key string) error {
	st, err := s.storeFor(containerID)
	if err != nil {
		return err
	}
	return st.Delete(key)
}

// ReadPngDataURL 给 FE thumbnail 用. 返最大分辨率 variant 的 PNG (area DESC first).
// FE 若要拿特定分辨率走 ReadVariantPngDataURL (新加, Task 4.4).
func (s *Service) ReadPngDataURL(containerID, key string) (string, error) {
	st, err := s.storeFor(containerID)
	if err != nil {
		return "", err
	}
	vs := st.ListVariants(key)
	if len(vs) == 0 {
		return "", fmt.Errorf("template %q has no variants", key)
	}
	first := vs[0]
	b, err := st.ReadVariantPng(key, first.Resolution)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b), nil
}

// Capture 截取游戏窗口当前帧 (跟 containerID 无关, 共享 capture backend).
func (s *Service) Capture() (string, error) {
	pngData, err := s.capture.Capture()
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData), nil
}

// ListVariants 列指定 key 的全部 variant. 按 resolution area DESC.
func (s *Service) ListVariants(containerID, key string) ([]VariantMeta, error) {
	st, err := s.storeFor(containerID)
	if err != nil {
		return nil, err
	}
	return st.ListVariants(key), nil
}

// DeleteVariant 删指定 (key, resolution) 单 variant. 其他 variant + _meta 保留.
func (s *Service) DeleteVariant(containerID, key string, resolution [2]int) error {
	st, err := s.storeFor(containerID)
	if err != nil {
		return err
	}
	return st.DeleteVariant(key, resolution)
}

// ReadVariantPngDataURL 读指定 variant PNG. FE 后续 multi-variant UI 用.
func (s *Service) ReadVariantPngDataURL(containerID, key string, resolution [2]int) (string, error) {
	st, err := s.storeFor(containerID)
	if err != nil {
		return "", err
	}
	b, err := st.ReadVariantPng(key, resolution)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b), nil
}
