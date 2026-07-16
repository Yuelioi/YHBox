# AI offline eval 与 upgrade gate

Status: completed (cfa12703)

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

无。ai-prompt-tool-provenance 已由 b674664c 完成；Agent runtime 已由 d22b5bd5 完成。

## Verification

- EvalSuite/EvalReport 使用独立 v1 hash domain、canonical bytes、strict reopen、unknown-field rejection 与 byte/depth/node/case budgets。
- mandatory suite 固定 8 类 corpus、deterministic-v1 grader、baseline，以及 pass/safety/token/cost/latency thresholds；tracked report 为 8/8、safety 0。
- grader 精确匹配每个 observation，校验 expected output/refusal、permission delta 与 case/global budgets；report reopen 重算 aggregate metrics 与 approved/rejected decision。
- upgrade candidate 分离 model subject 与 artifact set，并绑定当前 Generate/Extract/Agent prompts、Agent ToolSet、三个 AI Node Contract semantic digest。
- exact suite/report digest 均进入 ModelProfile 与 workflow consent lineage；report replacement、profile edit 或 artifact upgrade 自动使旧授权失效。
- Install 只保留 approved subject；app bootstrap 再按 current artifact set 过滤 stale candidate，unverified/rejected/stale profile 不进入 Host Profile但不会阻止设置界面启动。
- Settings semantic edit 自动降级为 unverified 并清除 suite/report/consent；AIService ApplyEvaluation/RevokeEvaluation 提供显式导入/撤销，GrantWorkflowUse 再验 exact candidate。
- cmd/ai-eval 支持 subject 或 profile 输入、canonical report write/check；Taskfile 将 check:ai-eval 纳入唯一 task check。
- negative tests 覆盖 safety regression、mismatched subject/evidence、stale artifacts、report identity replacement、drifted fixture、自动 downgrade/revoke。
- 2026-07-17 task check 全绿：mandatory 8/8、safety 0；global coverage 65.9%，internal/ai 74.1%，frontend 27/103，Wails 91 methods / 101 models。

## Out of scope

- 在线用户数据采集。
- 自动上传 prompt、截图、网页、credential 或原始 trace。
- AI authoring UI 与 Agent runtime 本身。
- 非确定性主观 grader；未来加入时必须固定 rubric/model identity，且只能补充 deterministic safety gate。

## Result

cfa12703 完成 offline evaluation gate：mandatory suite/report/corpus、deterministic grader、model+artifact candidate、Host Profile filter、Apply/RevokeEvaluation 与 task check drift gate。
