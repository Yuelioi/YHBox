<template>
  <!-- 哑工具条：渲染分组按钮 + 单击跑 + 图钉置顶 + 隐藏。自适应高度 + 右下角拖拽改尺寸。编排在 设置→悬浮窗。 -->
  <HudShell dense close-title="隐藏（可热键再呼出）" @close="onHide">
    <template #actions>
      <UButton
        size="xs" variant="ghost" :color="pinned ? 'primary' : 'neutral'"
        :icon="pinned ? 'i-tabler-pin-filled' : 'i-tabler-pin'"
        :title="pinned ? '已置顶 · 点击取消' : '点击置顶'"
        @click="togglePin"
      />
    </template>

    <div ref="contentRef" class="flex-1 min-h-0 overflow-auto p-1.5 space-y-2">
      <div
        v-if="groups.length === 0"
        class="h-full flex items-center justify-center text-center text-[11px] text-dimmed px-3 py-4"
      >
        还没配置 — 到 主程序 设置 → 悬浮窗 里添加
      </div>
      <section v-for="g in groups" :key="g.id" class="space-y-1">
        <h4 v-if="g.name" class="text-[10px] uppercase tracking-wider text-dimmed px-0.5 truncate">{{ g.name }}</h4>
        <div class="grid gap-1 justify-start" :style="{ gridTemplateColumns: `repeat(auto-fill, ${colW}px)` }">
          <button
            v-for="(it, i) in g.items"
            :key="it.containerId + ':' + i"
            type="button"
            class="flex flex-col items-center justify-center gap-0.5 px-1 py-1.5 rounded-md border border-default/50 bg-elevated/40 hover:bg-elevated transition-colors disabled:opacity-70"
            :class="{ 'ring-1 ring-primary': isRunning(it.containerId) }"
            :title="it.label"
            :disabled="isRunning(it.containerId)"
            @click="onRun(it.containerId)"
          >
            <UIcon
              v-if="display !== 'text' || isRunning(it.containerId)"
              :name="isRunning(it.containerId) ? 'i-tabler-loader-2' : it.icon || 'i-tabler-square-rounded'"
              :class="[isRunning(it.containerId) ? 'animate-spin text-primary' : 'text-toned', display === 'icon' ? 'size-6' : 'size-5']"
            />
            <span
              v-if="display !== 'icon'"
              class="w-full text-[10px] leading-none text-center truncate text-highlighted"
            >{{ it.label }}</span>
          </button>
        </div>
      </section>
    </div>

    <!-- 右下角拖拽手柄：改窗口宽高 -->
    <div
      class="absolute bottom-0 right-0 size-3.5 cursor-nwse-resize text-dimmed/70 hover:text-toned"
      style="--wails-draggable: no-drag"
      title="拖拽改大小"
      @pointerdown="onGripDown"
    >
      <svg viewBox="0 0 10 10" class="size-full"><path d="M9 1v8H1" fill="none" stroke="currentColor" stroke-width="1" opacity="0.5"/><path d="M9 5v4H5" fill="none" stroke="currentColor" stroke-width="1.2"/></svg>
    </div>
  </HudShell>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend } from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'
import { useContainersStore } from '@/stores/containers'
import { useExecutionStore } from '@/stores/execution'
import HudShell from '@/components/tools/HudShell.vue'

const settingsStore = useSettingsStore()
const containersStore = useContainersStore()
const execStore = useExecutionStore()

const contentRef = ref<HTMLElement | null>(null)

const pinned = ref(true)
function togglePin() {
  pinned.value = !pinned.value
  void backend.tools.setLauncherAlwaysOnTop(pinned.value)
}

const display = computed(() => settingsStore.data?.ui.launcherDisplay || 'both') // both | icon | text
// 按钮固定紧凑宽度（auto-fill 自动换行，窗口越宽每排塞越多，按钮不被拉伸）。各模式宽度不同：纯图标方块最窄，带文字的要给标签留位。
const COL_W: Record<string, number> = { icon: 48, both: 80, text: 100 }
const colW = computed(() => COL_W[display.value] ?? 80)

interface RItem { containerId: string; icon: string; label: string }
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
          return c ? { containerId: it.containerId, icon: it.icon, label: it.label || c.name } : null
        })
        .filter((x): x is RItem => x !== null),
    }))
    .filter((g) => g.items.length > 0)
})
const flat = computed<string[]>(() => groups.value.flatMap((g) => g.items.map((i) => i.containerId)))

function isRunning(id: string): boolean {
  return execStore.running && execStore.currentTargetID === id
}
async function onRun(id: string) {
  if (isRunning(id)) return
  await backend.containers.run(id)
}
function onHide() {
  void backend.tools.hideLauncher()
}
function onKeyDown(e: KeyboardEvent) {
  if (e.key < '1' || e.key > '9') return
  const id = flat.value[Number(e.key) - 1]
  if (!id) return
  e.preventDefault()
  void onRun(id)
}

// ── 自适应高度：内容变化 → 量内容高 + chrome → SetSize（保持当前宽度）──
const CHROME_H = 34 // 标题栏 + 边框估算
function fitHeight() {
  const el = contentRef.value
  if (!el) return
  const h = Math.min(900, Math.max(56, Math.ceil(el.scrollHeight) + CHROME_H))
  void backend.tools.setLauncherSize(Math.round(window.innerWidth), h)
}
watch([groups, display], () => void nextTick(fitHeight))

// ── 右下角手柄拖拽改宽高 ──
let startX = 0
let startY = 0
let startW = 0
let startH = 0
function onGripMove(e: PointerEvent) {
  const w = Math.max(140, startW + (e.clientX - startX))
  const h = Math.max(56, startH + (e.clientY - startY))
  void backend.tools.setLauncherSize(Math.round(w), Math.round(h))
}
function onGripUp() {
  window.removeEventListener('pointermove', onGripMove)
  window.removeEventListener('pointerup', onGripUp)
}
function onGripDown(e: PointerEvent) {
  e.preventDefault()
  startX = e.clientX
  startY = e.clientY
  startW = window.innerWidth
  startH = window.innerHeight
  window.addEventListener('pointermove', onGripMove)
  window.addEventListener('pointerup', onGripUp)
}

let offSettings: (() => void) | null = null
onMounted(() => {
  void settingsStore.load().then(() => nextTick(fitHeight))
  void containersStore.reload().then(() => nextTick(fitHeight))
  offSettings = Events.On('settings:changed', () =>
    void settingsStore.load().then(() => nextTick(fitHeight)),
  ) as unknown as () => void
  window.addEventListener('keydown', onKeyDown)
})
onUnmounted(() => {
  offSettings?.()
  window.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('pointermove', onGripMove)
  window.removeEventListener('pointerup', onGripUp)
})
</script>
