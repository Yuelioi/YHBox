<template>
  <!-- 哑工具条：渲染块序列 (容器按钮 / 文字标题 / 水平·垂直分隔符) + 单击跑 + 图钉置顶 + 隐藏。
       自适应高度 + 右下角拖拽改尺寸。编排在 设置→悬浮窗。 -->
  <HudShell dense icon="i-tabler-rocket" :title="t('floatingLauncher.title')" :close-title="t('floatingLauncher.hide')" @close="onHide">
    <template #actions>
      <UButton
        size="xs" variant="ghost" :color="pinned ? 'primary' : 'neutral'"
        :icon="pinned ? 'i-tabler-pin-filled' : 'i-tabler-pin'"
        :title="pinned ? t('floatingLauncher.unpin') : t('floatingLauncher.pin')"
        @click="togglePin"
      />
    </template>

    <div ref="contentRef" class="flex-1 min-h-0 overflow-auto p-1.5">
      <div
        v-if="blocks.length === 0"
        class="h-full flex items-center justify-center text-center text-[11px] text-dimmed px-3 py-4"
      >
        {{ t('floatingLauncher.empty') }}
      </div>
      <div v-else class="flex flex-wrap items-stretch gap-1">
        <template v-for="b in blocks" :key="b.id">
          <!-- 容器按钮 -->
          <button
            v-if="b.type === 'container'"
            type="button"
            class="flex flex-col items-center justify-center gap-0.5 px-1 py-1.5 rounded-md border border-default/50 bg-elevated/40 hover:bg-elevated transition-colors disabled:opacity-70 shrink-0"
            :class="{ 'ring-1 ring-primary': isRunning(b.containerId!) }"
            :style="{ width: colW + 'px' }"
            :title="b.label"
            :disabled="isRunning(b.containerId!)"
            @click="onRun(b.containerId!)"
          >
            <UIcon
              v-if="display !== 'text' || isRunning(b.containerId!)"
              :name="isRunning(b.containerId!) ? 'i-tabler-loader-2' : b.icon || 'i-tabler-square-rounded'"
              :class="[isRunning(b.containerId!) ? 'animate-spin text-primary' : 'text-toned', display === 'icon' ? 'size-6' : 'size-5']"
            />
            <span
              v-if="display !== 'icon'"
              class="w-full text-[10px] leading-none text-center truncate text-highlighted"
            >{{ b.label }}</span>
          </button>
          <!-- 文字标题：占整行 -->
          <div
            v-else-if="b.type === 'label'"
            class="basis-full text-[10px] uppercase tracking-wider text-dimmed px-0.5 pt-1 pb-0.5 truncate"
          >{{ b.label }}</div>
          <!-- 水平分隔符：占整行的横线 (把后面挤到下一排) -->
          <div v-else-if="b.type === 'hsep'" class="basis-full border-t border-default/60 my-0.5" />
          <!-- 垂直分隔符：同排按钮之间的竖线 -->
          <div v-else-if="b.type === 'vsep'" class="self-stretch border-l border-default/60 mx-0.5" />
        </template>
      </div>
    </div>

    <!-- 右下角拖拽手柄：改窗口宽高 -->
    <div
      class="absolute bottom-0 right-0 size-3.5 cursor-nwse-resize text-dimmed/70 hover:text-toned"
      style="--wails-draggable: no-drag"
      :title="t('floatingLauncher.resize')"
      @pointerdown="onGripDown"
    >
      <svg viewBox="0 0 10 10" class="size-full"><path d="M9 1v8H1" fill="none" stroke="currentColor" stroke-width="1" opacity="0.5"/><path d="M9 5v4H5" fill="none" stroke="currentColor" stroke-width="1.2"/></svg>
    </div>
  </HudShell>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Events } from '@wailsio/runtime'
import { backend } from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'
import { useContainersStore } from '@/stores/containers'
import { useExecutionStore } from '@/stores/execution'
import HudShell from '@/components/tools/HudShell.vue'

const settingsStore = useSettingsStore()
const containersStore = useContainersStore()
const execStore = useExecutionStore()
const { t } = useI18n()

const contentRef = ref<HTMLElement | null>(null)

const pinned = ref(true)
function togglePin() {
  pinned.value = !pinned.value
  void backend.tools.setLauncherAlwaysOnTop(pinned.value)
}

const display = computed(() => settingsStore.data?.ui.launcherDisplay || 'both') // both | icon | text
// 按钮固定紧凑宽度（flex-wrap 自动换行）。各模式宽度不同：纯图标方块最窄，带文字的要给标签留位。
const COL_W: Record<string, number> = { icon: 48, both: 80, text: 100 }
const colW = computed(() => COL_W[display.value] ?? 80)

// 渲染块：container 解析容器名/图标（容器没了则跳过该块）；label/hsep/vsep 原样。
interface RBlock { id: string; type: 'container' | 'label' | 'hsep' | 'vsep'; containerId?: string; icon?: string; label?: string }
const blocks = computed<RBlock[]>(() => {
  const raw = settingsStore.data?.ui.launcherItems ?? []
  const out: RBlock[] = []
  for (const b of raw) {
    if (b.type === 'container') {
      const c = containersStore.list.find((x) => x.id === b.containerId)
      if (!c) continue
      out.push({ id: b.id, type: 'container', containerId: b.containerId, icon: b.icon, label: b.label || c.name })
    } else {
      out.push({ id: b.id, type: b.type, label: b.label })
    }
  }
  return out
})
// 数字热键 1-9 跑第 N 个容器块（按出现顺序）。
const flat = computed<string[]>(() =>
  blocks.value.filter((b) => b.type === 'container').map((b) => b.containerId!),
)

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
watch([blocks, display], () => void nextTick(fitHeight))

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
