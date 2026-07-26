# 版本维护主流实践调研

调研日期：2026-07-24

## 范围与方法

本调研只讨论：

1. 产品 SemVer 的单一事实源与 release automation；
2. Go 构建时注入版本和 runtime build info；
3. npm package version 与桌面产品版本的关系；
4. Windows executable、manifest、installer 的静态版本元数据；
5. Wails v3 的版本配置和构建入口；
6. JSON/JSON Schema 持久化格式的 `format + version`；
7. Host/plugin ABI 与 wire protocol 的独立版本和协商；
8. 数据迁移的逐版本策略。

只采用规范、官方文档或成熟项目的官方源码/设计文档。每节先列“来源事实”，再列“对 Yotta
的推论”，避免把项目决策伪装成外部规范。

## 结论摘要

- 产品发布号只是一次应用发布的身份，不应自动成为任何持久化格式、插件接口、wire protocol
  或 store layout 的兼容性版本。
- 对 Yotta 最稳妥的产品版本权威源是一个只含 SemVer 的根目录 `VERSION` 文件。Wails、
  Go、Windows 和 installer 所需的值都是它的受控 projection，不允许人工分别维护。
- `frontend/package.json` 是未发布的私有 npm workspace 元数据，不应继续充当桌面产品版本副本。
- Windows 不同字段的格式约束并不相同：产品 `3.1.0` 应投影为 manifest
  `3.1.0.0`、二进制 `FILEVERSION 3,1,0,0`，而展示字符串仍可为 `3.1.0`。
- Go 产品版本宜通过 `-ldflags -X` 注入；`runtime/debug.ReadBuildInfo` 提供 commit、构建时间和
  dirty 状态等来源证明，两者互补而不是互相替代。
- 每种持久化 artifact 自己拥有 `format + version`，Schema `$id` 包含该格式版本；
  `$schema` 只表示 JSON Schema dialect，不能拿来表示 Yotta artifact 版本。
- Host Interface 与 wire protocol 都必须独立于产品版本。插件包自己的 SemVer、Host Interface
  范围和具体 wire protocol 代际是三个不同维度。
- 正式发布后的迁移应登记显式的相邻版本步骤，并逐步执行、逐步验证；缺少任何一步就应拒绝修改
  用户数据。当前未稳定的开发期 `3.1` 数据可以按已定决策直接停止支持。

## 1. 产品 SemVer、单一事实源与 release automation

### 来源事实

Semantic Versioning 要求先声明被版本号描述的 public API，随后才用
`MAJOR.MINOR.PATCH` 表达不兼容、向后兼容功能和向后兼容修复。已经发布的某个版本内容不得再被
修改。[Semantic Versioning 2.0.0](https://semver.org/)

这意味着 SemVer 描述的是一个明确的兼容性承诺，而不是仓库里所有接口的全局代号。一个产品发布、
一个插件协议和一个持久化文件格式若有不同的 public API，就没有规范依据要求它们保持同一个版本。

Google 的 Release Please 是成熟的 release automation 例子：它根据提交历史维护 Release PR；
合并 Release PR 后更新 changelog 和适用的版本文件、给该提交打 tag，并创建 GitHub Release；
发布到 package registry 是后续独立步骤。它还支持 `simple` 类型，即以一个简单版本文件维护版本。
[Release Please 官方仓库](https://github.com/googleapis/release-please)

GitHub Actions 可以由 `release.published` 事件触发，此时 `GITHUB_REF` 是
`refs/tags/<tag_name>`，因此构建、签名和发布可以绑定到一个不可含糊的 tag。
[GitHub Actions release 事件](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#release)

Wails v3 则把项目产品元数据放在 `build/config.yml`，Taskfile 在生成图标、manifest、installer
等资产时读取它；官方提供 `generate/update build-assets` 刷新平台资产。
[Wails v3 Build System](https://v3.wails.io/concepts/build-system/)

### 对 Yotta 的推论

Yotta 需要区分三个角色：

- **源代码中的权威值：** 根目录 `VERSION`，只含例如 `3.1.0`；
- **release identity：** `v3.1.0` tag，必须与 `VERSION` 相符；
- **构建 projection：** Wails config、Windows resource、installer、Go runtime version 等，由脚本
  从 `VERSION` 生成或校验。

选择根目录 `VERSION` 而不是某个语言或平台配置作为权威源，是 Yotta 的工程决策，不是 SemVer
规范要求。理由是 Yotta 同时包含 Go、npm、Wails 和 Windows 构建，纯文本版本文件没有 YAML/JSON
解析依赖，也不会让某个平台配置成为其他模块的反向依赖。

本地命令和 release automation 应分层：

```text
task version:show
task version:check
task version:sync
task version:bump -- patch|minor|major|<exact-version> [DRY_RUN=true]
task release:prepare
```

- `version:bump` 只更新 `VERSION` 并运行 `version:sync`，默认不 commit、不 tag；
- `release:prepare` 要求 clean tree，运行版本检查、生成检查、适用测试、打包和产物元数据检查；
- commit/tag/release 是单独的显式 release 动作，或由 Release PR 合并后的 CI 完成；
- 若保留本地“一键 commit + tag”，应使用额外的显式开关，而不是让普通 bump 隐含地产生 Git
  历史和 tag。

这比当前脚本一次正则修改多个文件再 commit/tag 更容易审查，也避免部分写入后失败留下半同步状态。

## 2. Go 构建时注入和 runtime build info

### 来源事实

Go linker 的 `-X importpath.name=value` 可以设置一个 string 变量。它只对未初始化变量或由常量
字符串表达式初始化的变量生效，不能修改 `const`。
[Go linker `-X` 官方文档](https://pkg.go.dev/cmd/link)

`go build` 的 `-ldflags` 会把参数传给 linker；`-buildvcs` 控制是否把版本控制信息写入 binary。
[Go command 官方文档](https://pkg.go.dev/cmd/go)

运行中的 Go 程序可用 `runtime/debug.ReadBuildInfo` 读取嵌入的构建信息。已定义的设置包括
`vcs.revision`、`vcs.time` 和 `vcs.modified`，以及 Go toolchain、GOOS、GOARCH 等。
[runtime/debug.ReadBuildInfo](https://pkg.go.dev/runtime/debug#ReadBuildInfo)

### 对 Yotta 的推论

`pkg/version` 中的产品版本应从 `const` 改为可由 linker 覆盖的变量，例如：

```go
var Product = "dev"
```

正式 Task build 从 `VERSION` 构造：

```text
-ldflags "-X <module>/pkg/version.Product=3.1.0"
```

应用内展示和 updater 比较只读取 `Product`。诊断信息另外读取 `debug.ReadBuildInfo`，展示：

```text
product=3.1.0
revision=<full commit>
modified=false
builtAt=<vcs time or CI-provided time>
go=<toolchain>
```

产品版本与 commit 必须分开：

- `3.1.0` 回答“这是什么发布”；
- commit/dirtiness 回答“它由哪份源码构建”。

开发构建可使用 `Product=dev`，但仍保留 build info。release gate 应使用
`wails3 tool buildinfo` 或 `go version -m` 检查最终 EXE，而不是只验证源码文件。
Wails v3 CLI 明确提供 `tool buildinfo`。
[Wails v3 CLI Reference](https://v3.wails.io/reference/cli/)

## 3. npm package version 是否应与桌面产品版本耦合

### 来源事实

npm 官方文档说明：若 package 要发布，`name + version` 共同构成 package identity；若不发布，
name 和 version 都是可选的。设置 `"private": true` 会让 npm 拒绝发布，从而防止私有项目被意外
发布。[npm package.json](https://docs.npmjs.com/files/package.json/)

因此 `package.json.version` 的语义首先是 npm package 的发布版本，不天然是包含该前端的桌面
产品版本。

### 对 Yotta 的推论

Yotta 前端是 Wails 应用的私有 build workspace，并没有作为独立 npm package 发布。它的
`package.json` 应保持：

```json
{
  "private": true,
  "version": "0.0.0"
}
```

或删除 version（前提是现有工具链确实不要求它）。更保守的做法是保留固定 `0.0.0`，不再由产品
版本脚本修改。`package-lock.json` 根 package version 也不再承担产品含义。

前端显示产品版本应继续通过 Go/Wails application service 获得，不能从
`package.json` import。只有未来把前端组件拆成真正发布的 npm package 时，那个 package 才拥有
自己的独立 SemVer。

## 4. Windows executable、manifest 和 installer 元数据

### 来源事实

Win32 `VERSIONINFO` 资源包含 `FILEVERSION` 和 `PRODUCTVERSION`。二进制版本由四个 16-bit
整数组成，同时还可提供 `FileVersion`、`ProductVersion` 等展示字符串。
[Microsoft VERSIONINFO resource](https://learn.microsoft.com/en-us/windows/win32/menurc/versioninfo-resource)

Windows application manifest 的 `assemblyIdentity.version` 是必填字段，必须使用四段
`major.minor.build.revision`，每段为 `0..65535`。
[Microsoft Application manifests](https://learn.microsoft.com/en-us/windows/win32/sbscs/application-manifests)

Windows Installer 的 `ProductVersion` 是三段 `major.minor.build`，第一、第二段最大 255，
第三段最大 65535，并且第四段会被忽略。
[Microsoft Installer ProductVersion](https://learn.microsoft.com/en-us/windows/win32/msi/productversion)

Wails v3 的 Windows packaging 文档说明：应用 metadata 来自 `build/windows/info.json`，
NSIS installer 位于 platform build pipeline；Wails CLI 的 `generate syso` 会生成 icon、
manifest 和 version info 的 Windows `.syso` resource。
[Wails v3 Windows Packaging](https://v3.wails.io/guides/build/windows/),
[Wails v3 CLI Reference](https://v3.wails.io/reference/cli/)

### 对 Yotta 的推论

不能再用“每个文件都等于 `3.1.0`”作为一致性定义。`version:sync` 必须按目标格式投影：

| 目标 | `VERSION=3.1.0` 的输出 |
|---|---|
| Wails 产品版本 | `3.1.0` |
| VERSIONINFO binary `FILEVERSION` | `3,1,0,0` |
| VERSIONINFO binary `PRODUCTVERSION` | `3,1,0,0` |
| VERSIONINFO string `FileVersion` | `3.1.0` |
| VERSIONINFO string `ProductVersion` | `3.1.0` |
| application manifest `assemblyIdentity.version` | `3.1.0.0` |
| installer 展示版本 | `3.1.0` |

Yotta 当前只接受纯数字三段 SemVer，因此这个映射没有歧义。若以后允许
`3.2.0-beta.1`，必须另外定义 Windows numeric projection；不能把 prerelease label 填入要求纯
数字的 manifest 或 binary fields。

`version:check` 应：

- 校验 SemVer；
- 校验 Windows 数字段范围；
- 校验 manifest 是四段；
- 读取生成后的 `.syso`/EXE 和 installer metadata；
- 区分 `FileVersion` 与 `ProductVersion`，即便 Yotta 当前让它们取相同发布号；
- 不把生成文件中语法不同但语义等价的版本误报为 mismatch。

## 5. Wails v3 的版本配置和构建入口

### 来源事实

Wails v3 把 Taskfile 作为构建 orchestrator。`wails3 build` 是 `wails3 task build` 的薄封装，
平台、输出、图标和 packaging 由项目 Taskfile 与 `build/config.yml` 控制。官方文档还明确说
`wails3 build` 没有直接的 `-ldflags` 选项；自定义 linker flags 应在项目 Task 中配置。
[Wails v3 Build System](https://v3.wails.io/concepts/build-system/)

Wails v3 的 `build/config.yml` 是产品 name、identifier、version、NSIS 和平台 metadata 的来源；
`generate/update build-assets` 用它刷新平台资产。Wails 同时鼓励“bring your own tooling”，允许
项目调整 Taskfile 和平台构建步骤。
[Wails v3 Build Customization](https://v3.wails.io/guides/build/customization/)

### 对 Yotta 的推论

版本自动化应作为 Task dependency，而不是要求开发者记住先运行脚本：

```text
build/package
  -> version:check-source
  -> version:sync-or-verify-projections
  -> Wails generate syso / platform resources
  -> frontend build
  -> Go build with product ldflag
  -> post-build metadata verification
```

开发 build 可以按需生成；release build 必须从 clean tree 进行 deterministic generation 并校验
没有未提交 drift。

Wails 的 build assets 允许项目定制，Yotta 已经有自己的 Windows manifest。因此不应在每次
构建时无条件用默认模板覆盖整份目录。可选择以下任一可重复方案：

1. `VERSION` 写入 Wails config 后，使用受版本控制的 Yotta 模板生成 Windows assets；
2. 在临时目录运行 Wails build-assets generator，再把明确管理的 metadata 字段投影到定制模板；
3. 对结构化 JSON/YAML 使用解析器更新，并对 XML manifest 使用 XML 解析器或唯一占位符模板。

不要继续使用能误匹配任意 `"version"` 字段的全仓正则替换。

## 6. JSON/JSON Schema 持久化格式

### 来源事实

JSON Schema 中，`$schema` 指定所使用的 JSON Schema dialect；`$id` 为 schema 设置唯一 URI，
供 schema registry 和 `$ref` 引用。官方建议 `$id` 使用绝对 URI。
[JSON Schema Getting Started](https://json-schema.org/learn/getting-started-step-by-step),
[JSON Schema schema identification](https://json-schema.org/understanding-json-schema/structuring)

`additionalProperties: false` 会拒绝 schema 未声明的属性，所以“只增加一个 optional field”
对旧的 strict reader 也可能不兼容。
[JSON Schema object reference](https://json-schema.org/understanding-json-schema/reference/object)

Kubernetes 的对象显式携带 `apiVersion + kind`；API server 可同时提供多个 API representation，
并在它们与实际 storage representation 之间转换。Kubernetes 的产品发布版本与资源的
`apiVersion` 不相等。
[The Kubernetes API](https://kubernetes.io/docs/concepts/overview/kubernetes-api/),
[Kubernetes Storage Versions](https://kubernetes.io/docs/concepts/overview/working-with-objects/storage-version/)

### 对 Yotta 的推论

每个持久化对象根部保留：

```json
{
  "format": "yotta.workflow",
  "version": "1"
}
```

两个字段一起构成 reader/migration dispatch key。每个 owning module 自己声明：

```text
Format = "yotta.workflow"
CurrentVersion = "1"
```

Schema 应类似：

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://schemas.yotta.app/workflow/v1/workflow.schema.json"
}
```

这里：

- JSON `$schema` 是第三方标准 dialect；
- `$id` 标识 Yotta schema resource；
- instance 中的 `format + version` 决定应用 reader 和 migration；
- 目录 `/v1/` 只是当前 Schema 的物理组织，不应暴露给普通前端消费者。

Yotta 的 strict/closed Schema 政策意味着版本提升规则必须按实际 reader/writer matrix 制定：

- 新 writer 输出的新字段若会被旧 reader 拒绝，就是 breaking；
- 只修改 description、example、生成顺序，不改变可接受 instance 集合，不提升；
- 修复 validator、canonicalizer 或语义解释，即便 JSON shape 未变，也可能需要提升该 artifact
  version；
- 产品 `3.1.1 -> 3.2.0` 不触发任何 artifact version 自动提升。

## 7. Host/plugin ABI 与 wire protocol

### 来源事实

Terraform 官方明确说明，其 plugin protocol 是 Terraform CLI 与 plugins 之间的独立 versioned
interface。protocol major 划分兼容性，minor 是 additive；Registry 还把 plugin 支持的 protocol
versions 作为选择 plugin release 的兼容性 metadata。Terraform CLI 1.x 产品发布与 plugin
protocol 5/6 显然是两个版本域。
[Terraform Plugin Protocol](https://developer.hashicorp.com/terraform/plugin/terraform-plugin-protocol),
[Terraform Registry protocol metadata](https://developer.hashicorp.com/terraform/registry/providers/publishing#terraform-registry-manifest-file)

HashiCorp `go-plugin` 在握手层提供 protocol version，接口签名或底层协议发生不兼容变化时提升，
不兼容时返回可读错误。
[HashiCorp go-plugin](https://github.com/hashicorp/go-plugin)

Kafka 使用 `ApiVersionsRequest` 让 broker 返回每个 API key 支持的版本集合；若 client 和 broker
有多个共同版本，推荐使用双方支持的最新版本。这是一种显式 capability/range negotiation。
[Apache Kafka Protocol](https://kafka.apache.org/28/design/protocol/)

Protocol Buffers 进一步区分 wire-safe、wire-unsafe 和需要 rollout 管理的条件兼容变化，说明
“schema 文件改了”不等于“必须跟产品一起升级”。
[Protocol Buffers proto3 update rules](https://protobuf.dev/programming-guides/proto3/#updating)

### 对 Yotta 的推论

Yotta 应至少区分：

- **Plugin Package Version：** 某个插件 release 自己的 SemVer；
- **Host Interface Version：** 插件期望宿主暴露的 API/ABI，建议从 `1.0` 开始；
- **Wire Protocol Version：** 子进程握手和消息结构，例如 `yotta.script.worker/1`；
- **Provider-specific ABI：** 现有 capability/provider URI 的 `/v1`，继续由 provider domain 拥有。

Host Interface 由独立 module 拥有：

```text
Current = 1.0
Supported = [1.0, 2.0)
```

插件 manifest 声明最小值和 max-exclusive，loader 计算交集并选择共同支持的最高版本。若当前只实现
一个版本，也应经过同一兼容函数，而不是比较产品 `"3.1"`。

Host minor 只允许 additive 且旧插件可忽略的能力；删除、改签名、改变安全/授权语义等必须提升
Host major。wire protocol 版本在握手发生在解析普通业务消息之前，不兼容就明确失败。

不要使用以下关系：

```text
Yotta 3.2.0 => Host 3.2 => Script Worker /3.2
```

正确关系是独立 inventory，例如：

```text
product                 3.2.0
host-interface          1.0
script-worker           1
plugin-protocol         1
```

## 8. 逐版本数据迁移

### 来源事实

SQLite 为应用保留 `PRAGMA user_version`，SQLite 自身不解释该整数。这是 storage engine 与应用
schema version 分离的直接例子。
[SQLite PRAGMA user_version](https://www.sqlite.org/pragma.html#pragma_user_version)

Android Room 把 migration 明确定义为 `startVersion -> endVersion`，示例分别注册
`1 -> 2` 和 `2 -> 3`；它要求导出历史 schema 并测试 migration。找不到完整 migration path 时默认
失败，destructive recreation 必须由应用显式开启且会永久删除数据。
[Android Room database migrations](https://developer.android.com/training/data-storage/room/migrating-db-versions)

Terraform resource state 把 schema version 保存进 state；当保存版本与当前版本不同时，请求
provider 执行对应 prior version 的 upgrade，缺少该 prior version 的 upgrader 就返回错误。
[Terraform State Upgrade](https://developer.hashicorp.com/terraform/plugin/framework/resources/state-upgrade)

Kubernetes 可以同时识别多个 storage versions，并在读取/更新时转换；旧对象被重新写入时使用当前
storage version，也提供主动重写旧 storage version 的 migration。
[Kubernetes CRD versioning](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/),
[Kubernetes storage version migration](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/storage-version-migration/)

Terraform 还明确区分产品版本和 state format version：只有 state format 真正变化时才更新后者，
不是每次 Terraform 产品升级都改变 state version。
[Terraform version and state format](https://developer.hashicorp.com/terraform/tutorials/configuration-language/versions)

### 对 Yotta 的推论

正式 `v1` 后，每个 artifact 使用独立 registry：

```text
(format, 1 -> 2)
(format, 2 -> 3)
```

加载流程应为：

```text
读取最小 envelope
  -> 校验 format
  -> 若 version > current，拒绝且不写盘
  -> 查找 from -> from+1
  -> 在内存或临时文件迁移
  -> 按目标 Schema + 领域规则验证
  -> 重复直到 current
  -> 原子提交新内容
```

默认只要求相邻步骤，避免一个 `1 -> 7` 函数复制所有历史知识。可以增加经过等价性测试的 fast path，
但不能替代 canonical chain。

每次版本提升应提交：

- 旧版本 Schema/fixture；
- 新版本 Schema/fixture；
- 一个有名字的 migration step；
- 单步测试；
- 每个仍受支持旧版本到 current 的 chain 测试；
- migration 后 strict parse、canonicalize 和 domain validation；
- 中途失败不修改原文件的测试；
- future version 拒绝且不修改原文件的测试。

用户数据上线后，迁移提交应使用 temp/write/fsync/atomic replace 或对应 store 的原子提交机制，
并在第一次破坏性迁移前创建可识别的备份/恢复点。迁移成功后只写 current version；不要长期运行
两套 runtime reader。

当前阶段按既定决定执行一次破坏性 cutover：

- 开发期 `3.1` artifact 不进入 migration registry；
- 当前合同从独立 `v1` 开始；
- 不添加 fallback reader；
- 但 migration registry、错误分类和测试 seam 现在就内置，为以后 `1 -> 2` 使用。

## 对 Yotta 的最终建议

### 自动化命令

建议由一个小型 Go CLI（例如 `cmd/yotta-versions`）承载解析和决策，PowerShell/Task 只负责调用。
这样 Windows 本地和 CI 共用同一实现，并可直接单元测试。

```text
yotta-versions show
yotta-versions check
yotta-versions sync [--check|--write]
yotta-versions bump patch|minor|major|3.2.0 [--dry-run]
yotta-versions inventory [--json]
yotta-versions verify-binary <path>
```

`inventory` 应汇总但不强制相等：

```text
product                  3.1.0
yotta.workflow           1
yotta.node-contract      1
yotta.catalog            1
yotta.program            1
host-interface           1.0
script-worker            1
plugin-protocol          1
```

### 门禁

`task versions:check` 至少验证：

- `VERSION` 是允许的 SemVer；
- 所有 product projections 与 `VERSION` 语义一致；
- Windows 三段/四段和数字范围正确；
- 私有 frontend package version 未被误当成产品版本；
- 每个 artifact `format` 在 inventory 中唯一；
- 每个 artifact owner 只有一个 `CurrentVersion`；
- 生成 Schema 的 `$id`、目录和 instance enum 与 owner 一致；
- Host/wire/store version 的生产代码只能从所属 module 引用；
- 非 legacy allowlist 的生产代码和当前文档不得再出现裸 `"3.1"`；
- 产品版本变化不会改变 contract schema/digest；
- contract schema 发生不兼容变化时必须提升对应 owner version；
- stable contract 版本提升必须存在 migration step 和 fixture。

### 升级决策表

| 变化 | 应提升 |
|---|---|
| UI 修复、普通功能、应用发布 | Product Release Version |
| Workflow Source 持久化结构或语义不兼容 | Workflow Source version |
| 单个 Node Type 输入/输出或运行语义变化 | 该 Node Type SemVer + digest |
| Compiler 实现变化、Program shape 不变 | Compiler build digest |
| Program JSON shape 不兼容 | Program artifact version |
| 插件宿主 API 新增可选能力 | Host Interface minor |
| 插件宿主 API 删除或改变语义 | Host Interface major |
| 子进程握手/消息协议不兼容 | 对应 Wire Protocol version |
| store 目录、marker 或提交语义变化 | 对应 Storage Layout version |
| 同一实体内容保存一次 | Revision，不叫 version |

### 实施优先级

1. 先引入 `VERSION`、版本 CLI 和 `version:check`，让产品版本只有一个可编辑源；
2. 修正 Windows projection，尤其是 manifest 的四段 version；
3. 把 frontend package version 从产品同步清单移除；
4. 完成开发期 `3.1 -> v1` artifact cutover；
5. 抽出 Host Interface 和各 wire/store owner；
6. 把 literal ownership 门禁接入 `task check`；
7. 最后接 release PR/tag/CI automation，避免在基础版本模型未稳定时自动发布错误 projection。
