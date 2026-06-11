<template>
  <div class="space-y-3">
    <!-- Toolbar -->
    <div class="flex items-center gap-2 flex-wrap">
      <UInput
        v-model="search"
        icon="i-tabler-search"
        :placeholder="t('template.manager.search')"
        class="flex-1 min-w-48 max-w-md"
      />
      <USelect
        v-model="sortKey"
        :items="sortItems"
        class="w-44"
        :ui="{ content: 'min-w-[200px]' }"
      />
      <UButton
        size="xs"
        variant="ghost"
        :icon="sortDesc ? 'i-tabler-sort-descending' : 'i-tabler-sort-ascending'"
        :title="sortDesc ? t('template.manager.sort_desc') : t('template.manager.sort_asc')"
        @click="sortDesc = !sortDesc"
      />
      <span class="text-xs text-dimmed">{{ filtered.length }} / {{ entries.length }}</span>
      <!-- 主操作放最右，跟其他 tab 一致 -->
      <slot name="toolbar-right" />
    </div>

    <!-- 空态 -->
    <div
      v-if="entries.length === 0"
      class="rounded-md border border-dashed border-default/60 bg-elevated/40 py-12 px-6 text-center"
    >
      <UIcon name="i-tabler-photo-off" class="size-8 text-dimmed mx-auto mb-2" />
      <p class="text-sm text-dimmed">{{ t('template.manager.empty') }}</p>
      <p class="text-[11px] text-dimmed mt-1">{{ t('template.manager.empty_hint') }}</p>
    </div>

    <!-- 过滤后空态 -->
    <div
      v-else-if="filtered.length === 0"
      class="rounded-md border border-dashed border-default/60 bg-elevated/40 py-10 px-6 text-center text-sm text-dimmed"
    >
      {{ t('template.manager.no_match', { search }) }}
    </div>

    <!-- 网格 -->
    <div
      v-else
      class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6 gap-3"
    >
      <div
        v-for="s in filtered"
        :key="s.guid"
        class="group rounded-md border border-default bg-elevated/30 overflow-hidden hover:border-primary/60 transition-colors flex flex-col"
      >
        <!-- 缩略图 -->
        <button
          type="button"
          class="relative w-full aspect-[4/3] bg-elevated flex items-center justify-center overflow-hidden cursor-zoom-in"
          @click="openPreview(s)"
        >
          <img
            v-if="thumbCache[s.guid]"
            :src="thumbCache[s.guid]"
            class="max-w-full max-h-full object-contain"
            :alt="s.name"
          />
          <UIcon v-else name="i-tabler-photo" class="size-8 text-dimmed" />
          <div class="absolute top-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity">
            <span
              class="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[10px] bg-elevated text-toned"
            >
              <UIcon name="i-tabler-zoom-in" class="size-3" /> {{ t('template.manager.preview') }}
            </span>
          </div>
        </button>

        <div class="p-2 space-y-0.5 min-w-0">
          <div class="text-xs text-highlighted truncate" :title="s.name">{{ s.name }}</div>
          <div v-if="s.tags?.length" class="text-[10px] text-dimmed truncate">
            {{ s.tags.join(', ') }}
          </div>
          <div class="text-[10px] text-dimmed flex items-center gap-1.5 truncate">
            <span class="inline-flex items-center gap-0.5">
              <UIcon name="i-tabler-layers" class="size-3" />
              {{ s.variantCount }} {{ t('template.manager.variant_count') }}
            </span>
          </div>
        </div>

        <div class="px-2 pb-2 flex justify-end gap-1">
          <UButton
            size="xs"
            variant="ghost"
            :color="copiedGuid === s.guid ? 'success' : 'neutral'"
            :icon="copiedGuid === s.guid ? 'i-tabler-check' : 'i-tabler-copy'"
            :title="copiedGuid === s.guid ? t('common.copied') : t('template.manager.copy_key')"
            @click="copyGuid(s.guid)"
          />
          <UButton
            size="xs"
            variant="ghost"
            color="error"
            icon="i-tabler-trash"
            :title="t('template.manager.delete_template_tip', { key: s.name })"
            @click="onDelete(s)"
          />
        </div>
      </div>
    </div>

    <!-- Preview Modal -->
    <UModal
      :open="previewOpen"
      @update:open="(v: boolean) => (previewOpen = v)"
      :ui="{ content: 'sm:max-w-[820px]' }"
    >
      <template #content>
        <div class="bg-default flex flex-col">
          <header class="flex items-center gap-2 px-5 py-3 border-b border-default">
            <UIcon name="i-tabler-photo" class="size-4 text-primary" />
            <h3 class="text-sm font-medium truncate">{{ previewSummary?.name }}</h3>
            <span class="ml-auto text-[11px] text-dimmed font-mono truncate max-w-[200px]">{{ previewSummary?.guid }}</span>
            <UButton
              size="xs"
              variant="ghost"
              color="neutral"
              icon="i-tabler-x"
              class="ml-2"
              @click="previewOpen = false"
            />
          </header>
          <div class="p-5 space-y-3">
            <div
              class="rounded-md overflow-hidden border border-default bg-elevated flex items-center justify-center"
              style="min-height: 240px"
            >
              <img
                v-if="previewDataURL"
                :src="previewDataURL"
                class="max-w-full max-h-[60vh] object-contain"
              />
              <div v-else class="text-xs text-dimmed py-12">{{ t('common.loading') }}</div>
            </div>
            <div class="text-xs space-y-1">
              <div v-if="previewSummary?.tags?.length">
                <span class="text-dimmed">{{ t('template.manager.tags_label') }}</span>
                <span class="text-highlighted">{{ previewSummary.tags.join(', ') }}</span>
              </div>
              <div class="text-[11px] text-dimmed">
                {{ t('template.manager.variant_count') }}: {{ previewSummary?.variantCount }}
                · {{ t('template.manager.created_at', { time: formatTime(previewSummary?.createdAt) }) }}
              </div>
            </div>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTemplatesStore } from '@/stores/templates'
import { useToast } from '@nuxt/ui/composables'
import { useConfirm } from '@/composables/useConfirm'
import { backend, type AssetSummary } from '@/lib/backend'

const { t } = useI18n()

const store = useTemplatesStore()
const toast = useToast()
const { confirm } = useConfirm()

const search = ref('')
const sortKey = ref<'name' | 'createdAt' | 'variantCount'>('name')
const sortDesc = ref(false)
const thumbCache = ref<Record<string, string>>({})

const sortItems = computed(() => [
  { label: t('template.manager.view_by_name'), value: 'name' },
  { label: t('template.manager.view_by_created'), value: 'createdAt' },
  { label: t('template.manager.view_by_res'), value: 'variantCount' },
])

// entries: AssetSummary[] (template kind, map 已过滤)
const entries = computed(() => Object.values(store.map))

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  let arr = entries.value
  if (q) {
    arr = arr.filter((s) => {
      return (
        s.name?.toLowerCase().includes(q) ||
        s.guid.toLowerCase().includes(q) ||
        s.tags?.some((tag) => tag.toLowerCase().includes(q))
      )
    })
  }
  const sorted = [...arr]
  sorted.sort((a, b) => {
    let cmp = 0
    switch (sortKey.value) {
      case 'name':
        cmp = (a.name ?? '').localeCompare(b.name ?? '')
        break
      case 'createdAt':
        cmp = (a.createdAt ?? '').localeCompare(b.createdAt ?? '')
        break
      case 'variantCount':
        cmp = (a.variantCount ?? 0) - (b.variantCount ?? 0)
        break
    }
    return sortDesc.value ? -cmp : cmp
  })
  return sorted
})

async function loadThumb(s: AssetSummary) {
  if (thumbCache.value[s.guid] || !s.firstBlobSha) return
  const r = await store.readBlobDataURL(s.firstBlobSha)
  if (typeof r === 'string') thumbCache.value[s.guid] = r
}

// 监听 filtered 变化，懒加载未加载的缩略图
watch(
  filtered,
  async (list) => {
    for (const s of list) {
      if (!thumbCache.value[s.guid]) await loadThumb(s)
    }
  },
  { immediate: true },
)

async function onDelete(s: AssetSummary) {
  // 删前同步扫一遍引用 → 有引用则弹"被 N 处引用"确认.
  const refs = await backend.assets.referrers(s.guid)
  const n = refs?.length ?? 0
  const description =
    n > 0
      ? t('template.manager.delete_confirm_referenced', { key: s.name, n })
      : t('template.manager.delete_confirm', { key: s.name })
  const yes = await confirm({
    title: t('template.manager.delete_title'),
    description,
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  await store.remove(s.guid)
  delete thumbCache.value[s.guid]
}

const copiedGuid = ref('')
let copiedGuidTimer = 0
function copyGuid(guid: string) {
  navigator.clipboard?.writeText(guid).then(
    () => {
      copiedGuid.value = guid
      window.clearTimeout(copiedGuidTimer)
      copiedGuidTimer = window.setTimeout(() => {
        copiedGuid.value = ''
      }, 1500)
    },
    () => toast.add({ title: t('toast.copy_failed'), color: 'error' }),
  )
}

// Preview modal state
const previewOpen = ref(false)
const previewSummary = ref<AssetSummary | null>(null)
const previewDataURL = ref('')

async function openPreview(s: AssetSummary) {
  previewSummary.value = s
  previewDataURL.value = thumbCache.value[s.guid] ?? ''
  previewOpen.value = true
  if (!previewDataURL.value && s.firstBlobSha) {
    const r = await store.readBlobDataURL(s.firstBlobSha)
    if (typeof r === 'string') {
      previewDataURL.value = r
      thumbCache.value[s.guid] = r
    }
  }
}

function formatTime(s?: string): string {
  if (!s) return '—'
  try {
    return new Date(s).toLocaleString()
  } catch {
    return s
  }
}

onMounted(() => {
  void store.reload()
})
</script>
