# 录制与资源能力连续性

## Outcome

用户能从主窗口发现并使用现存 3.1 录制、Clip 和视觉模板能力，不依赖已删除的 Container 产品栈。

## Completion criterion

- 主导航存在资源库入口。
- 能选择 exact automation target 开始、暂停、完成录制并将 pending 结果保存为 InputClip。
- 能搜索、编辑和删除 Clip/模板元数据。
- 能从 exact target 打开模板截图选择器。
- 工作流节点 Inspector 继续以 immutable BlobRef 绑定 Clip 和模板变体。
- 明确记录旧复合模板节点与 3.1 原语之间的差异。

## Blocked by

无。本阶段不以模板实际缩略图为完成条件。

## Verification

`task check` 全绿；真实 Windows Wails WebView smoke 验证资源库和录制控件可达且无前端运行时错误；资源库截图人工检查通过。

## Out of scope

本阶段不重写 recorder/runtime，不引入 ambient window authority，不恢复依赖旧 Container graph 的 UI。安全 blob preview adapter 与 WaitTemplate / WaitTemplateGone / ClickTemplate 复合节点作为独立后续工作。

## Result

完成。录制与资产底层闭环继续复用 3.1 服务，新的独立资源库恢复了主窗口产品入口与管理闭环。
