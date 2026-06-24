# Detect/Click 节点能力补全(参考影刀 + MAA,10 项 + MatchMode 清理)

## 0. 定位 / 来源

研究两套同域工具的能力面,对照 YHFish 现有 Detect/Input 节点**穷举式** diff(已核实源码),筛出 10 个**落在「节点 config 补全 / 小节点」低成本范围内的真缺口**:

- **影刀 RPA**(`xbot.win32.Image` + 桌面点击):①②③④。
- **MaaFramework / MaaAssistantArknights**(基于图像识别的节点式自动化框架,与 YHFish 同命题):⑤⑥⑦⑧⑨⑩。

| # | 缺口 | 来源对应 | YHFish 现状 |
|---|---|---|---|
| ① | 锚点 + 偏移点击 | 影刀 anchor / MAA target_offset、rectMove | ClickTemplate 只点命中框中心 |
| ② | 等模板消失 | 影刀 wait_disappear | 全项目无(grep 零命中) |
| ③ | 组合键点击 | 影刀 keys / MAA — | ClickAt/ClickTemplate 无修饰键 |
| ④ | 双击 / 多击 | 影刀 dblclick | 全项目无双击 |
| ⑤ | 多命中选哪个 | MAA order_by + index | ClickTemplate 只点「最佳那个」 |
| ⑥ | 拖拽 / 滑动 | MAA Swipe | 无单一拖拽节点 |
| ⑦ | 输入字符串 | MAA InputText | 只有单键 KeyPress |
| ⑧ | ClickTemplate 搜索区 | MAA roi | ClickTemplate 无 ROI(找图走全帧) |
| ⑨ | 横向滚动 | MAA Scroll dx | Scroll 只竖直 notches |
| ⑩ | 关闭/杀进程 | MAA StopApp | RunProgram 只能开不能关 |

> **架构哲学(记一笔,不改)**:MAA 把「识别+动作+流程」揉进一个节点;YHFish 拆成 Detect →数据线→ Click,更解耦/更通用。本 spec 只借 MAA「该暴露成 config」的字段,不照搬其单节点融合模型。
>
> **穷举扫描已判出范围(留证)**:FeatureMatch / OCR / NeuralNetwork(大功能,需引擎,backlog)· green_mask 掩码(本轮未选,backlog)· MultiSwipe / TouchDown·Move·Up / Shell(多点触控 / ADB,PC 窗口实证不适用)· max_hit·rate_limit·cache·anchor跳转·baseTask·占位符替换(MAA 任务调度 DSL 特性,YHFish 用显式图连线/变量+Loop 表达,范式不同)· pre/post_wait_freezes(YHFish 已有 WaitStable/WaitChange 独立节点,更细,不内联)· **LongPress(冗余,= ClickAt.DurationMs 按住时长,见 §1 非目标)**。

## 1. 目标 / 非目标

**目标(In)**
- ClickTemplate 增:**锚点(九宫格)+ 偏移**(①)· **order_by + index** 多命中选哪个(⑤)· **ROI 搜索区**(⑧)。
- 新增 **WaitTemplateGone** 节点:等模板从画面消失再放行(②)。
- ClickAt 与 ClickTemplate 增:**组合键(Keys)** + **多击(ClickCount,含双击)**(③④)。
- 新增 **Swipe** 拖拽节点(⑥)· **InputText** 输入字符串节点(⑦)· **StopApp** 关闭进程节点(⑩)。
- Scroll 增**横向滚动**(⑨)。
- 清理:删 Check/Wait/ClickTemplate 的 `MatchMode`(any/all),多模板统一 OR,连带简化 VisionService 的 `mode` 参数。
- 全程 TDD;新增 pin 默认严格 = 旧行为(零回归)。**例外**:MatchMode 删除是有意行为变更 —— 原 `all` 用法需改用多节点串。

**非目标(Out — YAGNI,未发布无兼容包袱)**
- **LongPress 鼠标长按节点**:**冗余** —— `ClickAt.DurationMs` 即「按下时长」(源码 `down→hold(dur)→up`),`ClickAt(DurationMs=N)` 就是长按。不加独立节点(避免一处能力两处做)。
- OCR / FeatureMatch / 神经网络识别 / 结构化·AI 元素定位 / 网页自动化(独立大功能,需引擎/范式)。
- `green_mask` 模板掩码(本轮未选)。
- MultiSwipe / 多点触控 / Shell(ADB):PC 窗口 SendInput 实证不适用。
- MAA 的 inline `wait_freezes`(YHFish 用 WaitStable 独立节点)· `max_hit`/`rate_limit`/`cache`/anchor跳转/baseTask/占位符(任务调度 DSL,范式不同)。
- 通用 `OffsetPoint` 纯节点(锚点做进 ClickTemplate)· 把 magnitude-switch 单位预先推广到其它 pin · 给 WaitTemplate/CheckTemplate 加 bbox Rect 输出 · Swipe begin/end offset · ClickCount 逐击间隔做 pin。

## 2. 关键决策(已与用户拍板)

| 决策 | 选择 | 理由 |
|---|---|---|
| ① 锚点 | 做进 ClickTemplate | 单节点不拼链;命中框尺寸现成 |
| 偏移/坐标单位 | magnitude-switch(`|v|≤1`=比例`1`=100% / `|v|>1`=像素) | 一值两用、`1`=满幅;抽通用 helper(§11) |
| ② 等消失 | 新节点 WaitTemplateGone | 出口/轮询方向与 appear 相反,分开干净 |
| ③④ | Keys + ClickCount 加到 ClickAt + ClickTemplate | 暴露 config;ClickCount 比 dblclick bool 通用 |
| ⑤ 多命中 | order_by + index 加到 ClickTemplate;默认 `score/0`=旧单最佳路径,非默认走 MatchAll | 复用现成 MatchAll;默认零回归 |
| ⑥ 拖拽 | 新节点 Swipe,包现成 `pkg/input.MouseDrag` | 后端拖拽原语已存在 |
| ⑦ 输入 | 新节点 InputText + 新后端 `pkg/input.TypeText`(SendInput unicode) | 现有无打字符串能力 |
| ⑧ 搜索区 | ClickTemplate 加 ROI(Geometry) | `Matcher.Detect` 已吃 region(现传 nil);限范围更快更准,配 ⑤ 尤其需要 |
| ⑨ 横向滚 | Scroll 加 Axis(竖/横)+ 后端 WM_MOUSEHWHEEL | 现 `inputAdapter.Scroll` 只竖直 |
| ⑩ 关进程 | 新节点 StopApp(taskkill /F,按名或 PID) | RunProgram 只能开;无现成 kill helper |
| ⑩drop LongPress | **不做** | = ClickAt.DurationMs,冗余 |
| bbox 来源 | 上传 `Matcher.Detect` 现有 `regionOut`,改 WaitMatch/Match 签名 | 权威来源,含多尺度真实尺寸 |
| MatchMode | 整删,多模板恒 OR | `any` 不该是选项;`all` 罕用;WaitTemplateGone 消失语义也变干净 |

## 3. ① + ⑤ + ⑧ ClickTemplate 增强(锚点偏移 / 多命中 / 搜索区)

新增 pin(Advanced,默认 = 旧行为):

| pin | 类型 | 默认 | 说明 |
|---|---|---|---|
| `ROI` | Geometry | 全帧 | 找图搜索区(客户区 ratio,全 0 = 全屏) |
| `Anchor` | String 下拉 | `center` | 九宫格:落点从命中框哪个点起算 |
| `OffsetX` | Number | `0` | 锚点上水平偏移(单位 §11) |
| `OffsetY` | Number | `0` | 垂直偏移 |
| `OrderBy` | String 下拉 | `score` | 多命中排序:`score`(分↓,=旧最佳)/`horizontal`(左→右)/`vertical`(上→下)/`area`(面积↓)/`random` |
| `Index` | Number | `0` | 排序后取第几个(0 基) |

**Anchor 取值**(9):`topLeft topCenter topRight midLeft center midRight botLeft botCenter botRight`。

**落点算法**:命中框 bbox = `[bx,by,bw,bh]`(全帧归一化,左上角+宽高,§10)。
```
锚点(归一化):X = bx + bw*{0|0.5|1 按列};Y = by + bh*{0|0.5|1 按行}
最终落点 = clamp01(锚点 + 偏移ratio)
```

**两条匹配路径**:
- `OrderBy=score, Index=0`(默认)⇒ 走现有 `WaitMatch`(单最佳;ROI 经新增 roi 参数透传给 `Detect`),**零回归**。
- 非默认 ⇒ 轮询走 `MatchAll`(在 ROI 内多命中 []TemplateMatch)→ 按 OrderBy 排序(`score`=Conf↓ / `horizontal`=BBox.x↑ / `vertical`=BBox.y↑ / `area`=w*h↓ / `random`=洗牌)→ 取 `Index` → 对其 bbox 锚点+偏移 → 点击。每 `visionWaitPollMs` 一帧,命中数 ≤ Index = 继续等,超时走 `Timeout`。

**统一收尾**:选定命中后落点替换原中心,喂 `clickAt` / `settleAfterMatch` 重定位 / `MaxAttempts` 重点;Done/Timeout 出口 `Point` 改吐**最终落点**。

**删除**:本节点 `MatchMode` pin;多模板恒 OR。

## 4. ② WaitTemplateGone(新节点,Detect)

镜像 WaitTemplate 取反。Pin:`In` · `Templates`(必填,可多个) · `TimeoutMs`(5000) · `Threshold`(0.85)。出口:`Gone`(无数据)/ `Timeout`(Conf optional)。

**逻辑**:轮询(复用 `matchOnce`+`waitOrCancel`+`visionWaitPollMs`)→ `matchOnce` 返 `pt==nil` 即 `Gone`;到 `TimeoutMs` 仍命中走 `Timeout`(带 Conf)。`TimeoutMs<=0` → 单帧。可取消。**消失定义**:无任何模板命中(OR-absence),无 MatchMode;不需 bbox(§10 签名变更对它透明)。

## 5. ③ + ④ ClickAt / ClickTemplate 组合键 + 多击

两节点各加(Advanced,默认 = 旧行为):`Keys`(String,`+` 分隔修饰键,空=不按)· `ClickCount`(Number,默认 1)。

**实现**(抽公共 helper,两节点共用):
```
mods = parse(Keys)                      // split '+' / trim / 小写
for m in mods: Input.KeyDown(m)          // 顺序按下
for i in 1..ClickCount:
    Input.Click(x, y, button, durationMs)
    if i < ClickCount: 等 interClickGapMs  // 常量 ~60ms,< 系统双击时限
for m in reverse(mods): Input.KeyUp(m)   // 逆序松开
```
- 全用现成 `InputService.KeyDown/KeyUp/Click`,**后端零改动**(Click 内部已 down→hold→up)。
- ClickTemplate 现有 `clickAt` helper 扩展吃 keys+count;ClickAt 走该 helper。
- `interClickGapMs` 常量(60ms),不做 pin。
- Validate:Keys token ∈ `ctrl/shift/alt/win`(沿用 KeyHoldStart VK 命名),ClickCount<1 报错。

## 6. ⑥ Swipe(新节点,Input)

Pin:`In` · `Begin`(Point) · `End`(Point) · `DurationMs`(200) · `Button`(left)。出口 `Done`。NeedsWindow。
**实现**:`ctx.Input().Drag(begin.X,begin.Y,end.X,end.Y,button,durationMs)` → 包现成 `pkg/input.MouseDrag`(Down→分步 Move→Up,自带 duration)。需 InputService + inputAdapter 暴露 `Drag`(§10)。

## 7. ⑦ InputText(新节点,Input)

Pin:`In` · `Text`(String,必填,支持 unicode)。出口 `Done` / `Fail`(Error/Code)。NeedsWindow。
**实现**:`ctx.Input().TypeText(text)` → 新后端 `pkg/input.TypeText`:逐 rune 走 `SendInput` 的 `KEYEVENTF_UNICODE`(复用现有 `procSendInput`),不依赖 VK 映射。需 InputService + inputAdapter 暴露 `TypeText`(§10)。

## 8. ⑨ Scroll 横向滚动(改现有节点)

Scroll 加 `Axis` 下拉(`vertical` 默认 / `horizontal`);沿用现有 `Notches`(+/- 方向)。
**实现**:`InputService.Scroll` 增轴向参数;`pkg/input` 加横向滚动(`WM_MOUSEHWHEEL`,现 `MouseScroll` 只 `WM_MOUSEWHEEL`)。默认 vertical = 旧行为(零回归)。

## 9. ⑩ StopApp(新节点,IO)

Pin:`In` · `Target`(String,必填 —— 进程名如 `game.exe`,纯数字按 PID)。出口 `Done` / `Fail`(Error/Code)。
**实现**:Windows `taskkill /F /IM <name>` 或 `/PID <pid>`(小 helper,失败包 Fail)。与 RunProgram 同 IO 分组。

## 10. 底层改动

**VisionService**(internal/node/interfaces.go + runtime visionAdapter):
- `Match/WaitMatch`:**去 `mode` 参数**(恒 OR)、**加 `roi` 参数**(⑧)、返回携带 bbox(`*node.MatchHit{Point,BBox,Conf}` 或等价;形态留 plan)。`Matcher.Detect` 的 `regionOut`(命中框,现 `node_services.go ~L560/L579` 被 `_` 丢)带上来;`region` 参数填 ROI(现传 nil)。
- `visionAdapter.matchOnce`:删 `all` 分支(OR,按 keys 序取首个);透传 bbox + roi。
- `MatchAll` 已存在([]TemplateMatch 带 BBox/Conf/Point)→ ⑤ 直接复用。
- 同步改:`template_common.go`、`ClickTemplate`、`CheckTemplate/WaitTemplate`(删 MatchMode);mock/stub。

**InputService**(internal/node/interfaces.go + runtime inputAdapter):
- 新增 `Drag(x1,y1,x2,y2 float64, button string, durationMs int) error`(包 `pkg/input.MouseDrag`,已存在)。
- 新增 `TypeText(s string) error`(包新 `pkg/input.TypeText`)。
- `Scroll` 增轴向(竖/横)参数。

**pkg/input**:`MouseDrag` 已有;新增 `TypeText`(SendInput unicode)+ 横向滚动(WM_MOUSEHWHEEL)。StopApp 的 taskkill helper(可放节点旁或 sysutil)。

## 11. 单位约定:magnitude-switch(通用,首用于 OffsetX/Y)

按绝对值切单位,边界 `1`:`|v|≤1`=比例(`1`=100%,直接用)· `|v|>1`=像素(`v/clientW`、`v/clientH`)· `0`=零。
- `1` 收编为 100%(用户定),无 (1,2) 空档,`1.5`=1.5px(取用按整数舍入);负值合法(按绝对值判、留符号)。
- 抽 `internal/node` **唯一权威 helper**(`node.ResolveScalar(v,fullPx)→ratio`):本次用于 OffsetX/Y,通用可复用,但不预先改造现有 ratio pin(YAGNI)。
- 像素换算用 `ctx.Window().ClientSize()`(NeedsWindow 已注入)。
- 注:ROI 仍用现有 Geometry 类型(ratio+per-resolution override),不走 magnitude-switch。

## 12. 测试计划(TDD)

- **锚点**:mock vision 返固定 bbox → 9 Anchor × 偏移(像素/比例/负值)验落点;`center,0,0`==旧中心(回归);clamp01。
- **order_by/index**:mock MatchAll → 5 种排序 + Index 取对 + 越界继续等/超时;默认 `score/0` 走 WaitMatch(回归)。
- **ROI**:roi 透传给 Detect/MatchAll 的 region;全帧默认(回归)。
- **WaitTemplateGone**:先命中后 nil → Gone;一直命中 → Timeout 带 Conf;单帧两分支;取消 halt;多模板全消失才 Gone。
- **MatchMode 删除**:Check/Wait/ClickTemplate 多模板 OR;删/改原 `all` 旧测。
- **组合键+多击**:mock input 验 `KeyDown(ctrl)→KeyDown(shift)→Click×N(间隔)→KeyUp(shift)→KeyUp(ctrl)`;默认==旧单击(回归);非法 Keys/ClickCount<1 报错。
- **Swipe**:mock input 记录 `Drag` 参数。
- **InputText**:mock input 记录 `TypeText`;`pkg/input.TypeText` unicode 编码单测。
- **Scroll 横向**:mock input 记录轴向;默认竖直(回归)。
- **StopApp**:helper 按名/PID 组 taskkill 命令(可注入 exec 验参数);失败走 Fail。
- **catalog drift**:目录 i18n/结构测试随新 pin/新节点更新(`pnpm gen:node-i18n`)。
- 预存失败基线见 `knowledge/build/build.md`,判红时排除。

## 13. 验证 / 收尾

- `go build ./...` + `task build` 绿;前端 vitest(若触及)绿。
- node-catalog 重导出核对新 pin/4 个新节点(WaitTemplateGone/Swipe/InputText/StopApp)+ Scroll 改动齐全。
- 真机 smoke:① 锚点偏移点中 · ⑤ 多命中点最上/第 2 个 · ⑧ 限 ROI 找图 · ② 等图消失 · ③ ctrl+点 · ④ 双击 · ⑥ 拖滑块 · ⑦ 搜索框打字 · ⑨ 横向滚 · ⑩ 杀进程。
- 文档:有坑 → `knowledge/`;无坑则 catalog 自带。

## 14. 影响面 / 文件清单(预估)

- `internal/node/interfaces.go` — VisionService.Match/WaitMatch(去 mode、加 roi、+MatchHit/bbox);InputService 加 Drag/TypeText + Scroll 轴向。
- `internal/services/container/runtime/node_services.go` — visionAdapter(删 mode/all、透传 bbox+roi);inputAdapter(Drag/TypeText/Scroll 轴向);mock/stub 测试。
- `internal/nodes/detect/click_template.go` — ROI/Anchor/Offset/OrderBy/Index/Keys/ClickCount + 落点+排序 + clickAt 扩展。
- `internal/nodes/detect/wait_template_gone.go` — 新节点。
- `internal/nodes/detect/template_common.go` — 透传 bbox/roi。
- `internal/nodes/detect/{check,wait}_template.go` — 删 MatchMode + 适配新签名。
- `internal/nodes/input/click_at.go` — Keys/ClickCount + 公共 click helper。
- `internal/nodes/input/{swipe,input_text}.go` — 新节点;`scroll.go` — 加 Axis。
- `internal/nodes/io/stop_app.go`(或 input/)— 新节点 + taskkill helper。
- `pkg/input/input.go` — TypeText + 横向滚动(MouseDrag 已有)。
- catalog i18n(`frontend` gen:node-i18n)+ 各节点单测。
