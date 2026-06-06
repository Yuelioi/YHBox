<template>
  <!-- 哑工具条：渲染分组图标按钮 + 单击跑 + 图钉置顶 + 隐藏。编排在 设置→悬浮窗。 -->
  <HudShell dense close-title="隐藏（可热键再呼出）" @close="onHide">
    <template #actions>
      <UButton
        size="xs" variant="ghost" :color="pinned ? 'primary' : 'neutral'"
        :icon="pinned ? 'i-tabler-pin-filled' : 'i-tabler-pin'"
        :title="pinned ? '已置顶 · 点击取消' : '点击置顶'"
        @click="togglePin"
      />
    </template>

    <div class="flex-1 min-h-0 overflow-auto p-1.5 space-y-2">
      <div
        v-if="groups.length === 0"
        class="h-full flex items-center justify-center text-center text-[11px] text-dimmed px-3"
      >
        还没配置 — 到 主程序 设置 → 悬浮窗 里添加
      </div>
      <section v-for="g in groups" :key="g.id" class="space-y-1">
        <h4 v-if="g.name" class="text-[10px] uppercase tracking-wider text-dimmed px-0.5 truncate">{{ g.name }}</h4>
        <div class="grid gap-1" :style="{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }">
          <button
            v-for="(it, i) in g.items"
            :key="it.containerId + ':' + i"
            type="button"
            class="flex flex-col items-center gap-0.5 px-1 py-1.5 rounded-md border border-default/50 bg-elevated/40 hover:bg-elevated transition-colors disabled:opacity-70"
            :class="{ 'ring-1 ring-primary': isRunning(it.containerId) }"
            :title="it.name"
            :disabled="isRunning(it.containerId)"
            @click="onRun(it.containerId)"
          >
            <UIcon
              :name="isRunning(it.containerId) ? 'i-tabler-loader-2' : it.icon || 'i-tabler-square-rounded'"
              class="size-5"
              :class="isRunning(it.containerId) ? 'animate-spin text-primary' : 'text-toned'"
            />
            <span class="w-full text-[10px] leading-none text-center truncate text-highlighted">{{ it.name }}</span>
          </button>
        </div>
      </section>
    </div>
  </HudShell>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend } from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'
import { useContainersStore } from '@/stores/containers'
import { useExecutionStore } from '@/stores/execution'
import HudShell from '@/components/tools/HudShell.vue'

const settingsStore = useSettingsStore()
const containersStore = useContainersStore()
const execStore = useExecutionStore()

const pinned = ref(true) // 开窗即置顶（后端 AlwaysOnTop:true）
function togglePin() {
  pinned.value = !pinned.value
  void backend.tools.setLauncherAlwaysOnTop(pinned.value)
}

const DEFAULT_COLUMNS = 3
const columns = computed(() => {
  const n = settingsStore.data?.ui.launcherColumns ?? 0
  return n > 0 ? n : DEFAULT_COLUMNS
})

interface RItem { containerId: string; icon: string; name: string }
interface RGroup { id: string; name: string; items: RItem[] }
const groups = computed<RGroup[]>(() => {
  const raw = settingsStore.data?.ui.launcherGroups ?? []
  return raw
    .map((g) => ({
      id: g.id,
      name: g.name,
      items: g.items
        .map((it) => {
          const c = containersStore.list.find((x) => x.id === it.containerId)
          return c ? { containerId: it.containerId, icon: it.icon, name: c.name } : null
        })
        .filter((x): x is RItem => x !== null), // 过滤已删容器
    }))
    .filter((g) => g.items.length > 0) // 空组不渲染
})
const flat = computed<string[]>(() => groups.value.flatMap((g) => g.items.map((i) => i.containerId)))

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
// 数字键 1-9：跑扁平第 N 个（窗口聚焦时）
function onKeyDown(e: KeyboardEvent) {
  if (e.key < '1' || e.key > '9') return
  const id = flat.value[Number(e.key) - 1]
  if (!id) return
  e.preventDefault()
  void onRun(id)
}

let offSettings: (() => void) | null = null
onMounted(() => {
  void settingsStore.load()
  void containersStore.reload()
  // 设置在主程序窗口改 → 后端 emit settings:changed → 本独立窗口 reload（修 icon 不反应）
  offSettings = Events.On('settings:changed', () => void settingsStore.load()) as unknown as () => void
  window.addEventListener('keydown', onKeyDown)
})
onUnmounted(() => {
  offSettings?.()
  window.removeEventListener('keydown', onKeyDown)
})
</script>
