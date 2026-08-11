# V4 发布兼容性封板 context

## What matters

本地工作流及其依赖是用户资产。一次产品、节点合同、数据类型或持久格式升级，不得因为 reader 只接受
新版本、节点 semantic digest 原地变化、迁移未持久发布，或错误被列表层吞掉，而让资产静默消失或无法运行。

## Decisions

- 当前代码、schema、正式生成合同和可执行测试是事实源；文档与 Knowledge 只用于导航并必须反向核验。
- V4 是首个公开兼容底线。发布前的历史开发格式可以最后一次确定性升级，不长期保留无用户价值的兼容分支。
- V4 发布后，每个持久化版本域只允许显式的相邻 migration；成功后必须原子写回并通过关闭、重开验证。
- 节点同一 `(type, version)` 的语义合同不得原地变化。需要变化时提升节点版本，并为已有 NodeRef 提供确定性迁移。
- 可重建缓存与持久用户资产分开处理：缓存可以按 compiler/catalog/build identity 失效重建，用户资产必须迁移或给出可操作错误。
- 兼容检查放在保存、加载、编译或 Run 建立阶段，不在逐节点或逐操作热路径增加 authority/security 包装层。
- 迁移失败不得被 `continue`、空列表或泛化错误吞掉；错误至少能定位 artifact、版本和可恢复动作。
- 公开发布事实由三份 create-only snapshot 记录：版本域 floor、built-in NodeRef floor、built-in
  TypeRef/CapabilityRef floor；当前 projection/schema 不能替代 release snapshot。
- 已冻结 TypeRef/CapabilityRef 暂无引用 rewrite registry，后续语义变化必须保留旧 versioned URI 并增加新
  URI；NodeRef 则允许通过完整、唯一、无环的相邻 migration 链升级。
- 外部 Node Package 的 Workflow dependency 精确锁定 manifest digest；应用升级不会自动切换 package
  generation。Package 安装/更新仍是 preview owner 能力，未来产品化更新必须先定义旧依赖迁移或并存策略。
- 对外 `main` 从 `e330f47b`（V4 workflow core cutover）的等价无父提交开始，并保留其后的 V4 提交粒度；
  之前的完整 1900+ 提交历史由 `archive/pre-v4-full-history` 持有，不删除旧对象、不把旧历史继续挂在主线祖先上。

## Acceptance

- 所有持久化 artifact 与协议都有明确 owner、当前版本、reader 策略和升级路径；无意义的“版本字段”被识别为派生数据。
- 已冻结的节点合同发生同版本漂移时门禁失败；合法节点升级有回放旧 NodeRef 的测试。
- 工作流升级测试覆盖磁盘旧样本、迁移发布、关闭重开、编译和运行准备，不只检查内存对象。
- 调度、宏、设置等引用工作流的资产升级后仍可列出；无法迁移时返回可诊断错误而非静默丢失。
- 发布范围门禁通过，并保存明确的未完成发布阻断。

## Frozen V4 floor

- `contracts/releases/4.0.0/version-domains.json`：39 个 tracked public version domains。
- `contracts/node/releases/4.0.0/builtin-node-refs.json`：147 个 built-in NodeRefs。
- `contracts/catalog/releases/4.0.0/builtin-catalog-refs.json`：24 个 TypeRefs、5 个 CapabilityRefs。
