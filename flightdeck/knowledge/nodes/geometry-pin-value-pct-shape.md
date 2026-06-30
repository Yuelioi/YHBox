# ⚠ Geometry pin 的值是 `{ pct, overrides }`，不是扁平 `{x,y,w,h}`

SUMMARY: Geometry pin 存 `{pct, overrides}` 而非扁平 rect，前端回填/后端校验读错形状会全 0 或误报
READ WHEN: 读/写 Geometry 类型 pin 的值 (前端回填 / 后端校验); 撞"框选/截图设了值但 widget 显示全 0 / 校验永远报 ROI 错"

---

**坑（同次 smoke 挖出两处）**:

1. **前端回填写错形状** → DetectColor 截屏框选不显示。`NodeInspector` 的 `applyRect` 仍写扁平 `{x,y,w,h}` 到 `config.literal.Region`，但 `Region` 早迁成 `Geometry` 类型。`GeometryWidget.safeValue` 读 `v.pct`，读不到就**整体回退全 0** → 框选结果不显示。
2. **后端校验读错层级** → DetectColorHSV ROI 永远误报。`validateDetectColorHSV` 读 `roi["w"]`（顶层），但值在 `roi["pct"]["w"]`，顶层永远拿不到 → 不管框多大都报 `w/h>=1` 失败。而且 ratio 本就 0–1，"≥1 像素"阈值对 ratio 无意义。

**事实**:
- `Type: "Geometry"` + `Schema: node.GeometrySchema()`（`{Type:"object", Widget:"geometry"}`）的 pin，前端走 `StructuredInput` → `GeometryWidget`。
- 存储形状 = `GeometryValue { pct: {x,y,w,h}, overrides: [{resolution, px}] }`，其中 **`pct` 是 0–1 比例**（widget 显示成 X%/Y%/W%/H%），`overrides` 段才是按分辨率的**像素**值。
- `pct.w===0 && pct.h===0` = 全帧。后端 `in.Geometry()` → `ResolveGeometry` 按窗口尺寸 + override 折算成像素。

**怎么做**:
- 前端写 Geometry pin：写 `{ pct: {...}, overrides: 原样保留 }`，别写扁平 rect。
- 后端读/校验 Geometry literal：读 `.pct.*`，且记住是 ratio（别套像素阈值）；全帧是合法值不是错误。
- 用 `Type: "Geometry"` 的节点（DetectColor/DetectColorHSV/ROIColorScan/WaitStable/WaitChange/Screenshot/DualColorBarTrack）改 ROI 相关逻辑前，先认清是 ratio 不是像素。

相关: [pin-presence-check-must-mirror-pinvalue.md](pin-presence-check-must-mirror-pinvalue.md)、[node-validation-pipeline-bifurcation.md](node-validation-pipeline-bifurcation.md)。
