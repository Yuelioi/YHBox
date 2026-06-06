<template>
  <!-- 哑悬浮窗：只渲染按钮 + 单击跑 + 置顶 + 拖动 + 隐藏。编排在 主程序 设置→悬浮窗。 -->
  <div class="h-screen flex flex-col bg-default text-default select-none">
    <header
      class="shrink-0 flex items-center gap-1 px-1.5 py-1 border-b border-default"
      style="--wails-draggable: drag"
    >
      <UIcon name="i-tabler-grip-horizontal" class="size-3 text-dimmed" />
      <span class="ml-auto" />
      <UButton
        size="xs" variant="ghost" color="neutral" icon="i-tabler-x"
        style="--wails-draggable: no-drag" title="隐藏（可热键再呼出）" @click="onHide"
      />
    </header>

    <div class="flex-1 min-h-0 overflow-auto p-1.5">
      <div v-if="items.length === 0" class="h-full flex items-center justify-center text-center text-[11px] text-dimmed px-3">
        还没配置 —— 到 主程序 设置 → 悬浮窗 里添加容器
      </div>
      <div
        v-else
        class="grid gap-1.5"
        :style="{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }"
      >
        <button
          v-for="(it, i) in items"
          :key="it.containerId + ':' + i"
          type="button"
          class="flex flex-col items-center justify-center gap-1 px-2 py-2 rounded-md border border-default/60 bg-elevated/40 hover:bg-elevated transition-colors disabled:opacity-70"
          :class="{ 'ring-1 ring-primary': isRunning(it.containerId) }"
          :title="it.name"
          :disabled="isRunning(it.containerId)"
          @click="onRun(it.containerId)"
        >
          <UIcon
            v-if="isRunning(it.containerId)"
            name="i-tabler-loader-2"
            class="size-5 animate-spin text-primary"
          />
          <UIcon v-else-if="it.icon" :name="it.icon" class="size-5 text-toned" />
          <span class="text-[11px] leading-tight text-center line-clamp-2 break-all text-highlighted">{{ it.name }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { backend } from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'
import { useContainersStore } from '@/stores/containers'
import { useExecutionStore } from '@/stores/execution'

const settingsStore = useSettingsStore()
const containersStore = useContainersStore()
const execStore = useExecutionStore()

const DEFAULT_COLUMNS = 3
const columns = computed(() => {
  const n = settingsStore.data?.ui.launcherColumns ?? 0
  return n > 0 ? n : DEFAULT_COLUMNS
})

interface RenderItem { containerId: string; icon: string; name: string }
const items = computed<RenderItem[]>(() => {
  const raw = settingsStore.data?.ui.launcherItems ?? []
  return raw
    .map((it) => {
      const c = containersStore.list.find((x) => x.id === it.containerId)
      return c ? { containerId: it.containerId, icon: it.icon, name: c.name } : null
    })
    .filter((x): x is RenderItem => x !== null) // 过滤已删容器
})

function isRunning(id: string): boolean {
  return execStore.running && execStore.currentTargetID === id
}
async function onRun(id: string) {
  if (isRunning(id)) return // 点正在跑的 = no-op
  await backend.containers.run(id)
}
function onHide() {
  void backend.tools.hideLauncher()
}

// 数字键 1-9：跑第 N 个（窗口聚焦时）。纯快捷，无角标以保持紧凑。
function onKeyDown(e: KeyboardEvent) {
  if (e.key < '1' || e.key > '9') return
  const it = items.value[Number(e.key) - 1]
  if (!it) return
  e.preventDefault()
  void onRun(it.containerId)
}

onMounted(() => {
  void settingsStore.load()
  void containersStore.reload()
  window.addEventListener('keydown', onKeyDown)
})
onUnmounted(() => window.removeEventListener('keydown', onKeyDown))
</script>
