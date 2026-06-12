---
status: active
summary: 选中行 primary 高亮(bg-primary/15+ring) + 折叠/裸建子图默认名改「子图 N」序号递增(弃时间戳, 删折叠前缀键) + ConfirmDialog 输入模式打开时全选默认值
last_updated: 2026-06-12
---

# 子图库选中态强化 + 折叠序号命名

## 背景 (2026-06-12 真机反馈)

1. **选中很不明显**: 子图库列表选中行 `bg-elevated/60` 与 hover 色完全相同, 选没选分不出。
2. **折叠默认名「折叠 18:23」难看**: useFolding.ts:39 默认名 = `折叠 + 时间戳`; 裸加 Subgraph 节点另一路是 `子图 + 时间戳` (useSubgraphLifecycle.ts:32); 命名框聚焦但不全选, 想改名得先手动删默认值。

主流调研 (Unreal/Blender/Houdini/Figma): 默认名一律 **通用名词 + 递增序号**, 没人用时间戳; 改名要么静默+事后改, 要么弹框但默认值全选一字可替。用户拍板: **保留弹框 + 默认全选** (Unreal 式)。

## 设计

### 1. 选中态 (LibraryExplorerModal 行样式)

- 选中: `bg-primary/15 ring-1 ring-inset ring-primary/50`
- 未选: `bg-elevated/30 hover:bg-elevated/60` (现状不动)
- 符合 ui.md 配色铁律: 选中激活语义走 primary token。

### 2. 序号命名 (两路统一)

- 新纯函数 `nextSubgraphName(existingLabels: string[], base: string): string` — 扫 `^<base> (\d+)$` 取最大序号 +1, 从 1 起。放 `frontend/src/lib/subgraphNaming.ts`, vitest 单测。
- **折叠路** (useFolding): 弹框默认值 = `nextSubgraphName(全库具名子图名, t('subgraphLifecycle.default_name_prefix'))`。
- **裸加节点路** (useSubgraphLifecycle.autoCreateSubgraphForNewNode): 同一 helper。
- i18n: 统一用 `subgraphLifecycle.default_name_prefix` ('子图'/'Subgraph'); **删 `folding.default_name_prefix`** ('折叠') zh+en 两处 — 二号铁律一次切干净。
- 现有子图名单来源: `useContainerEditorStore().visibleSubgraphs` 的 label (两个调用点都已在编辑器上下文, store 已灌池)。

### 3. ConfirmDialog 输入模式打开时全选

- 打开 (open=true) 且带输入时: 聚焦后全选默认值 → 直接打字即替换, 回车即接受。
- 只在打开瞬间全选, **不**挂在 focus 事件上 (避免编辑中点回输入框被全选)。
- 通用组件改动, 所有 `confirm({inputDefault})` 调用方受益; 纯增强无行为破坏。

## 不做的事

- 不做静默创建 (用户已拍弹框案); 不做画布内联改名; 不动录制落盘命名 (另一条路, 没被点名)。

## 改动面

| 文件 | 改动 |
|---|---|
| `frontend/src/lib/subgraphNaming.ts` (+test) | 新 helper |
| `frontend/src/components/containers/LibraryExplorerModal.vue` | 选中行样式 |
| `frontend/src/components/common/ConfirmDialog.vue` | 打开时全选输入默认值 |
| `frontend/src/composables/containerEditor/useFolding.ts` | 默认名换 helper |
| `frontend/src/composables/containerEditor/useSubgraphLifecycle.ts` | 默认名换 helper |
| `frontend/src/i18n/zh.ts` / `en.ts` | 删 folding.default_name_prefix |

## 验收

代码全绿 (typecheck/vitest/lint 按在册口径) + 真机: ①子图库选中行 primary 高亮, 与 hover 可区分; ②折叠弹框默认名「子图 N」且全选, 直接打字替换, 连续折叠序号递增不重名; ③裸拖 Subgraph 节点自动名同为「子图 N」; ④其它带输入确认框 (如有) 打开同样全选不炸。
