<template>
  <div class="workspace-canvas h-full overflow-y-auto">
    <main class="mx-auto flex w-full max-w-[880px] flex-col px-6 py-8 sm:px-10 sm:py-10">
      <section
        class="relative overflow-hidden rounded-2xl border border-default bg-elevated px-6 py-7 sm:px-9 sm:py-9"
      >
        <YottaMark class="pointer-events-none absolute -right-10 -top-12 size-60 opacity-[0.045]" />
        <div class="relative flex flex-col gap-6 sm:flex-row sm:items-center sm:gap-7">
          <YottaMark class="size-20 shrink-0 shadow-lg shadow-black/20" />
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-3">
              <h1 class="text-3xl font-semibold tracking-[-0.025em] text-highlighted">
                {{ info?.name ?? 'Yotta' }}
              </h1>
              <span
                class="rounded-md border border-default bg-muted/45 px-2 py-1 font-mono text-[11px] tabular-nums text-muted"
              >
                v{{ info?.version ?? '...' }}
              </span>
            </div>
            <p class="mt-3 max-w-2xl text-[15px] leading-7 text-muted">
              {{ t('about.tagline') }}
            </p>
            <div class="mt-5 flex flex-wrap items-center gap-x-5 gap-y-3">
              <p v-if="info?.author" class="flex items-center gap-2 text-sm">
                <UIcon name="i-tabler-user-circle" class="size-4 text-dimmed" aria-hidden="true" />
                <span class="text-dimmed">{{ t('about.author_label') }}</span>
                <span class="font-medium text-highlighted">{{ info.author }}</span>
              </p>
              <button
                v-if="info?.repo"
                type="button"
                class="inline-flex max-w-full cursor-pointer items-center gap-2 rounded-lg border border-primary/30 bg-primary/10 px-3.5 py-2 text-sm font-medium text-primary transition-colors hover:border-primary/45 hover:bg-primary/15 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
                :aria-label="t('about.source_action')"
                @click="openExternal(info.repo)"
              >
                <UIcon name="i-tabler-brand-github" class="size-4 shrink-0" aria-hidden="true" />
                <span class="truncate">{{ repoLabel }}</span>
                <UIcon name="i-tabler-arrow-up-right" class="size-4 shrink-0" aria-hidden="true" />
              </button>
              <button
                v-if="info?.bilibili"
                type="button"
                class="inline-flex cursor-pointer items-center gap-2 rounded-lg border border-default bg-muted/25 px-3.5 py-2 text-sm font-medium text-toned transition-colors hover:bg-muted/45 hover:text-highlighted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
                :aria-label="t('about.author_home_action')"
                @click="openExternal(info.bilibili)"
              >
                <UIcon name="i-tabler-brand-bilibili" class="size-4 shrink-0" aria-hidden="true" />
                <span>Bilibili</span>
                <UIcon name="i-tabler-arrow-up-right" class="size-4 shrink-0" aria-hidden="true" />
              </button>
            </div>
          </div>
        </div>
      </section>

      <section class="mt-10">
        <div class="max-w-2xl">
          <h2 class="text-lg font-semibold text-highlighted">{{ t('about.concepts.title') }}</h2>
          <p class="mt-2 text-sm leading-6 text-muted">{{ t('about.concepts.subtitle') }}</p>
        </div>
        <div class="about-concept-grid mt-5 grid grid-cols-1 sm:grid-cols-2">
          <article v-for="c in concepts" :key="c.key" class="about-concept flex gap-3.5 py-5">
            <div
              class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-default bg-muted/30"
            >
              <UIcon :name="c.icon" class="size-4.5" :class="c.iconClass" aria-hidden="true" />
            </div>
            <div class="min-w-0">
              <h3 class="text-sm font-semibold text-highlighted">
                {{ t(`about.concepts.${c.key}.name`) }}
              </h3>
              <p class="mt-1.5 text-xs leading-5 text-muted">
                {{ t(`about.concepts.${c.key}.desc`) }}
              </p>
            </div>
          </article>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Browser } from '@wailsio/runtime'
import { backend } from '@/lib/backend'
import YottaMark from '@/components/common/YottaMark.vue'

const { t } = useI18n()

interface AppInfo {
  name: string
  version: string
  author: string
  repo: string
  bilibili: string
}

const info = ref<AppInfo | null>(null)
const repoLabel = computed(() => info.value?.repo.replace(/^https?:\/\//, '') ?? '')

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

<style scoped>
.about-concept {
  border-bottom: 1px solid var(--ui-border);
}

.about-concept:last-child {
  border-bottom: 0;
}

@media (min-width: 640px) {
  .about-concept:nth-child(odd) {
    padding-right: 2rem;
    border-right: 1px solid var(--ui-border);
  }

  .about-concept:nth-child(even) {
    padding-left: 2rem;
  }

  .about-concept:nth-child(n + 3) {
    border-bottom: 0;
  }
}
</style>
