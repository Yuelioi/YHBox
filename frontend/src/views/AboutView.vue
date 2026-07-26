<template>
  <div class="h-full overflow-y-auto">
    <div class="detail-form px-8 py-8">
      <!-- 产品介绍 -->
      <section class="flex items-start gap-4 border-b border-default pb-7">
        <div class="pt-0.5">
          <IconBadge icon="i-tabler-info-circle" size="lg" shape="round" color="primary" />
        </div>
        <div class="min-w-0">
          <h2 class="text-lg font-semibold text-highlighted mb-1.5">
            {{ info?.name ?? 'Yotta' }}
            <span class="text-muted font-normal ml-1 font-mono tabular-nums"
              >v{{ info?.version ?? '...' }}</span
            >
          </h2>
          <p class="max-w-xl text-sm text-muted leading-relaxed">
            {{ t('about.tagline') }}
          </p>
        </div>
      </section>

      <!-- 核心概念 -->
      <section class="detail-section">
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-bulb" class="size-4 text-dimmed" />
          <h2 class="text-sm font-medium text-highlighted">{{ t('about.concepts.title') }}</h2>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-5">
          <div v-for="c in concepts" :key="c.key">
            <div class="flex items-center gap-2 mb-1.5">
              <UIcon :name="c.icon" class="size-4" :class="c.iconClass" />
              <span class="text-sm font-medium text-highlighted">{{
                t(`about.concepts.${c.key}.name`)
              }}</span>
            </div>
            <p class="text-xs text-muted leading-relaxed">
              {{ t(`about.concepts.${c.key}.desc`) }}
            </p>
          </div>
        </div>
      </section>

      <!-- 作者 / 链接 -->
      <section class="detail-section">
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-user" class="size-4 text-dimmed" />
          <h2 class="text-sm font-medium text-highlighted">{{ t('about.section_author') }}</h2>
        </div>
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
              class="rounded text-default font-medium hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary transition-colors cursor-pointer truncate"
              @click="openExternal(info.repo)"
            >
              {{ info.repo.replace('https://', '') }} ↗
            </button>
          </div>
          <div class="flex items-center justify-between gap-4">
            <span class="text-muted inline-flex items-center gap-1.5">
              <UIcon name="i-tabler-brand-bilibili" class="size-3.5" /> B
              {{ t('about.label_site') }}
            </span>
            <button
              v-if="info?.bilibili"
              type="button"
              class="rounded text-default font-medium hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary transition-colors cursor-pointer truncate"
              @click="openExternal(info.bilibili)"
            >
              {{ info.bilibili.replace('https://', '') }} ↗
            </button>
          </div>
        </div>
      </section>

      <!-- 技术栈 -->
      <section class="detail-section">
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-stack-2" class="size-4 text-dimmed" />
          <h2 class="text-sm font-medium text-highlighted">{{ t('about.section_stack') }}</h2>
        </div>
        <div class="grid grid-cols-2 gap-x-6 gap-y-2 text-sm">
          <div class="flex items-center justify-between">
            <span class="text-muted">Runtime</span>
            <span class="text-default font-medium font-mono">Wails 3</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-muted">Frontend</span>
            <span class="text-default font-medium font-mono">Vue 3 + TS</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-muted">UI</span>
            <span class="text-default font-medium font-mono">NuxtUI v4</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-muted">Backend</span>
            <span class="text-default font-medium font-mono">Go 1.25</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-muted">Logger</span>
            <span class="text-default font-medium font-mono">zerolog</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-muted">CV / Input</span>
            <span class="text-default font-medium font-mono">pkg/vision + Win32</span>
          </div>
        </div>
      </section>

      <!-- 致谢 -->
      <section class="space-y-4 pt-1">
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-heart" class="size-4 text-dimmed" />
          <h2 class="text-sm font-medium text-highlighted">{{ t('about.section_thanks') }}</h2>
        </div>
        <div class="text-sm space-y-2">
          <div class="flex items-start justify-between gap-4">
            <span class="text-muted">{{ t('about.label_icon') }}</span>
            <button
              type="button"
              class="rounded text-default font-medium hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary transition-colors cursor-pointer"
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
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Browser } from '@wailsio/runtime'
import { backend } from '@/lib/backend'
import IconBadge from '@/components/common/IconBadge.vue'

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
  { key: 'workflow', icon: 'i-tabler-route', iconClass: 'text-primary' },
  { key: 'catalog', icon: 'i-tabler-schema', iconClass: 'text-fuchsia-300' },
  { key: 'program_run', icon: 'i-tabler-player-play', iconClass: 'text-emerald-300' },
  { key: 'installation', icon: 'i-tabler-plug-connected', iconClass: 'text-amber-300' },
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
