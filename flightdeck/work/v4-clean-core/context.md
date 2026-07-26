# V4 goal context

## Product truth

- Yotta 是本地优先的工作流工具，首要价值是让用户快速创建、配置和运行自动化。
- Workflow 是唯一一级内容对象；本地创建和文件导入不形成不同的工作流类型。
- Target 是 AI、HTTP、桌面应用、窗口或设备的本机配置；保存后即可被工作流引用。
- Schedule 是 Workflow 的触发配置；Run 是一次执行及其结果。
- Resource 是图片、Macro 或 InputClip；可归属 Workflow，也可存在于本机资源库。
- 来源地址、作者和上游版本只是可选 Origin metadata，不产生 Release/Installation 产品对象。

## Constraints

- 保留现有节点编辑、子图、Snippet、Macro、InputClip、录制、资源、目标、计划、调试与运行历史能力。
- 不为 Registry、在线分发、Publisher Proof、离线依赖包或第三方 Node Package 预建 V4 核心接口。
- 不增加 Workflow 运行授权。配置缺失只报告缺少的 Target、Credential 或 Node。
- V3 基线 `645e0bad` 必须始终可恢复；V4 删除旧实现时使用独立 commit。
- 现有 `fishing-v2` Workflow Source 与 Blob 数据必须保持可读。

## Design decisions

- V4 在现有主线原地替换产品路径，不建立永久 `internal/v4` 或第二套 runtime。
- 复用已稳定的 Source schema、compiler、node runtime、Windows adapter 和资源实现。
- 每个新外部 seam 必须有当前真实调用方；单一实现的内部细节不抽象成公共 interface。
- Wails 只暴露用户动作，不暴露 Release、Installation、digest、generation 或内部更新状态机。
