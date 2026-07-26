# V4 execution plan

## Outcome

用户打开 Yotta 后只面对一个工作流库；导入即成为本地工作流，配置目标后直接运行。内部由少量深 Module
承载持久化、编辑、执行和设备能力，Wails transport 不复制领域状态机。

## V4-A — Workflow-only cutover

- [x] [删除 Release/Installation 产品路径](slices/v4-a-workflow-only-cutover.md)：工作流列表、运行和计划只引用
  Workflow；导入创建本地 Workflow，移除设置/更新/回退/派生 Installation UI 和 RPC。
- [x] 从 desktop composition 和 application runtime 移除 Workflow Installation module。
- [x] 删除无调用的 Release/Installation catalog、离线包、导入协调器和测试代码；旧 SQLite migration
  只为已存在 profile 的 schema ledger 兼容保留，不再有 repository 或运行调用。

## V4-B — Target configuration

- [x] AI、HTTP、应用和自动化对象在产品层统一使用 Target/配置语言；应用路径与设备配置仍属于当前
  用户设置，保存后由现有热替换路径立即生效。
- [x] 设置 transport 保持 `Get`/`Update` 和按需测试动作，不向前端暴露 adapter installation
  draft、digest 或 generation。

## V4-C — Desktop and transport depth

- [x] 保留稳定的 `Application`/compiler/runtime 深 Module，不为 V4 再建立平行 composition；删除
  Workflow Installation 传递层和 schedule readiness seam。
- [x] Wails 从 149 个方法收敛到 138 个，Workflow gateway 只表达本地 Workflow 用户动作。
- [x] 删除 Installation runtime/interface、离线安装抽象和重复 library projection。

## V4-D — Product cutover

- [x] 产品版本切换至 4.0，清理 V3 专有文案、领域说明与不可达代码。
- [x] 验证现有 Workflow Source、资源、录制、计划、运行历史和设置数据可继续使用；旧内置节点契约
  在形状兼容时自动迁移，`fishing-v2` 真实数据编译通过。
- [x] 完成 `task check`、Windows build 与完整 WebView 桌面验收，建立独立 V4 基线。
