# Visual analysis recipe

## Outcome / Question

以 Analyze Color / Find Color Blobs 为第一条完整复杂节点黄金路径，证明 Authoring Surface 不只是更好看的数值表单，而是能把“从目标画面识别颜色”表达成可理解、可截图取样、可预览、可连接的任务流程。

## Completion criterion

- ColorRange 默认展示颜色空间、范围色样和人类可读摘要；RGB/HSV 六通道上下限进入高级区，不再作为第一屏唯一交互。
- 从节点当前有效 target 打开 ScreenPicker color 模式，复用 ExtractColorRange 回填严格 ColorRange typed value；取消不修改，失败就地解释。
- Region adapter 从同一 target 打开矩形框选并回填 ratio/px；Point adapter同理取点。
- Analyze Color / Find Color Blobs 的 Inspector 将图像来源、颜色范围、搜索区域和常用阈值按任务顺序展示；输出使用匹配像素数、覆盖比例、中心位置、色块等用户语言。
- 提供可搜索的视觉分析配方入口，至少能插入“捕获目标画面 → 分析颜色”和“捕获目标画面 → 查找色块”的普通 3.1 节点链；配方不创建新 runtime 或隐藏节点。
- 节点卡片只内联颜色摘要、区域或一个高频阈值，不铺满通道数值。
- 目标缺失、图片未连接、颜色未取样等状态在字段附近解释并给出下一步动作。

## Blocked by

- Slice 07 的 Authoring Surface interface、adapter registry 与 target context。

## Verification

- 组件测试模拟 picker success/cancel/failure，断言严格 ColorRange/Region/Point 值和无隐式 mutation。
- Source/Compiler 定向测试确认配方生成的仍是普通 Capture Frame + vision nodes 与 typed edges。
- WebView 中从空白工作流插入两条配方，绑定默认目标、框区域、取颜色、保存、重开并运行。
- 检查 100%/125%/150% 缩放和窄 Inspector 下无截断、重叠或横向滚动。
- Stage D 末统一运行 `task check`、`task webview:smoke`、`task build` 与 UAC 桌面截图验收。

## Out of scope

- 新增 OCR、模型推理或改变既有视觉算法。
- 在 editor adapter 内执行工作流或持有授权。
- 复制 3.0 DetectColor 节点/脚本系统。

## Result

- ColorRange 以色样、摘要和颜色空间为第一层，高级通道折叠；Point/Region/Color 共用当前 target 的 ScreenPicker 边界。
- success/cancel/failure 与 Point/Region/ColorRange 严格值映射均有独立测试，取消不会隐式修改值。
- 节点目录提供可搜索“捕获并分析颜色”“捕获并查找色块”配方，插入的仍是 Capture、Vision、Comparison、Collection 与 Branch 普通节点和 typed edges。
- 视觉配方插入经过真实 EditorSession 回归；WebView smoke 中目录入口、子图与资源工作区均正常加载且无前端错误。
