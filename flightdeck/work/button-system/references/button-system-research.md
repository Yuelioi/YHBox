## Research Read

研究问题：Yotta 的工作流、资源库和编辑器属于高密度桌面生产力界面，用户需要快速分辨“当前任务的
主动作”“局部工具”“当前位置”和“危险操作”。现状把所有 `primary + solid` 都渲染为亮绿色渐变块，
导致新建、录制、运行和分页选中态获得相同且过高的视觉权重。

约束：继续使用 Nuxt UI 与 Tabler 图标；适配现有暗色 surface token；保留 loading、disabled、键盘
focus 和图标按钮的可访问名称；不引入包装式 Button 组件。

## Source Matrix

### 官方设计系统

- [Fluent 2 Button](https://fluent2.microsoft.design/components/web/react/core/button/usage)：一个布局只保留
  一个 primary；大量次要动作应使用 outline、subtle 或 transparent，避免工具界面变得繁忙；文字和
  图标在交互状态下分别满足 4.5:1 与 3:1 对比度。
- [Carbon Button](https://carbondesignsystem.com/components/button/usage/)：primary 是页面主 CTA，
  tertiary 可独立承担局部正向动作，ghost 用于最低强调工具；三项以上动作应进入 toolbar 或 menu；
  每个 screen 只使用一个页面级 primary。
- [Primer Button](https://primer.style/product/components/button/)：primary 是最高优先级且应谨慎使用，
  default 承担普通动作，invisible 用于最小化工具界面和复合组件。

### 本地实现基础

- [Nuxt UI](https://ui.nuxt.com/)：Button 的 color、variant、size 与 compoundVariants 可从 Vite
  AppConfig 统一覆盖；语义色和 surface token 可以避免页面散写颜色。
- Nuxt UI 4.7.1 本地实现：Button 默认提供 solid / outline / soft / subtle / ghost / link 六级；
  Pagination 的当前页默认复用 `primary + solid`，可通过 `active-variant` 将它改为选择态。
- Yotta 现有设置中心大量使用 `primary + soft`，用户明确认为其比亮绿色渐变主按钮更融入软件。

## Patterns

### 1. 主操作是角色，不是品牌色的同义词

- 使用：页面或模态任务中唯一的提交、新建、运行。
- 外观：暗色 accent-contained 实体面、1px 内描边、primary 前景；hover 增加少量 tint，active 回压。
- 避免：渐变、彩色外投影、同一区域多个同权重主按钮。

### 2. 局部正向工具使用 quiet accent

- 使用：侧栏录制、截图模板、批量编辑等不会完成整个页面任务的操作。
- 外观：`primary + soft`；与 neutral 工具共享尺寸和圆角，只用色相表达正向含义。
- 避免：为了“可点击”而把每个局部工具都升级为 solid。

### 3. 普通工具用 neutral surface

- 使用：搜索、列设置、刷新、更多、返回、撤销、面板开关。
- 外观：`neutral + soft` 用于需要稳定容器的文字工具，`neutral + ghost` 用于高密度 toolbar 和图标工具。
- 避免：把 navigation、menu trigger 或 toolbar icon 做成 primary。

### 4. 当前位置不使用提交态

- 使用：分页当前页、scope/tab 选择。
- 外观：`primary + subtle/soft`，依靠 tint、内描边和字重表达 selected。
- 避免：复用 solid 主操作，尤其不要带 CTA 投影。

### 5. 危险色只表达危险

- 使用：删除、停止、丢弃等会中断或损坏用户工作结果的动作。
- 外观：默认 error ghost/soft；只有危险动作本身是确认流程最终主动作时才提高强调。
- 避免：用红色表达普通返回或无害取消。

### 6. 状态完整且几何稳定

- 所有角色覆盖 default、hover、active、focus-visible、loading、disabled。
- icon-only 保持正方形与居中，并提供 `aria-label`/tooltip；同一按钮组不混用高度。
- hover/active 只改变 surface、描边和前景，不产生位移和尺寸变化。

## Local Application

- 在 `vite.config.ts` 的 Button compound variant 中替换旧 `btn-primary-raised`，统一所有真正的
  `primary + solid`，从根源消除亮绿色网页 CTA。
- 在 `style.css` 用 semantic token 派生 contained 主操作的默认、hover、active、focus 和 disabled，
  不散写页面色值。
- 三个 `UPagination` 统一设置 `active-variant="subtle"`。
- `WorkflowResourceDock` 的录制/截图按钮明确为 `primary + soft`；资源库页头录制和编辑器运行仍是
  各自任务区的唯一主操作，消费新的 contained 角色。
- 保留设置中心现有 quiet accent 方向，避免无意义批量替换。

## Next Step

先实现全局 contained 主操作、分页 selected 状态和编辑器侧栏 quiet accent，随后用 Playwright CLI
并排检查工作流、资源库、编辑器和设置中心；依据真实密度只调整 token，不在单页追加补丁。
