<template>
  <div class="p-4 space-y-4 max-w-2xl">
    <div>
      <h3 class="text-sm font-medium text-highlighted mb-1">悬浮窗启动器</h3>
      <p class="text-xs text-dimmed">
        挑容器进悬浮窗，单击即跑。悬浮窗用「呼出/隐藏」热键（在 快捷键 页绑）或容器页「悬浮启动器」按钮打开。
      </p>
    </div>

    <div class="flex items-center gap-2">
      <label class="text-xs text-toned">每排按钮数</label>
      <UInputNumber
        :model-value="columns" :min="1" :max="12" size="sm" class="w-28"
        @update:model-value="setColumns"
      />
    </div>

    <div class="space-y-1.5">
      <label class="text-xs text-toned">按钮（拖动排序）</label>
      <div v-if="items.length === 0" class="text-xs text-dimmed py-2">还没添加容器。</div>
      <VueDraggable
        v-else v-model="items" :animation="150" handle=".drag-h" class="space-y-1" @end="persist"
      >
        <div
          v-for="(it, i) in items"
          :key="it.containerId + ':' + i"
          class="flex items-center gap-2 px-2 py-1.5 rounded-md border border-default/60 bg-elevated/40"
        >
          <UIcon name="i-tabler-grip-vertical" class="drag-h size-4 text-dimmed cursor-grab shrink-0" />
          <UPopover :ui="{ content: 'w-[280px] p-2' }">
            <UButton size="xs" variant="outline" color="neutral" class="shrink-0" title="选图标">
              <UIcon
                :name="it.icon || 'i-tabler-photo-plus'" class="size-4"
                :class="it.icon ? 'text-toned' : 'text-dimmed'"
              />
            </UButton>
            <template #content>
              <div class="space-y-2">
                <IconPicker :model-value="it.icon" @update:model-value="(v: string) => setIcon(i, v)" />
                <UButton v-if="it.icon" size="xs" variant="ghost" color="neutral" block @click="setIcon(i, '')">
                  清除图标
                </UButton>
              </div>
            </template>
          </UPopover>
          <span class="flex-1 min-w-0 truncate text-sm text-highlighted">{{ containerName(it.containerId) }}</span>
          <HotkeyCaptureInput
            class="w-32 shrink-0" :model-value="containerHotkey(it.containerId)"
            @update:model-value="(v: string) => setHotkey(it.containerId, v)"
          />
          <UButton size="xs" variant="ghost" color="error" icon="i-tabler-x" title="移除" @click="removeAt(i)" />
        </div>
      </VueDraggable>
    </div>

    <USelect
      v-if="addable.length"
      :model-value="undefined" :items="addable" size="sm" class="w-full sm:w-64"
      placeholder="+ 添加容器"
      @update:model-value="add"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { backend } from '@/lib/backend'
import { useSettingsStore, type LauncherItem } from '@/stores/settings'
import { useContainersStore } from '@/stores/containers'
import { useHotkeysStore } from '@/stores/hotkeys'
import { VueDraggable } from 'vue-draggable-plus'
import IconPicker from '@/components/containers/inline/IconPicker.vue'
import HotkeyCaptureInput from '@/components/hotkeys/HotkeyCaptureInput.vue'

const settingsStore = useSettingsStore()
const containersStore = useContainersStore()
const hotkeysStore = useHotkeysStore()

// 本地工作副本；本页是 launcherItems 的唯一编辑者，每次改动立即 persist。
const items = ref<LauncherItem[]>([])
function syncFromStore() {
  items.value = (settingsStore.data?.ui.launcherItems ?? []).map((it) => ({ ...it }))
}
watch(() => settingsStore.data?.ui.launcherItems, syncFromStore, { immediate: true })

const columns = computed(() => settingsStore.data?.ui.launcherColumns || 3)

function persist() {
  void settingsStore.patch({ ui: { launcherItems: items.value.map((it) => ({ ...it })) } })
}
function setColumns(v: number) {
  void settingsStore.patch({ ui: { launcherColumns: Math.max(1, Math.floor(v || 1)) } })
}
function containerName(id: string): string {
  return containersStore.list.find((c) => c.id === id)?.name ?? '(已删容器)'
}
function containerHotkey(id: string): string {
  return hotkeysStore.list.find((e) => e.key === 'container.' + id)?.hotkeyStr ?? ''
}
function setIcon(i: number, icon: string) {
  const it = items.value[i]
  if (it) { it.icon = icon; persist() }
}
function removeAt(i: number) {
  items.value.splice(i, 1)
  persist()
}
async function setHotkey(id: string, hk: string) {
  await backend.hotkeys.update('container.' + id, hk)
  await hotkeysStore.reload()
}
const addable = computed<{ label: string; value: string }[]>(() => {
  const have = new Set(items.value.map((it) => it.containerId))
  return containersStore.list.filter((c) => !have.has(c.id)).map((c) => ({ label: c.name, value: c.id }))
})
function add(cid: string) {
  if (!cid || items.value.some((it) => it.containerId === cid)) return
  items.value.push({ containerId: cid, icon: '' })
  persist()
}

onMounted(() => {
  void settingsStore.load()
  void containersStore.reload()
  void hotkeysStore.reload()
})
</script>
