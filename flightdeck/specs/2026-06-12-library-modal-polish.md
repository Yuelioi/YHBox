---
status: active
summary: hover 多选框 + 底部工具栏(双态: 计数/分页 ↔ 批量删/加标签/改分类) + 分页(每页条数记忆) + 右栏就地编辑(名称/描述双击, 标签/分类行内控件, 删编辑弹窗) + 插入引用单主 CTA + 删除/复制弱化 + LibraryBatchPanel 连根删
last_updated: 2026-06-12
---

# 子图库 modal 美化 v3 — 分页/工具栏/就地编辑

## 背景 (2026-06-12 真机反馈第三轮)

用户四点: ①没分页; ②不支持批量改分类; ③右栏要就地编辑(名称双击/标签描述加编辑入口), 去掉编辑按钮, 插入引用放醒目位, 删除/复制弱化; ④左侧加多选框 + 底部加工具栏。设计已过用户认可。uiuxpro 规则依据: 单屏单主 CTA / 破坏性动作分离弱化 / 50+ 列表分页 / 行内编辑 affordance。配色照 ui.md semantic token, 不另起体系。

## 设计

### 1. 左列表

- **hover 多选框**: 行首 hover 浮现 UCheckbox, 勾选后常显; 点 checkbox = toggle 多选 (免 Ctrl), 点行 = 单选出详情, Ctrl/Shift 照旧; checkbox 点击 stop 传播 (不触发行单击/双击)。
- **分页**: 对过滤后扁平列表切页, 页内再按分类分组。过滤/搜索/每页条数变化 → 回第 1 页。翻页选中自动清 (selection 的 visibleIds 剪枝天然覆盖, 跨页批量不做)。
- 纯函数 `paginate(items, page, size)` → `{ pageItems, total, totalPages, page(钳制后) }`, 放 libraryFilter.ts, vitest。

### 2. 底部工具栏 (列表栏底部, 双态)

- **无选中**: 左「共 N 个」; 右 分页控件 (上/下页 + x–y/N + 每页条数 USelect 20/50/100)。
- **选中 ≥1**: 左变批量区「已选 N」+ 批量删除(error) + 批量加标签 + **批量改分类(新)** + 取消选择; 右分页照常。
- 每页条数用 `useLocalStorage('library.pageSize', 50)` 记忆 (@vueuse 已有依赖)。
- **LibraryBatchPanel.vue 连根删** (批量归工具栏一处); 右栏恒为详情 (取锚点项 = 最后单击行), 0 选中时空态提示。

### 3. 右详情栏就地编辑 (LibraryDetailPanel 重做)

- **名称**: 双击标题 → input (回车/失焦存, Esc 撤), hover 标题尾部浮小铅笔提示可编辑。习语照 CommentBoxNode (editing ref + draft + nextTick focus)。
- **描述**: 双击 → textarea (同上); 无描述时显示 dimmed「双击添加描述」。
- **标签 / 分类**: 直接放行内控件 (UInputMenu multiple creatable / single creatable, 候选同库), 即改即存。
- **保存语义**: 每字段独立 `updateSilent(id, {字段}, rev)` (merge patch + rev 乐观锁), 失败 toast + reload; 名称空白不提交 (恢复原值)。
- **按钮层级**: 「插入引用」唯一 primary 实心 block 按钮, 紧贴 header 下; 「复制为新 / 删除」弱化为底部一行两个 size=xs soft 按钮 (删除 error 色)。
- **删编辑弹窗双轨**: 编辑信息 modal、详情面板「编辑信息」按钮、右键菜单「编辑信息」项、`library.explorer.edit_info`/`edit_title`/`tags_hint` i18n 键全删 (就地编辑全覆盖)。

### 4. 批量改分类

- 工具栏按钮 → 小弹窗 (BaseModal sm): UInputMenu single creatable, 候选 = 现有分类; 确认后对每个选中项 `updateSilent(id, {category}, rev)` 逐项提交, 失败聚合一条 toast + reload。

## 不做的事

- 跨页选中/全选所有页 (每页条数调大即可覆盖); 虚拟滚动 (分页已解); 行内改名进左列表 (右栏就地编辑已覆盖); 在线 tab 不动。

## 改动面

| 文件 | 改动 |
|---|---|
| `frontend/src/lib/libraryFilter.ts` (+test) | 加 `paginate` 纯函数 |
| `frontend/src/components/containers/LibraryExplorerModal.vue` | checkbox 列 + 分页接线 + 底部工具栏 + 批量改分类弹窗 + 删编辑弹窗/批量面板引用 + 右键菜单删编辑项 |
| `frontend/src/components/containers/LibraryDetailPanel.vue` | 就地编辑重做 + 按钮层级 |
| `frontend/src/components/containers/LibraryBatchPanel.vue` | **删除** |
| `frontend/src/i18n/zh.ts` / `en.ts` | +分页/批量改分类/就地编辑文案; −edit_info/edit_title/tags_hint |

## 验收

代码全绿 (typecheck/vitest/lint/i18n 按在册口径) + 真机: ①行首 hover 出勾选框, 勾选常显且批量不用按 Ctrl; ②底部工具栏: 无选中显总数+分页, 选中显批量四钮; ③每页条数 20/50/100 可调且重开记住, 过滤变化回第 1 页; ④批量改分类生效, 分组随之变; ⑤右栏名称/描述双击可改, 标签/分类行内直接改, 改完即存; ⑥「编辑信息」按钮/弹窗/右键项全无; ⑦插入引用是右栏唯一大绿钮, 复制/删除缩到底部小钮; ⑧双击行插入 + 右键其余菜单 + 过滤器全不回归。
