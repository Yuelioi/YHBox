# Comparable product patterns for Yotta V4

## Research read

Question: what do established visual/local automation products do to make first use, workflow discovery, target setup, editing, and upgrades feel direct and dependable?

Scope: official documentation only. The observations below describe the products; the recommendations are separate judgments for Yotta, a local-first, single-user desktop automation tool. Enterprise collaboration and cloud-governance features are not assumed to be relevant to V4.

## Source matrix

| Product | Primary official sources | What the sources establish |
| --- | --- | --- |
| n8n | [Create and run workflows](https://docs.n8n.io/build/understand-workflows/create-and-run-workflows), [Workflow templates](https://docs.n8n.io/build/ways-of-building-workflows/use-templates), [Create and edit credentials](https://docs.n8n.io/build/understand-workflows/create-and-edit-credentials), [Workflow history](https://docs.n8n.io/build/manage-workflows/view-change-history), [All executions](https://docs.n8n.io/workflows/executions/all-executions/) | A workflow can begin blank with “Add first step,” or from a template. Connections can be created in the node’s credential selector and are tested when saved. Workflow versions and run executions are presented as different concepts. |
| Node-RED | [Editor guide](https://nodered.org/docs/user-guide/editor/), [Creating your first flow](https://nodered.org/docs/tutorials/first-flow), [Concepts](https://nodered.org/docs/user-guide/concepts), [Nodes](https://nodered.org/docs/user-guide/editor/workspace/nodes), [Sidebar](https://nodered.org/docs/user-guide/editor/sidebar/), [Environment variables](https://nodered.org/docs/user-guide/environment-variables), [Import/export](https://nodered.org/docs/user-guide/editor/workspace/import-export), [Projects](https://nodered.org/docs/user-guide/projects/) | The core editor is palette, canvas, and supporting sidebars. Configuration nodes are reusable but stay off the main canvas. Runtime state and invalid configuration appear on nodes. Platform-specific values can live outside shared flow structure. Git-backed Projects are optional and deliberately expose only a small subset of Git. |
| Power Automate for desktop | [Console](https://learn.microsoft.com/en-us/power-automate/desktop-flows/console), [Create desktop flows](https://learn.microsoft.com/en-us/power-automate/desktop-flows/create-flow), [Flow designer](https://learn.microsoft.com/en-us/power-automate/desktop-flows/flow-designer), [Actions pane](https://learn.microsoft.com/en-us/power-automate/desktop-flows/actions-pane), [UI elements](https://learn.microsoft.com/en-us/power-automate/desktop-flows/ui-elements), [UI element collections](https://learn.microsoft.com/en-us/power-automate/desktop-flows/create-ui-elements-collections), [Update compatibility](https://learn.microsoft.com/en-us/power-automate/desktop-flows/how-to/update_power_automate), [v2 schema](https://learn.microsoft.com/en-us/power-automate/desktop-flows/schema) | The console separates home, owned flows, shared flows, and examples; flow rows expose Start, Edit, and Status. Action parameters open only after adding an action; advanced error handling is inside that action dialog. UI selectors are managed as flow resources and can be shared independently of whether an app is local or remote. Older flows remain usable after upgrades. |
| Zapier | [Zap drafts and versions](https://help.zapier.com/hc/en-us/articles/9693520498445-Create-Zap-drafts-and-versions) | Draft/publish/version separation is useful when a live automation must continue running while edits are prepared. This is a production-operation pattern, not a universal prerequisite for editing a personal workflow. |

## Observed facts

### 1. First run and home

- Power Automate’s console is the stable entry point. It separates **Home**, **My flows**, **Shared**, and **Examples**, while Start, Edit, and current Status are available from the flow list. An example is copied into **My flows** before it is changed; the original example is not another kind of installed runtime object. ([Console](https://learn.microsoft.com/en-us/power-automate/desktop-flows/console), [Create desktop flows](https://learn.microsoft.com/en-us/power-automate/desktop-flows/create-flow))
- n8n offers two creation paths: an empty workflow whose first prompt is **Add first step**, or an existing template. Its own template documentation says templates are starting material that may still require credentials and local configuration. ([Create and run workflows](https://docs.n8n.io/build/understand-workflows/create-and-run-workflows), [Workflow templates](https://docs.n8n.io/build/ways-of-building-workflows/use-templates))
- Node-RED’s first tutorial goes directly through drag node, configure, wire, deploy, and observe output. It teaches the product through one working loop rather than a prerequisite setup center. ([Creating your first flow](https://nodered.org/docs/tutorials/first-flow))

### 2. Workflow list and library hierarchy

- Power Automate treats examples as a source for creating a copy, while the user’s editable flows remain in one **My flows** collection. Its list-level actions are the user’s core jobs: start, edit, and read status. ([Console](https://learn.microsoft.com/en-us/power-automate/desktop-flows/console))
- n8n’s workflow list is the place to browse and search workflows. Execution history is a separate execution surface with workflow, status, and time filters. ([Workflow sharing](https://docs.n8n.io/workflows/sharing/), [All executions](https://docs.n8n.io/workflows/executions/all-executions/))
- Node-RED imports JSON, local-library items, and installed-node examples into either the current flow or a new flow. Once imported, they are ordinary editable flow content; the source library is not presented as a second runtime inventory. ([Import/export](https://nodered.org/docs/user-guide/editor/workspace/import-export))

### 3. Connections, applications, and run targets

- n8n lets a user select an existing credential or create one from the credential dropdown while editing the node that needs it. Saving the credential tests the connection. This keeps setup contextual while still allowing credentials to be reused. ([Create and edit credentials](https://docs.n8n.io/build/understand-workflows/create-and-edit-credentials))
- Node-RED represents a reusable connection as a configuration node referenced by ordinary nodes. Configuration nodes do not appear on the main canvas; they are managed in a sidebar or from the referencing node’s edit control. ([Concepts](https://nodered.org/docs/user-guide/concepts), [Configuration nodes](https://nodered.org/docs/creating-nodes/config-nodes))
- Node-RED supports environment variables at node, flow/group, and global scopes. Its own flow-structure guidance recommends keeping device-specific customization outside flow logic when the same flow runs on different devices. ([Environment variables](https://nodered.org/docs/user-guide/environment-variables), [Flow structure](https://nodered.org/docs/developing-flows/flow-structure))
- Power Automate keeps UI elements/selectors as resources used by actions. Shared UI-element collections omit the desktop itself so the same application elements can be imported under a local, RDP, or Citrix desktop in each user’s flow. ([UI elements](https://learn.microsoft.com/en-us/power-automate/desktop-flows/ui-elements), [UI element collections](https://learn.microsoft.com/en-us/power-automate/desktop-flows/create-ui-elements-collections))

### 4. Editor progressive disclosure

- Node-RED describes three permanent areas: header, main workspace, and sidebars. Supporting panels such as help, debug, explorer, and configuration nodes live in sidebars that can be rearranged, resized, or hidden. ([Editor guide](https://nodered.org/docs/user-guide/editor/), [Sidebar](https://nodered.org/docs/user-guide/editor/sidebar/))
- Node-RED exposes node-local state directly: undeployed changes use a marker, invalid configuration uses an error marker, and runtime connection state can appear below the node. ([Nodes](https://nodered.org/docs/user-guide/editor/workspace/nodes))
- Power Automate keeps the action catalog searchable and opens an action’s parameter dialog only after the action is added. Defaults reduce required input, while custom error handling is an **On error** section inside the action dialog. Variables and debugging tools have their own panes. ([Actions pane](https://learn.microsoft.com/en-us/power-automate/desktop-flows/actions-pane), [Flow designer](https://learn.microsoft.com/en-us/power-automate/desktop-flows/flow-designer))

### 5. Stability, versions, and migration

- Power Automate states that newer clients remain backward-compatible with flows made by older clients. A flow made by a newer client may refuse to open in an older client, and the error tells the user to update. ([Update compatibility](https://learn.microsoft.com/en-us/power-automate/desktop-flows/how-to/update_power_automate))
- During Power Automate’s v1-to-v2 storage transition, existing v1 flows continue to function; new, modified, or resaved flows use v2. This is an incremental migration path, not a requirement that every workflow be manually reinstalled before the product can be used. ([v2 schema](https://learn.microsoft.com/en-us/power-automate/desktop-flows/schema))
- n8n creates a workflow-history version when saving or restoring. Restoring first preserves the current version, and the UI distinguishes workflow versions from execution history. ([Workflow history](https://docs.n8n.io/build/manage-workflows/view-change-history))
- Node-RED’s Git-backed Projects are optional. Even there, the UI intentionally does not expose all Git operations. ([Projects](https://nodered.org/docs/user-guide/projects/))
- Zapier’s drafts let an already-live Zap continue running while edits are prepared, and publishing creates a version. The benefit depends on a continuously active production workflow. ([Zap drafts and versions](https://help.zapier.com/hc/en-us/articles/9693520498445-Create-Zap-drafts-and-versions))

## Recommendations for Yotta V4

These are product recommendations inferred from the observed patterns, not claims about the cited products.

### First run: reach a successful run in one short path

- Make **Workflows** the default home. If workflows exist, show them immediately; do not interpose a setup dashboard.
- For an empty profile, show exactly three obvious choices: **New workflow**, **Import JSON**, and **Start from example**. An example becomes a normal editable workflow as soon as it is chosen.
- A new blank workflow should open directly in the editor with one prompt: **Add first step**. Ask for a target/application only when the first node actually needs one.
- Preserve recent work and the last run result, but do not turn Home into a metrics dashboard before the user has runs to inspect.

### Library: one workflow inventory, secondary sources

- Keep one canonical workflow list. Imported, copied, or example-derived workflows are ordinary workflows, with a small origin label only if it helps the user understand where they came from.
- Treat examples/templates as a creation source, not a second type of workflow and not an “installed workflow” inventory.
- Optimize each row for the three primary jobs: **open/edit**, **run**, and **understand current condition**. Prioritize name, last result/running state, last modified time, and schedule state. Hide IDs, revisions, node count, schema, and origin behind details unless they resolve a real problem.
- Keep run history separate from workflow discovery. A workflow-specific history view may be reachable from the row/editor, but should not compete with the main list.

### Targets: reusable local configuration, requested in context

- Define a Target as the user’s local binding to an application/window/device; keep it outside portable workflow logic. A workflow node should reference a stable target role or target ID, never embed another user’s executable path as immutable workflow content.
- When a node first needs a target, offer **Choose target** and **Create target** in that node’s configuration. Test the binding immediately and return the user to the node.
- Provide one secondary Targets manager for reuse, repair, and bulk cleanup. Do not require users to visit it before creating or importing a workflow.
- Missing or stale targets should not make a workflow disappear or become read-only. Show the affected node and one repair action; unaffected editing remains available.

### Editor: canvas first, details on demand

- Keep the durable shell to four jobs: searchable node palette, canvas, selected-node inspector, and Run/Stop. Run output/debug can occupy a collapsible panel.
- Open only the selected node’s essential fields first. Put retries, timing, error policy, selector internals, and diagnostics under clear advanced sections.
- Show validity and runtime state where the user acts: on the node and in its inspector. Avoid global readiness pages that force the user to map an abstract failure back to a node.
- Use an explicit unsaved indicator and make Save predictable. Do not add publish/release terminology unless Yotta later supports a genuinely separate continuously running production version.

### Stability: compatibility and recovery are product features

- Adopt a hard V4 compatibility rule: upgrading Yotta must open and run older supported workflows without manual reinstall. Migrate a copy transactionally, keep the prior artifact/snapshot, and swap only after validation succeeds.
- Run safe schema/contract migrations automatically. Show no modal for a successful no-action migration; show a concise repair screen only when Yotta cannot preserve behavior automatically.
- Before every destructive save, migration, or automated repair, keep an automatic workflow snapshot. “Restore previous version” should create a new current snapshot rather than deleting later history.
- Distinguish **workflow history** from **run history**. The former answers “what changed?”; the latter answers “what happened when it ran?”
- Record the last successfully opened schema and runtime contract per workflow. If a workflow was created by a newer incompatible Yotta, keep the data untouched and state exactly which version is required.
- Versioning should be invisible insurance for the normal user. Draft/publish, release channels, dependency plans, and Git concepts should remain out of the V4 core until Yotta has a real multi-user or always-on deployment scenario that requires them.

## Product direction distilled

The comparable products converge on a simple shape:

1. One place contains the user’s workflows.
2. New/import/example all lead to the same editable workflow.
3. The editor asks for a connection or target at the node that needs it.
4. Advanced configuration, diagnostics, and history stay close but secondary.
5. Compatibility and snapshots protect the user silently; migration vocabulary appears only when intervention is unavoidable.

For Yotta V4, “清爽、快速、稳定” therefore means fewer concepts in the default path, not fewer capabilities: one Workflow model in the user interface, contextual Target binding, a canvas-first editor, visible run feedback, and automatic backward-compatible recovery underneath.
