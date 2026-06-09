<template>
  <div class="space-y-1.5">
    <!-- 触发按钮：显示已选模板数 + 首个缩略图，点击展开多选面板 -->
    <UPopover v-model:open="open" :ui="{ content: 'w-[360px] p-0' }">
      <UButton
        variant="outline"
        color="neutral"
        size="sm"
        class="w-full justify-start font-normal"
        :title="selectedNames.join(', ') || t('template.picker.not_selected')"
      >
        <div class="flex items-center gap-2 min-w-0 flex-1">
          <img
            v-if="firstThumb"
            :src="firstThumb"
            class="size-6 rounded object-contain bg-elevated shrink-0"
            alt=""
          />
          <UIcon
            v-else
            :name="selected.length ? 'i-tabler-photo' : 'i-tabler-photo-plus'"
            class="size-4 shrink-0"
            :class="selected.length ? 'text-dimmed' : 'text-amber-400'"
          />
          <div class="min-w-0 flex-1 text-left">
            <div class="text-xs text-highlighted truncate">
              {{ summaryLabel }}
            </div>
          </div>
          <UIcon name="i-tabler-chevron-down" class="size-3.5 text-dimmed shrink-0" />
        </div>
      </UButton>

      <template #content>
        <div class="flex flex-col max-h-[420px]">
          <div class="p-2 border-b border-default space-y-2">
            <UInput
              v-model="search"
              autofocus
              size="sm"
              class="w-full"
              icon="i-tabler-search"
              :placeholder="t('template.manager.search')"
            />
            <UButton
              size="xs"
              variant="soft"
              color="primary"
              icon="i-tabler-camera"
              class="w-full justify-center"
              @click="onCaptureNew"
            >
              {{ t('template.picker.capture_new') }}
            </UButton>
          </div>
          <div
            v-if="entries.length === 0"
            class="px-3 py-6 text-center text-[11px] text-amber-300/80"
          >
            <UIcon name="i-tabler-alert-triangle" class="size-4 mx-auto mb-1" />
            {{ t('template.picker.library_empty') }}
          </div>
          <div
            v-else-if="filtered.length === 0"
            class="px-3 py-6 text-center text-[11px] text-dimmed"
          >
            {{ t('template.manager.no_match', { search }) }}
          </div>
          <ul v-else class="flex-1 overflow-y-auto py-1">
            <li v-for="s in filtered" :key="s.guid">
              <button
                type="button"
                class="w-full px-2 py-1.5 flex items-center gap-2 hover:bg-elevated/60 transition-colors text-left"
                :class="isSelected(s.guid) ? 'bg-primary/10' : ''"
                @click="toggle(s.guid)"
              >
                <UIcon
                  :name="isSelected(s.guid) ? 'i-tabler-square-check' : 'i-tabler-square'"
                  class="size-4 shrink-0"
                  :class="isSelected(s.guid) ? 'text-primary' : 'text-dimmed'"
                />
                <img
                  v-if="thumbCache[s.guid]"
                  :src="thumbCache[s.guid]"
                  class="size-8 rounded object-contain bg-elevated shrink-0"
                />
                <div
                  v-else
                  class="size-8 rounded bg-elevated flex items-center justify-center shrink-0"
                >
                  <UIcon name="i-tabler-photo" class="size-3.5 text-dimmed" />
                </div>
                <div class="min-w-0 flex-1">
                  <div class="text-xs text-highlighted truncate">{{ s.name || s.guid }}</div>
                  <div v-if="s.tags?.length" class="text-[10px] text-dimmed truncate">
                    {{ s.tags.join(', ') }}
                  </div>
                </div>
              </button>
            </li>
          </ul>
        </div>
      </template>
    </UPopover>

    <!-- 已选模板 chips（可逐个移除）+ 缺失告警 -->
    <div v-if="selected.length" class="flex flex-wrap gap-1">
      <span
        v-for="guid in selected"
        :key="guid"
        class="inline-flex items-center gap-1 rounded bg-elevated/60 border border-default/40 pl-1.5 pr-1 py-0.5 text-[10px]"
        :class="tplStore.map[guid] ? 'text-toned' : 'text-rose-300/90 border-rose-500/30'"
      >
        <UIcon v-if="!tplStore.map[guid]" name="i-tabler-alert-triangle" class="size-3" />
        <span class="truncate max-w-[140px]">{{ tplStore.map[guid]?.name || guid }}</span>
        <UIcon
          name="i-tabler-x"
          class="size-3 cursor-pointer hover:text-error"
          @click="toggle(guid)"
        />
      </span>
    </div>
    <p v-else class="text-[10px] text-dimmed">{{ t('template.picker.no_template_selected') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTemplatesStore } from '@/stores/templates'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import { backend } from '@/lib/backend'

const { t } = useI18n()

const props = defineProps<{ modelValue: string[] }>()
const emit = defineEmits<{ 'update:modelValue': [v: string[]] }>()

const tplStore = useTemplatesStore()
const open = ref(false)
const search = ref('')
const thumbCache = ref<Record<string, string>>({})

// modelValue 容错: 可能是 undefined / 单 string / string[].
const selected = computed<string[]>(() => {
  const v = props.modelValue as unknown
  if (Array.isArray(v)) return v.filter((x): x is string => typeof x === 'string')
  if (typeof v === 'string' && v) return [v]
  return []
})

onMounted(() => {
  if (Object.keys(tplStore.map).length === 0) void tplStore.reload()
})

// entries: AssetSummary[] (template kind only, map 已过滤)
const entries = computed(() => Object.values(tplStore.map))

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return entries.value
  return entries.value.filter(
    (s) =>
      s.guid.toLowerCase().includes(q) ||
      s.name?.toLowerCase().includes(q) ||
      s.tags?.some((tag) => tag.toLowerCase().includes(q)),
  )
})

function isSelected(guid: string) {
  return selected.value.includes(guid)
}

// 选中的资产名称列表 (用于 title tooltip)
const selectedNames = computed(() =>
  selected.value.map((guid) => tplStore.map[guid]?.name || guid),
)

const firstThumb = computed(() =>
  selected.value[0] ? thumbCache.value[selected.value[0]] : undefined,
)
const summaryLabel = computed(() => {
  if (selected.value.length === 0) return t('template.picker.select_placeholder')
  if (selected.value.length === 1) return tplStore.map[selected.value[0]]?.name || selected.value[0]
  return t('template.picker.selected_count', { n: selected.value.length })
})

// toggle 加入/移除一个 guid（多选，面板不关闭）。
function toggle(guid: string) {
  const cur = selected.value
  const next = cur.includes(guid) ? cur.filter((x) => x !== guid) : [...cur, guid]
  emit('update:modelValue', next)
}

// "+ 现截一张" → 打开 ScreenPicker(template_save)，保存成功后把新 guid 追加到选中。
async function onCaptureNew() {
  const id = 'tpl-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now()
  open.value = false // 关掉下拉，避免遮 picker 窗口
  const waiter = awaitWailsEvent<{ id: string; mode: string; payload: any }>(
    'tools:picker-result',
    (p) => p?.id === id,
  )
  await backend.tools.openScreenPicker('template_save', id, tplStore.containerId)
  const result = await waiter
  if (!result.payload?.cancelled && result.payload?.guid) {
    await tplStore.reload()
    if (!selected.value.includes(result.payload.guid)) {
      emit('update:modelValue', [...selected.value, result.payload.guid])
    }
  }
}

async function loadThumb(summary: { guid: string; firstBlobSha?: string }) {
  if (thumbCache.value[summary.guid]) return
  if (!summary.firstBlobSha) return
  const r = await tplStore.readBlobDataURL(summary.firstBlobSha)
  if (typeof r === 'string') thumbCache.value[summary.guid] = r
}

// 打开面板时把可见列表里的缩略图都拉一下
watch(
  [open, filtered],
  () => {
    if (!open.value) return
    for (const s of filtered.value) {
      if (!thumbCache.value[s.guid]) void loadThumb(s)
    }
  },
  { immediate: true },
)

// 选中模板的缩略图：触发按钮/chips 也要显示
watch(
  selected,
  (guids) => {
    for (const guid of guids) {
      if (guid && !thumbCache.value[guid]) {
        const s = tplStore.map[guid]
        if (s) void loadThumb(s)
      }
    }
  },
  { immediate: true },
)
</script>
