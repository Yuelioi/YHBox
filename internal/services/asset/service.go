package asset

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/internal/blob"
)

// CaptureAdapter exposes trusted local authoring access to an installed target.
type CaptureAdapter interface {
	CapturePNG(context.Context, string) ([]byte, error)
	ResolveWindow(context.Context, string) (target.WindowHandle, error)
}

type AssetVariantSummary struct {
	Resolution [2]int       `json:"resolution"`
	Blob       blob.BlobRef `json:"blob"`
}

// AssetSummary is picker metadata plus every immutable content binding choice.
type AssetSummary struct {
	GUID         string                `json:"guid"`
	Kind         string                `json:"kind"`
	Name         string                `json:"name"`
	Description  string                `json:"description,omitempty"`
	Category     string                `json:"category,omitempty"`
	Tags         []string              `json:"tags,omitempty"`
	VariantCount int                   `json:"variantCount"`
	Variants     []AssetVariantSummary `json:"variants"`
	Blob         *blob.BlobRef         `json:"blob,omitempty"`
	CreatedAt    string                `json:"createdAt,omitempty"`
}

// Service owns global asset metadata and installed-target authoring capture.
type Service struct {
	store   *Store
	capture CaptureAdapter
}

func NewService(store *Store, capture CaptureAdapter) *Service {
	return &Service{store: store, capture: capture}
}

// SaveTemplateCapture 用户截模板 → 建新资产 (新 GUID) + 单变体 + blob.
// dataURL 必须 data:image/png;base64,...; recRes = 录制帧分辨率 [W,H];
// region = ratio [x,y,w,h] within recRes 帧 → 换算成 bbox 像素 (逐字搬旧 template/service.go).
// 返新分配的 GUID (用户看不到, FE 拿来 set 节点 pin).
func (s *Service) SaveTemplateCapture(dataURL, name, category string, tags []string, recRes [2]int, region [4]float32) (string, error) {
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		return "", fmt.Errorf("data URL must start with %q", "data:image/png;base64,")
	}
	pngData, err := base64.StdEncoding.DecodeString(dataURL[len("data:image/png;base64,"):])
	if err != nil {
		return "", fmt.Errorf("decode dataURL: %w", err)
	}
	// region (ratio in recRes frame) → bbox (pixel in recRes frame). 逐字搬旧 service.go:92-98.
	bbox := [4]int{
		int(region[0] * float32(recRes[0])),
		int(region[1] * float32(recRes[1])),
		int((region[0] + region[2]) * float32(recRes[0])),
		int((region[1] + region[3]) * float32(recRes[1])),
	}
	guid := uuid.NewString()
	rec := AssetRecord{
		GUID:      guid,
		Kind:      KindTemplate,
		Name:      name,
		Category:  category,
		Tags:      tags,
		Origin:    Origin{Kind: "user"},
		CreatedAt: time.Now().UTC(),
	}
	if _, err := s.store.CommitRecordBlob(context.Background(), "image/png", bytes.NewReader(pngData), func(ref blob.BlobRef) AssetRecord {
		rec.Variants = []Variant{{Resolution: recRes, BBox: bbox, Blob: ref}}
		return rec
	}); err != nil {
		return "", fmt.Errorf("commit template blob: %w", err)
	}
	return guid, nil
}

// AddTemplateVariant 给已有模板加/换一个分辨率档 (重拍同 GUID). 同 recRes 覆盖该变体 blob.
// 返目标 GUID — 给 FE 区分成功 (返 guid) 与失败 (RPC error), 跟 SaveTemplateCapture 对称.
func (s *Service) AddTemplateVariant(guid, dataURL string, recRes [2]int, region [4]float32) (string, error) {
	if _, ok := s.store.Get(guid); !ok {
		return "", fmt.Errorf("asset %q not found", guid)
	}
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		return "", fmt.Errorf("data URL must start with %q", "data:image/png;base64,")
	}
	pngData, err := base64.StdEncoding.DecodeString(dataURL[len("data:image/png;base64,"):])
	if err != nil {
		return "", fmt.Errorf("decode dataURL: %w", err)
	}
	bbox := [4]int{
		int(region[0] * float32(recRes[0])),
		int(region[1] * float32(recRes[1])),
		int((region[0] + region[2]) * float32(recRes[0])),
		int((region[1] + region[3]) * float32(recRes[1])),
	}
	if _, err := s.store.CommitVariantBlob(context.Background(), "image/png", bytes.NewReader(pngData), guid, recRes, bbox, nil); err != nil {
		return "", fmt.Errorf("commit template variant: %w", err)
	}
	return guid, nil
}

// RemoveVariant 删指定分辨率的单个变体档 (详情页"删这一档"). 返目标 GUID 给 FE 区分成功/失败.
// 守卫: 仅剩 1 档时拒删 (删它=废掉整个素材, 该走 Delete 整删). FE 也仅在 >1 档时给入口.
func (s *Service) RemoveVariant(guid string, w, h int) (string, error) {
	rec, ok := s.store.Get(guid)
	if !ok {
		return "", fmt.Errorf("asset %q not found", guid)
	}
	if len(rec.Variants) <= 1 {
		return "", fmt.Errorf("asset %q 仅剩 1 个分辨率档, 删它请用删除整个素材", guid)
	}
	if err := s.store.RemoveVariant(guid, [2]int{w, h}); err != nil {
		return "", err
	}
	return guid, nil
}

// List 返全局资产摘要 (template + clip).
func (s *Service) List() []AssetSummary {
	out := []AssetSummary{}
	for _, rec := range s.store.List() {
		sum := AssetSummary{
			GUID: rec.GUID, Kind: rec.Kind, Name: rec.Name,
			Description: rec.Description, Category: rec.Category, Tags: rec.Tags,
			VariantCount: len(rec.Variants), Variants: []AssetVariantSummary{},
		}
		if !rec.CreatedAt.IsZero() {
			sum.CreatedAt = rec.CreatedAt.UTC().Format(time.RFC3339)
		}
		for _, variant := range rec.Variants {
			sum.Variants = append(sum.Variants, AssetVariantSummary{Resolution: variant.Resolution, Blob: variant.Blob})
		}
		if rec.Blob != nil {
			ref := *rec.Blob
			sum.Blob = &ref
		}
		out = append(out, sum)
	}
	return out
}

// Get 返单条记录.
func (s *Service) Get(guid string) (AssetRecord, error) {
	rec, ok := s.store.Get(guid)
	if !ok {
		return AssetRecord{}, fmt.Errorf("asset %q not found", guid)
	}
	return rec, nil
}

// UpdateMeta 改资产显示名 + 描述 + 分类 + 标签 (记录级元数据, 不动变体/blob).
func (s *Service) UpdateMeta(guid, name, description, category string, tags []string) error {
	if _, ok := s.store.Get(guid); !ok {
		return fmt.Errorf("asset %q not found", guid)
	}
	if err := s.store.PutRecordMeta(guid, name, description, category, tags); err != nil {
		return err
	}
	return nil
}

// Delete removes asset metadata. Workflows retain immutable BlobRefs rather
// than asset GUIDs, so deletion never attempts graph-wide reference inference.
func (s *Service) Delete(guid string) error {
	if err := s.store.DeleteRecord(guid); err != nil {
		return err
	}
	return nil
}

// Capture captures the exact installed target selected by targetSlot.
func (s *Service) Capture(targetSlot string) (string, error) {
	if s.capture == nil {
		return "", fmt.Errorf("capture adapter 未注入")
	}
	pngData, err := s.capture.CapturePNG(context.Background(), targetSlot)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData), nil
}

// CurrentResolution resolves the installed target's current client size.
func (s *Service) CurrentResolution(targetSlot string) ([2]int, error) {
	if s.capture == nil {
		return [2]int{}, fmt.Errorf("capture adapter 未注入")
	}
	window, err := s.capture.ResolveWindow(context.Background(), targetSlot)
	if err != nil {
		return [2]int{}, err
	}
	if window.ClientW <= 0 || window.ClientH <= 0 {
		return [2]int{}, fmt.Errorf("automation target %q has invalid client size %dx%d", targetSlot, window.ClientW, window.ClientH)
	}
	return [2]int{window.ClientW, window.ClientH}, nil
}

// VariantPick 给详情页: 当前分辨率下推荐绑定的档位在 record.Variants[] 里的下标 + 是否精确命中当前分辨率.
type VariantPick struct {
	Index int  `json:"index"`
	Exact bool `json:"exact"`
}

// PickVariant 给定帧分辨率, 返推荐绑定的档位 (store.PickVariant: 精确命中优先, 否则长边比最近)
// 在 record.Variants[] 里的下标 + 是否精确命中该分辨率. 详情页进来据此自动切档 + 决定按钮"重拍/新增".
// 挑档算法权威在 store.PickVariant (有单测), 此处只把选中档换算成下标, 不复刻算法.
func (s *Service) PickVariant(guid string, w, h int) (VariantPick, error) {
	v, ok := s.store.PickVariant(guid, w, h)
	if !ok {
		return VariantPick{}, fmt.Errorf("asset %q 无可用变体 (%dx%d)", guid, w, h)
	}
	rec, ok := s.store.Get(guid)
	if !ok {
		return VariantPick{}, fmt.Errorf("asset %q not found", guid)
	}
	for i, vv := range rec.Variants {
		if vv.Resolution == v.Resolution {
			return VariantPick{Index: i, Exact: v.Resolution[0] == w && v.Resolution[1] == h}, nil
		}
	}
	return VariantPick{}, fmt.Errorf("picked variant %v not in record %q", v.Resolution, guid)
}
