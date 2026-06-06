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
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { backend } from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'
import { useContainersStore } from '@/stores/containers'
import { useExecutionStore } from '@/stores/execution'
import { useHotkeysStore } from '@/stores/hotkeys'

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
