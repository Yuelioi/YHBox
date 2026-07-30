# 统一操作按钮体系上下文

## 用户反馈

- 现有亮绿色 `primary + solid` 按钮像网页 CTA，色块和投影从新的暗色实体表面中“跳出来”，不符合
  Yotta 桌面生产力工具的克制、精确气质。
- 需要覆盖工作流“新工作流”、资源库录制、工作流编辑器运行、左侧截图/新模板工具、分页当前页等，
  不能只修三张截图里的实例。
- 设置中心的低饱和强调按钮是可参考方向，但最终仍需区分主操作与普通工具，不能把所有按钮做成同一层。
- 第一版 contained 方向得到认可，但 13% primary tint 的默认底仍偏亮、偏实心；最终默认态使用 7%
  tint、薄内高光和短距离深色阴影，形成更暗、更晶莹的实体边缘。

## 约束

- 继续使用 Nuxt UI `UButton`、`UDropdownMenu` 与 `UPagination`；共享外观优先放在 Nuxt UI 主题或
  少量语义 utility，不建立包一层却不增加语义的通用 Button 组件。
- 图标只使用当前安装的 Iconify Tabler 集合。图标按钮必须有 `aria-label`，固定尺寸按钮保留全局
  `justify-center` 基线。
- 暗色 surface 已固定为 canvas L17、surface L21、hover L24、strong L26；按钮必须从这些表面和
  semantic primary/error/warning 等角色派生，不散写 zinc/black/white。
- 颜色只增强操作层级，按钮仍需通过填充、边框、文字、图标和位置表达语义；键盘 focus 不能只靠颜色。
- 主操作可以使用有真实向下位移的深色短阴影建立实体分离，但不使用彩色发光、宽软阴影或营销式渐变。
- 不改变按钮的业务行为、权限、loading/disabled 条件、菜单结构或反馈生命周期。

## 初始角色假设

- 主操作：每个局部任务区至多一个，使用克制的 accent-contained 实体面、薄内高光与短位移阴影，
  不使用营销式渐变和彩色发光。
- 普通工具：使用 neutral/soft 或 ghost，依靠 surface hover 和边框变化融入工作台。
- 紧凑导航与分页：选中态表达“当前位置”，不冒充提交动作；使用 accent tint、边框和数字权重。
- 危险操作：继续使用 error 语义，只在确认执行或不可逆动作上出现，不把红色用于普通取消。
