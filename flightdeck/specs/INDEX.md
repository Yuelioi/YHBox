# specs/ — INDEX

<!-- AUTO:specs -->
### Backlog (idea)
- [bug.md](bug.md) — idea — 临时 bug 暂存池（不值得开 spec 的随手记）— 当前空
- [cv-perception-pool.md](cv-perception-pool.md) — idea — CV 感知节点 / OCR / ONNX 灵感池
- [editor-footgun-backlog.md](editor-footgun-backlog.md) — idea — 节点编辑器 footgun backlog — ① exec 出口 Data 字段连不上(前提已变, 见正文需重评) ② DetectColor「范围」裸 JSON textarea 跟 HSV 结构化输入不一致
- [misc-tools-backlog.md](misc-tools-backlog.md) — idea — 杂项小工具 backlog — i18n residue 清理 (悬浮窗工具→转正 floating-launcher; 截图 UI 美化已做; 从 scratch-backlog 抢救保留)

### Active · Done
- [2026-06-23-ai-nodes.md](2026-06-23-ai-nodes.md) — active — AI 功能 epic 第②块: 图里调 LLM。新 AI 节点(选 connection+model、提示词模板 {{Name}} 插值 + 任意个带类型动态输入、任意个带类型结构化输出逐字段可捕获、vision 图像输入);llm 包扩结构化输出(OpenAI json_schema / Anthropic 强制 tool-use / 提示词注入三模式)+ 多模态图像;Provider 按 connectionID 指纹缓存自愈 + 池调优;新框架机制 config.Outputs[] 驱动动态 Data 字段(DynamicDataFields,镜像 DynamicInputs);Image 成一等流动值(node.Image{Format,Data} 编码字节,Capture 可选 PNG/JPEG)+ Screenshot 拆 Capture/SaveImage/LoadImage 三个图像节点。一个 spec 分阶段 plan 落地。
<!-- /AUTO -->
