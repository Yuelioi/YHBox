# Phase 4:Point 手填控件 + WaitWindowGone 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development 逐 task 实现。步骤 `- [ ]`。两 task 独立;i18n 都改 zh.ts/en.ts,顺序做避免冲突。

**Goal:** (4.1) 新节点 WaitWindowGone(等窗口关闭,对称 WaitWindow);(4.2) 给 Point 类型补手填控件(x/y 两个 0-1 小数框),Swipe 起止点既能手填又能连线。

**两块来由:** 用户 review Phase 3 后拍板 —— Swipe 的 Point 起止点该能手填 x/y(现状只能连线 / 退化成文本框);StopApp 该能配"等它真关掉"。

## Global Constraints

- **不要兼容**:不留 shim。
- **TDD**:节点逻辑用 mock/可注入 seam 验出口;纯 WinAPI 投递层(窗口枚举)无 harness → 节点级 mock + 真机 smoke(诚实标注)。前端用 vitest(`cd frontend && ./node_modules/.bin/vitest run <路径>`,别用 `pnpm -C frontend test` 会炸 ENOENT)。
- **构建/验证**:`go build ./...` + `go test ./internal/...`;改 catalog → `cd frontend && pnpm gen:node-i18n` + 先补 zh.ts/en.ts 源 + `go test ./internal/catalog/...`;前端 `cd frontend && pnpm vue-tsc --noEmit`(类型)+ vitest。
- **验证基线**:见 `flightdeck/knowledge/build/build.md`; 当前 Go/前端测试应绿, 旧 runtime fixture 红记录已过期。
- **本机 Write 故障**:写文件尾可能混入 `</content>`,写完检查;改既有文件优先 Edit;诊断面板编辑期 stale,以 `go build`/`go test` 退出码为准。
- **坐标系**:Point/Rect/Geometry 全是 ratio 0-1(`node.Point` = `{X,Y float64}` ratio)。GeometryWidget 给用户显示 **0-100 百分比**(内部 ×100 显示 / ÷100 存),PointWidget 对齐这个 UX。

---

### Task 4.1: WaitWindowGone(新 System 节点 + winutil 等消失)

等指定窗口从系统消失再放行。对称 WaitWindow(等出现)。

**Files:**
- Modify: `pkg/winutil/window.go`(抽单帧查找 + 加 WaitWindowGone)
- Create: `internal/nodes/system/wait_window_gone.go` + 测试
- i18n: `frontend/src/i18n/zh.ts`、`en.ts`(加 WaitWindowGone 块)

**现状锚点(读了再动):**
- `WaitWindow`(`internal/nodes/system/wait_window.go`):Run 调 `winutil.ResolveWindow(ctx, spec, timeout, interval)`;返回 nil=找到→Found;`ErrWindowNotFound`=超时没出现→Timeout;其它(空 spec/regex 非法/ctx 取消)→裸 err。pin: Title/Class/ProcessName/TitleMatch(exact/regex)/TimeoutMs。`wwPollInterval = 500ms`。
- `winutil.ResolveWindow`(`pkg/winutil/window.go:140`):内部 EnumWindows callback 做单帧匹配(可见窗口 + Title/Class/ProcessName 匹配),poll 直到找到或超时。单帧枚举逻辑**内联**在 callback 里,需抽出复用。已有 `IsEmptyMatch`/`CompileTitle`。WaitWindow **无单测**(纯 WinAPI)。

- [ ] **Step 1:winutil 抽单帧 + 加等消失**
  - 把 ResolveWindow 内的单帧枚举匹配抽成可复用 `windowMatchOnce(spec MatchSpec, titleRe *regexp.Regexp) bool`(枚一遍可见窗口,任一匹配→true)。ResolveWindow 改用它(零行为变化,确认现有调用方仍过)。
  - 加 `func WaitWindowGone(ctx, spec MatchSpec, timeout, interval time.Duration) error`:空 spec→error(同 ResolveWindow 的 IsEmptyMatch 守卫);CompileTitle;循环每 interval 查一次 `windowMatchOnce` —— 不匹配(窗口没了)→ return nil;ctx 取消→ctx.Err();超 timeout 仍匹配→ return `ErrWindowStillPresent`(新 sentinel)。一开始就没→立即 nil。
- [ ] **Step 2:写失败测试** — winutil 纯逻辑可测的部分(IsEmptyMatch 守卫 / 空 spec→err);节点出口路由用**可注入 seam**:节点里 `var waitWindowGone = winutil.WaitWindowGone`,测试替换成 mock(返回 nil→Gone / ErrWindowStillPresent→Timeout / 其它 err→裸冒泡)。
- [ ] **Step 3:实现节点** `internal/nodes/system/wait_window_gone.go`:
  - 仿 wait_window.go。`Kind:"WaitWindowGone"`,`Category:"System"`。pin 同 WaitWindow(Title/Class/ProcessName/TitleMatch/TimeoutMs,TimeoutMs 默认可设 10000)。出口 **Gone** / **Timeout**(无 Fail;同 WaitWindow 注释逻辑:空 spec/regex 非法/ctx 取消 = 裸冒泡)。
  - Run:`waitWindowGone(ctx.Context(), spec, timeout, wwgPollInterval)`;nil→Gone;`errors.Is(err, winutil.ErrWindowStillPresent)`→Timeout;else→裸 err。
- [ ] **Step 4:测试通过** — `go test ./internal/nodes/system/... ./pkg/winutil/...`(真窗口枚举靠 smoke,诚实标注)。
- [ ] **Step 5:i18n** — zh.ts/en.ts 加 WaitWindowGone 块(label「等待窗口关闭」/「Wait window closed」,description 大白话:按标题/类名/进程名等某窗口从系统消失再放行,常接在「关闭程序」后确认真关掉;pin Title/Class/ProcessName/TitleMatch/TimeoutMs;output Gone/Timeout)。`pnpm gen:node-i18n` + `go test ./internal/catalog/...`。
- [ ] **Step 6:提交** — `git commit -m "feat(system): WaitWindowGone — 等窗口从系统消失 (对称 WaitWindow)"`

---

### Task 4.2: Point 手填控件(框架级,前端为主)

给 `Point` 类型补手填控件(x/y 两个 0-1 小数框),既能手填又能连检测节点的点输出。

**后端早已支持手填值解析**(关键,别重复造):`internal/services/container/runtime/data_pull.go` 的 `coerceToType(v, "Point")` 已把 `map{x,y}` / `[x,y]` / `expr.Point` 转成 `node.Point`(有测试 `TestCoerceToType_Point`)。所以手填的 Point 字面量(前端存 JSON `{x,y}`)运行时会被正确还原,`in.Point` 拿得到。**后端不动 in.Point / coerce。**

**Files:**
- Modify(后端,极小): `internal/node/schema.go`(加 `PointSchema()`)、`internal/nodes/input/swipe.go`(Begin/End 加 Schema)
- Create(前端): `frontend/src/components/containers/inline/PointWidget.vue` + vitest
- Modify(前端): `StructuredInput.vue`(加 point 分支)、`ContainerFlowNode.vue`(inlineLiteralPins 放行 point)、必要的 coerceLiteral / 类型映射
- i18n: 无新文案(Swipe 已有 Begin/End label;PointWidget 内 X/Y 标签可硬编或走 geometry.* 风格 key)

**现状锚点(读了再动):**
- `node.GeometrySchema()`(`internal/node/schema.go:37`)= `&FieldSchema{Type:"object", Widget:"geometry"}`。Geometry pin 用 `Type:"Geometry", Schema: node.GeometrySchema()`。
- 前端 `StructuredInput.vue`:`schema.widget==='geometry'` → `<GeometryWidget>`;`schema.type==='object'`/`'tuple'` → 递归字段 + JSON 文本模式切换。
- `GeometryWidget.vue`:x/y/w/h 用 `UInputNumber`,**显示 0-100(内部 `p.x*100` 显示 / `v/100` 存)**;值结构 `{pct:{x,y,w,h}, overrides}`;props `modelValue`/`fieldPath`;emit `update:modelValue`;有"截图框选"按钮。
- `ContainerFlowNode.vue` `inlineLiteralPins`(~205):`p.type !== 'point' && fieldFor(p.name)?.schema?.widget !== 'geometry' && ... !== 'code'` 才进内联框集合 —— **point 被一刀切排除**。
- `types.go:99`:Point 注册 `WidgetKind:"point-editor"`,但前端 `adapter.ts` 无此分支 → fallback `'text'`(这就是 point 退化成文本框的原因)。
- `node.Point` JSON 形状 = `{x, y}`(`types.go:7`)。

- [ ] **Step 1:后端 PointSchema** — `internal/node/schema.go` 加 `func PointSchema() *FieldSchema { return &FieldSchema{Type:"object", Widget:"point"} }`;`swipe.go` 的 Begin/End 改 `{Name: swInBegin, Type:"Point", Schema: node.PointSchema()}`(End 同)。`go build ./...` + catalog 测试确认 Swipe schema 带上 point widget。
- [ ] **Step 2:前端 PointWidget** — 新 `PointWidget.vue`:仿 GeometryWidget 砍成 **x、y 两个 UInputNumber**,label「X %」「Y %」,显示 0-100(`x*100` 显示 / `v/100` 存,round4),min 0 / max 100 / step 0.1;值结构 `{x, y}`(0-1);props `modelValue: PointValue | null`(`{x:number,y:number}`)/`fieldPath`;emit `update:modelValue`;空值默认 `{x:0,y:0}`。(可选增强:加"屏幕取点"按钮,复用 picker —— 本 task **不做**,先纯数字框,YAGNI。)
- [ ] **Step 3:StructuredInput 路由** — `StructuredInput.vue` 顶部加 `schema.widget==='point'` → `<PointWidget>`(仿 geometry 分支,最高优先级之一)。
- [ ] **Step 4:放行内联编辑** — `ContainerFlowNode.vue` `inlineLiteralPins`:让有 `schema.widget==='point'` 的 pin 进集合(不再因 `p.type==='point'` 一刀切排除)。改判断:point 类型**且有 point schema** 的走结构化内联(同 geometry);确认连线时不显示框(现有 unconnected 逻辑已处理)。Inspector(NodeInspector)路径确认也走 StructuredInput → PointWidget。
- [ ] **Step 5:测试** — 前端 vitest:PointWidget(0.5→显示 50;输 25→存 0.25;空→默认 0,0;emit `{x,y}`);StructuredInput point 路由。`pnpm vue-tsc --noEmit` 类型过。后端 `go test ./internal/node/... ./internal/nodes/input/... ./internal/catalog/...`。
- [ ] **Step 6:验证手填 + 连线** — vitest/类型层确认:Swipe Begin/End 出现 x/y 百分比框、手填存 {x,y}、连线时藏框。运行时 in.Point 拿到手填值(coerce 既有,有 TestCoerceToType_Point 背书,无需新后端测试)。
- [ ] **Step 7:提交** — `git commit -m "feat(frontend): Point 类型手填控件 (PointWidget x/y 百分比) — Swipe 起止点可手填+连线"`

---

## 收尾(两 task 后)

- catalog 重导出核对 WaitWindowGone + Swipe Begin/End 的 point widget。
- 真机 smoke:WaitWindowGone(开记事本→等它标题的窗口关闭)· Swipe 手填 x/y 拖拽 · Swipe 连 ClickTemplate 的点拖拽。
- 知识沉淀(如值得):Point 手填走 coerce 既有支持(后端零改)、point-editor WidgetKind 接通方式 → `knowledge/`。
