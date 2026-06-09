package asset

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CaptureAdapter 截取指定容器目标窗口当前帧 PNG bytes. main.go 注入 (截帧仍需窗口上下文).
type CaptureAdapter interface {
	Capture(containerID string) ([]byte, error)
}

// AssetSummary 列表项 — 给 picker. 不含像素, 只 metadata + 首 blob sha (FE 拉缩略图用).
type AssetSummary struct {
	GUID         string   `json:"guid"`
	Kind         string   `json:"kind"`
	Name         string   `json:"name"`
	Tags         []string `json:"tags,omitempty"`
	VariantCount int      `json:"variantCount"`
	FirstBlobSha string   `json:"firstBlobSha,omitempty"`
	CreatedAt    string   `json:"createdAt,omitempty"`
}

// Service 全局资产 Wails RPC. 无 containerID (资产全局), guid 寻址.
type Service struct {
	store   *Store
	capture CaptureAdapter

	// scanReferrers 注入式扫全部容器+子图引用某 guid 的位置 (wire 层用 dependency BFS 填).
	// nil → Delete 不返引用列表 (退化, 测试).
	scanReferrers func(guid string) []Referrer

	// onChange: 资产 Save/Delete/Rename 后触发. main.go 接 matcher 解码缓存失效.
	onChange func()
}

func NewService(store *Store, capture CaptureAdapter) *Service {
	return &Service{store: store, capture: capture}
}

// SetOnChange 注册资产变更回调 (Save/Delete/Rename 成功后触发).
func (s *Service) SetOnChange(fn func()) { s.onChange = fn }

// SetReferrerScanner 注入引用扫描器 (wire 层用 container/dependency BFS).
func (s *Service) SetReferrerScanner(fn func(guid string) []Referrer) { s.scanReferrers = fn }

func (s *Service) notifyChange() {
	if s.onChange != nil {
		s.onChange()
	}
}

// SaveTemplateCapture 用户截模板 → 建新资产 (新 GUID) + 单变体 + blob.
// dataURL 必须 data:image/png;base64,...; recRes = 录制帧分辨率 [W,H];
// region = ratio [x,y,w,h] within recRes 帧 → 换算成 bbox 像素 (逐字搬旧 template/service.go).
// 返新分配的 GUID (用户看不到, FE 拿来 set 节点 pin).
func (s *Service) SaveTemplateCapture(dataURL, name string, recRes [2]int, region [4]float32) (string, error) {
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		return "", fmt.Errorf("dataURL must be data:image/png;base64,...")
	}
	pngData, err := base64.StdEncoding.DecodeString(dataURL[len("data:image/png;base64,"):])
	if err != nil {
		return "", fmt.Errorf("decode dataURL: %w", err)
	}
	sha, err := s.store.Blobs().Put(pngData)
	if err != nil {
		return "", fmt.Errorf("put template blob: %w", err)
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
		Origin:    Origin{Kind: "user"},
		Variants:  []Variant{{Resolution: recRes, BBox: bbox, Blob: sha}},
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.PutRecord(rec); err != nil {
		return "", err
	}
	s.notifyChange()
	return guid, nil
}

// AddTemplateVariant 给已有模板加/换一个分辨率档 (重拍同 GUID). 同 recRes 覆盖该变体 blob.
func (s *Service) AddTemplateVariant(guid, dataURL string, recRes [2]int, region [4]float32) error {
	if _, ok := s.store.Get(guid); !ok {
		return fmt.Errorf("asset %q not found", guid)
	}
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		return fmt.Errorf("dataURL must be data:image/png;base64,...")
	}
	pngData, err := base64.StdEncoding.DecodeString(dataURL[len("data:image/png;base64,"):])
	if err != nil {
		return fmt.Errorf("decode dataURL: %w", err)
	}
	sha, err := s.store.Blobs().Put(pngData)
	if err != nil {
		return err
	}
	bbox := [4]int{
		int(region[0] * float32(recRes[0])),
		int(region[1] * float32(recRes[1])),
		int((region[0] + region[2]) * float32(recRes[0])),
		int((region[1] + region[3]) * float32(recRes[1])),
	}
	if err := s.store.PutVariant(guid, recRes, sha, bbox, nil); err != nil {
		return err
	}
	s.notifyChange()
	return nil
}

// List 返全局资产摘要 (template + clip).
func (s *Service) List() []AssetSummary {
	out := []AssetSummary{}
	for _, rec := range s.store.List() {
		sum := AssetSummary{
			GUID: rec.GUID, Kind: rec.Kind, Name: rec.Name, Tags: rec.Tags,
			VariantCount: len(rec.Variants),
		}
		if !rec.CreatedAt.IsZero() {
			sum.CreatedAt = rec.CreatedAt.UTC().Format(time.RFC3339)
		}
		switch {
		case len(rec.Variants) > 0:
			sum.FirstBlobSha = rec.Variants[0].Blob
		case rec.Blob != "":
			sum.FirstBlobSha = rec.Blob
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

// Rename 改资产显示名 (tags 保留).
func (s *Service) Rename(guid, name string) error {
	rec, ok := s.store.Get(guid)
	if !ok {
		return fmt.Errorf("asset %q not found", guid)
	}
	if err := s.store.PutRecordMeta(guid, name, rec.Tags); err != nil {
		return err
	}
	s.notifyChange()
	return nil
}

// Delete 删资产记录. 删前扫引用返回 Referrer 列表 (不阻断, FE 据此弹"被 N 处引用"警告).
func (s *Service) Delete(guid string) ([]Referrer, error) {
	var refs []Referrer
	if s.scanReferrers != nil {
		refs = s.scanReferrers(guid)
	}
	if err := s.store.DeleteRecord(guid); err != nil {
		return refs, err
	}
	s.notifyChange()
	return refs, nil
}

// ReadBlobDataURL 给 FE thumbnail: 按 sha 读 blob → data:image/png;base64,...
func (s *Service) ReadBlobDataURL(sha string) (string, error) {
	b, err := s.store.Blobs().Read(sha)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b), nil
}

// Capture 截取指定容器目标窗口当前帧 (制作模板时取底图). 保留 containerID — 截帧需窗口上下文.
func (s *Service) Capture(containerID string) (string, error) {
	if s.capture == nil {
		return "", fmt.Errorf("capture adapter 未注入")
	}
	pngData, err := s.capture.Capture(containerID)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData), nil
}

// GCBlobs 显式触发 blob GC (回收无记录引用的孤儿字节). 返回回收数.
func (s *Service) GCBlobs() (int, error) {
	return s.store.GCBlobs()
}
