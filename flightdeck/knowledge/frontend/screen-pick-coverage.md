# 截图拾取(取点/取坐标/取色)覆盖原则
## 原则(用户定)

**能有截图拾取的尽量有,方便用户。** 任何表达"屏幕上某个位置 / 区域 / 颜色"的输入,都应提供"截图后在画面上点/框/吸"的拾取入口 —— 别让用户对着窗口数像素再手填。

## 机制:拾取入口绑在**控件**上,不绑节点名

拾取按钮是各结构化控件**自带**的(镜像 GeometryWidget 自持 picker 的模式),所以**任何用该类型 pin 的节点自动获得**,无需在 NodeInspector 维护节点名白名单(旧的 `canPickPoint = kind==='ClickAt'||'Scroll'` 白名单已删)。

| 语义 | pin 类型 / widget | 自带拾取 | 当前覆盖 |
|---|---|---|---|
| 坐标点 | `Point`(Schema=`PointSchema()`)→ PointWidget | **截图取点**(`data-testid="point-pick-btn"`) | ClickAt · Scroll · MouseMoveTo · MouseHoldStart · Swipe(Begin/End) |
| 区域 | `Geometry` → GeometryWidget | 截图框选(rect) | DetectColor 等 |
| 颜色 | color widget | 屏幕吸色 | 颜色范围类 |

**关键**:`Point` pin 只要带 `Schema: node.PointSchema()`,前端经 `StructuredInput`(`schema.widget==='point'`)→ PointWidget,**就自动带取点按钮**。5 个坐标节点都该有;若某个看起来没有,先怀疑跑的是旧进程(没重启新 exe),而非代码缺失 —— 后端/前端对所有 Point pin 一视同仁,无单节点门控。

## 故意没有的

- **MakePoint**(组装坐标):输入是两个 `Number`(供计算/连线拼坐标),不是 `Point` pin,故无 PointWidget、无取点。要拾取直接在坐标节点的 Point pin 上点 —— MakePoint 是"用算出来的数字拼坐标"的路径,拾取与之语义不符。

## 加新坐标节点时

用 `Type: "Point"` + `Schema: node.PointSchema()` 声明坐标 pin → **自动获得截图取点 + %/px 单位开关**,零额外接线。区域用 `Geometry`,颜色用 color 类型,同理自动获得对应拾取。
