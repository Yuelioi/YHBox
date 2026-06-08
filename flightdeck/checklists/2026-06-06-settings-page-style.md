---
status: active
when_to_read: 写/改任何设置 tab（SettingsView 下的子页：General/Hotkeys/Input/Launcher 等）前；新增设置子页前；想让设置页风格统一时
applies_to: [frontend, vue, settings, style, consistency, nuxtui, SettingsView]
when_to_update: 改 SettingsView 设置子页的布局/风格基线 / 设置页公共组件时
last_updated: 2026-06-06
---

# 设置页视觉范式

**基准 = `SettingsGeneral.vue`（通用 tab）。** 所有设置子页照它来，别各写各的。下面每条都从 General 提炼，直接照抄。
组件级通用规范（Tailwind / Nuxt UI 用法）见 [[ui]]；独立工具 HUD 窗是另一套外壳，见 [[standalone-window-style]]，别混。

## 骨架（照抄）

```vue
<template>
  <!-- 页容器：左右 px-8、上下 py-6、分区间距 space-y-6；不要加 max-w-* 限宽（General 不限宽） -->
  <div class="px-8 py-6 space-y-6">
    <!-- 一个分区 = 一张卡片 -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <!-- 分区标题：icon + h2，固定结构 -->
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-xxx" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">{{ 标题 }}</h2>
      </div>

      <!-- 可选：分区说明一段 -->
      <p class="text-xs text-dimmed leading-relaxed">{{ 说明 }}</p>

      <!-- 同一分区内多个设置项之间用这条分隔 -->
      <div class="border-t border-default/60" />

      <!-- 设置行：左文案 + 右控件，两端对齐 -->
      <div class="flex items-center justify-between gap-6">
        <div>
          <div class="text-sm text-default">{{ 项名 }}</div>
          <p class="text-xs text-dimmed mt-0.5">{{ 项说明 }}</p>
        </div>
        <USelect class="w-32" ... />   <!-- 或 USwitch / UInput 等 -->
      </div>
    </section>
  </div>
</template>
```

## 硬性规则

- **页容器**：`px-8 py-6 space-y-6`。**不加 `max-w-*` / `mx-auto`**——General 内容靠 `justify-between` 自己撑开，不靠限宽。
- **分区卡片**：永远 `<section class="rounded-xl bg-default border border-default p-5">`。
  - 圆角只用 `rounded-xl`（卡片）；卡片里的列表项用 `rounded-md`。不要 `rounded-md`/`rounded-lg` 当卡片，不要 `bg-elevated/40` 当卡片底（那是列表项的底）。
  - 卡片内间距 `space-y-4`（纯设置项）或 `space-y-3`（列表型，如 hotkeys/分组）。
- **分区标题**：固定 `<div class="flex items-center gap-2"><UIcon class="size-4 text-dimmed"/><h2 class="text-sm font-medium text-highlighted">`。
  - 用 `<h2>`，不用 `<h3>/<h4>`；不用 `text-base`。**每个分区都配一个 tabler 图标**，别留空标题。
- **分区内分隔**：同卡片里多个设置项之间用 `<div class="border-t border-default/60" />`。
- **设置行**：`flex items-center justify-between gap-6`。左侧 `text-sm text-default` 项名 + `text-xs text-dimmed mt-0.5` 说明；右侧控件。
- **列表项**（hotkey 行 / 启动器分组项 / profile 行等）：`rounded-md bg-elevated/30 border border-default/60 px-3 py-2`。被选中/激活态再叠 `border-primary/50 bg-primary/5`。
- **字号**：正文 `text-sm`，次要说明 `text-xs`。**禁用魔法字号 `text-[10px]`/`text-[11px]`**——一律 `text-xs`。
- **底部操作按钮**（新建/添加）：`<UButton variant="soft" color="primary" icon="i-tabler-plus">`，放分区/列表末尾，参考 Input 的「添加 profile」。
- **空态**：`text-xs text-dimmed py-8 text-center border border-dashed border-default/60 rounded-xl`。

## 自查（改完逐条看）

- [ ] 页容器是 `px-8 py-6 space-y-6`，没有 `max-w-*`。
- [ ] 每个分区都是 `rounded-xl bg-default border border-default p-5` 的 `<section>`。
- [ ] 每个分区标题都是 icon + `<h2 class="text-sm ...">` 结构，且有图标。
- [ ] 全页搜不到 `text-[10px]` / `text-[11px]` / `text-base`（标题除外无）；次要文字都是 `text-xs`。
- [ ] 列表项底是 `bg-elevated/30 border-default/60`，不是 `bg-default/40`。
- [ ] 跟 General tab 并排切一眼：卡片底色、圆角、标题、间距对得上，不突兀。
- [ ] 视觉拿不准 → 用 [[headless-ui-verify]] 离屏渲染，把新页和 General 截图对比。

## 已对齐（2026-06-06）

四个 tab 全部对齐 General：Hotkeys（标题结构 + `space-y-6`）、Input（卡片/字号/去限宽，差最多）、Launcher（重排为卡片范式）。新增设置子页直接按本清单写，不要再发明新样式。
