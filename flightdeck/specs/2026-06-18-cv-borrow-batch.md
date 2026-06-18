---
status: active
summary: 按键精灵 CV 函数表借鉴的 3 个纯 Go 节点: FindColorSignature 颜色签名 / FindTemplateAll 模板全部命中 / DecodeQR 二维码解码; 零新增原生依赖。
last_updated: 2026-06-18
---

# CV 低依赖借鉴批 (颜色签名 / 模板全部命中 / QR 解码)

## 背景

来源: 通读按键精灵 (anjian) YOLO/CV 函数表后的对照 (见 [cv-perception-pool §5](cv-perception-pool.md))。
本批挑出**三个纯 Go、零新增原生依赖**的借鉴点 —— 跟 ONNX/YOLO 那条"引原生 .dll"的大路线区分开,
先把低风险、覆盖面广的 building block 补齐。**无具体驱动游戏场景**, 按用户「设计新节点三问」
(通用 / 性能 / 功能) 做通用框架能力, 参数暴露成 config 不锁死。

三个节点:
1. **`FindColorSignature`** 颜色签名 — 借 anjian `FindMultiColor` / `CmpColorEx`
2. **`FindTemplateAll`** 模板全部命中 — 借 anjian `OpenCV.FindImgAll`
3. **`DecodeQR`** 二维码解码 — 借 anjian `Image.QrDecode`

> **本稿已纳入 3 份外部 AI 评审两轮 (ds/gpt/claude, 2026-06-18)。** 评审不了解项目全貌, 已逐条对源码核实;
> 采纳点并入下文, 驳回点见末尾「[评审裁决摘要](#评审裁决摘要)」附理由。

## 贯穿约束 (三个节点共守)

- **纯 Go, 零新增原生依赖**。守住「除 `capture_wgc.dll` 外不引原生库」的 invariant (go.mod 现仅
  `golang.org/x/image`)。QR 仅引一个纯 Go 模块 (落地前 gate 验**传递依赖**无 CGO/原生 + 许可证, 见节点 3)。
- **集成点统一走 `VisionService`** (`internal/node/interfaces.go`) → 实现 `visionAdapter`
  (`internal/services/container/runtime/node_services.go`) → stub `stubVisionService`
  (`internal/node/services.go`)。**二号铁律: 三处一起改, 不留 shim**。
- **新增类型** (`ColorSignature`/`ColorPoint`/`TemplateMatch`/`QRResult`) 一律放 **`internal/node`** (跟
  `Point`/`BlobEntry`/`HSVRange` 同处); VisionService 返回与节点输出**共用同一结构体**, 不在接口层/数据层各立一份。
- 节点都在 `internal/nodes/detect/`: `Category:"Detect"` · `NeedsWindow:true` · 吃 `ROI Geometry`
  (零值=全帧, 经 `ResolveGeometry`) · **一次 `Run` 一张帧快照** (adapter 抓**一次** `CaptureFrameCached` 传给
  本次所有匹配, 不靠 100ms 缓存兜)。
- **坐标系统一**: 所有输出点/框一律 **全帧归一化 0..1** (对齐 `BlobEntry` + 现有模板 `Point`)。内部用 ROI 相对像素,
  adapter 统一换算。
- 产出走 **Spec C 的 `config.capture` 模型** (非旧捕获框): 节点**只在 exec 出口声明 `OutputSpec.Data` 字段 + `Run()` 里
  `.Set(field, 值)`**, **不加 `Capture<字段>` 输入框、不调 `node.Capture`** (这套 `Semantic:"capture"` + `node.Capture`
  已被 Spec C T4 `33fa43f` 整体删除)。捕获由框架做: 用户在 Inspector 把 Data 字段绑到变量 (`node.config.capture`
  map[字段]→变量), fire 时 `dispatch_v5.applyCaptures` 自动写 (scope auto); **只写该 exit 实际带的字段** (`NotFound`
  只带 `Count` → `Point` 那轮框架不写、变量留旧值, 稀疏 data 天然保证)。可绑字段 = `BindableFields` 从 Data 派生,
  `validator_capture_refs.go` 校验。范式见 `detect_color_blobs.go`。机械步骤照
  [add-node.md](../checklists/add-node.md), pin 命名照 [node-spec-style.md](../checklists/node-spec-style.md)。
- **错误 / 校验分工** (评审消歧): **结构性非法** (坏 JSON / 锚点 dx,dy≠0 / 点数超限 / 负 tol) → `Validate` **硬报错**;
  **数值越界** (`Threshold`/`Tolerance`/`MaxResults`/`MinDistance`) → `Run` 入口 **clamp 不报错** (同 `DetectColorBlobs`
  clamp pollMs; 故 `Threshold=2` → 变 1, 不报错)。`Validate` **预解析 Signature** 做上述结构检查 (与 `Run` 同一解析
  规则), `Run` 仍二次 parse 兜底 (同 `DetectColor.Run` `parseRange6`) —— **不出现 "Validate 过但 Run 报错" 割裂**。
- **参数边界**: `Threshold`→[0,1]; `Tolerance`→[0,255]; `MinDistance`<0→0(=自动); `MaxResults`<0→0。
  **`MaxResults=0` = 不限 (全返回), 不是 "取 0 条"**; 要截断填 ≥1 (评审消歧, 防实现者把 0 当空列表)。

---

## 节点 1 — `FindColorSignature` 颜色签名

> **命名**: 评审指出 `FindColorMulti` 歧义。本节点语义 = anjian `FindMultiColor`: 在区域内搜索**一个**满足
> 「锚点色 + N 偏移点色」组合的位置, **返回单个锚点坐标**。故定名 `FindColorSignature`。

**借鉴**: anjian `FindMultiColor` (区域搜索多点颜色组合) + `CmpColorEx` (定点多色比对的概念)。
**填补**: 现有 `detect_color`/`hsv`/`blobs`/`roi_color_scan` 都是「单颜色范围 + 区域统计」, 没有「锚点 + N 偏移点
的颜色签名」这种**比单点稳、比模板便宜**的中间档原语。

### 颜色签名模型 (已冻结)

```jsonc
// Signature (JSON): 点列表, 首项=锚点 (dx=dy=0, 必填且必须为 0, 否则 Validate 报错)。点数 1..64。
[
  {"dx": 0,  "dy": 0,  "r": 200, "g": 30,  "b": 30          },  // 锚点 (无 tol → 用节点默认)
  {"dx": 12, "dy": -4, "r": 255, "g": 255, "b": 255, "tol": 8}  // 偏移点 (显式 tol=8)
]
```
- **每点匹配**: 像素 RGB 与目标 RGB 的**逐通道绝对差都 ≤ tol** (tol 单位 = 灰度值 0–255)。**只比 RGB, 忽略 Alpha** (屏幕帧 alpha 恒不透明)。
- **tol 取值** (消歧, 评审采纳): 解析成 `Tol *int` **纯 nullable** —— JSON **缺省/null** → 用节点级 `Tolerance`
  默认; **给了值** (含 `0`=严格) → 显式。**不用 `<0` 哨兵** (避免 nullable+sentinel 双语义); 负值 → `Validate` 报错。
- **点数** ≥ 1 (只有锚点 = 单点定色检查), ≤ 64 (经验上界, 防极端签名退化成大规模随机访存; 无场景需要更多, 真有再调); 空列表/超限 → `Validate` 报错。
- **颜色模型说明**: 本节点用「目标 RGB + 逐通道容差」, **有意区别于** `DetectColor` 的「HSV/RGB min-max 区间」
  (签名是定点精确色, 不是区域统色); widget 提示, 避免用区间直觉套。**Signature 走 `json` widget** (同 blobs `Range`,
  无逐项 schema 约束), 锚点/点数规则是 Validate-时检查 + placeholder 文案提示。

### Spec

- **Inputs**: `In`(Exec) · `ROI`(Geometry, **仅锚点搜索范围; 偏移点采样整帧不受此限**, 零值=全帧) ·
  `Signature`(JSON, widget=`json`) · `Tolerance`(Number, 默认 `16`, 单位=每通道灰度绝对差 0–255, [0,255])
- **Outputs**: `Found`(Exec, Data: `Point` 命中锚点全帧归一化坐标) · `NotFound`(Exec)
- **可捕获产出**: `Point` (Data 字段自动可绑变量, 走 config.capture; 节点无捕获框)

### 行为 / 性能 (设计三问②)

- **ROI 只界定锚点搜索范围** (半开区间 `[x0,x1)×[y0,y1)`, 标准约定, 避免右下边界实现分歧)。偏移点按
  (锚点px + dx, dy) **采样整帧** (可超 ROI)。
- **偏移点落帧外 → 该点 miss → 整签名在该锚点失败, 继续搜索** —— **有意选严格** (签名点离屏 = 图案没完整可见 =
  非真命中), 非唯一答案也非 bug, 记入[非目标](#非目标--yagni)防后人当 bug 改。
- **先锚点色过滤候选**: 仅在 ROI 内锚点色命中像素上验偏移点, O(区域像素 × N) → O(锚点命中像素 × N)。
- 扫描序**固定行主序** (`image.RGBA.Pix` offset, 正确处理 ROI 非零起点的 Stride), 返回**第一个**完整命中 (确定性)。

### VisionService 扩展

```go
// FindColorSignature 在 roi (锚点搜索区) 找签名首个完整命中, 偏移点采样整帧。返回锚点全帧归一化坐标。
FindColorSignature(roi Geometry, sig ColorSignature, defaultTol int) (found bool, pt Point, err error)
// ColorSignature{ Points []ColorPoint }; ColorPoint{ DX,DY int; R,G,B int; Tol *int }
```

---

## 节点 2 — `FindTemplateAll` 模板全部命中

**借鉴**: anjian `OpenCV.FindImgAll` (对应 `FindImgBest` 单个)。
**填补**: 现有模板族 (`CheckTemplate`/`ClickTemplate`/`WaitTemplate`) 经 `Match`/`WaitMatch` 只返回**单个最佳点**
(`pkg/vision.Match` 全局 argmax)。无法回答「画面里有几个匹配 / 都在哪」。

### Spec (形态对齐 `DetectColorBlobs`)

- **Inputs**: `In`(Exec) · `Templates`(模板 GUID 列表, 沿用模板族多模板字段 + `templateDeps`) · `ROI`(Geometry, 零值=全帧) ·
  `Threshold`(Number, 默认 `0.85`, [0,1]) · `MaxResults`(Number, 默认 `0`=不限) · `MinDistance`(Number, 默认 `0`=自动, 像素)
- **Outputs**:
  - `Found`(Exec, Data: `Matches`(JSON `[]TemplateMatch`, 每条含 `point/conf/bbox/templateKey`) · `Count`(Number, NMS 后总数, 见[行为](#行为-1)) · `PrimaryPoint`(Point) · `PrimaryConf`(Number))
  - `NotFound`(Exec, Data: `Count`(Number, =0))
- **可捕获产出**: `Matches`/`Count`/`PrimaryPoint`/`PrimaryConf` (Data 字段自动可绑, config.capture; 绑 `Matches` 大列表建议设 `MaxResults` 限体积)

### TemplateMatch 类型 (已冻结)

```go
// 放 internal/node; VisionService.MatchAll 返回与节点输出共用。
type TemplateMatch struct {
    Point       Point      `json:"point"`       // 命中实例中心, 全帧归一化 0..1 (= 现有模板 Point 语义, wire_container.go:285)
    Conf        float64    `json:"conf"`
    BBox        [4]float64 `json:"bbox"`        // 命中实例外接框 [x,y(左上), w, h], 全帧归一化; Point = BBox 中心
    TemplateKey string     `json:"templateKey"` // 命中来自哪个模板 GUID (多模板区分)
}
```
> **BBox 语义分歧 (有意)**: 现有单模板 `Detect` 的 `[4]float64` 槽返回的是 **ROI 矩形** (`wire_container.go:287`)。
> 多命中下 ROI 都一样、无意义, 故 `FindTemplateAll` 改返**每个实例的外接框**。`Point` 由同一命中算出 (BBox 中心),
> 两者不会不一致。

### 行为

- 单帧匹配, 收集 ROI 内**所有 conf ≥ Threshold 的局部极大值**, 做 **NMS**, 结果按 **conf 降序** (并列按 `(y,x)` 定序)。
  - **局部极大定义 (已冻结)**: 候选 = `conf ≥ Threshold` **且** `conf ≥ 全部 8 邻域` (3×3); plateau (邻域相等) →
    取 `(y,x)` 最小那个, 防同峰多收。**边缘像素** (邻域不全) → 只跟**存在的**邻居比 (缩减邻域, 不漏 ROI 边命中)。
    这步把"逐像素都过阈值"的稠密候选先收敛, 再交 NMS。
  - **NMS 规格 (已冻结)**: 按 conf 降序遍历 (并列 `(y,x)`, 两层 tie-break 全确定), 保留 A、抑制低分 B 当
    `dist(centerA, centerB) < min(radiusA, radiusB)` (取 `<`: dist 恰=radius 时**保留** B)。`MinDistance>0` →
    `radius = MinDistance` (全局); `MinDistance=0`(自动) → 每命中 `radius = 自己模板 min(tplW,tplH)/2`。
- **多模板**: 各模板各自 `MatchAll` (同一帧快照) → 合并 → **统一 NMS**。同区域跨模板互相抑制 = **有意 dedup**;「按模板
  分组各自 NMS」属 YAGNI。
- `Count` = **NMS 后总数, 不受 `MaxResults` 截断** (列表才截断)。**与 `DetectColorBlobs.BlobCount` 一致**
  (`detect_color_blobs.go:154` 即此语义)。⚠ 故可能 `Count > len(Matches)` (如 Count=12/MaxResults=5), widget tooltip 需注明。
  **例外**: 候选触 4096 上限那条病态路径 (低阈值) 下 Count 是截断集近似, 见[底层改动 §1](#底层改动-本批唯一动-matcher-核心), 已 log。
- `PrimaryPoint`/`PrimaryConf` = NMS 后 conf 最高 = `Matches[0]` (conf 降序), **恒在列表内** (MaxResults≥1 或 =0 不限时不被截)。
- **路由**: `Count ≥ 1` → `Found`; 无峰 ≥ Threshold → `NotFound`(Count=0)。

### 底层改动 (本批唯一动 matcher 核心)

1. **`pkg/vision.MatchAll(img, iw, ih, tpl, parallel, threshold) []Match`** —— `Match` 的变体: 同样 CCOEFF_NORMED
   全扫 (积分图 + 行并行), 但收集 **3×3 局部极大 ≥ threshold** 的候选 (而非全局 argmax)。**不收 `scaleTolerance`**
   (见下, 它在 adapter 消费)。
   - **性能 (评审纠正, 已坐实)**: 生产单模板检测 (`templateMatcherAdapter.Detect` → `vision.Match` 在 ROI 内,
     `wire_container.go:281`) 早就付**完整 Match** 代价 (全屏 1080p 大模板 **3–5s/次**, `wire_container.go:260` 注释)。
     `MatchAll` 同 ROI 下**同量级**, 边际只是"收集峰 + NMS"(可忽略)。**成本驱动 = ROI 大小, 与现有模板节点一致** ——
     建议同款收窄 ROI。
   - **候选上界 (评审采纳)**: 经 3×3 局部极大预筛后候选已大幅收敛; 仍 >4096 (病态低阈值) → 按 **conf 降序 (并列 `(y,x)`)**
     保前 4096 + `log` 截断告知 (不静默)。此时 `Count` = 截断集 NMS 后数 (best-effort, 已 log, 文档注明可能失真)。单测覆盖此路径。
2. **`TemplateMatcher` 接口加 `DetectAll`** (`internal/services/container/runtime/interfaces.go`):
   ```go
   DetectAll(ctx, frame *image.RGBA, guid string, th float64, region []float64, scaleTolerance float64) ([]TemplateMatch, error)
   ```
   - **`scaleTolerance` 在 adapter `DetectAll` 消费, 不进 `MatchAll`** (评审追问已核实): 镜像 `Detect`
     (`wire_container.go:228-245`) —— `PickVariant`(选最近分辨率档) → `longEdgeScale` + `withinScaleTolerance` 门
     (超容差判太远 → miss) → `ScaleTemplate` 缩到帧尺度 → 对**这一个已备模板** `MatchAll`。**非"每变体各 MatchAll 合并"**
     (只 pick 一档)。
   - **多槽变体** (`variant.Regions` 非空, 如货架 grid): 现有 `Detect`/`detectMultiRegion` 取 reading-order **末位单个**;
     `DetectAll` 改**收全部 region hit** —— **每 region 每命中各自换算全帧归一化** (镜像 `wire_container.go:331-332` 的逐
     命中换算, **不只换第一个**), 天然对齐"全部命中"。
   - **4 个实现同步改** (二号铁律, 不留旧签名): `NoopMatcher`(`interfaces.go:31`) · **`templateMatcherAdapter`**
     (`wire_container.go:214`, 生产) · `mockMatcher`(`inspect_phase_test.go:26`) · `stubMatcher`(`node_services_test.go:213`)。
     改完**跑模板族测试** (`check/click/wait_template_test.go` + runtime) 确认无回归。
3. **`VisionService` 加 `MatchAll`** + `visionAdapter` 实现: 调**一次** `CaptureFrameCached` (同 `DetectColor` 取帧路径,
   `node_services.go` ~480) 拿 `*image.RGBA`, 把**同一帧**传给每个模板的 `DetectAll` (多模板循环不重复抓帧), 逐命中
   ROI 相对像素 → 全帧归一化。

---

## 节点 3 — `DecodeQR` 二维码解码

**借鉴**: anjian `Image.QrDecode`。**填补**: 全新能力。屏幕 QR (扫码登录 / 分享码 / 活动码) → 读出内容做后续判定。

### Spec

- **Inputs**: `In`(Exec) · `ROI`(Geometry, 零值=全帧)
- **Outputs**:
  - `Found`(Exec, Data: `Text`(String, 首个 QR 内容) · `Count`(Number, **成功解码出文本的 QR 数** = `len([]QRResult)`) ·
    `Points`(JSON, 首个 QR 的**解码器定位点**数组, 全帧归一化; **点数/顺序以库为准, 非四角**; goqr 回退时可能为空 —— `Text` 才是稳定契约))
  - `NotFound`(Exec, Data: `Count`(Number, =0))
- **可捕获产出**: `Text`/`Count`/`Points` (Data 字段自动可绑, config.capture; 节点无捕获框)

### 行为 / 实现

- 抓帧 → 裁 ROI → 纯 Go QR 解码 → 文本。服务返回**全部成功解码**的 `[]QRResult`; 节点 `Text`/`Points` 取**第一个**, `Count` = 成功解码数。
- **"第一个" 排序 (评审消歧)**: 取每个 QR **定位点外接 bbox 的左上角** (`min-x, min-y`), 按 `y` 再 `x` 升序取首
  —— **不用几何中心** (倾斜/透视 QR 的中心 y 会比另一个更靠下 QR 的左上 y 还大, 误排)。文档写死此规则 (非"视觉阅读顺序")。
- **Points 形态 (诚实标注)**: gozxing 返回的是**定位图案点** (`Result.ResultPoints`, QR 通常 3 个 finder + 可选
  alignment, **非严格四角 TL/TR/BR/BL**); `Points` 原样透传其归一化坐标, **点数/顺序以库为准** —— **impl 时核实 gozxing
  QR decoder 实际返回点**, 不在 spec 脑补四角。
- **错误分类 (评审采纳)**: 「没检出 QR」与「检出但解码失败 (模糊/残缺)」**都走 `NotFound`** (当前合并, 文档注明便于日后拆);
  仅帧/库**致命错误**走 Fail。(gozxing 以异常类型区分 `NotFoundException` vs `Format/ChecksumException`, impl 核实;
  本批一律映射 `NotFound` —— 信息可拆但暂不拆。)
- **VisionService 加** `DecodeQR(roi Geometry) ([]QRResult, error)`; `QRResult{ Text string; Points []Point }` (全帧归一化)。
- **依赖 + gate + 回退 (评审采纳)**: `github.com/makiuchi-d/gozxing` (用户 2026-06-18 拍板; 纯 Go, 顺带条码)。落地前 gate:
  `go mod why` + 检查**传递依赖**无 CGO/原生库 + 许可证 (含传递依赖) 兼容。**gate 失败 (查出 CGO) → 回退 `goqr`
  (`github.com/liyue201/goqr`, 纯 Go 仅 QR)**; 此风险列在落地顺序步 2。

---

## 测试策略

- **`pkg/vision.MatchAll` 单测**: 合成多目标灰度图 (同模板平铺 N 份 + 噪声) 验「全找到 + 3×3 极大去稠密 + NMS 不重 +
  conf 降序稳定」; 退化 (单目标 / 全 miss) 对齐 `Match`; **低阈值 (如 0.01) 人造 >4096 候选验截断 + log 不 panic**。
- **`FindColorSignature` 单测**: ROI **非零起点**行主序确定性; 偏移点**越界=miss** (严格); 多候选取**首个**;
  `tol=null=默认` vs `tol=0=严格` 两路; 点数超限/锚点≠0/坏 JSON → Validate 报错; **边界: 恰 64 点过 / 65 点报错 /
  仅锚点(1 点)走 Found-NotFound 非 error**。
- **节点单测** (`*_test.go`): stub Vision 验 Outputs 路由、捕获写值、参数 clamp。参考 `detect_color_blobs_test.go` / `check_template_test.go`。
- **runtime 集成**: 同 `TestDetectColorHSVTimeoutOnNoMatch`, 合成帧跑 execNode 验真实 adapter; `FindTemplateAll` 额外验
  **变体缩放 / 多槽 DetectAll / 模板族无回归**。
- **QR**: 手生成 QR PNG 解码验文本; **多 QR 取最左上 + 同 top-left-y 用 x 决胜的 tie-break**; ROI 裁切边界; Count。
- 已知预存失败按 [build.md](../checklists/build.md) 判红, 不混入本批回归。

## 落地顺序 (按风险递增)

1. **`FindColorSignature`** — 纯 Go 像素扫描, **零新依赖、零核心改动**, 风险最低, 先打通节点端到端模式。
2. **`DecodeQR`** — 逻辑独立, 但引入首个第三方视觉库。⚠ **gate 风险**: 若 gozxing 传递依赖含 CGO → 回退 goqr (见节点 3)。
3. **`FindTemplateAll`** — 改 `TemplateMatcher` 接口 + 全部 4 实现 + `pkg/vision`, 压轴; 改完即跑模板族测试隔离回归。

## 非目标 / YAGNI

- **不含下游「逐命中点迭代」**: 列表产出要「对每个命中点点一下」需要 ForEach 迭代器 —— 跟现有 blobs 列表**同一个**下游缺口,
  不在本批 (另议)。本批只保证产出 count + list + primary。
- **`FindColorSignature` 偏移点越界 = 整签名失败 (严格)** —— 有意选择 (点离屏=图案没全可见=非真命中), 非 bug, 别"宽松跳过"改掉。
- **不拆定点比色节点** (CmpColorEx) —— 不声称与 anjian CmpColorEx 严格等价 (那是定坐标验证, 本节点是区域搜索); 无偏移点的
  签名近似覆盖该用途即可, 真要纯定点验证再议。
- **不给 `FindColorSignature` 加 hsv 模式** —— 先 RGB + 逐通道容差; 有需求再加 mode。
- **`scaleTolerance` 沿用容器级配置, 本批不加节点级覆盖** —— 节点 Inputs 无缩放字段是**有意**, 非漏 (同现有模板族)。
- **不做按模板分组 NMS**; **不做 TotalCount/Count 双字段** (保持 blobs 单 Count=总数一致)。
- **不做二值化/旋转等独立预处理节点** —— `pkg/vision/preprocess.go` 已有 `Binarize` 内部件, 做 OCR pipeline 时再升节点。
- **不碰 ONNX/YOLO/OCR** —— cv-pool 的大依赖路线, 与本批分家。

## 评审裁决摘要

两轮评审逐条对源码核实。**采纳**已并入上文。以下为 **驳回 / 澄清** (附理由, 防后人重提):

| 评审点 | 裁决 | 理由 (源码锚点) |
|---|---|---|
| `scaleTolerance` 现有没有 / 是否真消费 | **驳回 (现有真参数)** | `Detect` 现有参数 (`interfaces.go:25`), adapter 在变体选择+缩放门消费 (`wire_container.go:228-245`); `MatchAll` 不收它, DetectAll 镜像 Detect 消费。 |
| Count 与 blobs「不一致」 | **驳回 (实为一致)** | blobs 本就 `Count=总数不截断` (`detect_color_blobs.go:154`); 沿用并文档标注。 |
| 加 `TotalCount`+`Count` 双字段 | **驳回** | 保持与 blobs 单 `Count`=总数一致; 用 widget tooltip + 文档提示 `Count>len(Matches)`, 不增字段制造跨节点不一致。 |
| `CaptureMatches` vs `Blobs` 命名不统一 | **驳回** | 均复数名词, 已一致。 |
| 阈值围绕 argmax 设计、全量收集不合理 | **驳回 (真担忧另解)** | conf 逐位、收集 ≥th 良定义; 候选爆炸用 3×3 极大预筛 + NMS 前 4096 硬上限解。 |
| `FindColorSignature` ≠ anjian `CmpColorEx` | **采纳 (去等价声称)** | 不再声称严格等价 (见非目标), 借鉴其概念而非复刻定点验证。 |
| 接口改动需 feature 分支隔离 | **部分** | 已在 `feat/v2-foundation`; 不另开, 改完跑模板族测试隔离回归 (落地步 3)。 |
| 多模板统一 NMS 丢失被抑制命中的 TemplateKey | **驳回** | dedup 是本节点目的 (一物不重复计数); 保留命中带 TemplateKey 即够, "还有哪些模板在此匹配过"无场景需求 (YAGNI)。 |
| Signature 自由 JSON widget 无字段级约束 = 产品缺口 | **驳回 (同既有)** | 与 blobs `Range` 同款 `json` widget; 结构性靠 Validate。做结构化 widget 属 editor-footgun 范畴, 不在本批。 |
| 源码锚点行号易随改动失效 | **接受为 tradeoff** | 锚点**符号名优先**, 行号带 `~` 仅辅助定位; 撞不上以符号 grep 为准。 |

## 源码锚点 (撞问题先核这些, 别脑补)

- 视觉服务接口 + 类型: `internal/node/interfaces.go` (`VisionService` / `Point` / `BlobEntry` / `ClusterEntry` / `HSVRange`)
- 视觉服务实现: `internal/services/container/runtime/node_services.go` (`visionAdapter`: `Match` ~497 / `WaitMatch` ~504 / `matchOnce` ~540 / `scaleTolerance` ~490; 缓存帧注释 ~480)
- 视觉服务 stub: `internal/node/services.go` (`stubVisionService`)
- 模板匹配接口 + 4 实现: `interfaces.go` (`TemplateMatcher`/`NoopMatcher:31`) · `wire_container.go:214` (`templateMatcherAdapter`, 生产) · `inspect_phase_test.go:26` (`mockMatcher`) · `node_services_test.go:213` (`stubMatcher`)
- 生产匹配路径 (Point=中心 / BBox 槽=ROI / 变体缩放 / 多槽): `wire_container.go:214-343` (`Detect`/`detectMultiRegion`; `vision.Match` 行 281/323; center 行 285-286; ROI 槽行 287)
- 模板匹配实现 (纯 Go): `pkg/vision/template.go` (`Match` 全局 argmax / `MatchFast` 2x / `ScaleTemplate`/`MatchBest`)
- 列表产出 + Count 语义先例: `internal/nodes/detect/detect_color_blobs.go` (`Blobs`/`BlobCount`(:154 不截断)/`PrimaryCenter` + 捕获框)
- 颜色节点先例: `internal/nodes/detect/detect_color.go` (`parseRange6` / `dcRangeSchema` / colorRange widget)
- 截图/抓帧先例: `internal/nodes/detect/screenshot.go` (`ctx.Capture().CaptureROI`)
- 多模板依赖抽取: `internal/nodes/detect/template_common.go` (`templateDeps`)
- 数据流/捕获硬约束: [node-data-flow](../checklists/2026-06-05-node-data-flow.md)
