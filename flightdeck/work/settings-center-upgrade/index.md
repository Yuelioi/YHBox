---
topic: settings-center-upgrade
title: Settings center upgrade
summary: Upgrade the six settings themes, AI credential security, and adjacent automation workspaces.
---
# Settings center upgrade

## State

设置中心与六个主题已完成结构性升级；随后完成 AI API Key 的系统凭据迁移，并升级容器、计划两个主工作台。设置查询和 AI 连接元数据不再返回密钥，Windows 使用 Credential Manager；容器页改为运行导向工作台，计划页改为触发与健康状态导向的控制台。独立工具窗现统一为精密仪器式的产品界面：录制、校准、鼠标定位、截图选择器与悬浮启动器共用窗口 chrome、状态语言和响应式尺寸策略。

本轮产品改动已提交为 `c4d1307e`，Flightdeck 文档已提交为 `d92c11b0`。工作台概览栏左右缺边问题已修复并提交为 `ec771606`。独立窗口升级已通过完整门禁并提交为 `88a845bd`。

悬浮启动器第一阶段商业化升级已提交为 `6b56deee`：实际窗口与设置预览共享命令表面，支持搜索、完整键盘操作、运行反馈及过期入口安全清理。

## Next

由用户进行真实桌面视觉与交互 smoke。重点检查：
- AI 连接旧明文密钥启动迁移、替换、删除与测试连接。
- 容器工作台在常用窗口宽度下的概览、筛选、卡片/列表与运行操作。
- 计划控制台的搜索、启停、目标摘要、编辑器双栏与窄窗布局。
- 设置中心原有 860px 响应式切换及六主题交互。
- 录制/校准 HUD 在真实 Windows 缩放下无裁切；鼠标定位与启动器最小尺寸可用；截图选择器在 760×520 下画布与属性栏均可操作。

悬浮启动器第一阶段已完成；下一阶段继续按“固定收藏 Dock + 全容器命令面板”的双层模型演进：
- 增加热键唤出的全容器命令面板与最近使用；悬浮窗保持为少量高频收藏，不把所有容器塞进常驻小窗。
- 评估拼音/模糊搜索与最近运行排序，保持当前精确子串搜索为可靠降级路径。
- 真实 Windows 多屏、缩放与置顶场景继续由用户 smoke；需要时再处理 AppBar/边缘吸附，不在第一阶段扩大窗口管理范围。

## Read now

- knowledge/frontend/settings-page-style.md
- knowledge/frontend/ui.md
- knowledge/architecture/settings-durability.md
- knowledge/security/ai-credentials-windows-credential-manager.md

## Read if

- knowledge/build/build.md — 重新构建或运行完整门禁前
- knowledge/frontend/headless-ui-verify.md — 需要离屏视觉回归时

## Progress

Done:
- 设置中心共享壳层、搜索/深链/访问恢复、键盘导航、响应式侧栏和六主题升级。
- 设置写入串行、保存状态反馈、AI 引用保护与 MCP 授权确认。
- Windows AI API Key 改存 Credential Manager；旧 settings 明文执行无损启动迁移。
- Settings RPC 只返回 AI 连接元数据；新增密钥存在状态、写入、删除 RPC，测试连接可使用一次性表单密钥或已保存密钥。
- 容器 Tab 增加工作台标题、运行/节点/分类概览、渐进筛选和更完整的运行卡片。
- 计划 Tab 增加启用/自动触发/目标概览、搜索与状态筛选、运行态列表，以及带行为预览的分区编辑器。
- 工作台概览从仅有上下边框改为完整四边框与圆角，容器和计划共用；新增 CSS 契约回归测试。
- 五个独立窗口全部迁移到增强版 `HudShell`，标题、副标题、状态胶囊、拖拽区域、关闭按钮与窄窗收缩统一。
- 新增共享 `HudStatePanel`，录制与校准的待机、倒计时、进行、暂停、完成状态使用同一语义系统。
- 截图选择器重排为顶部工具栏、画布优先和响应式属性侧栏；模板名称、分类、标签和选区操作统一使用 Nuxt UI。
- 鼠标定位补齐坐标/比例/颜色读数与复制反馈；悬浮启动器补齐空状态、运行态、快捷键提示与无障碍名称。
- 工具窗默认/最小尺寸集中到 `wailsToolsWindowOptions`，并增加 Go 与前端 presentation contract 回归测试。
- 修复录制与校准 HUD 状态卡按内容收缩、`mt-auto` 将空白堆在操作栏上方造成的垂直不对称；全部实时状态与加载态改为弹性填满并居中，新增布局回归契约。
- Wails RPC contract 更新为 14 services / 119 methods / 100 models。
- 悬浮启动器升级为默认命令列表、可选紧凑图标网格和纯文字列表；分组标题与旧分隔符统一解析为正式区段。
- 实际悬浮窗与设置实时预览共用 `LauncherSurface`；新增搜索、方向键、Enter、Esc、数字直达、快捷键提示及运行中/成功/失败反馈。
- 设置页新增配置健康统计、完整过期入口范围、单击清理与撤销；清理只删除不存在容器的启动器引用，不修改快捷键，也不删除真实容器。
- 启动器默认/最小尺寸更新为 `300×360` / `220×120`，后端开窗策略与运行时拖拽下限一致。
- 启动器模型新增分组、搜索和过期清理单测；实际窗口/设置预览共享实现由 presentation contract 锁定。

Verified:
- 修复后完整 `task check` 通过。
- Go test/vet/staticcheck、全局覆盖率门槛通过。
- 前端 format、oxlint、eslint、vue-tsc、i18n、bindings contract 全绿。
- Vitest 94 files / 624 tests 全绿。
- 生产构建与 bundle budget 通过；entry gzip 333,058 / 350,000，editor gzip 472,215 / 650,000。
- 浏览器视觉通道本轮不可用，未伪报自动化视觉截图。
- 独立窗口 focused 校验已通过：前端 lint/typecheck/i18n、9 个工具窗测试与 Go 窗口参数测试全绿；修正无效 Tabler 图标后完整 `task check` 全绿。
- HUD 垂直节奏修复按红绿回归完成，独立窗口契约 8/8 通过，随后完整 `task check` 再次全绿。
- 启动器第一阶段完整 `task check` 通过：95 个前端测试文件 / 631 项测试，Go test/vet/staticcheck、覆盖率、bindings contract、i18n、生产构建与 bundle budget 全绿。
- 双轴代码审查完成：旧分隔符兼容、清理事务边界、清理范围披露、输入即搜索、共享失效态与持久化热键冲突诊断均修正；复审无剩余 P1/P2。

## Open questions

- Windows 桌面真实 smoke 尚待用户验收。
- Linux/macOS 仍为预览平台，当前 secure store 明确返回 unavailable；后续若承诺完整支持，需要接入各平台原生凭据库。
