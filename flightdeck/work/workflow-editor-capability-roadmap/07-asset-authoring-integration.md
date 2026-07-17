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

Planned。上一阶段恢复入口，本 Slice 补齐视觉反馈和创作闭环。
