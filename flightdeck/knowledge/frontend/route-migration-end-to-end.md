# ⚠ vue router 路径迁移必须端到端追双端

SUMMARY: 改 vue router 路径必须端到端追双端(写入方+读取方), 只改 routes 数组是 1/N.
READ WHEN: 改任何 vue route 命名 / 路径模板 / params↔query 形态 / hash router URL / wails frameless 子窗 URL / 写 plan 涉及 router 迁移时

---


## 教训

改 vue router 路径不是"改 routes 数组"一行事. 涉及 N 个角色:

| 角色 | 责任 | 代码位置 |
|---|---|---|
| 写入方 - sidebar / list `router.push` | 生成新路径触发跳转 | `AppSidebar.vue`, `ContainersTab.vue` 等 |
| 写入方 - backend window adapter | wails 子窗 URL (hash) | `wire_container_editor.go` `OpenEditor` |
| 写入方 - main window 初始 URL | wails app 启动 | `main.go` `mainWin.URL` |
| 读取方 - view 内部 | `route.params.id` vs `route.query.id` | `ContainerEditorView.vue` setup() |
| 读取方 - App.vue isStandalone 判断 | meta vs query | `App.vue` computed |

**只改 routes 数组 = 1/N 完成**. 其它角色仍指向老路径, 表现:
- 读取方仍读 `route.query.id` 但新路径用 params → 永空 → view 死在"加载中"
- 写入方 backend 仍 push 老 hashURL (`/container-editor?id=`) → 新 router 无此 route → 404
- main.go URL 给 raw `/containers` 而不是 `/#/containers` → wails webview 找不到 asset 路径 → 404

**Build 全绿**: TypeScript 不知道 router 路径含义, 不会报 `route.query.id` 错. Backend Go 字符串不解析. 静态检查全过, 运行才挂.

## 怎么避免

写 plan / spec 涉及 router 迁移时, **必须列一个"端到端 trace 表"**, 每行一个角色:

```
路由变更: /old-path?id= → /new-path/:id
端到端 trace:
- [ ] router/index.ts: routes 配置改
- [ ] 写入方: sidebar / list 的 router.push 改
- [ ] 写入方: backend wire_*.go 的 hashURL 改 (含 query / params 切换)
- [ ] 写入方: main.go window URL 改 (注意 hash router 必须带 #)
- [ ] 读取方: view setup() 内 route.query → route.params
- [ ] 读取方: App.vue isStandalone 之类判断改
```

Subagent dispatch 时这个 trace 表必须在 prompt 里. 不在 = subagent 只做 plan 字面要求, 漏端到端 — 不是 subagent 错, 是 plan 设计错.

## 反模式 (Plan 长这样必踩)

```markdown
### Task N: router 改成 /containers/:id/edit
- 改 router/index.ts: 删老 route + 加新 /containers/:id/edit
- typecheck 绿
- commit
```

Spec compliant ✅, build ✅, 运行 ❌. typecheck 不抓动态 route key 错配.

## 正确版

```markdown
### Task N: router 迁移 /container-editor?id= → /containers/:id/edit
- 改 router/index.ts (routes 数组)
- 改 ContainerEditorView.vue:532: route.query.id → route.params.id
- 改 wire_container_editor.go:42: hashURL 改 /#/containers/<id>/edit?standalone=1
- 改 main.go URL: /containers → /#/containers (hash router 带 #)
- 改 ContainersTab.vue onEdit: router.push 新路径
- 改 App.vue isStandalone: query.standalone 判断
- dev verify (build 不抓动态 route 错配, 必须 task dev 真跑一遍)
```

## Case 1 — 2026-05-28 v2 主壳瘦身 Task 6 路由迁移漏端到端

Plan 把 Task 6 写成"加 OpenInWindow RPC + 工具栏按钮 + 改 ContainerEditorToolbar". 实施 subagent (sonnet) 完美 follow plan, spec reviewer + code reviewer 都过. ship 后用户跑 dev:

1. 主窗启动: `http://wails.localhost/containers` → **404** (main.go URL 漏 `#`, fix commit `de7daa6`)
2. 点容器编辑: 主壳跳 `/containers/X/edit` (router 对) → ContainerEditorView 渲染但 **永显"加载中"** (内部 line 532 仍 `route.query.id` 拿空 → containerID="" → store load 失败, fix `9af2d10`)
3. 子窗"在新窗口打开": **404** (wire_container_editor.go 仍 push 老 hashURL `/container-editor?id=`, router 已删此 route, fix 同 `9af2d10`)
4. 列表"编辑"按钮: 仍调老 backend `openEditorWindow` (开独立窗), 用户预期嵌入 (fix `9af2d10`)

5 个 fix commit (de7daa6, 6d1ddda, 9af2d10, b839926, 9b5c601) 全是 ship 后 review-uncovered. 全可在 plan 设计阶段防住, 如果 Task 6 spec 列了"端到端 trace 表".

教训: plan / spec 任何涉及 router 路径变更, 写下"6 端 trace 清单" 当 acceptance criteria, 而不是 "改 router 就完事". 撞过一次后必须把这个清单做成 [plan 模板的一部分](../checklists/) (待第 2 次再撞时 promote 成 playbook).
