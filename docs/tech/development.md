# 开发指南

## 构建

```powershell
task build       # frontend (vite) + Rust DLL (capture_wgc) + Go exe
task dev         # 开发模式 (vite HMR + wails3 dev)
```

依赖：Go 1.25+、Node 22+、Rust（msvc target）、wails3 CLI、[Task](https://taskfile.dev)。

## 项目结构

详见 [architecture.md §2](architecture.md#2-架构平面)。

调试输出在 `debug/`（gitignored）：
- `debug/captures/<bot>/<date>/` — Capture.DumpDebug toggle 开后落盘的带框 PNG
- `debug/rhythm-darkratio-*.csv` — rhythm-calibrate 产物
- `debug/rhythm-roi-T*.png` — rhythm-calibrate 启动头一帧的 4 个 ROI 截图
- `debug/docs/` — 设计 spec / plan 文档

## 加新分辨率

YHBox 不做跨分辨率自动缩放（实测 NCC conf 损失明显）。每个分辨率独立标 ROI + 抠模板，精确匹配。

### fish 加新分辨率（含 18 个模板 slot × N 张 PNG）

1. 游戏切到目标分辨率，**UI 缩放 100%**
2. 进每个相关 UI 界面（钓鱼主界面 / 商店 / 鱼饵商店 / 结算 / ...）截全屏 PNG
3. 在每张截图上量 18 个 slot 的 bbox（hook_icon / hook_text / start_fish / ...，完整 slot 列表见 [tools/fish/detect.go](../../tools/fish/detect.go) `requiredSlots`）
4. 用图片编辑工具按 bbox 抠出 18 张小图 → 命名 `<slot>_<H>.png` → 丢 `tools/fish/configs/zh/templates/`
5. 编辑 [tools/fish/configs/zh/templates.toml](../../tools/fish/configs/zh/templates.toml) 加 18 条 `[[fish.<slot>]]` 块，每条含 `file` / `resolution=[W,H]` / `bbox=[x1,y1,x2,y2]` / `note`
6. 编辑 [tools/fish/configs/zh/config.yaml](../../tools/fish/configs/zh/config.yaml) 的 `capture.bar_rois` 加耐力条 ROI
7. `task build` → 启动 exe → fish 跑一会儿看 log 有没有 MISS

如果 NCC conf 太低（< 0.85），多半是抠模板时 bbox 不精确 / 抠到了 anti-aliasing 边缘。从真实游戏截图抠，不要从 UI 资源文件抠（那个跟实际渲染像素级不对应）。

### rhythm 加新分辨率

rhythm 比 fish 简单得多——4 个 ROI 11×21 居中在命中点。

1. 游戏切到目标分辨率，**UI 缩放 100%**
2. 进任意超强音曲目，暂停或截没有音符落到命中圈的一帧
3. 量 4 个命中点的中心 (X, Y) 像素坐标
4. 编辑 [tools/rhythm/configs/zh/config.yaml](../../tools/rhythm/configs/zh/config.yaml) 加一条 resolution entry：

   ```yaml
   "<W>x<H>":
     rois:
       - [Xc1-5, Yc-10, Xc1+6, Yc+11]   # T1 居中 (Xc1, Yc)
       - [Xc2-5, Yc-10, Xc2+6, Yc+11]
       - [Xc3-5, Yc-10, Xc3+6, Yc+11]
       - [Xc4-5, Yc-10, Xc4+6, Yc+11]
   ```

5. 跑 `go run ./cmd/rhythm-calibrate` 30s 看 dark_ratio 分布
6. 如果 p95 跟阈值 0.06 差太近 → ROI 量错或太小，重新量；阈值在 [constants.go](../../tools/rhythm/constants.go) 不外置

### cook / battle 加新分辨率

跟 fish 同模式，但只有 1-8 个 slot：
- cook：`hammer_<H>.png` 一个 slot
- battle：`team_tag` / `team_enabled` / `team_select` / `team_switch_success` 四个 slot

### piano 不用加分辨率

piano 是 MIDI → 按键，不依赖视觉。

## 加新语言

YHBox 把 locale 维度分两层：

### 1. UI 字符串（前端 vue-i18n）

- [frontend/src/i18n/zh.ts](../../frontend/src/i18n/zh.ts) 主语言
- [frontend/src/i18n/en.ts](../../frontend/src/i18n/en.ts) 目前只是部分翻译占位（sidebar / controls / settings 共 25 keys 已翻；views 内部硬编码中文未提取）
- 加新语言：复制 zh.ts → ja.ts，翻译每个 key
- 在 [frontend/src/i18n/index.ts](../../frontend/src/i18n/index.ts) 加 `import ja from './ja'` + `messages: { ja, ... }`
- `frontend/src/i18n/check.cjs` 验证 zh/en（/ ja）键集一致

### 2. bot 视觉资源（后端）

英文游戏 UI 跟中文不一样 → 视觉模板要重做一套。每个 bot 的 `tools/<bot>/configs/<locale>/` 是 locale-aware：

```text
tools/fish/configs/
  ├── zh/
  │   ├── manifest.yaml          implemented: true
  │   ├── config.yaml
  │   ├── templates.toml
  │   └── templates/             36 PNG
  └── en/
      └── manifest.yaml          implemented: false
```

加新语言（如 ja）：

1. 把游戏切到 ja，每个 bot 重抠模板 + 写 templates.toml 同 [加新分辨率](#fish-加新分辨率含-18-个模板-slot--n-张-png) 流程
2. 创建 `tools/<bot>/configs/ja/` + 各文件
3. `manifest.yaml` 写 `implemented: true`

### locale_alias 跨语言复用

某些 bot（如 rhythm）纯像素检测，跨语言视觉无差异。直接 alias 不必重抠：

```yaml
# tools/rhythm/configs/en/manifest.yaml
version: 1
implemented: true
locale_alias: zh
reason: ""
```

`botcore.ResolveLocale` 检测到 alias → effective locale 重定向到 zh → 复用 zh 的 config.yaml / templates。

适用场景：bot 完全不依赖游戏文字 UI（只靠 ROI 坐标 + 像素颜色/亮度）。

## 加新 bot

步骤：

1. **创建 `tools/<newbot>/`** 包：至少有 `newbot.go` (Run 主循环)、`config.go` (LoadConfig)、`configs/zh/manifest.yaml`
2. **如有视觉模板**：加 `configs/zh/templates.toml` + `templates/*.png`，`config.go` 4-helper 拆分（loadManifest → loadConfig → loadTemplates → Validate）
3. **创建 service**：`cmd/yhbox/services/newbot_service.go`，参考 `cook_service.go` 模板（init() 里调 `RegisterBot(BotMeta{Kind: "newbot", Construct: func(app *App) (BotService, application.Service) {...}})`）
4. **前端 view**：`frontend/src/views/NewBotView.vue` + `frontend/src/stores/newbot.ts`，参考 `CookView.vue` / `cook.ts`
5. **侧栏入口**：`frontend/src/lib/bot-registry.ts` 加一条 `{ kind: 'newbot', label: '新功能', icon: 'i-lucide-xxx', route: '/newbot', useStore: useNewBotStore }`
6. **router**：`frontend/src/router/index.ts` 加 route entry
7. **main.go**：跟随 `RegisteredBots()` 自动循环加载，**不需要手动修 main.go**（registry 设计目标）
8. **测试**：写 `newbot_test.go` 覆盖核心逻辑

## 调参

任何 bot 调参（fish HSV 阈值 / rhythm dark_ratio / cook 间隔 / ...）推荐流程：

1. **录帧**：真游戏跑一次，设置开 `Capture.DumpDebug` toggle，相关 detect 路径会异步落盘带框 PNG 到 `debug/captures/<bot>/<date>/`
2. **切 Mock backend**：设置 → 截屏方式 → Mock，把帧复制到 `bin/mock-frames/`，重启 exe
3. **改阈值**：编辑 `tools/<bot>/configs/zh/config.yaml` 或对应 constants.go，重启 exe 立即看效果
4. **不必反复开游戏**

针对 rhythm 还有 `go run ./cmd/rhythm-calibrate` 输出 dark_ratio 直方图。

详细工具说明：

- Mock backend 实现 [pkg/capture/mock.go](../../pkg/capture/mock.go)
- Screenshot writer [pkg/screenshot/screenshot.go](../../pkg/screenshot/screenshot.go)
- rhythm-calibrate [cmd/rhythm-calibrate/main.go](../../cmd/rhythm-calibrate/main.go)

## Git 提交风格

- 一次提交一个大功能，不要每个小改动都提交
- commit message 不带 Claude / Co-Authored-By 之类的署名
- 不要 `--no-verify` 跳 pre-commit hook
- `debug/` 是 gitignored 的开发者本地目录，所有调试输出 / 设计文档放这里不进 GitHub

## 不要犯的错

详见 [architecture.md §8](architecture.md#8-不要犯的错)。

## 还可以做什么（未来工作）

详见 [debug/docs/superpowers/specs/2026-05-14-future-roadmap.md](../../debug/docs/superpowers/specs/2026-05-14-future-roadmap.md) — 8 项 future work，4 项已完成，4 项待做（COCO JSON 多分辨率 / 配置 GUI / 统计面板 / OCR）。
