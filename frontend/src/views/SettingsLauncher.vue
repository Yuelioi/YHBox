<template>
  <div class="px-8 py-6 space-y-6">
    <!-- 启动器总说明 + 全局选项 -->
    <section class="rounded-xl bg-default border border-default p-5 space-y-4">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-layout-grid-add" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">悬浮窗启动器</h2>
      </div>
      <p class="text-xs text-dimmed leading-relaxed">
        把常用容器编进分组，悬浮窗里单击即跑。用「呼出/隐藏」热键（在 快捷键 页绑）或容器页「悬浮启动器」按钮打开。
      </p>

      <div class="border-t border-default/60" />

      <div class="flex items-center justify-between gap-6">
        <div>
          <div class="text-sm text-default">按钮显示</div>
          <p class="text-xs text-dimmed mt-0.5">悬浮窗里每个按钮显示图标、文字，或两者都显示。</p>
        </div>
        <USelect :model-value="display" :items="displayItems" class="w-32" @update:model-value="(v: string) => setDisplay(v)" />
      </div>
    </section>

    <!-- 分组卡片：每个分组一张 -->
    <section
      v-for="g in editGroups"
      :key="g.id"
      class="rounded-xl bg-default border border-default p-5 space-y-3"
    >
      <div class="flex items-center gap-2">
        <UInput
          :model-value="g.name" class="flex-1 min-w-0" placeholder="分组名（如 战斗）"
          @update:model-value="(v: string | number) => renameGroup(g.id, String(v))"
        />
        <UButton size="xs" variant="ghost" color="error" icon="i-tabler-trash" title="删除分组（不影响容器）" @click="deleteGroup(g.id)" />
      </div>

      <div v-if="g.items.length === 0" class="text-xs text-dimmed px-1 py-1">空分组 — 下面添加容器。</div>
      <VueDraggable v-else v-model="g.items" :animation="150" handle=".drag-h" class="space-y-2" @end="persist">
        <div
          v-for="it in g.items"
          :key="it.containerId"
          class="flex flex-col gap-1 px-3 py-2 rounded-md bg-elevated/30 border border-default/60"
        >
          <div class="flex items-center gap-2">
            <UIcon name="i-tabler-grip-vertical" class="drag-h size-4 text-dimmed cursor-grab shrink-0" />
            <UPopover :ui="{ content: 'w-[300px] p-2' }">
              <UButton size="xs" variant="outline" color="neutral" square class="shrink-0" title="选图标">
                <UIcon :name="it.icon || 'i-tabler-photo-plus'" class="size-4" :class="it.icon ? 'text-toned' : 'text-dimmed'" />
              </UButton>
              <template #content>
                <div class="space-y-2">
                  <IconPicker :model-value="it.icon" @update:model-value="(v: string) => setIcon(g.id, it.containerId, v)" />
                  <UButton v-if="it.icon" size="xs" variant="ghost" color="neutral" block @click="setIcon(g.id, it.containerId, '')">
                    清除图标
                  </UButton>
                </div>
              </template>
            </UPopover>
            <UInput
              :model-value="it.label" size="sm" class="flex-1 min-w-0"
              :placeholder="containerName(it.containerId)" :title="containerName(it.containerId)"
              @update:model-value="(v: string | number) => setLabel(g.id, it.containerId, String(v))"
            />
            <HotkeyCaptureInput
              class="w-28 sm:w-32 shrink-0" :model-value="containerHotkey(it.containerId)"
              @update:model-value="(v: string) => setHotkey(it.containerId, v)"
            />
            <UButton size="xs" variant="ghost" color="neutral" icon="i-tabler-minus" title="移出分组" @click="removeItem(g.id, it.containerId)" />
          </div>
          <!-- 自定义了显示名时 placeholder 消失，常驻一行原容器名兜底，避免忘了它指向哪个容器 -->
          <p v-if="it.label" class="pl-7 text-xs text-dimmed truncate">来自容器：{{ containerName(it.containerId) }}</p>
        </div>
      </VueDraggable>

      <USelect
        v-if="addableFor(g).length"
        :model-value="undefined" :items="addableFor(g)" size="sm" class="w-full sm:w-56"
        placeholder="+ 添加容器"
        @update:model-value="(v: string) => addItem(g.id, v)"
      />
    </section>

    <div
      v-if="editGroups.length === 0"
      class="text-xs text-dimmed py-8 text-center border border-dashed border-default/60 rounded-xl"
    >
      还没有分组，点下面「新建分组」开始。
    </div>

    <UButton variant="soft" color="primary" icon="i-tabler-plus" @click="addGroup">
      新建分组
    </UButton>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { backend } from '@/lib/backend'
import { useSettingsStore, type LauncherGroup } from '@/stores/settings'
import { useContainersStore } from '@/stores/containers'
import { useHotkeysStore } from '@/stores/hotkeys'
import { VueDraggable } from 'vue-draggable-plus'
import IconPicker from '@/components/containers/inline/IconPicker.vue'
import HotkeyCaptureInput from '@/components/hotkeys/HotkeyCaptureInput.vue'

const settingsStore = useSettingsStore()
const containersStore = useContainersStore()
const hotkeysStore = useHotkeysStore()

// 本地工作副本（深拷贝），本页是 launcherGroups 唯一编辑者，每次改动立即 persist。
const editGroups = ref<LauncherGroup[]>([])
function copyGroups(gs: LauncherGroup[]): LauncherGroup[] {
  return gs.map((g) => ({ id: g.id, name: g.name, items: g.items.map((it) => ({ ...it })) }))
}
function syncFromStore() {
  editGroups.value = copyGroups(settingsStore.data?.ui.launcherGroups ?? [])
}
watch(() => settingsStore.data?.ui.launcherGroups, syncFromStore, { immediate: true })

const display = computed(() => settingsStore.data?.ui.launcherDisplay || 'both')
const displayItems = [
  { label: '图标 + 文字', value: 'both' },
  { label: '仅图标', value: 'icon' },
  { label: '仅文字', value: 'text' },
]
function setDisplay(v: string) {
  void settingsStore.patch({ ui: { launcherDisplay: v } })
}

function persist() {
  void settingsStore.patch({ ui: { launcherGroups: copyGroups(editGroups.value) } })
}
function genId(): string {
  return 'lg_' + Math.random().toString(36).slice(2, 10)
}
function group(id: string) {
  return editGroups.value.find((g) => g.id === id)
}
function addGroup() {
  editGroups.value.push({ id: genId(), name: `分组 ${editGroups.value.length + 1}`, items: [] })
  persist()
}
function renameGroup(id: string, name: string) {
  const g = group(id)
  if (g) { g.name = name; persist() }
}
function deleteGroup(id: string) {
  editGroups.value = editGroups.value.filter((g) => g.id !== id)
  persist()
}
function addItem(gid: string, cid: string) {
  const g = group(gid)
  if (g && cid && !g.items.some((it) => it.containerId === cid)) {
    g.items.push({ containerId: cid, icon: '', label: '' })
    persist()
  }
}
function removeItem(gid: string, cid: string) {
  const g = group(gid)
  if (g) { g.items = g.items.filter((it) => it.containerId !== cid); persist() }
}
function setIcon(gid: string, cid: string, icon: string) {
  const it = group(gid)?.items.find((x) => x.containerId === cid)
  if (it) { it.icon = icon; persist() }
}
function setLabel(gid: string, cid: string, label: string) {
  const it = group(gid)?.items.find((x) => x.containerId === cid)
  if (it) { it.label = label; persist() }
}
async function setHotkey(cid: string, hk: string) {
  await backend.hotkeys.update('container.' + cid, hk)
  await hotkeysStore.reload()
}
function containerName(id: string): string {
  return containersStore.list.find((c) => c.id === id)?.name ?? '(已删容器)'
}
function containerHotkey(id: string): string {
  return hotkeysStore.list.find((e) => e.key === 'container.' + id)?.hotkeyStr ?? ''
}
// 同组不可重复加；跨组允许。
function addableFor(g: LauncherGroup): { label: string; value: string }[] {
  const have = new Set(g.items.map((it) => it.containerId))
  return containersStore.list.filter((c) => !have.has(c.id)).map((c) => ({ label: c.name, value: c.id }))
}

onMounted(() => {
  void settingsStore.load()
  void containersStore.reload()
  void hotkeysStore.reload()
})
</script>
