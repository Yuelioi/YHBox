<template>
  <div class="px-8 py-6 space-y-6">
    <!-- 主介绍卡 -->
    <section class="rounded-xl bg-default border border-default p-8">
      <div class="flex flex-col items-center text-center">
        <div class="size-20 rounded-full bg-elevated flex items-center justify-center mb-5">
          <UIcon name="i-tabler-info-circle" class="size-9 text-muted" />
        </div>
        <h2 class="text-lg font-semibold text-highlighted mb-2">
          {{ info?.name ?? 'YHBox' }}
          <span class="text-muted font-normal ml-1">v{{ info?.version ?? '...' }}</span>
        </h2>
        <p class="text-sm text-muted leading-relaxed max-w-sm">
          异环 桌面端游戏自动化工具集。
        </p>
      </div>
    </section>

    <!-- 作者 / 链接 -->
    <section class="rounded-xl bg-default border border-default p-5">
      <h3 class="text-xs font-semibold uppercase tracking-wider text-dimmed mb-3">作者 · 链接</h3>
      <div class="text-sm space-y-2">
        <div class="flex items-center justify-between gap-4">
          <span class="text-muted inline-flex items-center gap-1.5">
            <UIcon name="i-tabler-user" class="size-3.5" /> 作者
          </span>
          <span class="text-default font-medium">{{ info?.author ?? '—' }}</span>
        </div>
        <div class="flex items-center justify-between gap-4">
          <span class="text-muted inline-flex items-center gap-1.5">
            <UIcon name="i-tabler-brand-github" class="size-3.5" /> 源码
          </span>
          <button
            v-if="info?.repo"
            type="button"
            class="text-default font-medium hover:text-sky-400 transition-colors cursor-pointer truncate"
            @click="openExternal(info.repo)"
          >
            {{ info.repo.replace('https://', '') }} ↗
          </button>
        </div>
        <div class="flex items-center justify-between gap-4">
          <span class="text-muted inline-flex items-center gap-1.5">
            <UIcon name="i-tabler-brand-bilibili" class="size-3.5" /> B 站
          </span>
          <button
            v-if="info?.bilibili"
            type="button"
            class="text-default font-medium hover:text-sky-400 transition-colors cursor-pointer truncate"
            @click="openExternal(info.bilibili)"
          >
            {{ info.bilibili.replace('https://', '') }} ↗
          </button>
        </div>
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
        <div class="flex items-center justify-between">
          <span class="text-muted">Logger</span>
          <span class="text-default font-medium">zerolog</span>
        </div>
        <div class="flex items-center justify-between">
          <span class="text-muted">CV / Input</span>
          <span class="text-default font-medium">pkg/vision + Win32</span>
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
import { onMounted, ref } from 'vue'
import { Browser } from '@wailsio/runtime'
import { backend } from '@/lib/backend'

interface AppInfo {
  name: string
  version: string
  author: string
  repo: string
  bilibili: string
}

const info = ref<AppInfo | null>(null)

onMounted(async () => {
  const r = await backend.appInfo.info()
  if (r) info.value = r as AppInfo
})

// 走 wails3 后端开默认浏览器；webview target=_blank 会被引到内嵌 webview 里
function openExternal(url: string) {
  Browser.OpenURL(url)
}
</script>
