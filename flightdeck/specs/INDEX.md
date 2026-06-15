# specs/ — INDEX

<!-- AUTO:specs -->
### 待启动（idea）
- [bug.md](bug.md) — idea — 临时 bug 暂存池（不值得开 spec 的随手记）— 当前空
- [cv-perception-pool.md](cv-perception-pool.md) — idea — CV 感知节点 / OCR / ONNX 灵感池
- [editor-footgun-backlog.md](editor-footgun-backlog.md) — idea — 节点编辑器 footgun backlog — ① exec 出口 Data 字段连不上(前提已变, 见正文需重评) ② DetectColor「范围」裸 JSON textarea 跟 HSV 结构化输入不一致
- [misc-tools-backlog.md](misc-tools-backlog.md) — idea — 杂项小工具 backlog — i18n residue 清理 (悬浮窗工具→转正 floating-launcher; 截图 UI 美化已做; 从 scratch-backlog 抢救保留)

### 进行中·完成（active·done）
- [2026-06-15-output-auto-capture.md](2026-06-15-output-auto-capture.md) — active — "输出自动捕获 + Inspector 输出组统一绑定 (Spec C)。取消'逐节点手声明 Semantic:capture 输入框 + Run 里手调 node.Capture()'这套, 改成框架在 dispatch routeResult 把 fire 出口的 OutputData[字段] 自动写进用户绑定的变量。前端 Inspector「输出」组合并掉 Part 2 的'速览'+'捕获'两套, 每个可绑产出一行: 翻译名 + 类型 + 「+绑变量」按钮 (绑后显 → \$var ✕ chip), 写 config.capture{字段:变量名}。所有执行节点的数据产出自动可绑 (含现在漏声明捕获框的 PlayClip.Error/Code)。核心不变量: 被捕获值必须是出口 Data 字段 (从 OutputData 读) —— 模板三件套的 Found 布尔补成显式 Data 字段。删 13 文件 27 个 capture 输入 + node.Capture 助手。迁移条件化 (旧 config.literal[Capture<X>] → config.capture[<X>], 没有就跳过)。边界: 不碰 vue-flow 画布/节点/连线/pin, 绑定全在 Inspector。"
<!-- /AUTO -->
