<template>
  <div class="px-8 py-6 space-y-6">
    <!-- 主介绍卡 -->
    <section class="rounded-xl bg-default border border-default p-8">
      <div class="flex flex-col items-center text-center">
        <div class="size-20 rounded-full bg-elevated flex items-center justify-center mb-5">
          <UIcon name="i-tabler-info-circle" class="size-9 text-muted" />
        </div>
        <h2 class="text-lg font-semibold text-highlighted mb-2">
          {{ info?.name ?? 'Yotta' }}
          <span class="text-muted font-normal ml-1">v{{ info?.version ?? '...' }}</span>
        </h2>
        <p class="text-sm text-muted leading-relaxed max-w-sm">
          {{ t('about.tagline') }}
        </p>
      </div>
    </section>

    <!-- 核心概念 -->
    <section class="rounded-xl bg-default border border-default p-5">
      <h3 class="text-xs font-semibold uppercase tracking-wider text-dimmed mb-3">{{ t('about.concepts.title') }}</h3>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div
          v-for="c in concepts"
          :key="c.key"
          class="rounded-lg bg-default/50 border border-default/60 p-4"
        >
          <div class="flex items-center gap-2 mb-2">
            <UIcon :name="c.icon" class="size-4" :class="c.iconClass" />
            <span class="text-sm font-medium text-highlighted">{{ t(`about.concepts.${c.key}.name`) }}</span>
          </div>
          <p class="text-xs text-muted leading-relaxed">{{ t(`about.concepts.${c.key}.desc`) }}</p>
        </div>
      </div>
    </section>

    <!-- 作者 / 链接 -->
    <section class="rounded-xl bg-default border border-default p-5">
      <h3 class="text-xs font-semibold uppercase tracking-wider text-dimmed mb-3">{{ t('about.section_author') }}</h3>
      <div class="text-sm space-y-2">
        <div class="flex items-center justify-between gap-4">
          <span class="text-muted inline-flex items-center gap-1.5">
            <UIcon name="i-tabler-user" class="size-3.5" /> {{ t('about.label_author') }}
          </span>
          <span class="text-default font-medium">{{ info?.author ?? '—' }}</span>
        </div>
        <div class="flex items-center justify-between gap-4">
          <span class="text-muted inline-flex items-center gap-1.5">
            <UIcon name="i-tabler-brand-github" class="size-3.5" /> {{ t('about.label_source') }}
          </span>
          <button
            v-if="info?.repo"
            type="button"
            class="text-default font-medium hover:text-primary transition-colors cursor-pointer truncate"
            @click="openExternal(info.repo)"
          >
            {{ info.repo.replace('https://', '') }} ↗
          </button>
        </div>
        <div class="flex items-center justify-between gap-4">
          <span class="text-muted inline-flex items-center gap-1.5">
            <UIcon name="i-tabler-brand-bilibili" class="size-3.5" /> B {{ t('about.label_site') }}
          </span>
          <button
            v-if="info?.bilibili"
            type="button"
            class="text-default font-medium hover:text-primary transition-colors cursor-pointer truncate"
            @click="openExternal(info.bilibili)"
          >
            {{ info.bilibili.replace('https://', '') }} ↗
          </button>
        </div>
      </div>
    </section>

    <!-- 技术栈 -->
    <section class="rounded-xl bg-default border border-default p-5">
      <h3 class="text-xs font-semibold uppercase tracking-wider text-dimmed mb-3">{{ t('about.section_stack') }}</h3>
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
      <h3 class="text-xs font-semibold uppercase tracking-wider text-dimmed mb-3">{{ t('about.section_thanks') }}</h3>
      <div class="text-sm space-y-2">
        <div class="flex items-start justify-between gap-4">
          <span class="text-muted">{{ t('about.label_icon') }}</span>
          <button
            type="button"
            class="text-default font-medium hover:text-primary transition-colors cursor-pointer"
            @click="openExternal('https://www.pixiv.net/artworks/120610310')"
          >
            Pixiv #120610310 ↗
          </button>
        </div>
        <p class="text-xs text-dimmed leading-relaxed">
          {{ t('about.icon_credit') }}
        </p>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Browser } from '@wailsio/runtime'
import { backend } from '@/lib/backend'

const { t } = useI18n()

interface AppInfo {
  name: string
  version: string
  author: string
  repo: string
  bilibili: string
}

const info = ref<AppInfo | null>(null)

// 核心概念（从原帮助页迁来；文案走 about.concepts.* i18n）。
// icon 色是概念分类识别色（非状态语义），配色统一时不要改成 warning/success 等状态色。
const concepts = [
  { key: 'container', icon: 'i-tabler-package', iconClass: 'text-primary' },
  { key: 'subgraph', icon: 'i-tabler-schema', iconClass: 'text-fuchsia-300' },
  { key: 'library', icon: 'i-tabler-books', iconClass: 'text-emerald-300' },
  { key: 'schedule', icon: 'i-tabler-calendar-clock', iconClass: 'text-amber-300' },
]

onMounted(async () => {
  const r = await backend.appInfo.info()
  if (r) info.value = r as AppInfo
})

// 走 wails3 后端开默认浏览器；webview target=_blank 会被引到内嵌 webview 里
function openExternal(url: string) {
  Browser.OpenURL(url)
}
</script>
