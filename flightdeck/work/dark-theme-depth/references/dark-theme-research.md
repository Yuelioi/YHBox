# 暗色主题层次调研

## Research Read

问题是：Yotta 的设置中心和管理页怎样在保留近黑、专业、低干扰气质的同时，通过稳定表面层、文字层级和
克制强调色摆脱“所有区域都掉进同一个黑洞”的观感。

目标界面是高密度桌面工具，用户任务是扫描、编辑、比较、定位状态和持续操作。限制包括 dark-only、
Nuxt UI 语义 token、现有 emerald 主色、状态色含义以及避免卡片层层嵌套。

## Source Matrix

- [Carbon Color](https://carbondesignsystem.com/elements/color/overview/)：Gray 100 暗色主题从
  `#161616` 起步，嵌套层依次使用 Gray 90、80、70；官方明确说明暗色层级应逐级变亮，且一般不要把
  组件做得比所在背景更暗。
- [Carbon Color Usage](https://carbondesignsystem.com/elements/color/usage/)：用 contextual layer
  token 让同一组件随嵌套层级映射到正确表面；最多三层，重大对比才使用 inline theme。
- [Material 3 theming](https://developer.android.com/develop/ui/compose/designsystems/material3)：
  用 surface container 与 tonal elevation 表达暗色层级，提升层级时增加更显著的色调，而不是只依赖
  阴影；primary 主要用于关键动作、活动状态和表面 tint。
- [Material 3 Surface](https://developer.android.com/reference/kotlin/androidx/compose/material3/Surface.composable)：
  暗色 surface 的 tonal elevation 越高越亮，shadow elevation 只在确实需要与复杂背景分离时使用。
- [Primer Color Usage](https://primer.style/product/getting-started/foundations/color-usage/)：
  通过功能 token 区分 default、muted 等表面，暗色与亮色共享角色而非共享固定色值。
- [Primer Theme Reference](https://primer.style/product/getting-started/react/theme-reference/)：
  `dark_dimmed` 示例明确区分 default `#22272e`、overlay/muted `#2d333b`、inset `#1c2128`、
  default border `#444c56`，说明“暗”不等于所有区域同黑。
- [Primer Color Considerations](https://primer.style/accessibility/design-guidance/color-considerations/)：
  提供默认、dimmed、高对比及色觉适配等暗色子主题，强调 token 与对比度而非单一审美值。

## Patterns

1. **先有稳定实体层，再谈阴影。** 画布、一级内容面、交互抬升面和浮层应有可重复的明度关系；阴影只
   辅助 overlay，不负责静态页面分组。
2. **层级是角色，不是组件专属颜色。** 同一 `surface` token 可用于设置区块、管理表格和侧栏；同一
   组件进入下一层时使用 contextual role，避免每个页面发明灰色。
3. **暗色层通常向上变亮，凹陷内容才向下变暗。** 日志、轨道和代码区可使用 sunken；普通卡片不要比
   所在画布更暗。
4. **层数限制在三层以内。** 页面画布、一级实体区块、overlay/强交互面已经足够；字段、列表和详情不要
   再各自套完整卡片。
5. **颜色稀缺才有力量。** 主色留给关键动作、活动导航和焦点；分类色只做局部 wayfinding，错误、警告、
   成功和信息继续使用固定语义色。
6. **文字与边框有独立层级。** 主标题、正文、次级说明和 disabled 不能只靠背景解决；边框主要定义相邻
   近色面，避免每层都同时用高对比描边与阴影。
7. **状态必须比静态层更清楚。** hover、selected、focus、error、disabled 分别需要稳定 token 和非颜色
   线索；不能只让 hover“稍微亮一点”却让键盘焦点消失。

## Local Application

- 保留 `.dark --ui-bg = zinc-950` 作为最深画布，但为普通实体表面提供比画布明显、克制的稳定明度。
- `raised-surface` 改为实体表面是正确方向；hover 只提升一个预定义层，overlay 继续拥有真实投影。
- 设置中心可按主题使用极低剂量 tint，但 tint 应建立在相同的 surface 明度上，不能让七个 tab 变成七套
  主题，也不能占用 semantic status 的颜色含义。
- `SettingsSection` 只允许成为一级区域；其内部 collection、field、detail 均应平铺或用分隔线，避免
  再次形成卡片嵌套。
- 工作流和资产管理的侧栏、工具条、表格主体、表头与 hover 行应映射到同一组全局 surface 角色。
- 不修改编辑器画布的专用景深、节点分类调色板或语法高亮，它们有独立的功能识别语义。

## Next Step

以现有补丁为实验版本，在真实桌面尺寸中同时观察设置中心、工作流管理和资产管理；只调整全局 surface
角色、一级区块和交互状态，先验证层级是否清楚，再决定是否保留每个设置分类的轻 tint。
