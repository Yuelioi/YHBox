# 容器包 Schema 设计

## 目标

破坏式重设计容器存储模型, 让每个容器从创建第一天起就是一个可发布包。

这个模型必须同时支持:

- 纯本地使用
- 本地直接投稿
- 在线安装
- 在线更新
- fork 后重新发布
- 私有 / 未列出 / 公开发布

不再维护“本地容器 schema”和“在线容器 schema”两套模型。本地、在线、投稿、导入、fork 都是同一种 package, 差别只在发布状态、来源状态和本机安装态。

## 当前事实

当前容器存储是单文件:

```text
data/containers/<id>/container.json
```

这个文件混了多类职责:

- 身份: `id`
- 展示元数据: `name`, `description`, `tags`
- 本机运行偏好: `hotkey`, `inputBackend`, `captureBackend`, `scaleTolerance`
- 可执行节点图: `graph.nodes`, `graph.edges`
- 容器变量: `vars`
- 本地时间: `createdAt`, `updatedAt`

当前 `Container` 没有 `category`; 单测还明确断言序列化 JSON 不应包含 `"category"`。但子图和资产已经有 `category + tags`, 说明产品语义是成熟的, 只是容器层当时删掉了 category。

底层节点模型不是纯 JSON schema。节点结构来自 Go 侧 `node.Spec`, 节点实例配置存在 `GraphNode.Config map[string]any`。因此新 schema 不能把节点语义抽到 manifest 顶层:

- 图继续拥有节点和边。
- 节点继续拥有自己的 config。
- manifest 只描述包身份、展示、发布、默认绑定槽位和运行默认值。
- lock 文件由扫描器总结依赖、权限和能力。

## 核心原则

容器包和本机安装态必须分开:

- package 是可移植、可投稿、可发布、可签名、可更新的内容。
- installation 是当前机器对这个 package 的绑定: 热键、窗口匹配、ADB 设备、AI 凭据映射、收藏、别名、更新策略、最近运行时间。

所有容器都是 package:

- 本地新建 = `publication.state = "draft"` 的 package。
- 投稿 = 同一个 package 补齐发布字段后打包上传。
- 在线安装 = 同一个 package 下载到本机, 额外生成 installation。
- fork = 新 package 身份 + provenance 指向来源。

节点图仍是节点图。不要把节点塞进 `package.json`。

## 本地存储布局

每个本机安装实例目录:

```text
data/containers/<instanceId>/
  package.json
  graph.json
  installation.json
  yotta-lock.json
```

职责:

- `package.json`: 包 manifest。描述这个容器包是什么、谁写的、版本、文档、发布、默认 target/AI 槽位。
- `graph.json`: 主容器节点图。只放 `id`, `version`, `nodes`, `edges`。
- `installation.json`: 本机安装态。不投稿、不签名、不被普通更新覆盖。
- `yotta-lock.json`: 生成文件。记录依赖闭包、权限、能力、hash, 用于投稿/更新校验。

## 投稿/导出包布局

投稿或导出时打包完整依赖闭包:

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

bundle 扩展名固定为:

```text
.yotta-container.zip
```

bundle 不能包含:

- `installation.json`
- 本机热键
- 本机 ADB serial
- 本机窗口标题覆盖
- 本机 AI connection id / API key
- 最近运行时间
- 本机更新检查缓存

## package.json

`package.json` 使用 npm 风格顶层字段, Yotta 专属字段放 `yotta`。

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

字段规则:

- `name`: 可发布包名, 使用 npm-like 规则。小写, 可选 scope, slash 只能出现在 scope 后, 不允许空格。例如 `@yhfish/daily-fishing`。
- `displayName`: UI 展示名。
- `version`: semver, 更新比较用。
- `category`: 单一主分类。
- `keywords`: 多标签/搜索词。
- `publisher`: 发布账号或组织。
- `author` / `contributors`: 实际作者。
- `homepage` / `repository` / `bugs` / `docs` / `changelog`: 在线详情页直接展示的链接。
- `yotta.uid`: Yotta registry 里的稳定包身份, 不能等同本机 `instanceId`。
- `yotta.vars`: 容器级变量声明。变量是 package API/data 声明, 跟包一起走。

`publication.state` 固定枚举:

- `draft`
- `submitted`
- `published`
- `rejected`
- `archived`

`publication.visibility` 固定枚举:

- `private`
- `unlisted`
- `public`

`provenance.origin` 固定枚举:

- `local`
- `registry`
- `import`
- `fork`

## graph.json

主图保持接近当前 `Graph`:

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

重点:

- `Win32WindowTarget` 和 `AndroidTarget` 仍然是图内节点, 不抽到 manifest 顶层。
- target 节点使用逻辑槽位, 例如 `config.Target = "game"`。
- AI 节点使用逻辑槽位, 例如 `config.Connection = "main"`。
- 运行时通过 `installation.json` 把逻辑槽位解析成本机窗口匹配、ADB 设备或 AI 连接。

## installation.json

本机安装态:

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

普通 package 更新默认保留整个 `installation.json`。

如果新版本新增 target 或 AI 槽位, 本机安装进入 `needs binding` 状态, 直到用户补齐绑定。

如果新版本删除槽位, 本机 stale binding 可以保留为无害冗余, 后续 walkaround/维护动作再清理。

## yotta-lock.json

lock 是生成文件, 不手写。

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

生成时机:

- 保存前校验
- 导出前
- 投稿前
- 发布前
- 更新包安装前

如果 lock 和当前 `package.json` / `graph.json` / 依赖闭包不一致, 投稿和发布必须拒绝。

## 节点与依赖模型

现有 `node.Dependencer` 是正确底座, 继续用:

- 主图节点作为根。
- `Subgraph` / `CollapsedNode` 通过 `SubgraphID` BFS 进入子图闭包。
- `CheckTemplate` / `ClickTemplate` / `WaitTemplate` / `WaitTemplateGone` / `FindTemplateAll` 抽 template GUID。
- `PlayClip` 抽 clip ID。
- `Script` 继续静态扫描字面 template / clip / subgraph 引用。

需要扩展:

- 当前 `ClosureResult` 把 template 和 clip 合成 `AssetGUIDs`, 发布包里不够用。必须拆成 `TemplateGUIDs` 和 `ClipIDs`。
- AI 节点需要扫描逻辑 AI 槽位, 但不能导出本机 credential。
- target 需求和 capability 需要从 `node.Spec` 派生。

建议闭包结果:

```go
type ClosureResult struct {
    SubgraphIDs   []string
    TemplateGUIDs []string
    ClipIDs       []string
    AIRefs        []AIRef
    TargetKinds   []string
    Capabilities  []string
}
```

## Target 绑定

目标选择是图行为, 不是 manifest 行为。manifest 只声明逻辑 target 槽位。

manifest:

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

installation:

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

这样既能投稿, 又不会把本机窗口标题、ADB serial 等环境事实写死到包主体。

## AI 绑定

AI 节点现在的 `Connection` 指向本机 settings 里的连接, 不能原样发布。

新模型中 `Connection` 是逻辑槽位:

```json
"ai": {
  "main": {
    "displayName": "默认 AI",
    "providerHint": "openai-compatible",
    "modelHint": "gpt-4.1-mini"
  }
}
```

本机安装绑定到真实 credential:

```json
"aiBindings": {
  "main": {
    "connectionId": "local-openai-connection"
  }
}
```

投稿校验:

- AI 节点引用未声明 slot: 失败。
- bundle 内出现本机 credential: 失败。

运行校验:

- AI slot 未绑定本机 connection: 显示需要绑定, 不静默失败。

## 权限模型

权限由节点 spec 和实际图扫描派生, 写入 `yotta-lock.json`。

初始映射:

- `screenCapture`: screenshot capability、截图、模板匹配、颜色检测、QR、帧差。
- `windowControl`: Win32 target、GetWindow、MoveResizeWindow、WindowState、CloseWindow、WaitWindow、WaitWindowGone、前台激活。
- `globalInput`: SendInput 类前台输入或需要 OS 全局注入的动作。
- `network`: AI 节点或未来 HTTP 节点。
- `ai`: AI 节点。

权限展示用于投稿/安装前审查。运行时仍以节点 spec、controller profile 和 validator 为准。

## 校验规则

分层校验:

1. Manifest
   - `kind == "yotta.container"`
   - 支持的 `schemaVersion`
   - 合法 package `name`
   - 合法 semver `version`
   - `yotta.entryGraph` 指向存在的图文件
   - target / AI slot 名称唯一且非空

2. Graph
   - 复用现有图结构校验
   - node kind / pin 校验
   - target capability 校验
   - AI prompt 校验

3. Binding
   - target 节点引用的逻辑 target 必须在 `package.yotta.targets` 声明
   - AI 节点引用的逻辑 AI slot 必须在 `package.yotta.ai` 声明
   - 已安装容器还要检查 required slot 是否已有本机绑定或可用默认值

4. Dependency
   - 子图存在且在 bundle 或本地池可解析
   - template record 和 blob 存在
   - clip record 和 blob 存在
   - lock 与当前闭包一致

5. Publish
   - bundle 不含 `installation.json`
   - bundle 不含本机 AI credential
   - bundle 不含本机 hotkey
   - bundle 不含本机运行/更新缓存
   - lock 中权限和依赖与生成结果一致

## 更新规则

包更新替换:

- `package.json`
- `graph.json`
- bundled subgraphs / assets / clips, 按导入策略合并
- `yotta-lock.json`

包更新保留:

- `installation.json`
- 本机热键
- 本机 target 绑定
- 本机 AI 绑定
- alias / favorite / hidden
- 本机更新策略

## UI 影响

本地容器列表从聚合视图读取:

- package: `displayName`, `description`, `category`, `keywords`, `version`, `author`, `publisher`, `updatedAt`
- graph: node count, target kinds, AI usage, validation status
- installation: hotkey, favorite, alias, hidden, update availability, last run time

在线容器和本地容器共用 package 元数据 shape。

区别只在 affordance:

- 本地: 运行、编辑、绑定、更新、删除。
- 在线: 安装、更新、详情、投稿状态。

## 破坏式实施策略

直接替换旧持久化模型:

- 删除持久化文件里的顶层 `name`, `description`, `tags`, `hotkey`, `inputBackend`, `captureBackend`, `scaleTolerance`, `vars`, `graph`。
- 删除旧 `container.json` 写入路径。
- 新增 `package.json`, `graph.json`, `installation.json`, `yotta-lock.json`。
- 后端 `Container` 变成 RPC 聚合视图, 不再等同单个 JSON 文件。
- 前端 `Container` interface 改成聚合视图, 并在需要时暴露 package / installation 明细。
- MCP authoring 示例改成新 layout。
- Store 的 load/save/reload/delete 改成多文件目录模型。

不在 runtime schema 里保留旧格式兼容分支。若后续要救旧数据, 做外部一次性 converter, 不污染主线。

## 测试计划

后端:

- 新建容器会创建四个文件。
- 加载目录会返回正确聚合视图。
- 保存 package 元数据不改 installation。
- 保存 installation 不改 package。
- 图校验仍能抓 dangling edge、invalid pin、missing Start、missing target、missing subgraph。
- 依赖闭包拆分 subgraphs / templates / clips。
- Script 字面依赖仍能被扫到。
- AI 未声明 slot 会报错。
- Target 未声明 slot 会报错。
- graph / 依赖 / package version 变化会导致 lock hash 变化。

前端:

- 容器列表从 package 读 category、keywords、version、author、updated time。
- hotkey 和 favorite 从 installation 读。
- 本机缺 binding 有明确状态。
- 在线容器卡片能只靠 package metadata 渲染。

验证命令:

```text
go test ./internal/services/container/...
go test ./internal/services/container/dependency/...
go test ./internal/node/...
pnpm --dir frontend typecheck
pnpm --dir frontend build:dev
```

## 实施默认值

- package name 使用 npm-like scoped name 规则: 小写、可选 scope、scope 后允许一个 slash、不允许空格。
- bundle 扩展名使用 `.yotta-container.zip`。
- `vars` 放在 `package.json` 的 `yotta.vars`。
- 具名子图运行时仍进入全局池; 发布 bundle 携带副本, 导入时按稳定 ID 写入全局池。
