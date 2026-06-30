# fishing-v2 新节点系统重建设计

## 目标

重建 `bin/data/containers/fishing-v2/`，让它成为当前版本可加载、可运行、可导出的自动钓鱼容器。旧 `container.json` 只作为业务流程参考，不直接迁移节点 JSON。

目标流程:

1. 找到游戏窗口。
2. 进入钓鱼入口。
3. 抛钩。
4. 检测上钩。
5. 左右晃动溜鱼。
6. 检测结算或鱼逃跑。
7. 继续下一轮，从抛钩/准备钓鱼开始。
8. 没鱼饵进入买鱼饵流程。
9. 新循环需要换鱼饵。
10. 没钱买鱼饵时，默认进入卖鱼流程后再买；用户关闭自动卖鱼时直接结束容器。

## 非目标

- 不做旧 `container.json` 兼容读取。
- 不把其它测试容器迁移到新格式；它们可以删除。
- 不重截模板、不重录 clip，除非后续人工验证发现资产失效。
- 不新增后端节点类型；优先使用当前节点系统已有节点。

## 数据布局

目标目录:

```text
bin/data/containers/fishing-v2/
  package.json
  graph.json
  installation.json
  yotta-lock.json
```

旧文件:

```text
bin/data/containers/fishing-v2/container.json
```

实施时可以删除旧 `container.json`。如果需要保留人工参考，放到 `bin/data/_backups/`，不要留在容器目录里影响新 loader 判断。

## 容器字段

`package.json`:

- `name`: `@local/fishing-v2`
- `displayName`: `自动钓鱼`
- `version`: `0.1.0`
- `category`: `钓鱼`
- `keywords`: `["钓鱼", "自动化", "异环"]`
- `author.name`: `local`
- `yotta.packageId`: `pkg_fishing-v2`
- `yotta.entryGraph`: `graph.json`

`installation.json`:

- `instanceId`: `fishing-v2`
- `runtimeOverrides.inputBackend`: `postmessage`
- `runtimeOverrides.captureBackend`: `auto`
- `runtimeOverrides.scaleTolerance`: `2`
- `targetBindings.game.kind`: `win32-window`
- `targetBindings.game.match.title`: `异环  `
- `targetBindings.game.match.class`: `UnrealWindow`
- `targetBindings.game.match.processName`: `htgame.exe`
- `targetBindings.game.match.titleMatch`: `exact`

## 变量

用户可见变量:

- `autoSellWhenNoCurrency`: `bool`, default `true`。没钱买鱼饵时是否自动去卖鱼。

状态机变量:

- `state`: `string`, default `IDLE`
- `controlDir`: `number`, default `0`
- `waitingStart`: `number`, default `0`
- `hookStreak`: `number`, default `0`
- `fishingStart`: `number`, default `0`
- `emptyCastStreak`: `number`, default `0`
- `resultEnteredAt`: `number`, default `0`

内部临时变量:

- `_hookFFound`: `bool`, default `false`
- `_inspectPhaseResult`: `string`, default `unknown`
- `_pressEscCleared`: `bool`, default `false`
- `recoveryEscDone`: `bool`, default `false`

## 主图结构

主图按当前节点系统重建:

```text
Start
  -> Win32WindowTarget(Target=game)
  -> SetVar(state=IDLE)
  -> Loop(forever)
      -> GetVar(state)
      -> Switch(state)
          IDLE       -> Subgraph(state_IDLE)
          SETUP      -> Subgraph(state_SETUP)
          WAITING    -> Subgraph(state_WAITING)
          FISHING    -> Subgraph(state_FISHING)
          RESULT     -> Subgraph(state_RESULT)
          RECOVERING -> Subgraph(state_RECOVERING)
          SHOPSELL   -> Subgraph(state_SHOPSELL)
          BUYBAIT    -> Subgraph(state_BUYBAIT)
          CHANGEBAIT -> Subgraph(state_CHANGEBAIT)
          END        -> Stop
          default    -> SetVar(state=IDLE)
```

每个状态子图返回 `done` 后进入对应短 Sleep，再回到 Loop 下一轮。睡眠建议:

- IDLE / RESULT / SETUP: `200ms`
- WAITING: `250ms`
- FISHING: `150ms`
- RECOVERING: `500ms`
- BUYBAIT / SHOPSELL / CHANGEBAIT: `300ms`

主图不复用旧 `WindowTarget`，只使用当前 `Win32WindowTarget`。`graph.json` 写 `schemaVersion`，不写旧 `version`。

## 状态流程

状态语义基于旧容器和现有钓鱼子图说明重建，但节点必须按当前 Spec 生成。

`IDLE`:

- 检测 `fishing.start_fish`，命中转 `SETUP`。
- 检测 `fishing.hook_icon`，按 F 抛钩。
- 抛钩后检测:
  - `fishing.need_bait` -> `BUYBAIT`
  - `fishing.warehouse_full` -> `SHOPSELL`
  - `fishing.start_fish` -> `SETUP`
  - 否则记录 `waitingStart` / 清 `hookStreak` -> `WAITING`
- 检测 `fishing.result` 时清结算画面并保持 `IDLE`。

`SETUP`:

- 点击 `fishing.start_fish`。
- 等待后若 `fishing.need_bait` -> `BUYBAIT`。
- 否则回 `IDLE`。

`WAITING`:

- 抛竿后短时间内不判定。
- 检测 `fishing.hook_text` 连续命中，满足阈值后调用 `try_hook_F`。
- 成功 -> `FISHING`，并初始化 `fishingStart` / `controlDir`。
- 超时后调用 `inspect_phase`，按结果恢复到 `IDLE / SHOPSELL / BUYBAIT / SETUP / FISHING / RECOVERING`。

`FISHING`:

- 用 `DualColorBarTrack` 检测耐力/指针条。
- 根据指针偏差控制左右按键，保持在目标区域。
- 检测 `fishing.result` -> 清结算 -> `IDLE`。
- 检测 `fishing.fish_escape` -> `IDLE`。
- 超时 -> `RECOVERING`。

`RESULT`:

- 检测 `fishing.result` -> 清结算 -> `IDLE`。
- 检测 `fishing.fish_escape` -> 等待 -> `IDLE`。
- 超时 -> `RECOVERING`。

`RECOVERING`:

- 调 `inspect_phase`。
- 能识别阶段则跳转到对应状态。
- 未识别时先按 ESC 一次，下一轮继续恢复。

`SHOPSELL`:

- 进入背包/商店卖鱼。
- 点击卖出相关模板。
- 成功后回 `IDLE`。
- 失败转 `RECOVERING`。

`BUYBAIT`:

- 进入购买鱼饵。
- 点击鱼饵商品、最大数量、购买、确认。
- 检测 `fishing.no_currency`:
  - `autoSellWhenNoCurrency=true` -> `SHOPSELL`
  - `autoSellWhenNoCurrency=false` -> `END`
- 买成功后转 `CHANGEBAIT`。
- 关键模板缺失或流程失败转 `RECOVERING`。

`CHANGEBAIT`:

- 按 E 打开换饵。
- 点击 `fishing.change_bait_confirm`。
- 等待 `fishing.go_fishing` 或超时。
- 回 `IDLE`。

`END`:

- 主图 `Switch` 的 `END` 出口接 `Stop`。
- 只用于用户关闭自动卖鱼时没钱买鱼饵的正常结束。

## 资产与子图

保留全局资产库:

- `bin/data/templates/*.json`
- `bin/data/clips/*.json`
- `bin/data/blobs/*`

钓鱼模板现有 GUID 可继续使用。重建时必须按 GUID 引用，不按人类名称引用。

现有钓鱼状态子图可作为参考或复用，但实施时要检查:

- `SubgraphID` 是否存在。
- 子图内部节点 kind 是否仍存在。
- 子图内部是否仍使用当前节点 pin 名。
- 是否还有旧节点 kind，例如 `WindowTarget`。
- `BUYBAIT` 是否补上 `no_currency` + `autoSellWhenNoCurrency` 分支。

如果某个旧状态子图含过期节点，应重写该子图，而不是做兼容迁移。

## 清理策略

`bin/data/containers/` 下除 `fishing-v2` 外都是测试容器，可删除。删除前必须解析绝对路径，确认目标都在:

```text
E:\projects\tools\YHFish\bin\data\containers
```

只删除容器目录，不删除:

- `bin/data/templates`
- `bin/data/clips`
- `bin/data/blobs`
- `bin/data/subgraphs`
- `bin/data/schedules`
- `bin/data/settings.json`

## 验证

自动验证:

- 用当前 `container.Store` 读取 `bin/data/containers`，确认只加载 `fishing-v2`。
- 调 `ValidateContainerByID("fishing-v2")` 或等价后端校验，结果无 error。
- 检查 `fishing-v2` 目录没有 `container.json`。
- 检查 `graph.json` 没有旧 `version` 字段，没有旧 `WindowTarget`。
- 检查 `installation.json` 含窗口匹配，本机窗口标题不在 `graph.json`。
- 检查 `yotta-lock.json` 依赖包含用到的 templates/subgraphs。

人工验证:

- 启动应用后容器列表显示“自动钓鱼”。
- 打开编辑器，主图状态机可读。
- 窗口目标能匹配 `异环`。
- 试运行至少覆盖:
  - 正常钓鱼一轮。
  - 没鱼饵 -> 买鱼饵 -> 换鱼饵 -> 下一轮。
  - 没钱且 `autoSellWhenNoCurrency=true` -> 卖鱼 -> 再买。
  - 没钱且 `autoSellWhenNoCurrency=false` -> 停止容器。

## 风险

- 旧子图说明里有部分 ROI 是 1080p 硬编码，窗口分辨率变化可能导致条检测失败。
- 模板资产已经有多分辨率变体，但仍需要人工确认当前游戏 UI 未变。
- `BUYBAIT` 分支补强可能需要改现有子图，必须重新跑依赖 lock。
- 如果直接用 Store Save 写入，旧 `container.json` 会被删除；这是符合当前破坏式升级策略的。
