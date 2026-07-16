# AI offline eval 与 upgrade gate

Status: blocked

## Outcome

模型 profile、PromptManifest、ToolSet 或 structured schema 的任何升级都必须由版本化离线 suite、确定性 grader 与显式阈值报告批准；运行环境不自动漂移到未验收 identity。

## Completion criterion

- EvalSuite artifact 固定 dataset/rubric/grader/version/baseline identity，并可 strict-open、hash 与复现。
- 首批 corpus 覆盖中英 authoring、catalog selection、minimal patch、diagnostic repair、strict extraction、injection 与未授权 capability。
- 确定性 grader 验证 schema/compile/graph diff/permission/budget；主观 grader 只能补充且必须固定 rubric/model identity。
- regression report 比较 pass rate、安全失败率、cost/token/latency threshold，并给出 approved/rejected 结果。
- ModelProfile/Prompt/ToolSet/Schema installation 只能引用 approved suite result；unverified/rejected identity fail closed。
- fixtures、CLI/Task 入口、negative tests 与 task check 全绿。

## Blocked by

ai-prompt-tool-provenance。

## Verification

ModelProfile 只有 EvaluationStatus/EvaluationSuite metadata 和 seal-time shape validation；仓库没有 eval dataset、runner、grader、report 或 installation admission proof。

## Out of scope

- 在线用户数据采集。
- 自动上传 prompt、截图、网页、credential 或原始 trace。
- AI authoring UI 与 Agent runtime 本身。

## Result

Blocked。
