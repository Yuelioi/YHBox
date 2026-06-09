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
	// Resolution 返目标窗口客户区分辨率 [宽,高]; 窗口没开/容器无 WindowTarget → error.
	// 走 GetClientRect, 不截帧 — 与截图帧尺寸 (recRes) 同源, 故可拿来精确匹配变体档.
	Resolution(containerID string) ([2]int, error)
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
func (s *Service) SaveTemplateCapture(dataURL, name string, tags []string, recRes [2]int, region [4]float32) (string, error) {
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
		Tags:      tags,
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
// 返目标 GUID — 给 FE 区分成功 (返 guid) 与失败 (RPC error), 跟 SaveTemplateCapture 对称.
func (s *Service) AddTemplateVariant(guid, dataURL string, recRes [2]int, region [4]float32) (string, error) {
	if _, ok := s.store.Get(guid); !ok {
		return "", fmt.Errorf("asset %q not found", guid)
	}
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		return "", fmt.Errorf("dataURL must be data:image/png;base64,...")
	}
	pngData, err := base64.StdEncoding.DecodeString(dataURL[len("data:image/png;base64,"):])
	if err != nil {
		return "", fmt.Errorf("decode dataURL: %w", err)
	}
	sha, err := s.store.Blobs().Put(pngData)
	if err != nil {
		return "", err
	}
	bbox := [4]int{
		int(region[0] * float32(recRes[0])),
		int(region[1] * float32(recRes[1])),
		int((region[0] + region[2]) * float32(recRes[0])),
		int((region[1] + region[3]) * float32(recRes[1])),
	}
	if err := s.store.PutVariant(guid, recRes, sha, bbox, nil); err != nil {
		return "", err
	}
	s.notifyChange()
	return guid, nil
}

// RemoveVariant 删指定分辨率的单个变体档 (详情页"删这一档"). 返目标 GUID 给 FE 区分成功/失败.
// 守卫: 仅剩 1 档时拒删 (删它=废掉整个素材, 该走 Delete 整删 — 带引用警告). FE 也仅在 >1 档时给入口.
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
	s.notifyChange()
	return guid, nil
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

// UpdateMeta 改资产显示名 + 标签 (记录级元数据, 不动变体/blob).
func (s *Service) UpdateMeta(guid, name string, tags []string) error {
	if _, ok := s.store.Get(guid); !ok {
		return fmt.Errorf("asset %q not found", guid)
	}
	if err := s.store.PutRecordMeta(guid, name, tags); err != nil {
		return err
	}
	s.notifyChange()
	return nil
}

// Referrers 只扫不删 — 返回引用某资产 GUID 的全部位置, 给 FE 删除前弹"被 N 处引用"确认弹窗.
// scanReferrers 未注入 (测试) → 返 nil.
func (s *Service) Referrers(guid string) []Referrer {
	if s.scanReferrers == nil {
		return nil
	}
	return s.scanReferrers(guid)
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

// CurrentResolution 返当前容器目标窗口客户区分辨率 [宽,高]. 详情页据此挑"运行时真会用的那档"
// + 显示当前分辨率 + 决定"重拍/新增". 窗口没开/无容器上下文 → error (FE 静默降级, 不弹 toast).
func (s *Service) CurrentResolution(containerID string) ([2]int, error) {
	if s.capture == nil {
		return [2]int{}, fmt.Errorf("capture adapter 未注入")
	}
	return s.capture.Resolution(containerID)
}

// VariantPick 给详情页: 当前分辨率下运行时真会用的那档在 record.Variants[] 里的下标 + 是否精确命中当前分辨率.
type VariantPick struct {
	Index int  `json:"index"`
	Exact bool `json:"exact"`
}

// PickVariant 给定帧分辨率, 返运行时真会用的那档 (store.PickVariant: 精确命中优先, 否则长边比最近)
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

// GCBlobs 显式触发 blob GC (回收无记录引用的孤儿字节). 返回回收数.
func (s *Service) GCBlobs() (int, error) {
	return s.store.GCBlobs()
}
