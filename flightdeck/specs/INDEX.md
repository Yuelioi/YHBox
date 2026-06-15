# specs/ — INDEX

<!-- AUTO:specs -->
### 待启动（idea）
- [bug.md](bug.md) — idea — 临时 bug 暂存池（不值得开 spec 的随手记）— 当前空
- [cv-perception-pool.md](cv-perception-pool.md) — idea — CV 感知节点 / OCR / ONNX 灵感池
- [editor-footgun-backlog.md](editor-footgun-backlog.md) — idea — 节点编辑器 footgun backlog — ① exec 出口 Data 字段连不上(前提已变, 见正文需重评) ② DetectColor「范围」裸 JSON textarea 跟 HSV 结构化输入不一致
- [misc-tools-backlog.md](misc-tools-backlog.md) — idea — 杂项小工具 backlog — i18n residue 清理 (悬浮窗工具→转正 floating-launcher; 截图 UI 美化已做; 从 scratch-backlog 抢救保留)

### 进行中·完成（active·done）
- [2026-06-15-output-auto-capture.md](2026-06-15-output-auto-capture.md) — active — "输出自动捕获 + Inspector 输出组统一绑定 (Spec C)。取消'逐节点手声明 Semantic:capture 输入框'这套, 捕获绑定改存 config.capture{字段:变量名}, 前端 Inspector「输出」组统一一行式绑定 (方案 A: 按钮绑+chip, 翻译统一)。**两条写路径** (节点形态决定, 非统一): ① fire-time 自动捕获 — 出口 Data 字段值在 dispatch routeResult 由框架自动写进绑定变量 (~11 个检测/截图/脚本节点, 零节点代码); ② region per-iteration 显式捕获 — Loop/ForEach 的 Index/Item 在 RunRegion 每轮由节点调 helper 读 config.capture 写 (不经 routeResult)。模板三件套 Found 布尔补成显式 Data 字段。**消费者审计** (config.capture 是新 var-ref 站): useVarMutations 5 处 (rename/count/listUsageNodeIDs/deleteVar-cascade/listUsageRefs) + 后端 validator + referrers 全改读 config.capture。迁移条件化 + per-node 映射。边界: 不碰 vue-flow 画布/节点/连线/pin。"
<!-- /AUTO -->
