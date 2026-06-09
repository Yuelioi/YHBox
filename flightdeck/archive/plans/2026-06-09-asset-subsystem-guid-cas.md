---
status: done
summary: 按相位实现 asset 子系统: 新 asset 包(blob 池+记录+PickVariant+GC) → 运行时匹配接全局 store → 节点 semantic/校验/依赖改 GUID → clip 并入 → 分享导入坍缩 → RPC/前端 picker/捕获 → 接线+smoke
last_updated: 2026-06-10
implements: specs/2026-06-09-asset-subsystem-guid-cas.md
---

# 统一资产子系统 — 实现 plan

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development`（每 task 一个新 subagent + task 间 review）。步骤用 `- [ ]` 勾选跟踪。
>
> **⛔ 头号铁律（本仓铁律，凌驾一切）**：每个 task 动手前**先读它列出的 `源码:行` 把现状读准再改**，不许凭"通常这样写"脑补。pin 名 / config 键 / 返回值必须 grep 实现确认。
> **⛔ 二号铁律**：不要兼容。不留双读/旧路径 fallback/schema 版本 if。删干净。
> **设计真源**：[../specs/2026-06-09-asset-subsystem-guid-cas.md](../specs/2026-06-09-asset-subsystem-guid-cas.md)。本 plan 与 spec 冲突时**以 spec 为准**并回头改 plan。

**Goal**：把模板/clip 从"容器按 `namespace.name` 拥有"重构为全局资产库（每资产稳定 GUID + name 仅标签）+ 全局 `assets/blobs/<sha256>` 内容寻址字节池；节点按 GUID 引用；资产独立于图引用存在；分享/导入坍缩成 GUID+sha 幂等合并。

**Architecture**：新 `internal/services/asset` 包持有全局 blob 池 + 记录库 + `PickVariant` + GC，单实例全局共享（非 per-container）。运行时匹配、节点依赖、validator、library 分享、RPC、前端 picker 全部改为按 GUID。`template` 包被 asset 取代，`inputclip` 存储层并入 asset 的 clip kind。

**Tech Stack**：Go（`github.com/google/uuid`、`crypto/sha256`、`encoding/json`、`sync.RWMutex`、temp+rename 原子写）；前端 Vue3 + Pinia + Wails bindings；视觉匹配 `internal/vision`（NCC，不动）。

**测试约定**：单元门 = `go test ./...`；编译门 = `go build ./...`；真机 smoke 走 [../checklists/build.md](../checklists/build.md)（最后相位）。每 task 末尾 commit（rules `commit without asking`）。

---

## 文件结构（决定分解）

**新建**
- `internal/services/asset/model.go` — `AssetRecord` / `Variant` / `Origin` 类型 + `kind` 常量。
- `internal/services/asset/blobstore.go` — blob 池：`PutBlob/ReadBlob/HasBlob`，CAS + temp/rename 原子。
- `internal/services/asset/store.go` — 记录库：`sync.RWMutex` + preload + `Get/List/Delete/PutRecord/PutRecordMeta/PutVariant/RemoveVariant/PickVariant`。
- `internal/services/asset/gc.go` — `GCBlobs`（mark-sweep）+ `ScanReferrers`（usage 扫描，复用 dependency BFS）。
- `internal/services/asset/service.go` — Wails RPC：`List/Get/Rename/Delete/Capture/SaveTemplateCapture/ReadBlobDataURL/...`。
- `internal/services/asset/*_test.go` — 各文件配套单测。
- `internal/services/container/library/bundle.go` — 新 bundle 打包/解包（替代 copy.go 的 strategy/conflict）。

**重写/吸收（删旧）**
- `internal/services/template/`（整包删，逻辑并入 asset）。
- `internal/services/inputclip/` 的存储层（`model.go`/store）→ asset clip kind；回放/录制逻辑保留。
- `internal/services/container/library/copy.go` + `store.go` — bundle 改 GUID 闭包 + 加性合并。
- `internal/services/container/validator_template_key.go`（删）、`keyvalidation.go`（删）。
- `wire_container.go` 的 `templateMatcherAdapter`、`wire_inputclip.go`、`main.go` 接线。

**机械改引用**
- `internal/nodes/detect/{check,click,wait}_template.go`（semantic）、`template_common.go`（deps）。
- `internal/nodes/io/play_clip.go`（clip kind 对齐）。
- `internal/services/container/validator_deps.go`（`hasTemplate/hasClip` 接 asset 存在性）。
- 前端 `stores/templates.ts`、`components/containers/TemplatePicker.vue`、`components/templates/TemplateCapture.vue`。

---

## Phase 0 — asset 包：类型 + blob 池（纯新建，隔离单测）

### Task 0.1：资产记录类型
**Files**：Create `internal/services/asset/model.go`；Test `internal/services/asset/model_test.go`

- [ ] **Step 1：写类型**（无行为，先定 schema）
```go
package asset

import "time"

const (
	KindTemplate = "template"
	KindClip     = "clip"
)

// Origin 来源溯源 (非身份字段). 沿用旧 template.TemplateOrigin 语义.
type Origin struct {
	Kind     string `json:"kind"` // "user" | "imported" | "subgraph"
	SourceID string `json:"sourceID,omitempty"`
}

// Variant 模板的单分辨率变体: 元数据在记录里, 像素在 blob 池 (Blob = sha256).
type Variant struct {
	Resolution [2]int   `json:"resolution"`        // [W,H] 录制帧尺寸
	BBox       [4]int   `json:"bbox"`              // [x1,y1,x2,y2] 源帧像素位置
	Regions    [][4]int `json:"regions,omitempty"` // 多槽检测, 空=单 BBox
	Blob       string   `json:"blob"`              // 像素 PNG 的 sha256
}

// AssetRecord 一资产一文件 assets/records/<guid>.json.
type AssetRecord struct {
	GUID      string    `json:"guid"`
	Kind      string    `json:"kind"`            // KindTemplate | KindClip
	Name      string    `json:"name"`            // 可变显示标签, 可重名
	Tags      []string  `json:"tags,omitempty"`
	Origin    Origin    `json:"origin"`
	Variants  []Variant `json:"variants,omitempty"` // 仅 template; 按 Resolution 唯一
	Blob      string    `json:"blob,omitempty"`     // 仅 clip: 事件流序列化字节的 sha256
	CreatedAt time.Time `json:"createdAt"`
}
```

- [ ] **Step 2：写 round-trip 测试**
```go
func TestAssetRecord_JSONRoundTrip(t *testing.T) {
	r := AssetRecord{GUID: "g1", Kind: KindTemplate, Name: "登录",
		Variants: []Variant{{Resolution: [2]int{1920, 1080}, BBox: [4]int{1, 2, 3, 4}, Blob: "abc"}}}
	b, _ := json.Marshal(r)
	var got AssetRecord
	if err := json.Unmarshal(b, &got); err != nil { t.Fatal(err) }
	if got.Variants[0].Blob != "abc" || got.Kind != KindTemplate { t.Fatalf("roundtrip mismatch: %+v", got) }
}
```
- [ ] **Step 3**：`go test ./internal/services/asset/ -run RoundTrip -v` → PASS。
- [ ] **Step 4**：commit `feat(asset): record/variant types`。

### Task 0.2：blob 池（CAS + 原子写 + 幂等）
**Files**：Create `internal/services/asset/blobstore.go`；Test `blobstore_test.go`
**先读**：`internal/services/template/store.go:129-135`（`atomicWriteFile` temp+rename 模式，照搬）。

- [ ] **Step 1：写失败测试**
```go
func TestBlobStore_PutIdempotentDedup(t *testing.T) {
	dir := t.TempDir()
	bs := NewBlobStore(dir)
	sha1, err := bs.Put([]byte("hello")); if err != nil { t.Fatal(err) }
	sha2, _ := bs.Put([]byte("hello"))
	if sha1 != sha2 { t.Fatal("same bytes must give same sha") }
	if !bs.Has(sha1) { t.Fatal("Has false after Put") }
	got, _ := bs.Read(sha1)
	if string(got) != "hello" { t.Fatalf("read mismatch %q", got) }
	// 去重: 目录里只有一个文件
	entries, _ := os.ReadDir(filepath.Join(dir, "blobs"))
	if len(entries) != 1 { t.Fatalf("expected 1 blob, got %d", len(entries)) }
}
```
- [ ] **Step 2**：`go test ... -run PutIdempotent` → FAIL（NewBlobStore 未定义）。
- [ ] **Step 3：实现**：`Put(bytes)` = sha256 hex → 若 `Has` 直接返回；否则 temp+rename 写 `blobs/<sha256>`（裸 sha，无扩展名）。`Read`/`Has` 直读。rename 撞已存在目标视为成功。
- [ ] **Step 4**：`go test ... -run PutIdempotent -v` → PASS。
- [ ] **Step 5**：commit `feat(asset): content-addressed blob store`。

### Task 0.3：记录库 store（preload + CRUD + 锁 + 坏文件容错）
**Files**：Create `internal/services/asset/store.go`；Test `store_test.go`
**先读**：`internal/services/template/store.go:28-107`（preload 容错跳坏文件 + warning 的模式）。

- [ ] **Step 1：失败测试**（preload + Get + 坏文件跳过）
```go
func TestStore_PreloadSkipsCorrupt(t *testing.T) {
	dir := t.TempDir()
	recs := filepath.Join(dir, "records"); os.MkdirAll(recs, 0o755)
	os.WriteFile(filepath.Join(recs, "good.json"), mustJSON(AssetRecord{GUID: "good", Kind: KindTemplate, Name: "x"}), 0o644)
	os.WriteFile(filepath.Join(recs, "bad.json"), []byte("{not json"), 0o644)
	s, err := NewStore(dir); if err != nil { t.Fatal(err) } // 坏文件不 fail 启动
	if _, ok := s.Get("good"); !ok { t.Fatal("good record missing") }
	if _, ok := s.Get("bad"); ok { t.Fatal("corrupt record must be skipped") }
}
```
- [ ] **Step 2**：`go test ... -run Preload` → FAIL。
- [ ] **Step 3：实现** `Store{mu sync.RWMutex; root string; recs map[string]AssetRecord}`：`NewStore` 建 `records/`+`blobs/` 目录、preload `records/*.json`（坏的 `fmt.Fprintf(os.Stderr, ...)` + 跳过）；`Get/List/PutRecord/PutRecordMeta/DeleteRecord` 锁内操作，记录写盘走 temp+rename。
- [ ] **Step 4**：PASS。
- [ ] **Step 5**：commit `feat(asset): record store with fault-tolerant preload`。

### Task 0.4：变体级写（堵并发 lost-update）
**Files**：Modify `store.go`；Test `store_test.go`
- [ ] **Step 1：失败测试**（同记录两分辨率并发不丢）
```go
func TestStore_PutVariantConcurrentNoLostUpdate(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	s.PutRecord(AssetRecord{GUID: "g", Kind: KindTemplate, Name: "n"})
	var wg sync.WaitGroup
	for _, res := range [][2]int{{1920, 1080}, {1024, 768}} {
		wg.Add(1); res := res
		go func() { defer wg.Done(); s.PutVariant("g", res, "blob-"+strconv.Itoa(res[0]), [4]int{0, 0, 1, 1}, nil) }()
	}
	wg.Wait()
	r, _ := s.Get("g")
	if len(r.Variants) != 2 { t.Fatalf("expected 2 variants, got %d (lost update)", len(r.Variants)) }
}
```
- [ ] **Step 2**：FAIL。
- [ ] **Step 3：实现** `PutVariant(guid, res, blobSha, bbox, regions)`：**锁内**读记录 → 按 `res` upsert 单条 Variant（同 res 覆盖、否则追加）→ 写回。`RemoveVariant(guid, res)` 同理。
- [ ] **Step 4**：`go test ... -run PutVariantConcurrent -race -v` → PASS。
- [ ] **Step 5**：commit `feat(asset): variant-level upsert (lock-guarded)`。

---

## Phase 1 — PickVariant + GC

### Task 1.1：PickVariant（PickBest 算法逐字上移）
**Files**：Modify `store.go`；Test `store_test.go`
**先读**：`internal/services/template/store.go:256-291`（PickBest：精确命中优先，否则长边比最近）。**逐字搬该算法**，只把数据源从 `s.vars[key]` 换成 `record.Variants`。

- [ ] **Step 1：失败测试**
```go
func TestStore_PickVariant(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	s.PutRecord(AssetRecord{GUID: "g", Kind: KindTemplate, Name: "n"})
	s.PutVariant("g", [2]int{1920, 1080}, "b1080", [4]int{0,0,1,1}, nil)
	s.PutVariant("g", [2]int{1280, 720}, "b720", [4]int{0,0,1,1}, nil)
	v, ok := s.PickVariant("g", 1920, 1080); if !ok || v.Blob != "b1080" { t.Fatal("exact hit failed") }
	v2, ok2 := s.PickVariant("g", 2560, 1440); if !ok2 { t.Fatal("fallback should find closest") } // 长边比最近
	_ = v2
	if _, ok3 := s.PickVariant("missing", 1920, 1080); ok3 { t.Fatal("missing guid must be ok=false") }
}
```
- [ ] **Step 2**：FAIL → **Step 3**：实现（搬算法）→ **Step 4**：PASS → **Step 5**：commit `feat(asset): PickVariant (ported PickBest)`。

### Task 1.2：GC + ScanReferrers
**Files**：Create `internal/services/asset/gc.go`；Test `gc_test.go`
**说明**：`ScanReferrers` 复用 `internal/services/container/dependency` BFS，但 asset 包不能直接依赖 dependency（避免环）→ **注入**一个 `func() []Referrer` 闭包（caller 在 wire 层用 dependency 扫全部容器+子图）。GC 同理注入 live-guid 集来源。

- [ ] **Step 1：失败测试**（GC 只回收无记录引用的 blob）
```go
func TestGCBlobs_ReclaimsOrphans(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	live, _ := s.Blobs().Put([]byte("live"))
	orphan, _ := s.Blobs().Put([]byte("orphan"))
	s.PutRecord(AssetRecord{GUID: "g", Kind: KindTemplate, Name: "n",
		Variants: []Variant{{Resolution: [2]int{1, 1}, Blob: live}}})
	n, _ := s.GCBlobs()
	if n != 1 { t.Fatalf("expected 1 reclaimed, got %d", n) }
	if s.Blobs().Has(orphan) { t.Fatal("orphan blob not reclaimed") }
	if !s.Blobs().Has(live) { t.Fatal("live blob wrongly reclaimed") }
}
```
- [ ] **Step 2**：FAIL → **Step 3**：实现 `GCBlobs()`：live set = 所有记录 `Variants[].Blob` ∪ clip `Blob`；扫 `blobs/` 删不在 live set 的，返回回收数。`ScanReferrers(guid, walkGraphs func)` 返回 `[]Referrer{ContainerID, SubgraphID, NodeID, NodeKind}` → **Step 4**：PASS → **Step 5**：commit `feat(asset): blob GC + referrer scan`。

---

## Phase 2 — 运行时匹配接全局 asset store

**先读**：`wire_container.go:202-326`（`templateMatcherAdapter`：`storeFor`/`Detect`/`loadVariantTemplate`/缓存）。整段把 per-container `template.Store` 换成单个全局 `asset.Store`。

### Task 2.1：Detect 改按 GUID + 全局 store + 缓存按 blob sha
**Files**：Modify `wire_container.go`；Test `wire_container_test.go`（若无则新建最小匹配测试）

- [ ] **Step 1：失败测试**：构造 asset.Store 塞一个 template 记录 + blob（用一张已知小 PNG fixture），`Detect(frame, guid)` 命中返回 true、坐标合理；未知 guid 返回 ok-miss 不崩。
- [ ] **Step 2**：FAIL。
- [ ] **Step 3：实现**：
  - `templateMatcherAdapter` 去掉 `storeFor(containerID)` 与 `m.stores` map，改持单个 `*asset.Store`。
  - `Detect(_ context.Context, frame *image.RGBA, guid string, threshold, region)` —— **删 `containerID` 参数**。`store.PickVariant(guid, W, H)` → 拿 `variant.Blob` → 解码缓存（key=`variant.Blob`，value=未缩放 `*vision.Template`）→ 现有 scaleTolerance/缩放/ROI/多槽逻辑**原样保留**。
  - `loadVariantTemplate` 改 `loadDecodedTemplate(blobSha)`：缓存 `map[string]*vision.Template` 键=blob sha；miss 时 `store.Blobs().Read(sha)` → `vision.DecodeTemplate`（用现解码路径）。
  - `emitMissingVariantWarning`/`emitScaleTooFarWarning` 参数从 `(containerID,key,...)` 改 `(guid,...)`，文案适配。
- [ ] **Step 4**：`go test ./... ` 该测试 PASS；`go build ./...` 绿。
- [ ] **Step 5**：commit `refactor(matcher): detect by GUID via global asset store`。

### Task 2.2：VisionAdapter 透传 GUID
**Files**：Modify `internal/services/container/runtime/node_services.go:483-574`
**先读**该段：`VisionAdapter.Match/WaitMatch` → `a.rt.Matcher.Detect(ctx, a.containerID(), frame, key, ...)`。
- [ ] **Step 1**：改调用点删 `a.containerID()` 实参（Detect 签名已变），`key`→`guid`（变量名改 guid，值透传不预处理）。
- [ ] **Step 2**：`go build ./...` 绿；现有 runtime 测试 `go test ./internal/services/container/runtime/...` 绿。
- [ ] **Step 3**：commit `refactor(runtime): vision adapter passes GUID`。

---

## Phase 3 — 节点 semantic + validator + 依赖

### Task 3.1：节点 semantic 改 TemplateGUID
**Files**：Modify `internal/nodes/detect/check_template.go`、`click_template.go`、`wait_template.go`
**先读**三个文件的 `Semantic: "TemplateKey"` 行。
- [ ] **Step 1**：三处 `Semantic: "TemplateKey"` → `"TemplateGUID"`（widget `template-picker` 不动，pin 名 `Templates` 不动，StringList 形态不动）。
- [ ] **Step 2**：`go build ./...` 绿 → **Step 3**：commit `refactor(nodes): template pin semantic → TemplateGUID`。

### Task 3.2：删格式校验，依赖抽取改 GUID
**Files**：Delete `internal/services/container/validator_template_key.go` + `internal/services/template/keyvalidation.go`（随 template 包删，见 Phase 6）；Modify `internal/nodes/detect/template_common.go`
**先读**：`template_common.go:13-38`（`validateTemplateKeys` + `templateDeps`）。
- [ ] **Step 1**：删 `validateTemplateKeys` 的 namespace.name 格式校验（GUID 合法性=存在性，归 validator_deps）；三个节点的 `Validate()` 里对 `validateTemplateKeys` 的调用删掉（或改成空校验）。`templateDeps(keys)` 逻辑不变（`Dependency{Kind:"template", Key:guid}`，Key 现在是 GUID）。
- [ ] **Step 2**：删 `validator_template_key.go`（其 `ValidateKey` 调用点已无）。
- [ ] **Step 3**：`go build ./...` + `go test ./internal/nodes/...` 绿 → **Step 4**：commit `refactor(validator): drop template key format check`。

### Task 3.3：validator_deps 接 asset 存在性
**Files**：Modify `internal/services/container/validator_deps.go`
**先读**：`validator_deps.go:19-54`（`hasTemplate(key)`/`hasClip(id)` 现怎么注入/查）。
- [ ] **Step 1**：`hasTemplate`/`hasClip` 的实现改成查全局 `asset.Store`（`Get(guid)` 存在且 kind 匹配）。注入点从 per-container template store 改成全局 asset store（wire 层传）。
- [ ] **Step 2**：`go test ./internal/services/container/...` 绿 → **Step 3**：commit `refactor(validator): asset existence via global store`。

---

## Phase 4 — clip 并入 asset clip kind

**先读**：`internal/services/inputclip/model.go`（`InputClip{ID,Label,Events}`）、`wire_inputclip.go`（`_clips/` + `library/clips` 接线）、`internal/nodes/io/play_clip.go:42-50`（`ctx.Clip().Play(ctx, clipID)` → Fail 出口）。

### Task 4.1：clip 存储改 asset 记录 + blob
- [ ] **Step 1**：clip 序列化字节（现 Events 的 json/csbf 格式**不动**）→ `blob.Put` 得 sha → 写 `AssetRecord{Kind:KindClip, GUID, Name:label, Blob:sha}`。`ClipResolver`（`ctx.Clip()` 背后）读取改为：按 guid `Get` 记录 → `Blobs().Read(record.Blob)` → 现有反序列化回放。
- [ ] **Step 2**：`PlayClip` 的 `ClipID` pin 值=guid（semantic `ClipID` 保留；`Dependencies` 返 `{Kind:"clip", Key:guid}` 不变）。clip 缺失运行期：`Play` 返错 → 现 `Failf(CodePlaybackFailed)` → Fail 出口（**不动，已是目标行为**）。
- [ ] **Step 3**：clip 相关单测绿；`go build ./...` 绿 → commit `refactor(clip): store as asset clip kind`。

---

## Phase 5 — 分享/导入坍缩

**先读**：`internal/services/container/library/copy.go`（ExportFromContainer / ImportToContainer + ImportConflict/strategy）、`store.go`（SubgraphPackage）、`dependency/scanner.go`（ScanSubgraphDependencies BFS）。

### Task 5.1：bundle 打包（GUID 闭包 + blob）
**Files**：Create `library/bundle.go`；rewrite `library/copy.go` export 段；Modify `library/store.go`（`SubgraphPackage.Templates/Clips []string` → `Assets []string` GUID 列表）
- [ ] **Step 1：失败测试**：导出含 1 模板节点的子图 → bundle 目录含 `graphs/`、`assets/<guid>.json`、`blobs/<sha256>`；record 引用的 blob 缺失 → 报错中止（不生成半包）。
- [ ] **Step 2**：FAIL → **Step 3**：实现 export：
  - 闭包根：**子图导出**= `ScanSubgraphDependencies(sgID)`；**容器导出**= 顶层图 BFS（递归 Subgraph-call 节点）。统一收 `Dependency{Kind, Key=guid}`。
  - 对每个资产 guid：写 `assets/<guid>.json`（从全局 asset.Store `Get`）+ 其 `Variants[].Blob`/clip `Blob` 的字节到 `blobs/<sha256>`。校验缺失 → error。
- [ ] **Step 4**：PASS → **Step 5**：commit `feat(library): GUID-closure bundle export`。

### Task 5.2：导入幂等合并（加性变体合并 + 提交点=图）
**Files**：rewrite `library/copy.go` import 段（删 ImportConflict/strategy/conflict 检测）
- [ ] **Step 1：失败测试**：
  - 导入新 bundle → 资产记录+blob+图落地。
  - 重复导入同 bundle → 幂等（无重复 blob、记录跳过、图重写）。
  - 本地已有 GUID@1080，bundle 含同 GUID 新增 720 档 → 导入后记录有 2 档（**加性合并**）；本地 1080 不被覆盖。
- [ ] **Step 2**：FAIL → **Step 3**：实现 import：
  - blob 按 sha `Put`（去重）；record：GUID 不存在→整写；已存在→**按 resolution 加性合并变体**（本地缺的 res 加入；本地已有 res 保留本地），记录级 name/tags 保留本地。
  - 图写入为**提交点**；保留 `computeMissingGlobals`（RequiredGlobals diff，正交，逐字不变）。删 strategy/ImportConflict 类型 + 相关分支。
- [ ] **Step 4**：PASS；`go test ./internal/services/container/library/...` 绿 → **Step 5**：commit `feat(library): idempotent GUID/sha merge import`。

### Task 5.3：ExportContainer RPC
**Files**：Modify `library/service.go`
- [ ] **Step 1**：加 `ExportContainer(containerID, overwrite)` 走 Task 5.1 容器闭包路径（与 `ExportSubgraph` 共用打包）。
- [ ] **Step 2**：`go build ./...` 绿；test 绿 → **Step 3**：commit `feat(library): container-level export`。

---

## Phase 6 — asset RPC 服务 + 删旧 template 包

### Task 6.1：asset.Service（RPC 面）
**Files**：Create `internal/services/asset/service.go`；Test `service_test.go`
**先读**：`internal/services/template/service.go`（全签名，作为对照；新服务**去掉 containerID**）。
- [ ] **Step 1：失败测试**：`SaveTemplateCapture(dataURL,name,recRes,region)` → 建 GUID 记录+变体+blob，返回 GUID；`List()` 返全局 `[]AssetSummary`（guid/kind/name/tags/variantCount/firstBlob）；`Rename(guid,name)`；`Delete(guid)` 返 referrer 列表。
- [ ] **Step 2**：FAIL → **Step 3**：实现这些方法（dataURL 解码沿用旧 Service 逻辑；region→bbox 换算逐字搬 `template/service.go:92-98`）。`Delete` 先 `ScanReferrers` 返回引用列表（不阻断，调用方决定）。
- [ ] **Step 4**：PASS → **Step 5**：commit `feat(asset): RPC service`。

### Task 6.2：删 template 包
**Files**：Delete `internal/services/template/`（整目录）
- [ ] **Step 1**：grep 全仓 `services/template`、`template.NewService`、`template.NewStore`、`template.Store` 引用点，全部已被 asset 取代后删包。
- [ ] **Step 2**：`go build ./...` 绿（没有残留引用）→ **Step 3**：commit `refactor: remove template package (absorbed into asset)`。

---

## Phase 7 — 前端 picker / 捕获 / store

**先读**：`frontend/src/stores/templates.ts`、`components/containers/TemplatePicker.vue`、`components/templates/TemplateCapture.vue`、`lib/backend.ts`（wails binding 名）。

### Task 7.1：前端资产 store 改 GUID
**Files**：Modify `frontend/src/stores/templates.ts`（或新建 `stores/assets.ts`）
- [ ] **Step 1**：`map<string, TemplateMeta>` 键 `key`→`guid`；`list()` 调 `backend.asset.list()`（全局，无 containerID）；`save` 调 `backend.asset.saveTemplateCapture(...)` 返回 guid。
- [ ] **Step 2**：`cd frontend && pnpm type-check`（或 `vue-tsc`）绿 → **Step 3**：commit `refactor(fe): asset store keyed by GUID`。

### Task 7.2：picker + 捕获对话框
**Files**：Modify `TemplatePicker.vue`、`TemplateCapture.vue`
- [ ] **Step 1**：picker `v-for` 遍历 guid、显示 `name`+缩略图、`toggle(guid)`；搜索按 name/tags。
- [ ] **Step 2**：`TemplateCapture` 改为截图 → **单个命名对话框**填 name（GUID 后台分配，用户不可见）→ `save`；name 非阻断"已存在"提示（不强制唯一）。
- [ ] **Step 3**：`pnpm type-check` 绿；离屏渲染自检见 [../checklists/headless-ui-verify.md](../checklists/headless-ui-verify.md) → **Step 4**：commit `feat(fe): GUID picker + capture-then-name`。

---

## Phase 8 — 接线 + 全量构建 + smoke

### Task 8.1：main.go / wire 接线
**Files**：Modify `main.go`、`wire_container.go`、`wire_inputclip.go`
**先读**：`main.go:180-220`（templateSvc/templateMatcher/SetOnChange 接线）。
- [ ] **Step 1**：建单个全局 `asset.Store`（root=`<dataDir>/assets`）；注入 matcher（Task 2.1）、validator_deps（Task 3.3）、library bundle（Phase 5）、asset.Service（Task 6.1）、clip resolver（Phase 4）。删 `template.NewService`/`templateMatcher` 旧接线、`wire_inputclip` 的 `_clips/`+`library/clips` 旧路径。`SetOnChange` → matcher 解码缓存失效（按 guid 或全清）。
- [ ] **Step 2**：Wails bindings 重新生成（`wails generate module` 或 build 自动），前端 `backend.asset.*` 可用。
- [ ] **Step 3**：`go build ./...` 绿 → commit `refactor: wire global asset store, remove old template/clip wiring`。

### Task 8.2：全量测试 + 真机 smoke
- [ ] **Step 1**：`go test ./...` 全绿；`go vet ./...` 干净。
- [ ] **Step 2**：按 [../checklists/build.md](../checklists/build.md) 出 production artifact（task build / wails build），起 app。
- [ ] **Step 3：真机 smoke**（spec §13）：① 截两模板各得 GUID、同图截两次 blob 只一份；② 节点引用 GUID 跨分辨率命中；③ 重拍同 GUID 换图、所有引用自动用新图；④ 导出子图+导出整容器→bundle 带 assets/blobs→导入另一容器幂等无冲突弹窗（除 MissingGlobals）；⑤ 删引用共享模板的子图→资产仍在、其他引用不受影响；⑥ 库里删资产→弹"被 N 处引用"→确认删→GC 回收孤儿 blob。
- [ ] **Step 4**：smoke 全过 → commit `test: asset subsystem smoke green`。任一不过 → 回对应 Phase 修，不往下走。

---

## Self-review（写完本 plan 的自检）

- **spec 覆盖**：§3 数据模型→P0；§5 匹配→P2；§4 节点/§校验/依赖→P3；clip §7/§9→P4；分享 §7→P5；RPC §9→P6；前端 §8→P7；GC/生命周期 §6→P1.2+P8；迁移=零（无 task，符合 spec §2）。✅ 全覆盖。
- **变体 upsert（spec §3 修正点）**→ Task 0.4 `PutVariant` 锁内按 res upsert；导入加性合并→ Task 5.2。✅
- **类型一致**：`PutVariant`/`PickVariant`/`AssetRecord`/`Variant`/`KindTemplate` 全 plan 同名同签名。✅
- **YAGNI**：记录级 GC、反向索引、asset.repair、bundle 防篡改、picker 分页均不在 task 内（spec §14 backlog）。✅
