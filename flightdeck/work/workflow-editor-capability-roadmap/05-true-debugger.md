# Slice 5：唯一调度器上的真正调试器

## Outcome / Question

不建立第二运行时、不绕过 capability/target/资源边界，为同一 Program 提供暂停、单步、继续和停止。

## Completion criterion

- scheduler 调用可见节点前设置控制点；普通运行/调试共享 Program、Owner、journal、租约和 cleanup。
- pause-before-node、step-one-visible-invocation、continue、pause、stop 与当前节点高亮。
- stop/fail/cancel/owner.Go 均可靠释放资源。
- 输入输出、状态和队列快照有类型、大小、脱敏和 Blob 摘要边界。
- 断点为独立调试元数据，不进入 Workflow Source。
- 默认从 RunStarted 正常 admission；不提供未经证明的任意节点起跑。
- 副作用提示不替代 capability 授权。

## Blocked by

Slice 4；先审查 loop/retry/instruction 的“一个可见节点”定义。

## Verification

定向覆盖线性、分支、loop、retry、error、取消与 cleanup；Stage 2 末统一 race/门禁、build、GUI smoke。

## Out of scope

时间旅行、回滚副作用、任意节点起跑和远程调试。

## Result

Planned。旧 UI 心智可借鉴，旧 debug service/runner 不复活。
