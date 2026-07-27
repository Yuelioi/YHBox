# V4 后续稳定性收尾上下文

## 已确认事实

- 作者补丁执行器使用 `$<handle>` 引用同一批次刚创建、尚未分配持久化 ID 的节点。
- `set-config` 等命令的节点引用允许该语义；`connect`/`disconnect` 当前直接复用
  `schema.Edge`，其 Endpoint pattern 只允许正式 Source 节点 ID，导致生成补丁合同与执行器冲突。
- `启动已安装应用` 节点只保存设置中的逻辑应用槽位，例如 `htgame`；可执行文件与固定参数继续由设置
  档案拥有，工作流不应复制路径或命令行。
- 直接通过 Application/Workflow service 保存启动节点、`htgame` 槽位和相同 exec 连线可以成功，
  因此不要修改应用槽位规则或运行时启动适配器来规避本问题。

## 修复边界

- 使用作者补丁专用的节点引用/endpoint/edge contract 表达正式 ID 或 `$handle`。
- 不放宽持久化 Workflow Source 的 Endpoint grammar。
- 重新生成并检查 tracked Workflow authoring contract；不要手改 `frontend/bindings/`。
- 回归必须覆盖新增节点、设置配置、连接信号在同一 PatchRequest 中通过生成 schema 与执行器。
- 开发阶段只跑定向测试；全部修改完成后只运行一次 `task check`，不默认运行 `task check:full`。
