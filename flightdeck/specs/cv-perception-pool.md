---
status: idea
summary: CV 感知节点 / OCR / ONNX 灵感池
---

# CV 感知节点 / OCR / ONNX 灵感池

通用框架方向定下后 ([CLAUDE.md 项目方向 2026-05-26](../../CLAUDE.md)), CV 感知层是核心扩展面.
现有 `internal/nodes/detect/` 是钓鱼起点 (模板三件套 `check/click/wait_template` + `color_bar_track` + 点比色 + ROI 扫色), 通用框架要补"高级感知层".

**触发**: 用户灵感爆发 2026-05-26, 外部 AI 推荐 5 个 + OCR + ONNX + 自补.
**再看条件**: 通用框架第一批落地场景定 + 节点扩展模式 (framework owner 内置 / plugin / 用户 Go 自定义) 定后, 从此池挑 1-2 个做 spec. 不全做.
**关联**: 现有 `internal/nodes/detect/`, `capture_wgc.dll` (已有 D3D11, 后续 CV 可走 GPU compute shader).

---

## 1. 外部 AI 推荐的 5 个 CV 节点

| # | 节点 | 通用场景 | 实现路线 |
|---|---|---|---|
| 1 | 颜色直方图相似度 | 昼夜/天气/地图切换/整体氛围识别 | 纯 Go (RGB bin 统计 + 巴氏距离), 几十行 |
| 2 | pHash (感知哈希) | 模糊模板 / 装备-卡牌识别 / 黑屏掉线 | 纯 Go (DCT 或简易 average hash + 汉明距离) |
| 3 | 连通域几何 (blob) | 小地图标记 / 怪物点位 / UI element 定位 / 血条%测算 | 纯 Go 二值化 + flood fill + 外接矩形/质心 |
| 4 | 投影切字 (binary projection) | OCR 预处理 / 数字切割 / 血条分段 | 纯 Go, 但通用框架真要 OCR 直接接 PaddleOCR 收益更大 — 投影只做"半 OCR" |
| 5 | 光流法 | 卡死检测 / 移动感知 / 3D 跑图 / 音游 perfect | gocv (OpenCV C 依赖) 或 ONNX 模型; 纯 Go 不现实 |

---

## 2. OCR 路线 (待选)

| 选项 | 优 | 缺 |
|---|---|---|
| PaddleOCR (RapidOCR via ONNX) | 中文最强, ONNX 跨平台无 CGO 痛 | 模型 8M+, 启动慢 |
| Tesseract (CGO) | 老牌可靠 | CGO 依赖, 中文一般 |
| 自训轻量数字 CNN (via ONNX) | 游戏数字字体固定, 几 KB 模型够用, 跑得飞快 | 要标数据 + 训练; 只适合数字/固定字体 |
| 投影切字 (#4) + 模板比对 | 零依赖 | 字体/特效一变就崩 |

跟 #3 ONNX 路线绑死: 选 ONNX 即解锁 OCR + 后面的目标检测 / 场景分类.

---

## 3. ONNX 推理引擎接入

`onnxruntime-go` (微软官方 Go bindings) → 用户挂任意 `.onnx` 模型, 节点输入 ROI image → 输出 JSON (类别 + bbox + 置信度).

**节点形态拆**:
- `OnnxInfer` — 通用推理节点, 用户配模型路径 + 输入 tensor 形状
- `YoloDetect` — 封装好的 YOLOv8 目标检测 (内置 COCO + 用户挂自训)
- `ClassifyScene` — 场景分类 (登录 / 战斗 / 菜单 / 加载...)
- `OcrText` — 文本识别 (上面 OCR 路线)

**模型管理**: 项目 `models/` 目录管 onnx 文件; 用户脚本通过节点配置引用. 跨平台无 CGO (onnxruntime 是动态库, 跟 capture_wgc.dll 同套打包思路).

---

## 4. 灵感爆发补的 (我加的)

按"通用游戏自动化 CV 还缺什么" 倒推:

- **特征点匹配 (ORB)** — 旋转/缩放/光照不变, 比模板适应度强; 实现可纯 Go 也可 gocv
- **像素差分 (frame diff)** — 比光流便宜两个量级, 检测画面是否变化 (卡死 / 弹窗出现); 几十行纯 Go
- **模板金字塔 / 多分辨率匹配** — 应付分辨率切换 + DPI 缩放, 现有模板匹配死板
- **掩码 ROI** — 不规则区域排除 (HUD 元素遮挡时排除掉再检测), 现有 ROI 只支持矩形
- **颜色聚类 / 主色提取** — 自适应阈值, 用户不用手填 HSV
- **稳定帧检测 (frame stability)** — 等画面停止变化再操作, 避免动画中识别失败 (跟像素差分组合)
- **多帧投票** — 同一识别跑 N 帧多数投票, 抗瞬时干扰
- **像素↔地图坐标变换** — minimap → 世界坐标, 给路径规划用 (依赖 #3 blob 先定位)
- **节点级 capture 缓存** — 同一 tick 多个检测节点只 `wgc_session_grab` 一次, 不是性能优化是架构 invariant (Container 层做)
- **CV pipeline sub-graph 模板** — `capture → ROI → 二值化 → blob → 几何过滤` 这种链做成内置 sub-graph 模板, 用户拖一个就来一套
- **GPU 加速 hook** — capture_wgc 已经有 D3D11 device, 后续 CV 重操作 (大图卷积 / 模板匹配) 也走 GPU compute shader 不下 CPU; 但远期, 现在不动

---

## 5. 按键精灵 YOLO/CV 函数表 参考 (2026-06-18)

来源: help.anjian.com `expand/yolo/{SetModel,RunModel,ModelConvert}` + `basic/{color,ocr,pic}` + `expand/{opencv,image,ml}`. 通读结论: **大部分印证本池方向**, 我们 `detect/` 里 framediff (`framediff_nodes`)、色块 blob (`detect_color_blobs`)、拟人抖动 (jitter) **已实现**; 按键精灵真正能补的是 YOLO/OCR/多点找色. 拆点:

**(A) 最值得借鉴的是 API 形态, 不是节点**: `SetModel(index,targetSize,param,bin,label[,创建线程,运行线程,{modelType}])` 与 `RunModel(index,sim[,opt])` **分离** —— 加载(贵, 索引化模型句柄 0-9, **不跑不占资源, 停脚本自动销毁**) vs 推理(便宜, 循环里反复调). 启示: `YoloDetect`/`OnnxInfer` **绝不能每帧重载模型** → "加载模型(缓存句柄) vs 推理" 拆开, 跟 §4 的"节点级 capture 缓存"同属 **Container 架构 invariant** (性能 = 设计三问之一). 返回 `{text,x,y,w,h,score}` 数组 + `sim` 置信度阈值参数 + 线程数 config(暴露不锁死) —— 跟本池 §3 YoloDetect 规划的"JSON 类别+bbox+置信度"对得上.

**(B) ncnn 转换 footgun → 实锤"选 ONNX 不选 ncnn"**: 按键精灵 `.pt→.ncnn` 要**手改生成的 `_pnnx.py`**(把 `reshape(1,87,6400)` 改成 `reshape(1,87,-1).transpose(1,2)` 那段), 简化版(modelType=1)还**掉精度**, 标准版(=2)要按 ncnn 官方教程改参数. 我们 §2/§3 已定 ONNX 路线 (`yolo export format=onnx` 一行, 无需手改) —— 这页正是"为什么别走 ncnn"的反例, 强化 ONNX 决策.

**(C) 两个本池没列的新候选**:
- **多点找色签名** (`FindMultiColor`/`CmpColorEx`): 匹配"主点 + N 个相对偏移点的颜色组合", 比单点色稳、比模板便宜的**中间档**, 找纯色 UI 元素(按钮/图标)很省. 现有 `detect_color`/`hsv`/`blobs`/`roi_color_scan` 无此原语.
- **二维码 decode** (`Image.QrDecode`): 屏幕 QR→字符串, 纯 Go (gozxing) 自包含便宜. 小众, 放池子.

**(D) 配置泛化 —— 模板匹配返回 all vs best** (`OpenCV.FindImgAll`/`FindImgBest`): 现 `template_common.go:matchOnce` 只返回**单个 best 点**. 加"返回全部命中"模式 = 设计三问"别替用户锁死单一选择". 真有"批量点同类图标"场景再做.

**(E) 预处理原语拆分**: `Image.Binaryzation`/`GetScreenData`/`SetRotation` 各自独立 = 本池 §4 "CV pipeline sub-graph" 思路. 现 blob 内部已二值化但无独立 binarize 节点; 做 OCR pipeline 时补.

**不借鉴**: framediff / 色块 blob / jitter 已有; `Element.*`(无障碍元素树) 是手机 accessibility 模型, Windows 游戏不走 UIA, 不适用.

---

## YAGNI 提醒

按 [CLAUDE.md 二号铁律], 别开池子就堆 spec. 等第一批通用场景落地需求出来 (e.g. "要做 MMO 日常脚本" / "做卡牌抽卡" / "做音游"), 反推这次该挑哪 1-2 个先做.

按目前直觉的优先级排序 (无场景前的预判, 落地后可能洗牌):

1. **像素差分 + 稳定帧检测** — 几乎所有自动化都需要的"画面变了吗 / 稳了吗" 基础能力
2. **ONNX 接入 + YOLO 目标检测** — 一个能力解锁后面 OCR / 场景分类 / 自训模型一条线
3. **连通域几何 (blob)** — OpenCV 100 块经典中 ROI/blob 是其他高级 CV 的前置基础
4. **节点级 capture 缓存 (架构层)** — 不是节点是 Container invariant, 但 ROI 多了一定要做

剩下都先放池子里, 真有 case 再挑.
