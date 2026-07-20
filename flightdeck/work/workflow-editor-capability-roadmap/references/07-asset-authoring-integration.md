# Slice 7：安全资源预览与录制联动

## Outcome / Question

资源库、模板节点和录制结果形成 3.1 创作闭环，且不暴露任意文件读取能力。

## Completion criterion

- BlobRef 经受限缩略图 API 预览，限制类型、尺寸、解码时间/输出，不接受路径。
- inspector/资源选择器显示缩略图、名称和失效状态。
- 简易录制支持按键+点击，复杂录制保留有价值的轨迹信息，输出 3.1 source/草稿。
- 录制结果先预览，批量插入一个 undo，不自动保存、不成功 toast。
- 资源失效形成可定位编译诊断。
- Clip/检测/等待模板共用 BlobRef 协议。

## Blocked by

Slice 6 和现有资源/录制入口。

## Verification

定向验证内容安全、缩略图失败、引用和录制转换；Stage 3 末统一 task check、build、真机 smoke。

## Out of scope

云同步、任意文件浏览、无审核自动执行和永久旧库兼容层。

## Result

Completed。

- 新增只接受 BlobRef 的受限 PNG preview RPC；前端资源库与模板绑定共用 BlobPreview，并显示缺失/失效状态，不接收任意路径。
- compiler 对不存在的 BlobRef 形成可定位诊断；Clip、WaitTemplate、WaitTemplateGone、ClickTemplate 共用同一 BlobRef 协议。
- 录制停止结果包含 preview，支持按键/点击 steps 与轨迹草稿；前端先预览，再 finalize 并线性批量插入，整批只产生一个 undo，不自动保存且成功不 toast。
- 修复 recording completed event 漏发 preview、前端不安全 payload cast、HUD 打开失败被误报为录制启动失败、preview 加载永久悬挂等收口问题。
- Stage 3 定向 Go 套件、recording/EditorSession Vitest 与 typecheck 通过；完整 task check 通过（Go global coverage 65.6%，前端 35 files / 134 tests）。
- 最终调试 generation 回归测试后，EditorSession 定向 12 tests 与 typecheck 通过；正式 task build 生成 bin/Yotta.exe。
- Windows WebView smoke 通过：103 catalog nodes、3 canvas nodes、AI review 与资源/录制入口可达，编辑器和资源页 PNG 已人工目检。隔离 smoke 不具备已安装 Win32 target，真实 OS 输入捕获仍按 native-target smoke 触发条件执行。
