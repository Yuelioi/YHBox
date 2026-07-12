<template>
  <div class="px-8 py-6 space-y-6">
    <!-- 搜索框 + 批量操作 -->
    <div class="flex items-center gap-2">
      <UInput
        v-model="searchText"
        :placeholder="t('hotkeys.search_placeholder')"
        icon="i-tabler-search"
        class="flex-1"
      />
      <UButton color="neutral" variant="outline" icon="i-tabler-restore" @click="onResetSystem">
        {{ t('hotkeys.reset_system') }}
      </UButton>
      <UButton color="neutral" variant="outline" icon="i-tabler-eraser" @click="onClearContainers">
        {{ t('hotkeys.clear_containers') }}
      </UButton>
    </div>

    <!-- 按 source 分组渲染 -->
    <section
      v-for="group in filteredGrouped"
      :key="group.source"
      class="rounded-xl bg-default border border-default p-5 space-y-3"
    >
      <div class="flex items-center gap-2">
        <UIcon :name="groupIcon(group.source)" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">{{ groupLabel(group.source) }}</h2>
      </div>
      <div class="space-y-2">
        <div
          v-for="entry in group.entries"
          :key="entry.key"
          class="flex items-center gap-3 px-3 py-2 rounded-md bg-elevated/30 border border-default/60"
        >
          <div class="flex-1 min-w-0">
            <div class="text-sm text-default truncate">
              {{ t(entry.label, entry.labelParams ?? {}) }}
            </div>
            <div
              v-if="entry.lastError"
              class="text-xs text-error mt-0.5 truncate"
              :title="entry.lastError"
            >
              ⚠ {{ t('hotkeys.status.register_failed') }}: {{ entry.lastError }}
            </div>
            <div v-else-if="entry.readonlyReason" class="text-xs text-dimmed mt-0.5">
              {{ entry.readonlyReason }}
            </div>
            <div v-else-if="entry.status === 'unbound'" class="text-xs text-dimmed mt-0.5">
              {{ t('hotkeys.status.unbound') }}
            </div>
          </div>
          <div class="w-56 shrink-0">
            <HotkeyCaptureInput
              :model-value="entry.hotkeyStr"
              :disabled="!!entry.readonlyReason"
              @update:model-value="(v: string) => onUpdate(entry.key, v)"
            />
          </div>
        </div>
      </div>
    </section>

    <p v-if="filteredGrouped.length === 0" class="text-xs text-dimmed text-center py-8">
      {{ t('hotkeys.empty') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useHotkeysStore } from '@/stores/hotkeys'
import { useConfirm } from '@/composables/useConfirm'
import { backend } from '@/lib/backend'
import HotkeyCaptureInput from '@/components/hotkeys/HotkeyCaptureInput.vue'

const { t } = useI18n()
const store = useHotkeysStore()
const toast = useToast()
const { confirm } = useConfirm()
const searchText = ref('')

onMounted(() => {
  void store.reload()
})

const filteredGrouped = computed(() => {
  const q = searchText.value.trim().toLowerCase()
  const filtered = q
    ? store.list.filter((e) => {
        const label = t(e.label, e.labelParams ?? {})
        return label.toLowerCase().includes(q) || e.hotkeyStr.toLowerCase().includes(q)
      })
    : store.list
  // 顺序: system → action → container → schedule (key 顺序即 UI 顺序)
  const groups: Record<string, typeof filtered> = {
    system: [],
    recording: [],
    action: [],
    container: [],
    schedule: [],
    editor: [],
  }
  for (const e of filtered) {
    if (groups[e.source]) groups[e.source].push(e)
  }
  for (const k of Object.keys(groups)) {
    groups[k].sort((a, b) =>
      t(a.label, a.labelParams ?? {}).localeCompare(t(b.label, b.labelParams ?? {}), 'zh'),
    )
  }
  return Object.entries(groups)
    .filter(([_, entries]) => entries.length > 0)
    .map(([source, entries]) => ({ source, entries }))
})

function groupIcon(source: string): string {
  switch (source) {
    case 'system':
      return 'i-tabler-tool'
    case 'recording':
      return 'i-tabler-player-record'
    case 'action':
      return 'i-tabler-bolt'
    case 'container':
      return 'i-tabler-box'
    case 'schedule':
      return 'i-tabler-calendar-clock'
    case 'editor':
      return 'i-tabler-edit'
    default:
      return 'i-tabler-keyboard'
  }
}
function groupLabel(source: string): string {
  switch (source) {
    case 'system':
      return t('hotkeys.group.system')
    case 'recording':
      return t('hotkeys.group.recording')
    case 'action':
      return t('hotkeys.group.action')
    case 'container':
      return t('hotkeys.group.container')
    case 'schedule':
      return t('hotkeys.group.schedule')
    case 'editor':
      return t('hotkeys.group.editor')
    default:
      return source
  }
}

async function onUpdate(key: string, hotkeyStr: string) {
  const ok = await store.update(key, hotkeyStr)
  if (ok) {
    toast.add({
      title: hotkeyStr ? t('hotkeys.toast.bound', { hk: hotkeyStr }) : t('hotkeys.toast.cleared'),
      icon: 'i-tabler-check',
      color: 'neutral',
    })
  }
  // 失败 invoke wrapper 已 toast 带 [conflict]/[reserved]/[invalid] 前缀，不再重复
}

// 重置内置热键 (强停/校准/录制停止/录制暂停) 为出厂默认。容器热键不动。
async function onResetSystem() {
  const ok = await confirm({
    title: t('hotkeys.confirm.reset_title'),
    description: t('hotkeys.confirm.reset_desc'),
    confirmText: t('hotkeys.confirm.reset_ok'),
    cancelText: t('common.cancel'),
    color: 'warning',
  })
  if (ok !== true) return
  await backend.hotkeys.resetSystemDefaults()
  await store.reload()
  toast.add({ title: t('hotkeys.toast.reset_done'), icon: 'i-tabler-check', color: 'neutral' })
}

// 清空所有容器的热键绑定 (容器/蓝图保留)。
async function onClearContainers() {
  const ok = await confirm({
    title: t('hotkeys.confirm.clear_title'),
    description: t('hotkeys.confirm.clear_desc'),
    confirmText: t('hotkeys.confirm.clear_ok'),
    cancelText: t('common.cancel'),
    color: 'error',
  })
  if (ok !== true) return
  const n = await backend.containers.clearAllHotkeys()
  await store.reload()
  if (typeof n === 'number' && n > 0) {
    toast.add({
      title: t('hotkeys.toast.containers_cleared', { n }),
      icon: 'i-tabler-check',
      color: 'neutral',
    })
  } else {
    toast.add({
      title: t('hotkeys.toast.containers_none'),
      icon: 'i-tabler-info-circle',
      color: 'neutral',
    })
  }
}
</script>
