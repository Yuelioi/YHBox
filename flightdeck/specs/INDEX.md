# specs/ — INDEX

<!-- AUTO:specs -->
### 待启动（idea）
- [bug.md](bug.md) — idea — 临时 bug 暂存池（不值得开 spec 的随手记）— 当前空
- [cv-perception-pool.md](cv-perception-pool.md) — idea — CV 感知节点 / OCR / ONNX 灵感池
- [editor-footgun-backlog.md](editor-footgun-backlog.md) — idea — 节点编辑器 footgun backlog — ① exec 出口 Data 字段连不上(前提已变, 见正文需重评) ② DetectColor「范围」裸 JSON textarea 跟 HSV 结构化输入不一致
- [misc-tools-backlog.md](misc-tools-backlog.md) — idea — 杂项小工具 backlog — i18n residue 清理 (悬浮窗工具→转正 floating-launcher; 截图 UI 美化已做; 从 scratch-backlog 抢救保留)

### 进行中·完成（active·done）
- [2026-06-14-ui-uplift-editor.md](2026-06-14-ui-uplift-editor.md) — active — "UI 升级第二轮 (Spec B): 容器编辑器 restyle + 布局/UX 重设计。方向 = A 双栏停靠 + Inspector 选中才出 (全量重做, 解 4 大痛点: modal 盖画布 / 三层边栏挤画布 / Inspector 扁平 / Toolbar 主次乱)。① 面板 IA: 左侧自适应宽度停靠区收纳 节点库·变量·Snippets·资产浏览器 (窄列表↔宽网格自适应, 始终挤画布不盖画布), 小弹窗 restyle 留 modal; ② Toolbar 三区分层 (导航 / 主操作 hero / 文档+⋯收纳) + 底部问题条; ③ Inspector 三态收起规则 + SectionHeader 分组。复用 Spec A 设计语言 + 共用组件 (含 AlertBox/SectionHeader/ListRow)。边界: vue-flow 画布·节点框·连线·pin·画布右键 只继承 token, 不重设计。"
<!-- /AUTO -->
