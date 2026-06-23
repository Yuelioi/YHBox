# specs/ — INDEX

<!-- AUTO:specs -->
### Backlog (idea)
- [bug.md](bug.md) — idea — 临时 bug 暂存池（不值得开 spec 的随手记）— 当前空
- [cv-perception-pool.md](cv-perception-pool.md) — idea — CV 感知节点 / OCR / ONNX 灵感池
- [editor-footgun-backlog.md](editor-footgun-backlog.md) — idea — 节点编辑器 footgun backlog — ① exec 出口 Data 字段连不上(前提已变, 见正文需重评) ② DetectColor「范围」裸 JSON textarea 跟 HSV 结构化输入不一致
- [misc-tools-backlog.md](misc-tools-backlog.md) — idea — 杂项小工具 backlog — i18n residue 清理 (悬浮窗工具→转正 floating-launcher; 截图 UI 美化已做; 从 scratch-backlog 抢救保留)

### Active · Done
- [2026-06-23-mcp-node-exec.md](2026-06-23-mcp-node-exec.md) — active — AI 功能 epic 第③块(MCP 对外暴露 / AI 调我们): GUI 内置 Streamable HTTP MCP server, 暴露通用 run_node(单动作节点探测, 复用 held-output 缓存收割输出, 含 Capture 图像) + find_window + 写图四件套(后端换 GUI 真实 store), 闭合「AI 跑节点实验 → save_container 生成容器」环; 全局 arm 安全开关默认关; 不做整容器执行/有状态会话/模板工具(YAGNI)。
<!-- /AUTO -->
