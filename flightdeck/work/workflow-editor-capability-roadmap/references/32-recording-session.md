# Slice 32：Recording Session 纵向闭环

## Outcome / Question

把 recorder、HUD、canonicalization、finalize、codec、asset save/reload 和 playback 收回一个会守住持久化不变量的深模块，恢复显式 simple/precise 两种录制任务。

## Completion criterion

- session 提供权威 snapshot + event stream；HUD 晚挂载会 reconcile 当前状态。
- simple/precise 是显式策略，不由 incidental mouse move 猜测。
- finalize 前统一首事件归零、严格排序、pause time、合法 key/button release。
- 保存链真实使用 recorder events，并完成 codec、asset、reload、playback round-trip。
- 失败遵循 Slice 30 错误边界；取消/丢弃不污染资源库。

## Blocked by

Slices 30–31。

## Verification

- 状态机、canonicalizer、codec property/invariant tests。
- HUD late-mount、pause/resume、cancel/finalize integration tests。
- G07/G08 使用真实 Windows recorder events 完成 Stage R1 contract gate；R2 完成 native playback。

## Out of scope

- 不在页面组件内复制 recording state machine。
- 不通过放宽 codec invariant 接受无效 clip。
- 不把 simple/precise 合并成不可预测的自动模式。

## Result

Completed。

- `RecordingMode(simple|precise)` 现在是 Start、session snapshot、pending payload、InputClip v3 carrier 与 Workflow draft 的显式契约，不再根据 incidental mouse move 推断。
- Recording Service 独占 `idle → recording/paused → finalizing → pending → idle`，每次转换发布 monotonic revision 全量快照；页面/store/HUD 只镜像，HUD 晚挂载会主动 `GetState` 对账。
- Start 持有 exact automation generation lease 到 Stop/Cancel；鼠标校准来自该 installation profile，不再读取独立全局 snapshot。
- recorder 优先使用 native event clock；统一 canonicalizer 负责排序、首事件归零、Seq 重建、simple 过滤、非法/重复 transition 清理与 dangling key/button release。
- Stop 只生成单一 pending；Finalize 先通过严格 InputClip v3 codec encode/decode 自校验，再原子提交 blob/asset；失败恢复 pending，Discard/Cancel 不创建资产。
- simple 生成完整按键/点击线性 draft；precise 始终生成持完整 BlobRef 的 Play Input Clip draft。真实 Windows hook/playback 仍按计划在 Slice 34 的 G07/G08 native gate 验收。
- 定向实现检验通过：5 个相关 Go packages；`vue-tsc`；5 个前端测试文件 35 tests；Wails contract 14 services/119 methods/170 models；i18n 1782 keys；相关 oxfmt 与 diff check。
- Slice 内按阶段批量验收原则没有重复运行整仓门禁；Slice 33 完成后，R1 统一 `task check` 已通过。

