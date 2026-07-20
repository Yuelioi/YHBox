# Slice 38：用户真机测试 1 解决方案

## Outcome / Question

以源码、真机数据和旧版能力为事实基线，解释失败是否属于架构问题，并给出 3.1 发布前可执行、验证和回滚的解决方案。

结论：核心分层（Source → Compiler → Program、platform-neutral automation adapter、run journal、asset store）可以继续使用；问题集中在五个浅边界未收拢：录制 completion 有两个 UI owner、InputClip 缺少创作投影、target 只有节点级显式值、run event 一次性投影到散落面板、持久目录带版本名。应深化这些 Module/Interface，而不是推倒底层或继续在页面 watcher 上叠兜底。

## Confirmed facts

- `App.vue` keep-alive 使资源库和编辑器同时订阅 `recording:completed`；后端 pending 只能 finalize 一次，第二个隐藏 modal 报 `pending recording ... not found`。真机数据证明第一次保存已经成功。
- 简易录制当前只保留键盘与鼠标按钮，旧版还保留滚轮；精准录制和 InputClip playback 底层仍在，但没有统一可编辑的录后界面。
- 资源库入口展示“保存并添加到画布”是调用上下文泄漏。
- 模板 picker 已有搜索/分页和 variant callback，但没有 selected state 与固定确认。
- Workflow Source 没有 default target；Inspector 只能逐节点设置 slot。
- 普通 Run 强制 `runTimelineOpen = true`；Timeline 一次读取并 v-for 最多 65,536 条 journal。
- 三节点 Debug run 的 Start 已成功后被取消：Step 停在 Click 执行前；UI 未区分 previous/current。
- production bootstrap 硬编码 `workspace-3.1`；不是测试模式，但版本泄漏进稳定数据根。

## Architecture decisions

1. **RecordingSession 单 owner。** 后端维护 pending；前端 store 只持 handle/snapshot。页面不各自复制 pending，save/discard 后一次性释放。
2. **InputClip 单模型。** 简易与精准只是采集 preset，共享 normalize/validate；简易为可编辑 actions，精准将采样折叠为 MovePath。
3. **保存与插入分离。** 资源库只保存；编辑器可在保存后插入 InputClip 节点。模板使用“选择 variant → 固定确认”。
4. **EffectiveTargetResolver 深 Module。** Source 保存 workflow defaults/node overrides；Projection 与 Compiler 共用规则，Program 仍含显式 slot。
5. **RunEventStore 单写多投影。** Logs、Timeline、Debug 共享 runID/sequence 事实，分别查询/呈现；3.1 只做底部 tab 工作台。
6. **三层数据边界。** store retention、RPC cursor/range、UI bounded window + virtualization/aggregation 缺一不可。
7. **Debug snapshot 表达语义。** previous/executed 与 current/will-execute 分离；Step 等待同 runID 的新 sequence。
8. **稳定 workspace path。** canonical root 为 `workspace`；仅旧目录存在时同父目录迁移；两者并存不静默合并。

## UX rules

- 普通 Run 成功不打开面板；失败开 Logs；pause 开 Debug；Timeline 仅手动或显式偏好打开。
- 简易宏可增删、前后插入、重排，编辑键/按钮、持续时间和前置 delay；恢复滚轮。
- 精准录制保留轨迹/时序；主列表只显示可折叠 MovePath。
- workflow 只选一次默认 target；Inspector 显示 inherited/overridden/missing。
- 模板卡片单击选中，variant 可选，footer 固定“使用此模板”。

## Implementation slices

- 39：RecordingSession、InputClip 编辑、双调用语义、模板 picker。
- 40：Source schema、EffectiveTargetResolver、Compiler/Projection parity、Inspector。
- 41：Run Event cursor/range、底部工作台、打开状态机、Debug previous/current。
- 42：稳定 workspace 根与一次性迁移。
- 37：统一真机验收、完整门禁、stage commit 和发布判定。

## Verification

阶段中只跑受影响 seam 的定向测试；四个 Slice 完成后统一验收：

- 跨路由/keep-alive 不二次 finalize；两种入口动作正确。
- 简易录制编辑/滚轮/时长/延迟回放；精准轨迹折叠编辑/回放；异常序列拒绝。
- 1000 templates 搜索分页并明确确认。
- 一个 default 驱动 Click/Keys/Template，单节点 override 正确。
- Run 不弹面板，失败开 Logs，Step 显示 previous/current；长运行查询/渲染有界。
- legacy workspace 迁移、并存冲突、clean/current/corrupt 矩阵通过。
- 最后运行 `task check`、build/native/UAC 和工作台完整性检查。

## Out of scope

- 不做 VS/Blender 级任意停靠、多屏同步或独立 Trace Server。
- 不把精准录制拆成第二种资产/运行时。
- 不使用 last-used target 隐式状态。
- 不在测试通过前直接移动用户真机目录。

## Result

源码审计、真机数据核对与官方一方资料调研完成；实施边界固定为 Slices 39–42。计划完成不等于产品完成。
