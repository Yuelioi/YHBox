package asset

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yottaapp/yotta/internal/apperr"
	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/internal/blob"
	"golang.org/x/image/draw"
)

const (
	previewMaxSourceBytes = 16 << 20
	previewMaxPixels      = 16 << 20
	previewMaxEdge        = 256
	previewMaxOutputBytes = 512 << 10
	previewTimeout        = 750 * time.Millisecond
)

// CaptureAdapter exposes trusted local authoring access to an installed target.
type CaptureAdapter interface {
	CapturePNG(context.Context, string) ([]byte, error)
	ResolveTarget(context.Context, string) (target.Target, error)
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
	Thumbnail    *blob.BlobRef         `json:"thumbnail,omitempty"`
	CreatedAt    string                `json:"createdAt,omitempty"`
}

// BlobPreview is a bounded presentation artifact, not a durable asset value.
type BlobPreview struct {
	MediaType string `json:"mediaType"`
	Base64    string `json:"base64"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type AssetQuery struct {
	Search          string   `json:"search"`
	Kind            string   `json:"kind"`
	Category        string   `json:"category"`
	Tags            []string `json:"tags"`
	Sort            string   `json:"sort"`
	Page            int      `json:"page"`
	PageSize        int      `json:"pageSize"`
	ThumbnailBudget int      `json:"thumbnailBudget"`
	RecentGUIDs     []string `json:"recentGUIDs"`
}

type AssetPage struct {
	Items      []AssetSummary `json:"items"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"pageSize"`
	Revision   uint64         `json:"revision"`
	Categories []FacetValue   `json:"categories"`
	Tags       []FacetValue   `json:"tags"`
}

type FacetValue struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// AssetBinding is authoring-only presentation metadata for one exact BlobRef.
// Workflow Source persists only Blob; GUID and name remain mutable library identity.
type AssetBinding struct {
	Found      bool         `json:"found"`
	GUID       string       `json:"guid"`
	Kind       string       `json:"kind"`
	Name       string       `json:"name"`
	Resolution [2]int       `json:"resolution"`
	Blob       blob.BlobRef `json:"blob"`
	MatchCount int          `json:"matchCount"`
}

type BatchMetaRequest struct {
	GUID     string   `json:"guid"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

type BatchResult struct {
	GUID    string `json:"guid"`
	Updated bool   `json:"updated,omitempty"`
	Deleted bool   `json:"deleted,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Service owns global asset metadata and installed-target authoring capture.
type Service struct {
	store      *Store
	capture    CaptureAdapter
	references DurableReferenceSource
	emit       func(name string, data any)
}

func NewService(store *Store, capture CaptureAdapter, references DurableReferenceSource, emit ...func(name string, data any)) *Service {
	service := &Service{store: store, capture: capture, references: references}
	if len(emit) != 0 {
		service.emit = emit[0]
	}
	return service
}

func (s *Service) emitChangedSince(before uint64, guids ...string) {
	revision := s.store.Revision()
	if revision == before || s.emit == nil {
		return
	}
	s.emit("asset:changed", map[string]any{"revision": revision, "guids": append([]string(nil), guids...)})
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
	before := s.store.Revision()
	rec := AssetRecord{
		GUID:      guid,
		Kind:      KindTemplate,
		Name:      name,
		Category:  category,
		Tags:      tags,
		Origin:    Origin{Kind: "user"},
		CreatedAt: time.Now().UTC(),
	}
	_, err = s.store.CommitRecordBlob(context.Background(), "image/png", bytes.NewReader(pngData), func(ref blob.BlobRef) AssetRecord {
		rec.Variants = []Variant{{Resolution: recRes, BBox: bbox, Blob: ref}}
		return rec
	})
	s.emitChangedSince(before, guid)
	if err != nil {
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
	before := s.store.Revision()
	_, err = s.store.CommitVariantBlob(context.Background(), "image/png", bytes.NewReader(pngData), guid, recRes, bbox, nil)
	s.emitChangedSince(before, guid)
	if err != nil {
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
	before := s.store.Revision()
	err := s.store.RemoveVariant(guid, [2]int{w, h})
	s.emitChangedSince(before, guid)
	if err != nil {
		return "", err
	}
	return guid, nil
}

// List 返全局资产摘要 (template + clip).
func (s *Service) List() []AssetSummary {
	out := []AssetSummary{}
	for _, rec := range s.store.List() {
		out = append(out, assetSummary(rec))
	}
	return out
}

func (s *Service) QueryAssets(query AssetQuery) (AssetPage, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 24
	}
	if query.Page > 1_000_000 || query.PageSize > 100 || query.ThumbnailBudget < 0 || query.ThumbnailBudget > query.PageSize {
		return AssetPage{}, apperr.New(apperr.CodeAssetQueryInvalid, map[string]any{"reason": "pagination or thumbnail budget"})
	}
	if query.Kind != "" && query.Kind != KindTemplate && query.Kind != KindClip && query.Kind != KindMacro {
		return AssetPage{}, apperr.New(apperr.CodeAssetQueryInvalid, map[string]any{"reason": "kind"})
	}
	if len([]rune(query.Search)) > 200 || len([]rune(query.Category)) > 100 || len(query.Tags) > 16 || len(query.RecentGUIDs) > 64 {
		return AssetPage{}, apperr.New(apperr.CodeAssetQueryInvalid, map[string]any{"reason": "filter budget"})
	}
	search := strings.ToLower(strings.TrimSpace(query.Search))
	category := strings.ToLower(strings.TrimSpace(query.Category))
	wantedTags := normalizedTags(query.Tags)
	recentRank := make(map[string]int, len(query.RecentGUIDs))
	for index, guid := range query.RecentGUIDs {
		if _, exists := recentRank[guid]; !exists {
			recentRank[guid] = index
		}
	}
	records, revision := s.store.ListWithRevision()
	categories, tags := assetFacets(records, query.Kind)
	items := make([]AssetSummary, 0)
	for _, record := range records {
		if query.Kind != "" && record.Kind != query.Kind {
			continue
		}
		if category != "" && strings.ToLower(record.Category) != category {
			continue
		}
		if !containsEveryTag(record.Tags, wantedTags) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(strings.Join(append([]string{record.Name, record.Description, record.Category, record.GUID}, record.Tags...), " ")), search) {
			continue
		}
		items = append(items, assetSummary(record))
	}
	switch query.Sort {
	case "", "name_asc":
		sort.SliceStable(items, func(i, j int) bool { return assetNameLess(items[i], items[j]) })
	case "name_desc":
		sort.SliceStable(items, func(i, j int) bool { return assetNameLess(items[j], items[i]) })
	case "created_desc":
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].CreatedAt == items[j].CreatedAt {
				return items[i].GUID < items[j].GUID
			}
			return items[i].CreatedAt > items[j].CreatedAt
		})
	case "recent_desc":
		sort.SliceStable(items, func(i, j int) bool {
			left, leftRecent := recentRank[items[i].GUID]
			right, rightRecent := recentRank[items[j].GUID]
			if leftRecent != rightRecent {
				return leftRecent
			}
			if leftRecent && left != right {
				return left < right
			}
			return assetNameLess(items[i], items[j])
		})
	default:
		return AssetPage{}, apperr.New(apperr.CodeAssetQueryInvalid, map[string]any{"reason": "sort"})
	}
	total := len(items)
	start := min((query.Page-1)*query.PageSize, total)
	end := min(start+query.PageSize, total)
	pageItems := append([]AssetSummary(nil), items[start:end]...)
	thumbnailCount := 0
	for index := range pageItems {
		if thumbnailCount >= query.ThumbnailBudget || pageItems[index].Kind != KindTemplate || len(pageItems[index].Variants) == 0 {
			continue
		}
		ref := pageItems[index].Variants[0].Blob
		pageItems[index].Thumbnail = &ref
		thumbnailCount++
	}
	return AssetPage{
		Items: pageItems, Total: total, Page: query.Page, PageSize: query.PageSize, Revision: revision,
		Categories: categories, Tags: tags,
	}, nil
}

func assetFacets(records []AssetRecord, kind string) ([]FacetValue, []FacetValue) {
	categoryCounts := make(map[string]int)
	tagCounts := make(map[string]int)
	categoryLabels := make(map[string]string)
	tagLabels := make(map[string]string)
	for _, record := range records {
		if kind != "" && record.Kind != kind {
			continue
		}
		if category := strings.TrimSpace(record.Category); category != "" {
			key := strings.ToLower(category)
			categoryCounts[key]++
			if categoryLabels[key] == "" {
				categoryLabels[key] = category
			}
		}
		seen := make(map[string]struct{}, len(record.Tags))
		for _, raw := range record.Tags {
			tag := strings.TrimSpace(raw)
			key := strings.ToLower(tag)
			if key == "" {
				continue
			}
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			seen[key] = struct{}{}
			tagCounts[key]++
			if tagLabels[key] == "" {
				tagLabels[key] = tag
			}
		}
	}
	return sortedAssetFacetValues(categoryCounts, categoryLabels), sortedAssetFacetValues(tagCounts, tagLabels)
}

func sortedAssetFacetValues(counts map[string]int, labels map[string]string) []FacetValue {
	values := make([]FacetValue, 0, len(counts))
	for key, count := range counts {
		values = append(values, FacetValue{Value: labels[key], Count: count})
	}
	sort.Slice(values, func(i, j int) bool { return strings.ToLower(values[i].Value) < strings.ToLower(values[j].Value) })
	return values
}

func assetNameLess(left, right AssetSummary) bool {
	leftName, rightName := strings.ToLower(left.Name), strings.ToLower(right.Name)
	if leftName == rightName {
		return left.GUID < right.GUID
	}
	return leftName < rightName
}

// ResolveBinding maps one durable BlobRef back to optional mutable library
// presentation. Duplicate assets may intentionally share the same content.
func (s *Service) ResolveBinding(ref blob.BlobRef) (AssetBinding, error) {
	if err := ref.Validate(); err != nil {
		return AssetBinding{}, apperr.New(apperr.CodeAssetQueryInvalid, map[string]any{"reason": "blob reference"})
	}
	matches := make([]AssetBinding, 0, 1)
	for _, record := range s.store.List() {
		if record.Blob != nil && *record.Blob == ref {
			matches = append(matches, AssetBinding{Found: true, GUID: record.GUID, Kind: record.Kind, Name: record.Name, Blob: ref})
		}
		for _, variant := range record.Variants {
			if variant.Blob == ref {
				matches = append(matches, AssetBinding{Found: true, GUID: record.GUID, Kind: record.Kind, Name: record.Name, Resolution: variant.Resolution, Blob: ref})
			}
		}
	}
	if len(matches) == 0 {
		return AssetBinding{Blob: ref}, nil
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].GUID < matches[j].GUID })
	matches[0].MatchCount = len(matches)
	return matches[0], nil
}

func (s *Service) BatchUpdateMeta(requests []BatchMetaRequest) []BatchResult {
	before := s.store.Revision()
	results := make([]BatchResult, 0, len(requests))
	seen := make(map[string]struct{}, len(requests))
	for _, request := range requests {
		result := BatchResult{GUID: request.GUID}
		if _, duplicate := seen[request.GUID]; duplicate {
			result.Error = "duplicate asset update request"
		} else if record, ok := s.store.Get(request.GUID); !ok {
			result.Error = "asset not found"
		} else {
			seen[request.GUID] = struct{}{}
			if err := s.store.PutRecordMeta(record.GUID, record.Name, record.Description, strings.TrimSpace(request.Category), cleanTags(request.Tags)); err != nil {
				result.Error = err.Error()
			} else {
				result.Updated = true
			}
		}
		results = append(results, result)
	}
	s.emitChangedSince(before)
	return results
}

func (s *Service) BatchDelete(guids []string) []BatchResult {
	before := s.store.Revision()
	results := make([]BatchResult, 0, len(guids))
	seen := make(map[string]struct{}, len(guids))
	for _, guid := range guids {
		result := BatchResult{GUID: guid}
		if _, duplicate := seen[guid]; duplicate {
			result.Error = "duplicate asset delete request"
		} else {
			seen[guid] = struct{}{}
			if _, ok := s.store.Get(guid); !ok {
				result.Error = "asset not found"
			} else if err := s.store.DeleteRecord(guid); err != nil {
				result.Error = err.Error()
			} else {
				result.Deleted = true
			}
		}
		results = append(results, result)
	}
	s.emitChangedSince(before, guids...)
	return results
}

// Get 返单条记录.
func (s *Service) Get(guid string) (AssetRecord, error) {
	rec, ok := s.store.Get(guid)
	if !ok {
		return AssetRecord{}, fmt.Errorf("asset %q not found", guid)
	}
	return rec, nil
}

// PreviewBlob renders a bounded PNG thumbnail for an exact immutable BlobRef.
func (s *Service) PreviewBlob(ref blob.BlobRef) (BlobPreview, error) {
	if ref.MediaType != "image/png" {
		return BlobPreview{}, fmt.Errorf("preview media type %q is not supported", ref.MediaType)
	}
	if ref.Size <= 0 || ref.Size > previewMaxSourceBytes {
		return BlobPreview{}, fmt.Errorf("preview source size %d exceeds budget", ref.Size)
	}
	ctx, cancel := context.WithTimeout(context.Background(), previewTimeout)
	defer cancel()
	content, err := s.store.ReadBlob(ctx, ref)
	if err != nil {
		return BlobPreview{}, fmt.Errorf("read preview blob: %w", err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		return BlobPreview{}, fmt.Errorf("decode preview header: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > previewMaxPixels/config.Height {
		return BlobPreview{}, fmt.Errorf("preview dimensions %dx%d exceed budget", config.Width, config.Height)
	}
	if err := ctx.Err(); err != nil {
		return BlobPreview{}, fmt.Errorf("preview deadline: %w", err)
	}
	source, err := png.Decode(bytes.NewReader(content))
	if err != nil {
		return BlobPreview{}, fmt.Errorf("decode preview: %w", err)
	}
	width, height := previewDimensions(config.Width, config.Height)
	thumbnail := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(thumbnail, thumbnail.Bounds(), source, source.Bounds(), draw.Over, nil)
	if err := ctx.Err(); err != nil {
		return BlobPreview{}, fmt.Errorf("preview deadline: %w", err)
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, thumbnail); err != nil {
		return BlobPreview{}, fmt.Errorf("encode preview: %w", err)
	}
	if encoded.Len() > previewMaxOutputBytes {
		return BlobPreview{}, fmt.Errorf("preview output exceeds byte budget")
	}
	return BlobPreview{
		MediaType: "image/png",
		Base64:    base64.StdEncoding.EncodeToString(encoded.Bytes()),
		Width:     width,
		Height:    height,
	}, nil
}

func previewDimensions(width, height int) (int, int) {
	if width <= previewMaxEdge && height <= previewMaxEdge {
		return width, height
	}
	if width >= height {
		return previewMaxEdge, max(1, height*previewMaxEdge/width)
	}
	return max(1, width*previewMaxEdge/height), previewMaxEdge
}

// UpdateMeta 改资产显示名 + 描述 + 分类 + 标签 (记录级元数据, 不动变体/blob).
func (s *Service) UpdateMeta(guid, name, description, category string, tags []string) error {
	if _, ok := s.store.Get(guid); !ok {
		return fmt.Errorf("asset %q not found", guid)
	}
	before := s.store.Revision()
	err := s.store.PutRecordMeta(guid, name, description, category, tags)
	s.emitChangedSince(before, guid)
	return err
}

// Delete removes asset metadata. Workflows retain immutable BlobRefs rather
// than asset GUIDs, so deletion never attempts graph-wide reference inference.
func (s *Service) Delete(guid string) error {
	before := s.store.Revision()
	err := s.store.DeleteRecord(guid)
	s.emitChangedSince(before, guid)
	return err
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

// CurrentResolution resolves the installed target's current automation space.
func (s *Service) CurrentResolution(targetSlot string) ([2]int, error) {
	if s.capture == nil {
		return [2]int{}, fmt.Errorf("capture adapter 未注入")
	}
	resolved, err := s.capture.ResolveTarget(context.Background(), targetSlot)
	if err != nil {
		return [2]int{}, err
	}
	if resolved.Resolution.W <= 0 || resolved.Resolution.H <= 0 {
		return [2]int{}, fmt.Errorf("automation target %q has invalid resolution %dx%d", targetSlot, resolved.Resolution.W, resolved.Resolution.H)
	}
	return [2]int{resolved.Resolution.W, resolved.Resolution.H}, nil
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

func assetSummary(rec AssetRecord) AssetSummary {
	summary := AssetSummary{
		GUID: rec.GUID, Kind: rec.Kind, Name: rec.Name,
		Description: rec.Description, Category: rec.Category, Tags: append([]string(nil), rec.Tags...),
		VariantCount: len(rec.Variants), Variants: []AssetVariantSummary{},
	}
	if !rec.CreatedAt.IsZero() {
		summary.CreatedAt = rec.CreatedAt.UTC().Format(time.RFC3339)
	}
	for _, variant := range rec.Variants {
		summary.Variants = append(summary.Variants, AssetVariantSummary{Resolution: variant.Resolution, Blob: variant.Blob})
	}
	if rec.Blob != nil {
		ref := *rec.Blob
		summary.Blob = &ref
	}
	return summary
}

func normalizedTags(tags []string) map[string]struct{} {
	result := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if normalized := strings.ToLower(strings.TrimSpace(tag)); normalized != "" {
			result[normalized] = struct{}{}
		}
	}
	return result
}

func containsEveryTag(tags []string, wanted map[string]struct{}) bool {
	if len(wanted) == 0 {
		return true
	}
	available := normalizedTags(tags)
	for tag := range wanted {
		if _, ok := available[tag]; !ok {
			return false
		}
	}
	return true
}

func cleanTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		key := strings.ToLower(tag)
		if tag == "" {
			continue
		}
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, tag)
	}
	return result
}
