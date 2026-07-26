# 真机反馈跟进 — 2026-07-21

## Summary

| ID | 用户观察 | 根因或设计判断 | 本轮处理 | 真机复测 |
| --- | --- | --- | --- | --- |
| FD-11 | 八个数学/文本节点缺少图标 | Catalog 写入了当前 Tabler 集合不存在的名称 | 替换为可解析的近义 Tabler 图标，并校验全部 Catalog 图标 | 目录、Tab 添加器和节点卡片均显示图标 |
| FD-12 | 编译诊断固定在顶部，和运行信息割裂 | 验证面板独立于底部 runtime workbench | 诊断与日志、时间线、调试合并为四个底部页签 | 编译失败后自动打开诊断页签，定位节点正常 |
| FD-13 | Run State 的按键码不能接组合键；多种类型不能创建 | 创建 UI 只枚举部分 ref 类型，组合键实际要求 `list<key-code>` | 暴露所有有安全默认值的 durable 类型，并增加组合键选项和编辑器 | 创建组合键并连接“按下组合键”；逐个创建复杂类型 |
| FD-14 | Switch 左侧 case 标签数量/文案与右侧不一致 | 动态输入复制了 prototype 的“待匹配值”标题 | 每个实例端口投影为独立 `case-N` 标题 | 调整 case 数量后左右一一对应，旧 Source 仍兼容 |
| FD-15 | 所有节点右键都有“视觉模板” | 类型专属动作泄漏进通用节点菜单 | 删除通用菜单入口，保留 Inspector 类型字段与资源 Dock 的模板流程 | 普通节点无模板菜单；模板节点仍可选择/捕获 |
| FD-16 | 左右侧栏宽度固定且不能隐藏 | 编辑器布局没有持久的折叠/resize seam | 两侧均可折叠、拖拽缩放并支持键盘调整 | 不同窗口宽度、DPI 和 WebView 缩放下无内容遮挡 |
| FD-17 | Run State 打开后点击节点仍停留在状态侧栏 | 节点 selection 没有声明右侧面板优先级 | 点击、右键、搜索或诊断定位节点都会关闭状态/AI 面板并显示 Inspector | 从状态面板直接点击任一节点，立即显示节点属性 |
| FD-18 | 有悬空节点时整个工作流不能运行 | Compiler 把缺少必填输入的孤立 pull 节点当执行根并阻塞编译 | 有事件根时忽略不完整的孤立 pull 草稿；合法独立 pull 根和纯数据工作流保持旧语义 | 主链可运行；未连接草稿不执行；重新接回后恢复编译校验 |

## Verification completed

- 前端五组定向测试：5 files / 60 tests passed。
- `go test ./internal/workflow/compiler ./internal/nodes ./internal/nodeauthoring ./internal/noderuntime -count=1`。
- contract generation、Go 23 个受影响/反向依赖包及 vet 通过。
- frontend format、lint、typecheck、i18n 与全量 Vitest 通过：67 files / 276 tests。
- 完整门禁曾捕获 pure-data 执行根回归；修复后原失败的 compiler/application/service fixture 全部通过。
  按新的门禁分层，普通修复不重复全仓 coverage、Rust 与 production bundle。

## Boundaries

- FD-11 只使用已安装的 Tabler outline 集，不建立平行自绘图标系统。
- FD-13 的 object 类型只有存在 contract example 时才可创建，避免伪造不合法默认值。
- FD-18 不静默吞掉主执行链错误；只忽略事件工作流中缺少必填输入的孤立 pull 草稿节点。
