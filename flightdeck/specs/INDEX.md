# specs/ — INDEX

<!-- AUTO:specs -->
### 待启动（idea）
- [bug.md](bug.md) — idea — 临时 bug 暂存池（不值得开 spec 的随手记）— 当前空
- [cv-perception-pool.md](cv-perception-pool.md) — idea — CV 感知节点 / OCR / ONNX 灵感池
- [editor-footgun-backlog.md](editor-footgun-backlog.md) — idea — 节点编辑器 footgun backlog — ① exec 出口 Data 字段连不上(前提已变, 见正文需重评) ② DetectColor「范围」裸 JSON textarea 跟 HSV 结构化输入不一致
- [misc-tools-backlog.md](misc-tools-backlog.md) — idea — 杂项小工具 backlog — i18n residue 清理 (悬浮窗工具→转正 floating-launcher; 截图 UI 美化已做; 从 scratch-backlog 抢救保留)

### 进行中·完成（active·done）
- [2026-06-11-script-template-dep-extraction.md](2026-06-11-script-template-dep-extraction.md) — active — Script 节点扫 Code 里的模板/clip/subgraph GUID 字面量当依赖,堵住"脚本引用的资产被 GC 误删、库里删不警告"的盲区
- [2026-06-11-script-call-subgraph.md](2026-06-11-script-call-subgraph.md) — active — 把 Subgraph 暴露成脚本绑定函数 Subgraph({SubgraphID, ...params}),让脚本当编排层复用子图库(下一阶段,gated 在 Stage 1 之后)
<!-- /AUTO -->
