# UI (NuxtUI 偏好) checklist
写/改 `.vue` 组件 / Tailwind class 时**前置**读这份.

### 配色从哪来 — 新组件用色决策树 (2026-06-11 配色统一后)

色源只有两类: **chrome 色 = NuxtUI semantic token; 功能识别色 = 集中 TS 调色板**. 按顺序问:

1. **灰阶/文字/边框?** → 下面 semantic token 表 (`bg-default` / `text-muted` / `border-default` …)
2. **状态语义 (警告/错误/成功/提示/选中激活)?** → `warning` / `error` / `success` / `info` / `primary` (透明度照常: `bg-warning/10`)
3. **要比主背景更深 (终端/轨道/凹陷面)?** → `bg-sunken` (style.css 自定 utility)
4. **多彩"身份/区分"色 (节点分类/用户可选色板/pin 类型/日志 TAG)?** → 从集中调色板取, **不自己发明色相**:
   - 节点分类 + 用户色板: `visualRegistry.ts` (PALETTE / GROUP_VISUAL; class + hex 双形态)
   - pin 类型色: `nodeRegistry/index.ts` TYPE_COLOR
   - 日志 TAG/流: `logFormat.ts` / LogPanel
5. **Tailwind class 够不着 (CodeMirror JS 主题 / scoped CSS / 内联 style / canvas 绘制)?** → 直接写 `var(--ui-bg)` / `var(--ui-primary)` 等 CSS 变量; 要透明度用 `color-mix(in oklab, var(--ui-primary) 20%, transparent)`. 范例: `editorTheme.ts` (chrome 全走 var, 语法高亮是有意的 VSCode Dark+ 内容色)

**定义位置** (改基调只动这几处): 三个色相钉在 `vite.config.ts` ui.colors (primary=emerald / neutral=zinc / warning=amber) → NuxtUI 生成 `--ui-*` 变量; `bg-sunken` 在 `style.css`; 画布景深色 (背景渐变/网点/minimap mask) 是有意的非 token, 集中在 ContainerEditorView 带注释.

### 表面分层 — 偏冷近黑 + 实体明度阶梯 (2026-07-31)

暗色层级集中在 `style.css`，组件禁止散写 zinc/black/white。阴影不能替代暗色里的实体明度差：

1. **canvas** `bg-default` = `--ui-bg`，使用 OKLCH L17 的偏冷近黑，不再使用 zinc-950 纯黑。
2. **sunken** `bg-sunken` 只给终端、时间轴轨道等确实凹陷的内容边界。
3. **surface** `--ui-surface` 约 L21，供普通卡片、侧栏、表格主体和一级设置区块使用。
4. **hover / selected** `--ui-surface-hover` 约 L24，交互时必须比静态 surface 明显；选择态仍需文字、
   图标或边框等非颜色线索。
5. **strong / overlay** `--ui-surface-strong` 约 L26。真正遮挡内容的 overlay 可使用有位移的柔和阴影；
   静态卡片不再使用顶光、白色内高光或阴影伪造层级。

设置主题的低剂量 tint 建立在同一 surface 阶梯上，只用于导航、标题、焦点和 wayfinding；成功、警告、
错误和信息继续走语义状态色。

### 设置/表单类页面层级

设置中心统一使用 `SettingsView`、`SettingsPageHeader`、`SettingsSection` 和 `SettingsRow`，详细约定见
[settings-page-style.md](settings-page-style.md)。`SettingsSection` 只建立一级实体区块；其内部字段、
collection、详情和动作平铺或用分隔线组织，不得再套完整卡片。`AppCard` 留给独立列表项或真正可复用的
实体卡，不作为设置页默认分组外壳。

### 组件必须用 NuxtUI

有 NuxtUI 对应组件就用它. **禁止**裸 `<button>` / `<input>` / `<div role="dialog">` 自搭样式. 自研复合组件 (ConfirmDialog 等) 也只能由 NuxtUI 原子组合: `UButton` / `UInput` / `UModal` / `UPopover` / `UCheckbox` / `UTooltip` / `UTabs` / `UIcon` / `UTextarea` / `USelect` / `UDropdownMenu`.

**唯一例外**: 真无对应 NuxtUI 组件, 或其内部 dom 阻碍原生交互 (e.g. NodePalette 拖拽列表项 — `UButton` 包装层吞 drag 事件). 裸 HTML OK, 但视觉 token 仍走 `bg-elevated` / `text-toned` 等 semantic class.

### 通用 modal 用共享外壳 `components/common/BaseModal.vue`

普通 modal (确认 / 表单 / 面板 / 浏览器) **不要各自手写 `UModal` + `#content`**, 用 `BaseModal`:

- props: `open`(v-model) / `title` / `icon` / `iconColor`(primary|error|warning|success|info, 给删除/错误类上严重度色) / `size`(md..5xl) / `showClose`
- slots: 默认 = body、`#footer` = 按钮行、`#header-extra` = 标题右侧 (tabs / 步骤指示)
- 风格 = **纯黑平铺**: `bg-default` + header(border-b) + body 内容平铺(px-5 py-4) + footer(border-t)。层次靠**内容自身元素** (列表/卡片用 `bg-elevated` 自抬升), 外壳不套浅色面板/不上描边 (试过包裹面板/emerald 边, 用户定回纯黑)。
- 开关跟 `useDialogOpen()` 正交。`ConfirmDialog` (useConfirm Promise API) 也基于它。
- **例外不套 BaseModal**: 搜索面板 (CommandPalette / NodeSearch 结构特殊)、大复合 (TemplatePicker 资产浏览 / TemplateManager 预览)。
- **frameless 独立工具窗 (HUD) 不用这个** → 走 [standalone-window-style.md](../wails/standalone-window-style.md)。

### 永远 dark mode — 四件套一个不能少

1. **`<html class="dark">`** (index.html 静态写)
2. **`<div id="app" class="isolate dark">`** (`isolate` 创 stacking context; `dark` 双保险)
3. **`vite.config.ts` 配 colors + 已批准的全局组件基线**；不要在页面里重复修公共组件:

   ```ts
   ui({
     ui: {
       colors: { primary: 'emerald', neutral: 'zinc' },
       button: { slots: { base: 'justify-center' } },
     },
   })
   ```

   `UButton` 的全局主轴居中规则见 [nuxt-ui-icon-button-alignment.md](nuxt-ui-icon-button-alignment.md)。

4. **`main.ts` 强制 `useDark().value = true`** — **最关键**. NuxtUI v4 plugin 内部用 @vueuse `useDark()`, 启动时读 localStorage + 系统 `prefers-color-scheme` 动态改 `<html>` class, 会把 index.html 的 `class="dark"` 覆盖. 不锁就在用户 light mode 下全失效.

   ```ts
   app.use(pinia).use(router).use(ui).use(i18n)
   useDark().value = true  // 锁 dark
   ```

### 只用 semantic token, 禁止裸 zinc/black/white

| 视觉            | 用                                                    |
| --------------- | ----------------------------------------------------- |
| 凹陷面 (比主背景更深: 终端日志区/时间轴轨道) | `bg-sunken` (自定 utility, style.css 从 --ui-bg 往黑混) |
| 主背景 (最深 semantic) | `bg-default`                                   |
| 卡片/弹层       | `bg-elevated`                                         |
| 分隔区/muted    | `bg-muted`                                            |
| 强调背景 (最高) | `bg-accented`                                         |
| 主文字 (最亮)   | `text-highlighted`                                    |
| 普通文字        | `text-default`                                        |
| 强调文字        | `text-toned`                                          |
| 弱化文字        | `text-muted`                                          |
| 提示文字 (最弱) | `text-dimmed`                                         |
| 边框            | `border-default` / `border-muted` / `border-accented` |

状态色: `bg-primary` / `bg-error` / `bg-warning` / `bg-success` / `bg-info`. **状态语义 (警告/错误/成功/提示/选中激活) 一律走这些, 不许直写 `text-amber-300` / `bg-rose-500/10` 这类色相**; 透明度照常加 (`bg-warning/10`)。直写色相只允许"功能识别色"且必须来自集中调色板: 节点分类 (visualRegistry PALETTE) / pin 类型 (TYPE_COLOR) / 日志 TAG·流 (logFormat / LogPanel) / 编辑器语法色 (editorTheme)。编辑器 chrome 色 → editorTheme.ts 只写 `var(--ui-*)`。

**禁用 list** (写新组件前 grep 自己):

- `bg-zinc-*` / `bg-black` / `bg-white`
- `text-zinc-*` / `text-white` / `text-black`
- `border-zinc-*` / `ring-zinc-*` / `divide-zinc-*`

**唯一例外**: 录制叠加层这类"完全黑" scrim. 模态遮罩 **不要** override — Modal / Slideover 自带 overlay slot, 信任默认.

### 输入框宽度: 默认撑满, 短输入直接加宽度类

style.css 全局把含 input/textarea 的 NuxtUI 包装层默认 `width: 100%` (表单不用处处写 w-full), 这条放在 `@layer components` — **组件上直接写 `w-36` / `max-w-*` 就能压过它** (utilities 层更后), 不用包 div。数字步进 / 端口号 / 短数值类输入**不要**放任它撑满 (UInputNumber 拉满一行, ± 按钮分居两端很难用), 给个 `w-28`~`w-40`。

**底层规律 (踩过两次)**: Tailwind v4 工具类全在 `@layer utilities`, 而 style.css 里**不分层的自定义规则永远压过任何工具类 — 跟优先级数值无关, `:where()` 也救不了**。想让工具类能覆盖的全局默认值, 必须写进 `@layer base/components`; 想压过工具类的 (如改 display), 留在层外。

### 自定义 `@utility` 能加 variant 前缀 (v4 实证)

`style.css` 里 `@utility raised-surface { ... }` 这类自定义工具类, **可直接加 `hover:` / `focus:` 等 variant** —— `hover:raised-surface` 真生效 (v4 自动给 `@utility` 生成 variant), 不必为了 hover 态把配方拆成普通 class 再手写 `:hover`。2026-06-14 UI 升级 Part 1 ListRow 实证 (`raised-surface` / `overlay-surface` / `bg-sunken` 同理)。
> 副记: 想用 DOM 验「某 class 到底生没生成 CSS 规则」, 必须**递归** `CSSLayerBlockRule.cssRules` —— v4 把 utilities 全塞进 `@layer`, 平铺遍历 `document.styleSheets[].cssRules` 查不到 (假阴性)。离屏视觉自检时按此判。

### `:ui="{ base: '...' }"` 是替换不是 merge

调 NuxtUI 组件加 width / 样式 → `class="w-24"`, **不是** `:ui="{ base: 'w-24' }"`. 后者会把 base slot 默认那一长串 (`bg-default text-default ring-default ...`) 清空, 只剩 `w-24`, 背景变裸 HTML 白色. 踩过 Step editor 输入框白底的坑.

固定尺寸 icon button 同理写 `<UButton icon="..." class="size-7 p-0" />`，居中由全局 Button base 保证；需要左对齐的菜单/导航按钮显式加 `justify-start`。

### `UInputMenu` 让用户创建新项 → `create-item` + `@create`, 不是 `creatable`

`creatable` 是 NuxtUI v2/headless 旧 prop 名, **v4 改名 `create-item` 且静默忽略旧名** → 写 `creatable` 不报错但创建项压根不出现 (下拉只显 "No matching data", 用户输新分类/标签建不出来)。v4 正确写法:

- `:create-item="'always'"` (输入值非现有精确项就显「创建」; 写 `true` 只在候选**全空**时才显, 部分匹配时反而不给创建, 不够)
- **必须配 `@create` 处理器** —— 选「创建」只 `emit('create', searchTerm)`, **不自动写 model-value**。单值: `@create="(v) => setX(v)"`; 多值 (`multiple`): `@create="(v) => model = [...model, v]"`。
- **单值 menu 还要让新值进入 `items`**。`@create` 只给原始文本, Nuxt UI 不会替你把新项 push 进候选列表；如果 `v-model` 设成一个不在 `items` 里的值, 后续可能显示/选择状态不稳定。做法: 维护 `createdX` 本地数组, `items = unique(existing, createdX, [model])`, `@create` 里先 trim/dedupe/push, 再把 model 设成该值。容器分类用 `categoryOptions.ts` 的 `addCreatedCategory()` / `uniqueCategoryOptions()` 锁这个规则。

范例: `SwitchInspector.vue` (`:create-item="'always'"` + `@create`)。2026-06-13 全仓 13 处 `creatable` 全是坏的 (分类/标签建不了新项), 已统一修 (含 ContainerSettings 一处有 create-item 但漏 @create 同样建不出)。

### 普通枚举 Select 宽度要同时计算 label 与控件 chrome

Nuxt UI 的 Select 弹层默认跟随 trigger 宽度。基于最长选项做自适应时，不能只算文字：leading icon、gap、左右 padding 和 trailing chevron 都占固定空间；中日韩文字还要按双宽估算。项目普通枚举统一使用 `AdaptiveSelect`，最长 label 之外预留共享 chrome 余量并设置最大宽度。

页面不要再给需要自适应的筛选器叠加 `w-*` + `width-mode="fixed"`，否则共享计算会被直接绕过。只有每页数量、紧凑日志级别或明确受表单栅格约束且已验证完整显示的控件才使用 fixed；数百条实体选择仍走可搜索 picker，不用 Select。

### UFormField 的 hint 不是字段说明

`UFormField.hint` 与 label 同行，只适合“可选”“只读”等极短状态；不要放完整句子。字段用途、约束和
操作建议一律放 `description`，显示在 label 下方，窄侧栏里也不能与参数名争夺横向空间。检查器使用
12px 参数名、11px 次级说明、再下方独占一行的控件；必填状态使用 `required` 的 `*`，不要把
`required/optional` 当说明正文。

### 反馈方式: 一个事件只用一个反馈表面

用户明确不喜欢成功类 toast (顶上弹出干扰视线、和操作点割裂)。**新代码不加成功 toast** —— 按这棵决策树:

1. **操作结果在当前视野立即可见?** (删除→列表项消失 / 自动布局→画布重排 / 折叠/融合/导入→画布变 / 重拍→详情图更新) → **什么都不弹**, 变化本身就是反馈。
2. **复制到剪贴板** (不可见) → **优先原地反馈, 别 toast**:
   - 触发点是**持久按钮** = 按钮内联闪「已复制 ✓」(本地 `copied` ref + ~1500ms 恢复, icon 切 check, 文案 `common.copied`)。
   - 触发点是**下拉菜单项** = `onSelect(e)` 里 `e.preventDefault()` **留住菜单**, 被点项 label/icon 原地切「已复制 ✓」~1500ms (范例 `NodeInspector.vue` 复制下拉)。菜单能留住就别弹 toast。
   - 仅当触发点**真的留不住**(点完必然关闭、无处可显) 才退而用短 toast (`duration: 1500`)。
   - 失败如果已由当前字段或区域内 alert 持久展示，不再重复 toast；其余错误才用 toast。
3. **有明确持久触发按钮的操作** (保存) → 按钮内联「已保存 ✓」(参 ContainerEditorToolbar 保存按钮 + useEditorSave.saveFlash, success soft ~1.6s 恢复)。
4. **后台/跨窗口/跨位置事件, 或系统替你做了非显式动作的告知** (热键触发运行入队、导出/分享到库、录制落盘、自动重连了线、校准值变更) → **保留 toast** (这是 toast 的正当用途)。
5. **错误/警告** → 字段级错误放字段旁；当前区域可恢复错误用一条 inline alert；后台或跨位置失败才用 toast；阻塞且需要选择才用 modal。完整规则见 [feedback-surfaces.md](feedback-surfaces.md)。

2026-06-11 已按此把全仓 ~20 处成功 toast 收口 (删/内联), 并清掉随之死掉的 i18n 文案 key。

写完新 view 前跑:

```bash
grep -rn "bg-zinc-\|text-zinc-\|border-zinc-\|bg-black\|bg-white\|text-white\|text-black" frontend/src/
```

必须**零行**.
