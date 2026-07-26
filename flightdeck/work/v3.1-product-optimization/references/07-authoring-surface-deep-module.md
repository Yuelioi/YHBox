# Authoring surface deep module

## Outcome / Question

把严格 Data Type / Node Contract 投影成普通用户可发现、可操作的节点创作界面。页面不得按节点 ID 或 kind 堆条件分支；复杂输入差异隐藏在类型级 Editor Adapter 内部，运行语义仍只属于 3.1 Source、Compiler 和 Program。

## Completion criterion

- Authoring Projection 提供非语义 presentation facts：group、order、importance、unit、inlinePriority、preset、help 与 editorAdapter；这些字段不能改变类型、binding、carrier、capability 或执行行为。
- 建立单一 Authoring Surface interface，把 config 与 typed input 归一为稳定分组、排序、主次、默认值和已解析 adapter；Inspector 与节点卡片消费同一投影。
- 类型级 registry 首批统一 Point、Region、Duration、KeyChord、Asset、Target；页面只渲染 adapter host，不知道具体节点 ID。
- Inspector 按“必填/常用/高级/输出”渐进披露；默认值、单位、帮助、连接状态和资源状态就地表达。
- 节点卡片最多显示 1–3 个高频、未连线、可安全内联编辑的输入；其他输入仍保留 typed port，不能因隐藏 UI 丢失连接能力。
- Point/Region 支持 ratio/px 的人类单位和目标截图拾取；Duration 默认以 ms/s/min 可读编辑；KeyChord 使用按键捕获；Asset 使用可搜索资源选择；Target 继承工作流默认目标并允许节点覆盖。
- adapter 解析与 presentation 投影有纯函数测试；未知 adapter 安全退回通用 typed editor。

## Blocked by

- Slice 06 已证明真实调试链路可用，Stage D 可以只聚焦创作表达。
- 复用现有 ScreenPicker、AssetPicker、AdaptiveSelect 和 capability target projection，不新增第二套服务。

## Verification

Slice 内只执行支持开发的聚合定向检查：

- Go contract/projection 测试覆盖 metadata 规范化、确定性生成和非语义隔离。
- 前端纯函数/组件测试覆盖 adapter 解析、分组排序、inline 限额、fallback 和各复杂类型。
- 用自动化输入节点与视觉节点验证继承目标、Point、Region、Duration、KeyChord、Asset 均走同一 host。
- Slice 08 完成后统一执行完整 task check、WebView smoke、production build 和桌面截图验收。

## Out of scope

- 修改类型可赋值规则、Compiler、Program、Admission 或 Scheduler。
- 恢复 3.0 Container runtime、按 kind 分发或脚本字面量编辑器。
- 自由停靠、完整节点 UI 插件 ABI 或第三方前端代码加载。

## Result

- Data Type / Node Contract 已生成并验证 presentation metadata，且保持 semantic digest 隔离。
- `authoringSurface.ts` 统一投影 config/input/output；Inspector 与节点卡片只消费同一 interface。
- Point、Region、Duration、KeyChord、Asset、Target 进入统一 adapter host；复杂 adapter 按需加载，编辑器初始 gzip 为 212,537 bytes，低于 220,000 上限。
- ratio/px 切换会用当前 target 客户区尺寸显式换算；缺少 target、必填未绑定和未知 adapter 均有安全反馈或 fallback。
