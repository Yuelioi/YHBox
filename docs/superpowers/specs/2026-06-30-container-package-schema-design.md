# Container Package Schema Design

## Goal

Redesign the container storage model so every container is a publishable package from day one.

The design must support local-only use, direct local submission, registry installation, updates, forks, and future online browsing without maintaining separate "local container" and "online container" schemas.

This is a breaking schema change. The implementation should not preserve the old `container.json` shape.

## Current Findings

The current backend stores one file per container:

```text
data/containers/<id>/container.json
```

That file currently mixes several responsibilities:

- Identity: `id`
- Display metadata: `name`, `description`, `tags`
- Local runtime preference: `hotkey`, `inputBackend`, `captureBackend`, `scaleTolerance`
- Executable graph: `graph.nodes`, `graph.edges`
- Container variables: `vars`
- Local timestamps: `createdAt`, `updatedAt`

The current `Container` intentionally has no `category`; tests assert that `"category"` is absent from serialized container JSON.

Subgraphs and assets already use richer metadata:

- Subgraphs have `label`, `description`, `category`, `tags`, `inputParams`, `outputPins`, `requiredGlobals`, and `graph`.
- Assets have stable GUID identity, `name`, `description`, `category`, `tags`, `origin`, variants, blobs, and created time.

The node layer is not a pure JSON-schema system. Node shape comes from Go `node.Spec`, while node instance config is stored as `GraphNode.Config map[string]any`. The package schema must respect that:

- The graph owns nodes and edges.
- Nodes own their config.
- The manifest can summarize required targets, permissions, and dependencies, but must not replace graph semantics.

## Design Principles

Container package and installation are different concepts:

- A package is portable, publishable, signable, and suitable for registry submission.
- An installation is this machine's binding of that package: hotkeys, local target matches, local AI credentials, aliases, favorites, update settings, and last-run timestamps.

There is still only one container product model:

- Local draft containers are packages with `publication.state = "draft"`.
- Published containers are the same package shape with registry fields filled.
- Imported or installed containers are the same package shape plus an installation file.
- Forks are packages with new identity and provenance pointing to the source.

The graph remains graph data. Do not move nodes into the manifest. The manifest describes the package; `graph.json` describes execution.

## Storage Layout

Each local installed container directory uses this layout:

```text
data/containers/<instanceId>/
  package.json
  graph.json
  installation.json
  yotta-lock.json
```

`package.json` is the package manifest. It is hand-editable enough to inspect, but owned by the app.

`graph.json` is the main executable graph. It contains the same graph shape as today: `id`, `version`, `nodes`, and `edges`.

`installation.json` is machine-local state. It is not uploaded, not signed, and not overwritten by package updates unless the user explicitly resets local settings.

`yotta-lock.json` is generated from `package.json`, `graph.json`, global subgraphs, assets, clips, and node specs. It is the publish/update integrity and safety summary.

## Publish Bundle Layout

Submission and export should package a complete dependency closure:

```text
package.json
graph.json
yotta-lock.json
subgraphs/<subgraphId>.json
assets/records/<guid>.json
assets/blobs/<sha256>
clips/<clipId>.json
clips/blobs/<sha256>
```

The bundle must not contain `installation.json`, local AI credentials, local hotkeys, or local run history.

## package.json

The manifest intentionally follows npm naming where it fits, then uses a `yotta` namespace for Yotta-specific data.

```json
{
  "schemaVersion": 2,
  "kind": "yotta.container",
  "name": "@yhfish/daily-fishing",
  "displayName": "每日钓鱼",
  "version": "0.1.0",
  "description": "自动完成每日钓鱼循环",
  "summary": "每日钓鱼自动化",
  "keywords": ["fishing", "daily", "mumu"],
  "category": "daily",
  "license": "MIT",
  "author": {
    "name": "yl",
    "email": "",
    "url": ""
  },
  "contributors": [],
  "publisher": {
    "id": "yhfish",
    "name": "YHFish"
  },
  "homepage": "",
  "repository": {
    "type": "git",
    "url": ""
  },
  "bugs": {
    "url": "",
    "email": ""
  },
  "docs": {
    "url": ""
  },
  "changelog": {
    "url": ""
  },
  "yotta": {
    "uid": "pkg_01jz_daily_fishing",
    "entryGraph": "graph.json",
    "publication": {
      "state": "draft",
      "visibility": "private",
      "registryUrl": "",
      "updateUrl": "",
      "downloadUrl": "",
      "publishedAt": "",
      "contentHash": "",
      "signature": ""
    },
    "provenance": {
      "origin": "local",
      "upstreamName": "",
      "upstreamVersion": "",
      "forkedFrom": "",
      "importedAt": ""
    },
    "runtimeDefaults": {
      "inputBackend": "postmessage",
      "captureBackend": "auto",
      "scaleTolerance": 1
    },
    "vars": [],
    "targets": {
      "game": {
        "kind": "win32-window",
        "displayName": "游戏窗口",
        "defaultMatch": {
          "title": "",
          "class": "",
          "processName": "",
          "titleMatch": "contains"
        }
      }
    },
    "ai": {
      "main": {
        "displayName": "默认 AI",
        "providerHint": "openai-compatible",
        "modelHint": ""
      }
    }
  },
  "createdAt": "",
  "updatedAt": ""
}
```

### Manifest Field Rules

`name` is the portable package name. It should be stable, unique in the registry, and can use npm-like scope syntax.

`displayName` is the user-facing name shown in Yotta.

`version` follows semver and is the update comparison value.

`category` is a single primary grouping key. `keywords` are multi-tag search terms. The UI may present them as category and tags.

`publisher` is the publishing account or organization. `author` and `contributors` are human authors.

`homepage`, `repository`, `bugs`, `docs`, and `changelog` are top-level because they are common package metadata and should be visible on online detail pages.

`yotta.uid` is Yotta's registry-stable package identity. It must not be the local `instanceId`.

`yotta.publication.state` is one of:

- `draft`
- `submitted`
- `published`
- `rejected`
- `archived`

`yotta.publication.visibility` is one of:

- `private`
- `unlisted`
- `public`

`yotta.provenance.origin` is one of:

- `local`
- `registry`
- `import`
- `fork`

`yotta.targets` declares logical target slots used by the graph. It does not select a live local window or device.

`yotta.ai` declares logical AI slots used by the graph. It does not contain credentials.

## graph.json

The main container graph stays close to the current `Graph` model.

```json
{
  "id": "graph_uuid",
  "version": 1,
  "nodes": [
    {
      "id": "start",
      "kind": "Start",
      "x": 100,
      "y": 120,
      "config": {},
      "createdAt": "2026-06-30T00:00:00Z"
    },
    {
      "id": "target",
      "kind": "Win32WindowTarget",
      "x": 100,
      "y": 260,
      "config": {
        "Target": "game"
      },
      "createdAt": "2026-06-30T00:00:00Z"
    },
    {
      "id": "ai1",
      "kind": "AI",
      "x": 400,
      "y": 260,
      "config": {
        "Connection": "main",
        "Model": "",
        "User": "判断当前界面"
      },
      "createdAt": "2026-06-30T00:00:00Z"
    }
  ],
  "edges": [
    {
      "from": "start.Done",
      "to": "target.In"
    }
  ]
}
```

Graph nodes remain the executable truth. Package metadata may summarize requirements but must not duplicate node behavior.

Target selection nodes should support logical binding IDs:

- `Win32WindowTarget.config.Target = "game"`
- `AndroidTarget.config.Target = "emulator"`

AI nodes should support logical connection IDs:

- `AI.config.Connection = "main"`

For a first breaking implementation, old direct fields can be removed from newly written data. Runtime can resolve the logical binding through `installation.json` and fall back to manifest defaults when no installation binding exists.

## installation.json

Installation state is local to this machine.

```json
{
  "schemaVersion": 1,
  "instanceId": "local_uuid",
  "packageName": "@yhfish/daily-fishing",
  "installedVersion": "0.1.0",
  "display": {
    "favorite": false,
    "hidden": false,
    "alias": ""
  },
  "runtimeOverrides": {
    "hotkey": "Ctrl+Shift+1",
    "inputBackend": "",
    "captureBackend": "",
    "scaleTolerance": 0
  },
  "targetBindings": {
    "game": {
      "kind": "win32-window",
      "match": {
        "title": "Blue Archive",
        "class": "",
        "processName": "",
        "titleMatch": "contains"
      }
    }
  },
  "aiBindings": {
    "main": {
      "connectionId": "local-openai-connection"
    }
  },
  "updates": {
    "autoCheck": true,
    "pinned": false,
    "lastCheckedAt": "",
    "availableVersion": ""
  },
  "installedAt": "",
  "lastRunAt": "",
  "updatedAt": ""
}
```

Local fields never go into a submission bundle:

- Hotkeys
- Favorite and hidden status
- Display alias
- Local target matches
- ADB serials
- Local AI connection IDs
- Last run time
- Local update check cache

Package updates must preserve `installation.json` by default.

## yotta-lock.json

The lock file is generated, not hand-authored.

```json
{
  "schemaVersion": 1,
  "packageName": "@yhfish/daily-fishing",
  "version": "0.1.0",
  "graphHash": "sha256-...",
  "nodeKinds": ["Start", "Win32WindowTarget", "CheckTemplate", "ClickTemplate", "Subgraph", "AI"],
  "targetKinds": ["win32-window"],
  "capabilities": ["screenshot", "click", "key-state"],
  "permissions": {
    "screenCapture": true,
    "windowControl": true,
    "globalInput": false,
    "network": false,
    "ai": true
  },
  "dependencies": {
    "subgraphs": [],
    "templates": [],
    "clips": [],
    "ai": []
  }
}
```

The lock file should be regenerated before validation, export, submission, and publishing.

The app should reject a submission if the lock file is stale relative to `package.json`, `graph.json`, or bundled dependency contents.

## Node And Dependency Model

The existing `node.Dependencer` interface remains the right primitive. It already lets node implementations extract dependencies from config without hard-coding node kinds in the scanner.

The dependency scanner should be extended, not replaced:

- Keep scanning root graph nodes.
- Keep BFS into subgraphs through `Subgraph` and `CollapsedNode`.
- Keep script static extraction for literal template, clip, and subgraph IDs.
- Change closure output so template and clip dependencies are separate arrays.
- Add an AI requirement scan.
- Add a target requirement and capability scan from node specs.

The current closure result merges template and clip into `AssetGUIDs`. That is too coarse for publish bundles because templates and clips have different record formats and runtime services. The new result should keep:

```go
type ClosureResult struct {
    SubgraphIDs  []string
    TemplateGUIDs []string
    ClipIDs []string
    AIRefs []AIRef
    TargetKinds []string
    Capabilities []string
}
```

AI refs should not export credentials. They should only identify logical slots and model/provider hints.

## Target Binding

Targets are graph behavior, not manifest behavior. The graph still decides when to switch target and which target node is used.

The manifest declares logical slots:

```json
"targets": {
  "game": {
    "kind": "win32-window",
    "displayName": "游戏窗口",
    "defaultMatch": {
      "title": "",
      "class": "",
      "processName": "",
      "titleMatch": "contains"
    }
  }
}
```

The installation binds those slots to local reality:

```json
"targetBindings": {
  "game": {
    "kind": "win32-window",
    "match": {
      "title": "Blue Archive",
      "class": "",
      "processName": "",
      "titleMatch": "contains"
    }
  }
}
```

This avoids publishing local ADB serials or local window titles while still allowing a package author to provide useful defaults.

## AI Binding

AI nodes currently use `Connection` to point at a local settings connection. That cannot be published as-is.

New packages should treat `Connection` as a logical slot name:

```json
"ai": {
  "main": {
    "displayName": "默认 AI",
    "providerHint": "openai-compatible",
    "modelHint": "gpt-4.1-mini"
  }
}
```

The installation maps the logical slot to a local credential:

```json
"aiBindings": {
  "main": {
    "connectionId": "local-openai-connection"
  }
}
```

Submission validation must fail if an AI node references an undeclared logical AI slot.

Runtime must fail with a user-facing binding error if an AI slot has no local `aiBindings` entry.

## Permissions

Permissions should be derived from node specs and stored in `yotta-lock.json`.

Initial mapping:

- `screenCapture`: any node with `TargetCapabilityScreenshot`, image capture, template matching, color detection, QR detection, or frame diff.
- `windowControl`: `Win32WindowTarget`, `GetWindow`, `MoveResizeWindow`, `WindowState`, `CloseWindow`, `WaitWindow`, `WaitWindowGone`, foreground actions.
- `globalInput`: sendinput-style foreground input or any action requiring global OS input.
- `network`: AI nodes or future HTTP nodes.
- `ai`: AI nodes.

The permission display is for review and submission safety. Runtime still relies on node specs and controllers.

## Validation Rules

Package validation should run in layers:

1. Manifest validation
   - `kind == "yotta.container"`
   - supported `schemaVersion`
   - valid package `name`
   - valid semver `version`
   - `yotta.entryGraph` points to an existing graph file
   - all declared target and AI slot names are unique and non-empty

2. Graph validation
   - existing graph structural validators
   - node kind and pin validation
   - target capability validation
   - AI prompt validation

3. Binding validation
   - every logical target used by target nodes exists in `package.yotta.targets`
   - every logical AI connection used by AI nodes exists in `package.yotta.ai`
   - installed containers also validate that required target and AI slots are bound or have usable defaults

4. Dependency validation
   - every referenced subgraph is bundled or available locally
   - every referenced template record and blob are present
   - every referenced clip record and blob are present
   - generated lock matches the current graph and dependency closure

5. Publish validation
   - no `installation.json` in the bundle
   - no local AI credential values
   - no local hotkey
   - no local run/update cache
   - permissions and dependencies in the lock match generated values

## Update Rules

Package update replaces:

- `package.json`
- `graph.json`
- bundled subgraphs and assets according to import policy
- `yotta-lock.json`

Package update preserves:

- `installation.json`
- local hotkey
- local target bindings
- local AI bindings
- alias, favorite, hidden
- local update settings

If a new package version adds a target or AI slot, the installation becomes "needs binding" until the user fills it.

If a new package version removes a slot, stale local binding entries can remain harmlessly or be cleaned by a maintenance pass.

## UI Impact

The local container list should read display fields from `package.json`:

- `displayName`
- `description`
- `category`
- `keywords`
- `version`
- `author`
- `publisher`
- `updatedAt`

The list can read graph summary from `graph.json`:

- node count
- target kinds
- AI usage
- validation status

The list can read installation state from `installation.json`:

- hotkey
- favorite
- alias
- hidden
- update availability
- last run time

Online container cards and local container cards can share the same package metadata shape. Local cards have installation affordances; online cards have install/update/open detail affordances.

## Breaking Implementation Strategy

This change should intentionally replace the old model:

- Remove top-level container `name`, `description`, `tags`, `hotkey`, `inputBackend`, `captureBackend`, `scaleTolerance`, `vars`, and `graph` from the persisted container file.
- Replace `container.json` with `package.json`, `graph.json`, `installation.json`, and generated `yotta-lock.json`.
- Update backend `Container` to become an aggregate view used by RPC, not a direct mirror of one JSON file.
- Update frontend `Container` interface to mirror the aggregate view returned by backend, while keeping package/installation data accessible where needed.
- Update MCP authoring examples to produce the new package and graph layout.
- Update store load/save/reload/delete to work on container directories with multiple files.

No migration code is required for old local data in this design. A later implementation plan can decide whether to provide a one-off external converter, but the runtime schema itself should not carry backward compatibility branches.

## Testing

Backend tests:

- New store creates all four files.
- Loading a package directory returns the expected aggregate container view.
- Saving package metadata does not mutate installation state.
- Saving installation state does not mutate package metadata.
- Graph validation still catches dangling edges, invalid pins, missing Start, missing target, and missing subgraph.
- Dependency closure separates subgraphs, templates, and clips.
- Script dependencies still mark literal template, clip, and subgraph references.
- AI logical slot validation rejects undeclared slots.
- Target logical slot validation rejects undeclared slots.
- Generated lock changes when graph nodes, dependency references, or package version changes.

Frontend tests:

- Container list reads category, keywords, version, author, and updated time from package metadata.
- Hotkey and favorite come from installation state.
- Missing binding state is visible in local containers.
- Online container cards can render from package metadata without installation state.

Verification commands:

```text
go test ./internal/services/container/...
go test ./internal/services/container/dependency/...
go test ./internal/node/...
pnpm --dir frontend typecheck
pnpm --dir frontend build:dev
```

## Implementation Defaults

Package names use an npm-like rule: lowercase, optional scope, one slash allowed only after a scope, and no spaces. Example: `@yhfish/daily-fishing`.

Bundle files use the `.yotta-container.zip` extension.

Container variable declarations live in `package.json` under `yotta.vars`. They are package-level API/data declarations, usually small, and should travel with the package manifest.

Named subgraphs remain in the global runtime pool after install. Publish bundles carry copies of the referenced subgraphs, and import writes them into the global pool with stable IDs.
