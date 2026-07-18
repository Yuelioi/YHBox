---
slice: "31"
title: Installation Manifest 与 Automation Target Runtime
status: completed
---

# Slice 31：Installation Manifest 与 Automation Target Runtime

## Outcome / Question

用 adapter-owned、版本化 installation manifest 取代中央 ProfileDraft 和 capability 手工二次映射；由单一 runtime owner 原子发布 authoring、admission、policy 和 provider generation。

## Completion criterion

- manifest 是 authoring descriptor、Host Profile、provider registry、policy digest、health 和 Settings editor schema 的唯一事实源。
- 新 operation/target kind 不需要修改 composition root capability switch。
- Adapter 自己拥有 profile version、payload schema、seal、runtime factory、health 和 editor descriptor。
- Target Runtime 独占 prepare/publish/lease/reclaim/shutdown；Settings 只持久化用户意图。
- capture/install/edit/delete/consent 同进程生效；旧 Run 持 lease，空闲 generation 可回收。
- Slice 20/26 的 exact/regex、UAC、热更新和真实诊断被吸收，不丢失。

## Blocked by

Slice 29；Stage R1 验收还依赖 Slice 30。

## Verification

- manifest conformance 覆盖 capability、operation、resource、authoring、admission 和 provider parity。
- generation 测试覆盖 atomic publish、失败 rollback、old-run lease、idle reclaim、shutdown。
- 用 Press Keys + target fixture 证明不存在 `builtinHostProfile` 漏投影。
- G04 contract/integration 通过；R2 再做 Windows native smoke。

## Out of scope

- 不在此 Slice 实现所有 Windows 操作或 Android native smoke。
- 不把 HWND、PID、path、credential 或 endpoint 写进 Workflow Source。
- 不保留正常修改后要求重启的兼容流程。

## Result

Completed。

- Adapter registration 现在拥有 profile version、Settings intent codec、sealed payload、editor fields、capability/resource/operation manifest、verifier、runtime factory 与 health。
- 中央 Settings 只持久化稳定 envelope + opaque payload；未知 target 由 manifest fields 提供基础编辑器，Win32/ADB/Browser 专用发现只是增强。
- 每个 sealed installation manifest 是 Host Profile、provider registry、Policy/consent digest、health 与 workflow-facing descriptor 的单一事实源；Press Keys 不再依赖 composition-root 手工映射。
- AutomationTargetRuntime 独占 prepare/publish/rollback/retire/reap/shutdown；Application 在 admission 前获取 exact provider generation lease，旧 Run 终态后才释放。
- generation tests 覆盖 active lease、idle reclaim、失败 rollback、关闭竞态与同进程替换；custom Adapter conformance 同时覆盖 intent、manifest、provider verifier 与 health。
- 聚合验收通过：go test 覆盖 installed/application/appbootstrap/services/desktopapp/nodes/browser smoke command；vue-tsc 通过；3 个相关 Vitest 文件共 13 tests 通过；oxfmt 与定向 diff check 通过。
- 按阶段验收原则未运行整仓 task check；它与 G04/G07/G08/G09/G11 contract gate 一起留到 R1 的 Slices 30–33 全部完成后。
