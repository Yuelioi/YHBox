---
when_to_read: 改 view 从 standalone window 改嵌入主 router-view / 看到内嵌 view 出现奇怪纵向滚动条 / fixed bottom-right 元素 (如 MiniMap) 显不全 / 调 flex 父子高度链
applies_to: [frontend, vue, tailwind, layout, h-screen, flex, vue-flow]
last_updated: 2026-05-28
status: active
---

# 嵌入主壳的 view 不能用 h-screen

## 教训

独立 webview 窗 (wails 子窗) 内的 view 用 `h-screen` (100vh) 撑满 — 没问题, 窗的整个 client area 就是 100vh.

但同一 view **改嵌入主壳** (主 webview 的 `<router-view />`) 后, 实际可用高度 = `vh - AppTitleBar (56) - LogPanel (28/220) - AppStatusBar (xx)`. View 还用 `h-screen` 强行 100vh → 撑出父 `<main>` 范围 → 触发主壳 main `overflow-auto` 出纵向滚动条.

雪上加霜:
- View 内 vue-flow canvas 占满 → MiniMap (`position="bottom-right"`) 跟 canvas 一起被滚动条 gutter 切 12px → 用户感觉"右下角 minimap 显不全"
- Vue-flow 滚轮被 canvas 捕获做缩放, 用户**滚动条根本拖不了**
- `scrollbar-gutter: stable` 让宽度预留 — 不滚也 12px 边距, 看似排版 bug

## 正解

嵌入主壳的 view 顶层:

```vue
<div class="flex flex-col h-full min-h-0 overflow-hidden bg-default text-default">
```

3 个关键 class:
- `h-full` — 撑满父 (父是 main 区, 已经被 flex-1 算好高度)
- `min-h-0` — flex item 默认 `min-height: auto` 不让 shrink, 加 min-h-0 才能让 flex child overflow-hidden 真生效
- `overflow-hidden` — 防 minimap / canvas / 任何 absolute 元素撑出来触发主壳滚动条

子窗口 (standalone) 形态用 **同样** class 也行 — 子窗 App.vue 那 `<div v-if="isStandalone" class="h-screen">` 是父, view h-full 撑满 = 100vh, 等价.

**结论**: 嵌入 + 独立窗双形态共用同款顶层 class, 不要 v-if 分两套.

## 反模式

```vue
<!-- ❌ 嵌入主壳后必撑出 main 范围 -->
<div class="flex flex-col h-screen bg-default text-default">
  ...
</div>
```

```vue
<!-- ❌ 加 overflow-hidden 但漏 min-h-0, flex 子仍可能超 -->
<div class="flex flex-col h-full overflow-hidden">
  <div class="flex-1">...</div>  <!-- flex item 默认 min-height:auto 撑出父 -->
</div>
```

```vue
<!-- ❌ 给 main 改成 overflow-hidden 让全局所有 view 不能滚 — 跨 view 风险 -->
<!-- App.vue main -->
<main class="flex-1 overflow-hidden">
```

## 正解

```vue
<div class="flex flex-col h-full min-h-0 overflow-hidden bg-default text-default">
  <header v-if="isStandalone" class="h-14 shrink-0">...</header>
  <div class="flex flex-1 min-h-0">  <!-- 内部 flex 也加 min-h-0 -->
    <aside>...</aside>
    <div class="flex-1 min-w-0">...</div>
    <aside>...</aside>
  </div>
</div>
```

## Case 1 — 2026-05-28 ContainerEditorView 嵌入主壳后 minimap 切 + 纵滚条 + 滚轮失效

v2 主壳瘦身 Task 5 把 ContainerEditorView header 改成 `v-if="isStandalone"` (嵌入态不渲染), 但顶层 div 仍是老的 `class="flex flex-col h-screen"`. 嵌入主壳后:

- ContainerEditorView 强行 100vh, 主 `<main class="flex-1 overflow-auto pr-3">` 父高 = `vh - title(56) - logpanel(220) - statusbar` ≈ vh - 320. ContainerEditorView 比父高 320px → 触发 main 出滚动条.
- MiniMap 在 canvas 内 `position="bottom-right"`, canvas 跟着 ContainerEditorView 被推下去 → minimap 视觉跑屏外
- 用户想拖滚动条: 滚轮在 canvas 上被 vue-flow 吃做缩放, 拖滚动条只能用鼠标 hover 滚动条本体 — 用户体验崩

Fix [commit `b839926`](https://localhost/b839926): `h-screen` → `h-full min-h-0 overflow-hidden`. 一行 class 修. 子窗形态不受影响.

教训: 把 view 从独立窗改嵌入主壳, **第一件事不是改路由, 是确认顶层 size 计算是否还成立**. 嵌入 = 父容器算好高度交给你, view 必须 h-full 接收, 不能再脑补 100vh.
