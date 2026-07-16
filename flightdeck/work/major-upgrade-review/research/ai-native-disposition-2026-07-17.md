# AI-native design disposition

Basis: main at 44b01390 plus implementation history through 27e01b17.

## Product and architecture disposition

| Design claim | Disposition | Evidence / reason |
| --- | --- | --- |
| Yotta is a local-first, auditable AI automation workbench | retained | Workflow 3.1 Compiler, capability admission, durable Run facts and provider-native AI all implement this destination. |
| Human Studio, AI agent and MCP/CLI share one authoring core | partially completed | EditorSession and MCP use the same Application/authoring/Compiler seam; an AI authoring client/loop does not exist yet. |
| Provider wire semantics stay native behind a shared domain contract | completed | ab1f4cf4 and 99c3f5ff; internal/ai has no generic Chat or prompt JSON fallback. |
| Exact internal/ai subpackage topology is mandatory | obsolete implementation sketch | The flat internal/ai package already forms a cohesive provider/profile/schema/resource module. Remaining outcomes justify new modules only when their interfaces exist. |
| Generate, Extract/Classify and Agent are stable node families | partially completed | Generate and strict Extract are in the Catalog; Agent is absent. Classify can be expressed by strict Extract schema and does not require a duplicate runtime family. |
| Session resource is a 3.1 requirement | deferred by the design itself | The design says to add Session only when durable conversation is a proven product need. It is not a stable blocker. |
| MCP is an adapter, not a second business layer | completed | b4aa17aa projects Application commands, validates input/output schemas, pages catalog/workflow data and has no production listener. |
| Evals govern model/prompt/schema/tool upgrades | remaining | ModelProfile carries evaluation metadata only; no dataset, grader, threshold report or admission gate exists. |
| Contributor instructions have one canonical owner | completed | tracked AGENTS.md is short and routes mechanical rules to task check/Flightdeck. |

## Completion-definition disposition

| # | Completion definition | Disposition | Concrete evidence or gap |
| --- | --- | --- | --- |
| 1 | OpenAI Responses, Anthropic Messages, no ModePrompt/JSON extraction | completed | ab1f4cf4 implements the two native protocols; 99c3f5ff removes internal/services/llm and structured prompt fallback. |
| 2 | AI nodes reference slot/profile and credentials stay outside settings/frontend | completed | 4b630f70 binds logical slot requirements; 7e9fb87c installs content-addressed profiles and AISecrets uses securestore. UI receives status, never stored key material. |
| 3 | PromptManifest, ModelProfile, Schema and ToolSet are versioned, hashed, traced and rollbackable | remaining | ModelProfile and schema are content-addressed. PromptManifest and ToolSet do not exist; trace cannot identify their versions; rollback policy is absent. |
| 4 | Dynamic/untrusted data cannot enter system/developer blocks | remaining | nodes31runtime.aiRequest copies workflow config instructions into GenerateRequest.Instructions; native adapters map it directly to OpenAI instructions and Anthropic system. |
| 5 | Extract/Classify validates atomically; Agent has tool/time/token/cost/iteration budgets | split | Strict Extract validates provider output before sealing one result envelope. No Agent node, tool execution loop, approval state or multidimensional budget exists. |
| 6 | AI and UI share typed patch, Compiler, transaction and permission manifest | partially completed | The shared authoring/Application/MCP substrate is complete. There is no AI authoring client that performs the bounded patch/compile/repair loop. |
| 7 | MCP default off, strict schemas, on-demand catalog, no whole-graph save | completed | b4aa17aa removes whole-document tools. BuildProtocol is transport-neutral, schema-validating and not wired to a desktop listener. |
| 8 | Model/prompt/tool/schema upgrades pass offline eval and safety regression | remaining | EvaluationStatus/EvaluationSuite are declarative fields only; no executable gate verifies or controls upgrades. |
| 9 | Users can inspect AI diff, diagnostics, permission change and redacted trace | remaining | EditorSession shows compiler diagnostics, preview exposes capability plan and Run actions record provider IDs/usage. No AI patch review, permission delta, prompt/tool/schema lineage or redacted AI trace view exists. |

## Approved remaining outcomes

1. ai-prompt-tool-provenance
   - content-addressed PromptManifest and ToolSet artifacts;
   - trusted instruction rendering with typed untrusted blocks;
   - provider requests can only receive trusted rendered instructions;
   - lineage/rollback identity for prompt, schema, toolset and model profile.
2. ai-agent-budget-runtime
   - Agent node and bounded provider-native tool loop;
   - exact ToolSet allowlist, schema validation and capability approval;
   - token, cost, time, iteration and tool-call budgets with durable terminal facts.
3. ai-eval-upgrade-gate
   - versioned offline suites, deterministic graders and regression thresholds;
   - upgrade admission for model profile, prompt, schema and toolset identities.
4. ai-authoring-review-trace
   - bounded AI authoring loop over catalog/inspect/apply-patch/compile/preview;
   - review model for graph diff, diagnostics and permission delta;
   - redacted trace linking model/prompt/schema/toolset, approvals, patches and Run outcome.

These are sibling Slices under the existing major-upgrade-review destination. None requires a new Topic.

## Explicit non-work

- Do not recreate generic Chat, ModePrompt, whole-document editing or a production MCP listener.
- Do not split internal/ai merely to match the old directory diagram.
- Do not add a durable conversation Session without a separately demonstrated product need.
- Do not create a separate Classify runtime when strict Extract plus enum schema has the same contract.
