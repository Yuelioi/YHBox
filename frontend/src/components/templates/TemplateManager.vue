<template>
  <div class="space-y-3">
    <!-- Toolbar -->
    <div class="flex items-center gap-2 flex-wrap">
      <UInput
        v-model="search"
        icon="i-tabler-search"
        placeholder="搜索 key / 名称 / 描述..."
        class="flex-1 min-w-48 max-w-md"
      />
      <USelect
        v-model="resolutionFilter"
        :items="resolutionItems"
        class="w-40"
        :ui="{ content: 'min-w-[180px]' }"
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
        :title="sortDesc ? '降序' : '升序'"
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
      <p class="text-sm text-dimmed">模板库还是空的</p>
      <p class="text-[11px] text-dimmed mt-1">点上面「截图新模板」开始收录。</p>
    </div>

    <!-- 过滤后空态 -->
    <div
      v-else-if="filtered.length === 0"
      class="rounded-md border border-dashed border-default/60 bg-elevated/40 py-10 px-6 text-center text-sm text-dimmed"
    >
      没有匹配「{{ search }}」的模板
    </div>

    <!-- 网格 -->
    <div
      v-else
      class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6 gap-3"
    >
      <div
        v-for="[key, meta] in filtered"
        :key="key"
        class="group rounded-md border border-default bg-elevated/30 overflow-hidden hover:border-primary/60 transition-colors flex flex-col"
      >
        <!-- 缩略图 -->
        <button
          type="button"
          class="relative w-full aspect-[4/3] bg-zinc-900/60 flex items-center justify-center overflow-hidden cursor-zoom-in"
          @click="openPreview(key, meta)"
        >
          <img
            v-if="thumbCache[key]"
            :src="thumbCache[key]"
            class="max-w-full max-h-full object-contain"
            :alt="meta.name"
          />
          <UIcon v-else name="i-tabler-photo" class="size-8 text-dimmed" />
          <div class="absolute top-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity">
            <span
              class="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[10px] bg-zinc-900/80 text-zinc-200"
            >
              <UIcon name="i-tabler-zoom-in" class="size-3" /> 预览
            </span>
          </div>
        </button>

        <div class="p-2 space-y-0.5 min-w-0">
          <div class="text-[11px] font-mono text-toned truncate" :title="key">{{ key }}</div>
          <div class="text-xs text-highlighted truncate" :title="meta.name">{{ meta.name }}</div>
          <div class="text-[10px] text-dimmed flex items-center gap-1.5 truncate">
            <span class="inline-flex items-center gap-0.5">
              <UIcon name="i-tabler-aspect-ratio" class="size-3" />
              {{ meta.width }}×{{ meta.height }}
            </span>
            <span class="text-default/40">·</span>
            <span class="inline-flex items-center gap-0.5">
              <UIcon name="i-tabler-device-desktop" class="size-3" />
              {{ meta.recordedResolution[0] }}×{{ meta.recordedResolution[1] }}
            </span>
          </div>
          <div
            v-if="meta.description"
            class="text-[10px] text-dimmed truncate"
            :title="meta.description"
          >
            {{ meta.description }}
          </div>
        </div>

        <div class="px-2 pb-2 flex justify-end gap-1">
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-copy"
            title="复制 key"
            @click="copyKey(key)"
          />
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-pencil"
            :title="`编辑 ${key}`"
            @click="openEdit(key, meta)"
          />
          <UButton
            size="xs"
            variant="ghost"
            color="error"
            icon="i-tabler-trash"
            :title="`删除 ${key}`"
            @click="onDelete(key)"
          />
        </div>
      </div>
    </div>

    <!-- Edit Modal -->
    <UModal
      :open="editOpen"
      @update:open="(v: boolean) => (editOpen = v)"
      :ui="{ content: 'sm:max-w-[480px]' }"
    >
      <template #content>
        <div class="bg-default flex flex-col">
          <header class="flex items-center gap-2 px-5 py-3 border-b border-default">
            <UIcon name="i-tabler-pencil" class="size-4 text-primary" />
            <h3 class="text-sm font-medium text-highlighted">编辑模板</h3>
            <span class="ml-auto" />
            <UButton
              size="xs"
              variant="ghost"
              color="neutral"
              icon="i-tabler-x"
              @click="editOpen = false"
            />
          </header>
          <div class="p-5 space-y-4">
            <div class="space-y-1.5">
              <label class="block text-[11px] text-toned">key</label>
              <UInput :model-value="editKey" disabled class="w-full font-mono" />
              <p class="text-[10px] text-dimmed">
                key 不能改（容器节点引用了它）；想换 key 请重新截图
              </p>
            </div>
            <div class="space-y-1.5">
              <label class="block text-[11px] text-toned">显示名</label>
              <UInput v-model="editName" class="w-full" placeholder="例：上钩图标" />
            </div>
            <div class="space-y-1.5">
              <label class="block text-[11px] text-toned">描述</label>
              <UTextarea
                v-model="editDescription"
                :rows="3"
                class="w-full"
                placeholder="什么场景下匹配？阈值有什么注意事项？"
              />
            </div>
          </div>
          <footer class="px-5 py-3 border-t border-default flex items-center justify-end gap-2">
            <UButton variant="ghost" color="neutral" @click="editOpen = false">取消</UButton>
            <UButton color="primary" icon="i-tabler-check" :loading="editSaving" @click="onSaveEdit"
              >保存</UButton
            >
          </footer>
        </div>
      </template>
    </UModal>

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
            <h3 class="text-sm font-medium font-mono truncate">{{ previewKey }}</h3>
            <span class="ml-auto text-[11px] text-dimmed"
              >{{ previewMeta?.width }}×{{ previewMeta?.height }}</span
            >
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
              class="rounded-md overflow-hidden border border-default bg-zinc-900/60 flex items-center justify-center"
              style="min-height: 240px"
            >
              <img
                v-if="previewDataURL"
                :src="previewDataURL"
                class="max-w-full max-h-[60vh] object-contain"
              />
              <div v-else class="text-xs text-dimmed py-12">加载中...</div>
            </div>
            <div class="text-xs space-y-1">
              <div>
                <span class="text-dimmed">显示名：</span
                ><span class="text-highlighted">{{ previewMeta?.name }}</span>
              </div>
              <div v-if="previewMeta?.description">
                <span class="text-dimmed">描述：</span>{{ previewMeta?.description }}
              </div>
              <div class="text-[11px] text-dimmed">
                录制分辨率 {{ previewMeta?.recordedResolution?.[0] }}×{{
                  previewMeta?.recordedResolution?.[1]
                }}
                · 创建于 {{ formatTime(previewMeta?.createdAt) }}
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
import { useTemplatesStore } from '@/stores/templates'
import { useToast } from '@nuxt/ui/composables'
import { backend, type TemplateMeta } from '@/lib/backend'

const store = useTemplatesStore()
const toast = useToast()

const search = ref('')
const sortKey = ref<'key' | 'name' | 'resolution' | 'createdAt'>('key')
const sortDesc = ref(false)
const resolutionFilter = ref<string>('__all__')
const thumbCache = ref<Record<string, string>>({})

const sortItems = [
  { label: '按 key 排序', value: 'key' },
  { label: '按显示名排序', value: 'name' },
  { label: '按分辨率排序', value: 'resolution' },
  { label: '按创建时间排序', value: 'createdAt' },
]

const entries = computed(() => Object.entries(store.map))

// 从模板库收集 unique 录制分辨率
const resolutionItems = computed(() => {
  const seen = new Map<string, number>()
  for (const [, m] of entries.value) {
    const w = m.recordedResolution?.[0] ?? 0
    const h = m.recordedResolution?.[1] ?? 0
    if (!w || !h) continue
    const k = `${w}x${h}`
    seen.set(k, (seen.get(k) ?? 0) + 1)
  }
  const list = [...seen.entries()]
    .sort((a, b) => b[1] - a[1])
    .map(([k, n]) => ({ label: `${k.replace('x', ' × ')} (${n})`, value: k }))
  return [{ label: '全部分辨率', value: '__all__' }, ...list]
})

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  let arr = entries.value
  if (q) {
    arr = arr.filter(([k, m]) => {
      return (
        k.toLowerCase().includes(q) ||
        m.name?.toLowerCase().includes(q) ||
        m.description?.toLowerCase().includes(q)
      )
    })
  }
  if (resolutionFilter.value !== '__all__') {
    const [fw, fh] = resolutionFilter.value.split('x').map(Number)
    arr = arr.filter(
      ([, m]) => m.recordedResolution?.[0] === fw && m.recordedResolution?.[1] === fh,
    )
  }
  const sorted = [...arr]
  sorted.sort(([ak, am], [bk, bm]) => {
    let cmp = 0
    switch (sortKey.value) {
      case 'key':
        cmp = ak.localeCompare(bk)
        break
      case 'name':
        cmp = (am.name ?? '').localeCompare(bm.name ?? '')
        break
      case 'resolution':
        cmp = am.width * am.height - bm.width * bm.height
        break
      case 'createdAt':
        cmp = (am.createdAt ?? '').localeCompare(bm.createdAt ?? '')
        break
    }
    return sortDesc.value ? -cmp : cmp
  })
  return sorted
})

async function loadThumb(key: string) {
  if (thumbCache.value[key]) return
  const r = await backend.templates.readPngDataURL(key)
  if (typeof r === 'string') thumbCache.value[key] = r
}

// 监听 filtered 变化，懒加载未加载的缩略图
watch(
  filtered,
  async (list) => {
    for (const [k] of list) {
      if (!thumbCache.value[k]) await loadThumb(k)
    }
  },
  { immediate: true },
)

async function onDelete(key: string) {
  if (!confirm(`删除模板 "${key}"？`)) return
  await store.remove(key)
  delete thumbCache.value[key]
}

function copyKey(key: string) {
  navigator.clipboard?.writeText(key).then(
    () => toast.add({ title: '已复制 key', color: 'success', duration: 1500 }),
    () => toast.add({ title: '复制失败', color: 'error' }),
  )
}

// Edit modal state
const editOpen = ref(false)
const editKey = ref('')
const editName = ref('')
const editDescription = ref('')
const editSaving = ref(false)

function openEdit(key: string, meta: TemplateMeta) {
  editKey.value = key
  editName.value = meta.name ?? ''
  editDescription.value = meta.description ?? ''
  editOpen.value = true
}

async function onSaveEdit() {
  editSaving.value = true
  try {
    const ok = await store.updateMeta(
      editKey.value,
      editName.value.trim(),
      editDescription.value.trim(),
    )
    if (ok) editOpen.value = false
  } finally {
    editSaving.value = false
  }
}

// Preview modal state
const previewOpen = ref(false)
const previewKey = ref('')
const previewMeta = ref<TemplateMeta | null>(null)
const previewDataURL = ref('')

async function openPreview(key: string, meta: TemplateMeta) {
  previewKey.value = key
  previewMeta.value = meta
  previewDataURL.value = thumbCache.value[key] ?? ''
  previewOpen.value = true
  if (!previewDataURL.value) {
    const r = await backend.templates.readPngDataURL(key)
    if (typeof r === 'string') {
      previewDataURL.value = r
      thumbCache.value[key] = r
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
