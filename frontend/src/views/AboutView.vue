<template>
  <div class="px-8 py-6 space-y-6">
    <!-- 主介绍卡 -->
    <section class="rounded-xl bg-default border border-default p-8">
      <div class="flex flex-col items-center text-center">
        <div class="size-20 rounded-full bg-elevated flex items-center justify-center mb-5">
          <UIcon name="i-tabler-info-circle" class="size-9 text-muted" />
        </div>
        <h2 class="text-lg font-semibold text-highlighted mb-2">YHBox v1.1</h2>
        <p class="text-sm text-muted leading-relaxed max-w-sm">
          基于 Wails 3 + Vue 3 + NuxtUI v4 + zerolog
          构建。一个干净、专业、好用的桌面端游戏自动化工具。
        </p>
      </div>
    </section>

    <!-- 技术栈 -->
    <section class="rounded-xl bg-default border border-default p-5">
      <h3 class="text-xs font-semibold uppercase tracking-wider text-dimmed mb-3">技术栈</h3>
      <div class="grid grid-cols-2 gap-x-6 gap-y-2 text-sm">
        <div class="flex items-center justify-between">
          <span class="text-muted">Runtime</span>
          <span class="text-default font-medium">Wails 3</span>
        </div>
        <div class="flex items-center justify-between">
          <span class="text-muted">Frontend</span>
          <span class="text-default font-medium">Vue 3 + TS</span>
        </div>
        <div class="flex items-center justify-between">
          <span class="text-muted">UI</span>
          <span class="text-default font-medium">NuxtUI v4</span>
        </div>
        <div class="flex items-center justify-between">
          <span class="text-muted">Backend</span>
          <span class="text-default font-medium">Go 1.25</span>
        </div>
      </div>
    </section>

    <!-- 致谢 -->
    <section class="rounded-xl bg-default border border-default p-5">
      <h3 class="text-xs font-semibold uppercase tracking-wider text-dimmed mb-3">致谢</h3>
      <div class="text-sm space-y-2">
        <div class="flex items-start justify-between gap-4">
          <span class="text-muted">软件图标</span>
          <button
            type="button"
            class="text-default font-medium hover:text-sky-400 transition-colors cursor-pointer"
            @click="openExternal('https://www.pixiv.net/artworks/120610310')"
          >
            Pixiv #120610310 ↗
          </button>
        </div>
        <p class="text-xs text-dimmed leading-relaxed">
          图标来源于 Pixiv 公开作品，版权归原作者所有。本工具仅作个人使用，如作者要求请联系替换。
        </p>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { Browser } from '@wailsio/runtime'

// 走 Wails3 后端打开默认浏览器；webview 自己用 target=_blank 会被引到内嵌 webview 里
function openExternal(url: string) {
  Browser.OpenURL(url)
}
</script>
