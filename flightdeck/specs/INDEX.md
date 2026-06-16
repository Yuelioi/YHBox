# specs/ — INDEX

<!-- AUTO:specs -->
### Backlog (idea)
- [bug.md](bug.md) — idea — 临时 bug 暂存池（不值得开 spec 的随手记）— 当前空
- [cv-perception-pool.md](cv-perception-pool.md) — idea — CV 感知节点 / OCR / ONNX 灵感池
- [editor-footgun-backlog.md](editor-footgun-backlog.md) — idea — 节点编辑器 footgun backlog — ① exec 出口 Data 字段连不上(前提已变, 见正文需重评) ② DetectColor「范围」裸 JSON textarea 跟 HSV 结构化输入不一致
- [misc-tools-backlog.md](misc-tools-backlog.md) — idea — 杂项小工具 backlog — i18n residue 清理 (悬浮窗工具→转正 floating-launcher; 截图 UI 美化已做; 从 scratch-backlog 抢救保留)

### Active · Done
- [2026-06-17-yt-scripting-console.md](2026-06-17-yt-scripting-console.md) — active — 编辑器内 JS 脚本控制台 (命名空间根 yt, 对标 blender bpy): 对当前容器主图+所有子图全节点批量改 config。v1: yt.nodes/selected/container/log + 预留 yt.ops。new Function 执行, set 收集后一次性 applyDraftMutation(一步撤销)+标脏, 抛错零变更, Ctrl+S 落盘。复用 CodeInput(CodeMirror)/walkAllGraphs/PIN_SPECS。
<!-- /AUTO -->
