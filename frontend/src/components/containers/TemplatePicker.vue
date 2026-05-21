<template>
  <div class="space-y-1.5">
    <!-- 触发按钮：显示当前模板的"显示名 + 缩略图"，点击展开选择面板 -->
    <UPopover v-model:open="open" :ui="{ content: 'w-[360px] p-0' }">
      <UButton
        variant="outline"
        color="neutral"
        size="sm"
        class="w-full justify-start font-normal"
        :title="modelValue || '未选择'"
      >
        <div class="flex items-center gap-2 min-w-0 flex-1">
          <img
            v-if="foundThumb"
            :src="foundThumb"
            class="size-6 rounded object-contain bg-elevated shrink-0"
            alt=""
          />
          <UIcon
            v-else
            :name="foundMeta ? 'i-tabler-photo' : 'i-tabler-photo-question'"
            class="size-4 shrink-0"
            :class="foundMeta ? 'text-dimmed' : 'text-amber-400'"
          />
          <div class="min-w-0 flex-1 text-left">
            <div class="text-xs text-highlighted truncate">
              {{ foundMeta?.name || modelValue || '选择模板...' }}
            </div>
            <div v-if="foundMeta" class="text-[10px] text-dimmed font-mono truncate">
              {{ modelValue }}
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
              placeholder="搜索 key / 名称 / 描述..."
            />
            <UButton
              size="xs"
              variant="soft"
              color="primary"
              icon="i-tabler-camera"
              class="w-full justify-center"
              @click="onCaptureNew"
            >
              + 现截一张
            </UButton>
          </div>
          <div
            v-if="entries.length === 0"
            class="px-3 py-6 text-center text-[11px] text-amber-300/80"
          >
            <UIcon name="i-tabler-alert-triangle" class="size-4 mx-auto mb-1" />
            模板库为空，去 任务 → 模板 tab 先截图
          </div>
          <div
            v-else-if="filtered.length === 0"
            class="px-3 py-6 text-center text-[11px] text-dimmed"
          >
            没有匹配「{{ search }}」的模板
          </div>
          <ul v-else class="flex-1 overflow-y-auto py-1">
            <li v-for="[k, m] in filtered" :key="k">
              <button
                type="button"
                class="w-full px-2 py-1.5 flex items-center gap-2 hover:bg-elevated/60 transition-colors text-left"
                :class="k === modelValue ? 'bg-primary/10' : ''"
                @click="select(k)"
              >
                <img
                  v-if="thumbCache[k]"
                  :src="thumbCache[k]"
                  class="size-8 rounded object-contain bg-elevated shrink-0"
                />
                <div
                  v-else
                  class="size-8 rounded bg-elevated flex items-center justify-center shrink-0"
                >
                  <UIcon name="i-tabler-photo" class="size-3.5 text-dimmed" />
                </div>
                <div class="min-w-0 flex-1">
                  <div class="text-xs text-highlighted truncate">{{ m.name || k }}</div>
                  <div class="text-[10px] text-dimmed font-mono truncate">{{ k }}</div>
                  <div class="text-[10px] text-dimmed">
                    {{ m.width }}×{{ m.height }} · {{ m.recordedResolution[0] }}×{{
                      m.recordedResolution[1]
                    }}
                  </div>
                </div>
                <UIcon
                  v-if="k === modelValue"
                  name="i-tabler-check"
                  class="size-3.5 text-primary shrink-0"
                />
              </button>
            </li>
          </ul>
        </div>
      </template>
    </UPopover>

    <!-- 状态行 -->
    <p v-if="!modelValue" class="text-[10px] text-dimmed">未选择模板</p>
    <p v-else-if="!foundMeta" class="text-[10px] text-rose-300/80">
      <UIcon name="i-tabler-x" class="size-3 inline" />
      模板 "{{ modelValue }}" 不存在
    </p>
    <p v-else class="text-[10px] text-dimmed truncate">
      {{ foundMeta.width }}×{{ foundMeta.height }} · 录制于 {{ foundMeta.recordedResolution[0] }}×{{
        foundMeta.recordedResolution[1]
      }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useTemplatesStore } from '@/stores/templates'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import { backend } from '@/lib/backend'

const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const tplStore = useTemplatesStore()
const open = ref(false)
const search = ref('')
const thumbCache = ref<Record<string, string>>({})

onMounted(() => {
  if (Object.keys(tplStore.map).length === 0) void tplStore.reload()
})

const entries = computed(() => Object.entries(tplStore.map))

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return entries.value
  return entries.value.filter(
    ([k, m]) =>
      k.toLowerCase().includes(q) ||
      m.name?.toLowerCase().includes(q) ||
      m.description?.toLowerCase().includes(q),
  )
})

const foundMeta = computed(() => tplStore.map[props.modelValue])
const foundThumb = computed(() => thumbCache.value[props.modelValue])

function select(k: string) {
  emit('update:modelValue', k)
  open.value = false
}

// "+ 现截一张" → 打开 ScreenPicker(template_save)，保存成功后用新 key 回填本组件
async function onCaptureNew() {
  const id = 'tpl-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now()
  open.value = false // 关掉下拉，避免遮 picker 窗口
  const waiter = awaitWailsEvent<{ id: string; mode: string; payload: any }>(
    'tools:picker-result',
    (p) => p?.id === id,
  )
  await backend.tools.openScreenPicker('template_save', id, tplStore.containerId)
  const result = await waiter
  if (!result.payload?.cancelled && result.payload?.key) {
    await tplStore.reload()
    emit('update:modelValue', result.payload.key)
  }
}

async function loadThumb(key: string) {
  if (thumbCache.value[key]) return
  const r = await tplStore.readPng(key)
  if (typeof r === 'string') thumbCache.value[key] = r
}

// 打开面板时把可见列表里的缩略图都拉一下；模板很多时只拉 filtered（搜索过滤后）
watch(
  [open, filtered],
  () => {
    if (!open.value) return
    for (const [k] of filtered.value) {
      if (!thumbCache.value[k]) void loadThumb(k)
    }
  },
  { immediate: true },
)

// 选中模板的缩略图：触发按钮也要显示
watch(
  () => props.modelValue,
  (v) => {
    if (v) void loadThumb(v)
  },
  { immediate: true },
)
</script>
