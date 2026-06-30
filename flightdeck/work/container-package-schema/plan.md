# 容器包 Schema 实现计划

## 目标

按 `design.md` 破坏式落地容器包模型:

- 本地和线上使用同一套 package 模型。
- 本机绑定从 package 中剥离到 `installation.json`。
- 主图单独存为 `graph.json`。
- 依赖闭包、能力、权限和 hash 由 `yotta-lock.json` 生成。
- 允许不迁移旧 `container.json`; 旧数据可以在开发期直接丢弃或重建。

## 非目标

- 不兼容旧容器文件。
- 不实现线上 registry 服务。
- 不实现 npm package 语义; `package.json` 只是 Yotta 容器 manifest, 不读取 npm deps/scripts。
- 不把节点 schema 搬进 manifest; 节点仍由 Go 侧 `node.Spec` 和 graph config 驱动。

## 阶段 1: 后端模型和 JSON 形状

文件:

- `internal/services/container/model.go`
- `internal/services/container/model_test.go`

任务:

- 新增 `PackageManifest`, `PackageYotta`, `Publication`, `SourceRef`, `TargetSlot`, `AISlot`, `PackageAuthor`, `PackageRepository`, `Installation`, `RuntimeOverrides`, `YottaLock` 等类型。
- 将 `Graph.version` 破坏式改名为 `Graph.schemaVersion`。
- `PackageManifest` 不包含 `createdAt` / `updatedAt`; 时间只留在 installation 或未来 registry 响应。
- `Installation` 使用 `packageId` 作为稳定关联键, `packageName` 只作为展示/诊断缓存。
- `RuntimeOverrides` 使用 `null` 或省略表达继承默认值, 不用空字符串或 `0` 表达未设置。

验证:

```powershell
go test ./internal/services/container
```

提交点:

```text
feat(containers): add package schema model types
```

## 阶段 2: 依赖闭包和 lock 生成

文件:

- `internal/services/container/dependency/types.go`
- `internal/services/container/dependency/closure.go`
- `internal/services/container/dependency/*_test.go`
- `internal/services/container/validator*.go`

任务:

- 将闭包结果拆出 `templates`, `clips`, `subgraphs`, `aiSlots`, `targetSlots`。
- 保留 Script 静态扫描, 但明确动态拼接引用不进入闭包; 发布校验遇到无法静态证明的动态引用时给 warning 或 reject。
- lock 写入 `manifestHash`, `graphHash`, `closureHash`, 并保留细分闭包列表。
- lock 只在导出、投稿、发布、安装、更新前强制刷新; 普通保存允许标 stale。
- Target/AI 槽位由节点 config 的逻辑引用反推, 不再把本机窗口标题、进程名、ADB serial、AI connection 写进 package。

验证:

```powershell
go test ./internal/services/container/dependency
go test ./internal/services/container
```

提交点:

```text
feat(containers): generate package dependency lock
```

## 阶段 3: Store 改为多文件包目录

文件:

- `internal/services/container/store.go`
- `internal/services/container/store_test.go`

任务:

- 目录保持 `data/containers/<instanceId>/`。
- 读写文件改为:
  - `package.json`
  - `graph.json`
  - `installation.json`
  - `yotta-lock.json`
- `Load` 只扫描新布局; 不读取旧 `container.json`。
- `Save` 以 manifest + graph + installation 为核心, lock stale 时重新生成或标记。
- `List/Get` 可以继续返回聚合后的 `Container` 视图, 但聚合视图必须从新文件拼出来。
- 写文件继续使用原子写策略, 避免中途损坏多文件目录。

验证:

```powershell
go test ./internal/services/container
```

提交点:

```text
feat(containers): store containers as package directories
```

## 阶段 4: Service/RPC 兼容聚合视图

文件:

- `internal/services/container/service.go`
- `internal/services/container/*_test.go`
- `internal/app/*` 中调用容器服务的 RPC 绑定

任务:

- `Create` 生成完整 package/graph/installation/lock 四件套。
- `Update` 按字段归属分发:
  - 包身份、展示、分类、tag、版本、作者、文档、发布字段写 `package.json`。
  - 节点图写 `graph.json`。
  - 热键、target 绑定、AI 绑定、运行覆盖、收藏、最近运行时间写 `installation.json`。
- `Delete/Duplicate/Import/Export` 使用新布局。
- 对前端暂时保留一个聚合 DTO, 降低一次性改动面积。

验证:

```powershell
go test ./internal/services/container
go test ./...
```

提交点:

```text
feat(containers): route container service fields by ownership
```

## 阶段 5: 前端字段和容器列表

文件:

- `frontend/src/types/container.ts`
- `frontend/src/composables/useContainers.ts`
- 容器列表、详情、设置弹窗相关 Vue 文件

任务:

- 前端类型对齐聚合 DTO。
- 容器列表字段读取:
  - `category`, `keywords`, `version`, `author` 来自 package。
  - `installedAt`, `lastRunAt`, 本地 `updatedAt` 来自 installation 或列表响应。
- 筛选支持分类和 tag。
- 排序、视图模式、分页大小继续写 localStorage。
- 分页器放在列表区域底部; 大屏布局按容器宽度自适应, 不固定 3 列。

验证:

```powershell
pnpm --dir frontend test
pnpm --dir frontend typecheck
pnpm --dir frontend i18n:check
```

提交点:

```text
feat(frontend): consume package-backed container fields
```

## 阶段 6: MCP 和投稿/导出入口

文件:

- `internal/services/mcp/*`
- `internal/services/container/export*`
- 相关前端设置或按钮入口

任务:

- MCP `save_container` 写入新布局。
- 导出产物包含 `package.json`, `graph.json`, `assets/`, `subgraphs/`, `templates/`, `clips/`, `yotta-lock.json`。
- 投稿前强制 lock 校验。
- registry 未实现前, 先保证本地导出包的结构就是未来线上上传结构。

验证:

```powershell
go test ./internal/services/mcp ./internal/services/container
```

提交点:

```text
feat(containers): export package bundle layout
```

## 阶段 7: 全量验证和收尾

命令:

```powershell
go test ./...
pnpm --dir frontend i18n:check
pnpm --dir frontend test
pnpm --dir frontend typecheck
pnpm --dir frontend build
```

人工检查:

- 新建容器目录只生成四件套, 不生成 `container.json`。
- 容器列表显示分类/tag/版本/作者。
- 排序、卡片模式、分页大小刷新后保留。
- 没有本机窗口/ADB/AI 凭据泄露进 `package.json`。
- `yotta-lock.json` 的 `manifestHash`, `graphHash`, `closureHash` 会随对应内容变化。

提交点:

```text
feat(containers): finish package schema rollout
```

## 执行顺序

1. 先做阶段 1 和 2, 锁定 JSON 形状和 lock 语义。
2. 再做阶段 3 和 4, 让后端持久化完整切到新布局。
3. 然后做阶段 5, 修前端展示、筛选、排序和分页体验。
4. 最后做阶段 6 和 7, 对齐 MCP/导出并跑全量验证。

