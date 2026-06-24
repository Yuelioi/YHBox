# Point 坐标 %/px 单位 + 截图取点(4 坐标节点统一到 Point 类型)

SUMMARY: 给 node.Point 加 Unit(%/px),px→ratio 只在 InputService.ResolvePoint 一处换算;ClickAt/Scroll/MouseMoveTo 由 XRatio/YRatio slider 统一成单个 Point pin,Swipe 沿用;PointWidget 自带单位开关+截图取点;新增 PureFunc 节点 MakePoint(2 Number+Unit→Point)恢复 X/Y 分别连线。
READ WHEN: 实现/改 Point 坐标节点、px 像素坐标、PointWidget、屏幕取点(point picker)、ResolvePoint 时。

---

## 目标 / 诉求(用户)

1. Point 类型支持**像素**语义(不只百分比),用**显式单位**(不靠值大小魔法)。
2. 所有带坐标、能截图取点的节点都尽量支持,方便用户。

## 范围

带坐标的**输入**节点共 5 个,全部统一到 `Point` 类型:

| 节点 | 现状 | 改成 |
|---|---|---|
| ClickAt | `XRatio`/`YRatio` 两个 slider Number pin | 1 个 `Point` pin(Schema=PointSchema) |
| Scroll | 同上 | 1 个 `Point` pin |
| MouseMoveTo | 同上 | 1 个 `Point` pin |
| MouseHoldStart | 同上(`MouseDown(x,y,btn)`) | 1 个 `Point` pin(执行期补漏:设计普查漏读 mouse_hold.go) |
| Swipe | 已是 Point(Begin/End) | 结构不动,自动获得单位+取点 |

> **MouseHoldStop 不受影响**(只松开按键、无坐标)。

**不在范围**:
- 检测节点的 Point 是**输出**(比例),不动:detect_color / detect_color_blobs(primaryCenter)/ wait_template / click_template / check_template / find_template_all / find_color_signature。
- `detect_color_blobs.RefPoint`(Advanced 排序参考点,比例),不动。
- backend 原语、Drag 接口、检测节点一行不改(见 §3)。
- **联合类型 / 类型系统安全(YAGNI 延后,不并入)**:现有 `PinTypeCompat`(`validator_pin_types.go`:相等/`any` 放行 + number↔bool↔string 转换警告)+ `*` 通配已够用;Log 的 `Message:*` 真的什么都能收(`fmt.Sprint`),换 `[number,string]` 反是退化。等出现"必须限定 2-3 种类型、接错有静默 bug"的具体节点再单开一条线。

## 1. 数据模型(Go)

`internal/node/types.go` 的 `Point` 加 Unit,用**具名类型 + 常量**(不撒裸 `"px"`/`""` 魔法字面量,Go 侧编译期可检):

```go
type PointUnit string
const (
    UnitRatio PointUnit = ""   // 百分比/比例 (X/Y 0-1) — 默认
    UnitPx    PointUnit = "px" // 客户区像素 (X/Y 像素值)
)
type Point struct {
    X    float64   `json:"x"`
    Y    float64   `json:"y"`
    Unit PointUnit `json:"unit,omitempty"`
}
```

- 单位是**值的属性,不是类型的属性** → 不拆两个类型,连线 / 检测输出全兼容(CSS `<length-percentage>` 路子)。
- **`""`=比例默认是有意为之**:检测节点输出的 Point 无单位(就是比例),`omitempty` + 空默认让它们天然=比例、零改动。
- `PointSchema()`(schema.go)不变(单位在值里,不在 schema)。
- coercion `asNodePoint`(`internal/services/container/runtime/data_pull.go:244`):map 分支读 `t["unit"]` 填 `Point.Unit`(转 PointUnit);`expr.Point` 分支无单位(算出来的比例),Unit 留空;`node.Point` 分支原样(已带 Unit)。
- §2 ResolvePoint、§7 MakePoint 的 Unit 判断全走 `UnitPx`/`UnitRatio` 常量。

## 2. px→ratio 换算:只在一处(node.ResolvePoint helper,复用 ctx.Window().ClientSize())

> **设计修订(规划期读源码发现,优于原 InputService 方案)**:`ctx.Window().ClientSize()`(`WindowService`,`interfaces.go:296`)已在每个节点 Ctx 上、已在各处 stub;`click_template.go:225` 已用它做 px→ratio。故**不动 InputService**(省接口改 + 6 个 stub),改用 node 包一个共享 helper。单一来源不变。

新增 `internal/node` 的 helper(放 `scalar.go` 旁或同文件):

```go
// ResolvePoint 把 Point 按 Unit 归一成 0-1 客户区比例。
// UnitRatio → 原样(不碰窗口);UnitPx → 取 ClientSize 后 px÷宽高。
func ResolvePoint(ctx Ctx, p Point) (xRatio, yRatio float64, err error) {
    if p.Unit != UnitPx {
        return p.X, p.Y, nil
    }
    w, h, err := ctx.Window().ClientSize()
    if err != nil {
        return 0, 0, err
    }
    if w <= 0 || h <= 0 {
        return 0, 0, fmt.Errorf("client size %dx%d invalid for px point", w, h)
    }
    return p.X / float64(w), p.Y / float64(h), nil
}
```

- **不复用 `node.ResolveScalar`**:它带 `|v|≤1 ⇒ 当比例` 的魔法启发(用户明确拒绝的那套),对显式 px 的小像素值(如 px=1)会误判成比例。显式单位下直接除,不走启发。

4 个节点的 `Run` **开头**调一次 `node.ResolvePoint(ctx, pt)` 拿比例,**err 直接 return**(px 模式下窗口/客户区失败要上报到节点层),之后全保持纯比例:
- ClickAt:`xr,yr,err := node.ResolvePoint(ctx, pt); if err!=nil {return nil,err}` → `moveCursor(ctx,xr,yr,...)` + `node.ClickWithMods(ctx, node.Point{X:xr,Y:yr}, ...)`。
- Scroll:`xr,yr,err := node.ResolvePoint(ctx, pt)` → `ctx.Input().Scroll(xr,yr,...)`。
- MouseMoveTo:`xr,yr,err := node.ResolvePoint(ctx, pt)` → `moveCursor(ctx,xr,yr,...)`。
- Swipe:Begin/End 各 ResolvePoint(各自查 err) → `ctx.Input().Drag(bxr,byr,exr,eyr,...)`。

**不变**:`moveCursor`(takes ratio)、`ClickWithMods`(takes ratio Point,detect 共用)、`Scroll`/`Drag`/`MoveTo` 接口、**InputService 接口(不加方法 → 零 stub 改动)**、backend、检测节点。
- ratio 点(Unit 空)走 early-return 不碰 Window → 现有 ratio 测试零改动;只有新增 px 测试需 stub ClientSize(node 包已有 `stubWindowService`,detect 包已有 `stubWindow{w,h}` 先例)。

## 3. 控件 PointWidget(`frontend/src/components/containers/inline/PointWidget.vue`)

当前纯 X%/Y% 两个数字框。加:

- **%/px 单位开关**(整点一个单位,非 X/Y 各自)。
- **切单位保留框里数字、不换算**(前端拿不到游戏窗口实时尺寸):
  - `%`(unit 空):框显示 x×100(min0 max100 step0.1),存 ÷100;
  - `px`(unit="px"):框显示原始像素值(min0,无 max,step1),存原值。
- **截图取点按钮**(控件自带,镜像 GeometryWidget 自持 picker):直接 `backend.tools.openScreenPicker('point', id, tplStore.containerId)` → `awaitWailsEvent('tools:picker-result', p=>p.id===id)`:
  - `%`:存 `{x:xRatio, y:yRatio}`(unit 空);
  - `px`:存 `{x:xRatio×screenW, y:yRatio×screenH, unit:'px'}`。

→ 任何 Point pin(ClickAt/Scroll/MouseMoveTo/Swipe.Begin/End)经 StructuredInput(`schema.widget==='point'`)渲染,**自动获得**单位+取点,不靠节点名白名单。

`PointValue` 类型(`nodeRegistry/index.ts`)加 `unit?: 'px'`。

## 4. 截图回传尺寸(`ScreenPickerView.vue` 已就绪,无需改 view)

**规划期核实**:`ScreenPickerView.vue:765-773` 的 point 分支**已经 emit `screenW/screenH`**(= `natW.value`/`natH.value`,客户区像素)。所以 view 端 0 改动。
只需:PointWidget 自持 picker 时,接收 payload 里的 `screenW/screenH` 做 px 换算;PointWidget 内部声明的 PointPayload 类型带上 `screenW?/screenH?`。

## 5. 删旧路径(no-compat 铁律)

取点搬进 PointWidget 后,NodeInspector 顶部的 point 取点路径失效,删:
- `useScreenPick.ts`:删 `canPickPoint`/`onPickPoint`/`applyPoint`(rect/color 路径 `canPickRect`/`onPickRect`/`onPickColor` 不动)。
- `NodeInspector.vue`:删顶部"屏幕拾取→选点"按钮(line ~138-148)、`v-if="canPickPoint || canPickRect"` 收成 `canPickRect`、useScreenPick 调用里删 `applyPoint`(line ~1077)。

## 6. i18n / catalog

ClickAt/Scroll/MouseMoveTo 的 input pin 由 `XRatio`/`YRatio` → 单个 `Point` → label key 变;PointWidget 新增单位开关 / 取点按钮文案;**新增 MakePoint 节点**的 label/description/input(X/Y/Unit + 选项)文案。改 catalog:先补 zh.ts/en.ts → `cd frontend && pnpm gen:node-i18n` → `go test ./internal/catalog/...`(见 build.md)。

## 7. 构造节点 MakePoint(把"失去 X/Y 分别连线"抹平)

新增 PureFunc 节点 `MakePoint`:两个 Number(+ 一个 Unit 选项)→ 一个 Point 输出。谁要动态算坐标(连 Div/Add/检测输出等)就用它拼 Point 再喂坐标节点,不用把单位/连线复杂度塞进每个坐标节点。符合通用框架 building block 路子 + YAGNI 例外(价值高、量极小)。

- 位置:`internal/nodes/purefunc/`(IsPureData,套现有 `specBuilder`)。
- 输入:`X`(Number,默认 0)、`Y`(Number,默认 0)、`Unit`(dropdown:百分比 / 像素,默认百分比)。
- 输出:`Result`(Point)。
- `Evaluate`:`node.Point{X: in.Float64("X"), Y: in.Float64("Y"), Unit: <Unit 映射>}`;Unit 选项映射到 Point.Unit(百分比→空串、像素→`"px"`),dropdown 值编码细节实现时定(避免空串 option key)。
- 语义:X/Y 按 Point 内部表示**逐字透传**(百分比模式 = 0-1 比例原值,不×100;像素模式 = 像素值)。PointWidget 字面编辑里的 ×100 只是显示便利,不影响数据层。
- 反向 Split(Point→X/Y)用户没要 → YAGNI,不加。

## 8. 取舍(已被 §7 抹平)

ClickAt/Scroll/MouseMoveTo 从 **slider(0-1)+ 可分别连 Number** → **1 个 Point pin(PointWidget 两个框)**:失去 slider 手感;X/Y 分别连线由 §7 的 `MakePoint` 恢复(两个 Number → Point)。换来:px 语义 + 截图取点 + 可直接连检测节点的 Point 输出。

## 测试

- Go:`node.ResolvePoint` 单测(UnitRatio early-return 不碰 Window;UnitPx 经 stub ClientSize 验 px÷宽高;ClientSize err 上报);ClickAt/Scroll/MouseMoveTo node 测试改读单个 Point pin;`asNodePoint` unit coercion 单测;`MakePoint` Evaluate 单测(X/Y/Unit→Point,含 Unit 映射)。**InputService stub 不动**(§2 改后无接口变更)。
- FE:PointWidget 单位切换 / px 显示不×100 / 取点 %与px 存值 单测;coerceLiteral('point') unit 透传(现有测试已是 identity,补 unit case)。
- 预存红基线照 build.md 排除(runtime fixture 缺失那几个)。

## 关键源码索引(实现时直达)

- `internal/node/types.go:7`(Point)、`schema.go`(PointSchema)、`inputs.go:158`(in.Point)。
- `internal/node/interfaces.go:249-267`(InputService,**不改**)、`:296`(WindowService.ClientSize)、`click_mods.go:48,64`(ClickWithMods 纯比例)、`scalar.go:6`(ResolveScalar,**不复用** — 带 |v|≤1 启发)。
- `internal/nodes/detect/click_template.go:222-229`(已有 px→ratio 经 ctx.Window().ClientSize() 先例)。
- `internal/nodes/input/`:click_at.go、scroll.go、mouse_move_to.go(moveCursor:67 纯比例)、swipe.go:62(Drag)。
- `internal/services/container/runtime/node_services.go:286-350`(inputAdapter,有 hwnd)、`data_pull.go:244`(asNodePoint)。
- `internal/nodes/purefunc/purefunc.go`(specBuilder + init 注册 + Evaluate 模板,MakePoint 加这里)。
- FE:`inline/PointWidget.vue`、`inline/GeometryWidget.vue`(自持 picker 参考)、`inline/StructuredInput.vue:11`(widget==='point'路由)、`composables/containerEditor/useScreenPick.ts`、`views/tools/ScreenPickerView.vue`、`components/containers/NodeInspector.vue:138/1077`、`nodeRegistry/index.ts`(PointValue/PinType)、`nodeRegistry/adapter.ts:98`(Point→point)。
