<template>
  <div class="h-screen flex flex-col bg-default text-default select-none">
    <header
      class="shrink-0 flex items-center gap-2 px-3 py-2 border-b border-default"
      style="--wails-draggable: drag"
    >
      <UIcon name="i-tabler-layout-grid" class="size-3.5 text-primary" />
      <span class="text-xs font-medium text-highlighted">启动器</span>
      <span class="ml-auto" />
      <UButton
        size="xs" variant="ghost" :color="editing ? 'primary' : 'neutral'"
        :icon="editing ? 'i-tabler-check' : 'i-tabler-pencil'"
        style="--wails-draggable: no-drag" :title="editing ? '完成编辑' : '编辑分组'"
        @click="editing = !editing"
      />
      <UButton
        size="xs" variant="ghost" color="neutral" icon="i-tabler-x"
        style="--wails-draggable: no-drag" title="隐藏（不关闭，可热键再呼出）"
        @click="onHide"
      />
    </header>

    <div class="flex-1 min-h-0 overflow-y-auto p-2 space-y-3">
      <!-- 编辑态：建/删组、加容器、拖拽排序/移除、就地改容器热键 -->
      <template v-if="editing">
        <UButton
          size="xs" variant="soft" color="primary" icon="i-tabler-plus"
          class="w-full justify-center mb-2" @click="addGroup"
        >新建分组</UButton>
        <section v-for="g in editGroups" :key="g.id" class="space-y-1.5 rounded-md border border-default/60 p-2">
          <div class="flex items-center gap-1">
            <UInput
              :model-value="g.name" size="xs" class="flex-1" placeholder="分组名"
              @update:model-value="(v: string | number) => renameGroup(g.id, String(v))"
            />
            <UButton
              size="xs" variant="ghost" color="error" icon="i-tabler-trash"
              title="删除分组（不影响容器本身）" @click="deleteGroup(g.id)"
            />
          </div>
          <VueDraggable
            v-model="g.containerIds" :animation="150" handle=".drag-h"
            class="space-y-1" @end="onDragEnd"
          >
            <div
              v-for="cid in g.containerIds" :key="cid"
              class="flex items-center gap-1.5 px-1.5 py-1 rounded bg-elevated/40"
            >
              <UIcon name="i-tabler-grip-vertical" class="drag-h size-3.5 text-dimmed cursor-grab shrink-0" />
              <span class="flex-1 min-w-0 truncate text-xs text-highlighted">{{ containerName(cid) }}</span>
              <HotkeyCaptureInput
                class="w-32 shrink-0" :model-value="containerHotkey(cid)"
                @update:model-value="(v: string) => setContainerHotkey(cid, v)"
              />
              <UButton
                size="xs" variant="ghost" color="neutral" icon="i-tabler-x"
                title="移出分组" @click="removeFromGroup(g.id, cid)"
              />
            </div>
          </VueDraggable>
          <USelect
            :model-value="undefined" :items="addableFor(g)" size="xs"
            class="w-full" placeholder="+ 加容器"
            @update:model-value="(v: string) => addToGroup(g.id, v)"
          />
        </section>
      </template>

      <!-- 普通态：空态 + 分组运行按钮 -->
      <template v-else>
        <div v-if="groups.length === 0" class="h-full flex flex-col items-center justify-center gap-2 text-dimmed text-xs">
          <UIcon name="i-tabler-layout-grid-add" class="size-8" />
          <p>还没有分组</p>
          <UButton size="xs" variant="soft" color="primary" icon="i-tabler-plus" @click="editing = true">新建分组</UButton>
        </div>

        <section v-for="g in groups" :key="g.id" class="space-y-1">
          <h4 class="text-[10px] uppercase tracking-wider text-dimmed px-1">{{ g.name || '未命名分组' }}</h4>
          <button
            v-for="item in g.items"
            :key="g.id + ':' + item.id"
            type="button"
            class="w-full flex items-center gap-2 px-2 py-1.5 rounded-md border border-default/60 bg-elevated/40 hover:bg-elevated text-left transition-colors disabled:opacity-60"
            :disabled="isRunning(item.id)"
            @click="onRun(item.id)"
          >
            <span
              v-if="item.num"
              class="shrink-0 size-4 rounded text-[10px] font-mono flex items-center justify-center bg-default text-dimmed"
            >{{ item.num }}</span>
            <span class="flex-1 min-w-0 truncate text-xs text-highlighted">{{ item.name }}</span>
            <span v-if="isRunning(item.id)" class="shrink-0 inline-flex items-center gap-1 text-[10px] text-primary">
              <UIcon name="i-tabler-loader-2" class="size-3 animate-spin" /> 运行中
            </span>
            <span v-else-if="item.hotkey" class="shrink-0 text-[10px] font-mono text-dimmed">{{ item.hotkey }}</span>
          </button>
        </section>
      </template>
    </div>

    <footer class="shrink-0 flex items-center px-3 py-2 border-t border-default">
      <UButton
        size="xs" variant="soft" color="error" icon="i-tabler-player-stop-filled"
        class="w-full justify-center" :disabled="!execStore.running" @click="onStopAll"
      >全部停止</UButton>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { VueDraggable } from 'vue-draggable-plus'
import { backend } from '@/lib/backend'
import { useSettingsStore, type LauncherGroup } from '@/stores/settings'
import { useContainersStore } from '@/stores/containers'
import { useExecutionStore } from '@/stores/execution'
import { useHotkeysStore } from '@/stores/hotkeys'
import HotkeyCaptureInput from '@/components/hotkeys/HotkeyCaptureInput.vue'

const settingsStore = useSettingsStore()
const containersStore = useContainersStore()
const execStore = useExecutionStore()
const hotkeysStore = useHotkeysStore()

const editing = ref(false)

function containerName(id: string): string {
  return containersStore.list.find((c) => c.id === id)?.name ?? ''
}
function containerHotkey(id: string): string {
  return hotkeysStore.list.find((e) => e.key === 'container.' + id)?.hotkeyStr ?? ''
}
function containerExists(id: string): boolean {
  return containersStore.list.some((c) => c.id === id)
}

interface RenderItem { id: string; name: string; hotkey: string; num: number | null }
interface RenderGroup { id: string; name: string; items: RenderItem[] }
const groups = computed<RenderGroup[]>(() => {
  let flat = 0
  const raw = settingsStore.data?.ui.launcherGroups ?? []
  return raw.map((g) => ({
    id: g.id,
    name: g.name,
    items: g.containerIds
      .filter(containerExists)
      .map((id) => {
        flat += 1
        return { id, name: containerName(id), hotkey: containerHotkey(id), num: flat <= 9 ? flat : null }
      }),
  }))
})

// 扁平顺序（数字键用，Task 6）：跨组按显示序
const flatItems = computed<string[]>(() => groups.value.flatMap((g) => g.items.map((i) => i.id)))

// ── 编辑态 ──────────────────────────────────────────────
// source of truth = settings；editGroups 仅编辑用的工作副本，每次改动立即 persist。
const editGroups = ref<LauncherGroup[]>([])

function copyGroups(gs: LauncherGroup[]): LauncherGroup[] {
  return gs.map((g) => ({ id: g.id, name: g.name, containerIds: [...g.containerIds] }))
}
function persist() {
  void settingsStore.patch({ ui: { launcherGroups: copyGroups(editGroups.value) } })
}
// 进入编辑态：拷贝当前分组 + 清掉失效容器 id（唯一清理点），落盘一次。
watch(editing, (on) => {
  if (!on) return
  editGroups.value = copyGroups(settingsStore.data?.ui.launcherGroups ?? [])
    .map((g) => ({ ...g, containerIds: g.containerIds.filter(containerExists) }))
  persist()
})
function genGroupId(): string {
  return 'lg_' + Math.random().toString(36).slice(2, 10)
}
function addGroup() {
  editGroups.value.push({ id: genGroupId(), name: `分组 ${editGroups.value.length + 1}`, containerIds: [] })
  persist()
}
function renameGroup(id: string, name: string) {
  const g = editGroups.value.find((x) => x.id === id)
  if (g) { g.name = name; persist() }
}
function deleteGroup(id: string) {
  editGroups.value = editGroups.value.filter((g) => g.id !== id)
  persist()
}
function addToGroup(gid: string, cid: string) {
  const g = editGroups.value.find((x) => x.id === gid)
  if (g && !g.containerIds.includes(cid)) { g.containerIds.push(cid); persist() }
}
function removeFromGroup(gid: string, cid: string) {
  const g = editGroups.value.find((x) => x.id === gid)
  if (g) { g.containerIds = g.containerIds.filter((x) => x !== cid); persist() }
}
function onDragEnd() { persist() } // VueDraggable 已就地改 g.containerIds
async function setContainerHotkey(cid: string, hk: string) {
  await backend.hotkeys.update('container.' + cid, hk)
  await hotkeysStore.reload()
}
// 同组不可重复加；跨组允许（只过滤掉本组已有的）。
function addableFor(g: LauncherGroup): { label: string; value: string }[] {
  const inGroup = new Set(g.containerIds)
  return containersStore.list.filter((c) => !inGroup.has(c.id)).map((c) => ({ label: c.name, value: c.id }))
}

function isRunning(id: string): boolean {
  return execStore.running && execStore.currentTargetID === id
}
async function onRun(id: string) {
  if (isRunning(id)) return // 点正在跑的 = no-op
  await backend.containers.run(id)
}
async function onStopAll() {
  await backend.containers.stopAll()
}
function onHide() {
  void backend.tools.hideLauncher()
}

function isTypingTarget(): boolean {
  const el = document.activeElement as HTMLElement | null
  return !!el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable)
}
function onKeyDown(e: KeyboardEvent) {
  if (editing.value || isTypingTarget()) return // 编辑态 / 输入框聚焦时禁用
  if (e.key < '1' || e.key > '9') return
  const id = flatItems.value[Number(e.key) - 1]
  if (!id) return
  e.preventDefault()
  void onRun(id) // onRun 内对运行中已 no-op
}

onMounted(() => {
  void settingsStore.load()
  void containersStore.reload()
  void hotkeysStore.reload()
  window.addEventListener('keydown', onKeyDown)
})
onUnmounted(() => window.removeEventListener('keydown', onKeyDown))
</script>
