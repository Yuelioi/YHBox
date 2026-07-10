<template>
  <div class="h-screen flex flex-col bg-default text-default text-xs select-none">
    <!-- 拖区 + 关闭 -->
    <header
      class="h-7 shrink-0 flex items-center px-2 border-b border-default bg-elevated/30"
      style="--wails-draggable: drag"
    >
      <UIcon name="i-tabler-pointer" class="size-3.5 text-primary" />
      <span class="ml-1.5 text-[11px] text-toned">鼠标 HUD</span>
      <span class="ml-auto" />
      <button
        type="button"
        class="size-5 flex items-center justify-center text-muted hover:bg-error hover:text-highlighted rounded transition-colors"
        style="--wails-draggable: no-drag"
        title="关闭"
        @click="onClose"
      >
        <UIcon name="i-tabler-x" class="size-3.5" />
      </button>
    </header>

    <div class="flex-1 p-3 font-mono tabular-nums space-y-1.5">
      <p v-if="toolError" class="text-[10px] text-error break-words">{{ toolError }}</p>
      <div class="flex items-center gap-2">
        <span class="text-dimmed w-14">屏幕</span>
        <span class="text-highlighted">{{ pos.screenX }}, {{ pos.screenY }}</span>
      </div>
      <div v-if="pos.hasGame" class="space-y-1.5">
        <div class="flex items-center gap-2">
          <span class="text-dimmed w-14">客户区</span>
          <span class="text-highlighted">{{ pos.clientX }}, {{ pos.clientY }}</span>
          <span class="ml-auto text-[10px] text-dimmed">{{ pos.clientW }}×{{ pos.clientH }}</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="text-dimmed w-14">比例</span>
          <span class="text-primary">{{ pos.xRatio.toFixed(4) }}, {{ pos.yRatio.toFixed(4) }}</span>
        </div>
        <div class="pt-2 border-t border-default/40 mt-2 space-y-1.5">
          <div class="flex items-center gap-2">
            <button
              type="button"
              class="flex-1 px-2 py-1 rounded bg-elevated/60 hover:bg-elevated text-[10px] text-toned inline-flex items-center justify-center gap-1 transition-colors"
              title="复制 xRatio, yRatio 到剪贴板"
              @click="copyRatio"
            >
              <UIcon name="i-tabler-copy" class="size-3" />
              复制比例
            </button>
            <button
              type="button"
              class="flex-1 px-2 py-1 rounded bg-primary/15 hover:bg-primary/25 text-[10px] text-primary inline-flex items-center justify-center gap-1 transition-colors"
              title="读光标位置当前像素的 RGB/HSV"
              @click="pickPixel"
            >
              <UIcon name="i-tabler-color-picker" class="size-3" />
              取色
            </button>
          </div>

          <div v-if="pixel?.ok" class="rounded bg-elevated/40 px-2 py-1.5 space-y-1">
            <div class="flex items-center gap-2">
              <div
                class="size-5 rounded border border-default"
                :style="{ backgroundColor: pixel.hex }"
              />
              <code class="text-[11px] text-highlighted">{{ pixel.hex }}</code>
              <button
                type="button"
                class="ml-auto text-[10px] text-dimmed hover:text-primary"
                @click="copyHex"
              >
                copy
              </button>
            </div>
            <div class="text-[10px] text-dimmed">
              RGB <span class="text-toned">{{ pixel.r }} {{ pixel.g }} {{ pixel.b }}</span> · HSV
              <span class="text-toned">{{ pixel.h }} {{ pixel.s }} {{ pixel.v }}</span>
            </div>
          </div>
          <p v-else-if="pixel && !pixel.ok" class="text-[10px] text-warning">光标不在游戏窗口内</p>
        </div>
      </div>
      <div v-else class="text-warning/80 text-[11px] pt-1">
        <UIcon name="i-tabler-alert-triangle" class="size-3 inline" />
        未检测到游戏窗口（只显示屏幕坐标）
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { Window } from '@wailsio/runtime'
import { backend } from '@/lib/backend'

const route = useRoute()
const containerID = String(route.query.containerID ?? '')

interface MousePos {
  screenX: number
  screenY: number
  hasGame: boolean
  clientX: number
  clientY: number
  xRatio: number
  yRatio: number
  clientW: number
  clientH: number
}

const pos = ref<MousePos>({
  screenX: 0,
  screenY: 0,
  hasGame: false,
  clientX: 0,
  clientY: 0,
  xRatio: 0,
  yRatio: 0,
  clientW: 0,
  clientH: 0,
})
const toolError = ref('')

let timer: ReturnType<typeof setInterval> | null = null

async function poll() {
  try {
    const r = await backend.tools.mousePos(containerID)
    if (r) pos.value = r as any
    toolError.value = ''
  } catch (error) {
    toolError.value = error instanceof Error ? error.message : String(error)
  }
}

function copyRatio() {
  const v = `${pos.value.xRatio.toFixed(4)}, ${pos.value.yRatio.toFixed(4)}`
  navigator.clipboard?.writeText(v).catch(() => {})
}

interface PixelInfo {
  ok: boolean
  clientX: number
  clientY: number
  r: number
  g: number
  b: number
  h: number
  s: number
  v: number
  hex: string
}
const pixel = ref<PixelInfo | null>(null)
async function pickPixel() {
  try {
    const r = await backend.tools.pixelAt(containerID)
    if (r) pixel.value = r as any
    toolError.value = ''
  } catch (error) {
    toolError.value = error instanceof Error ? error.message : String(error)
  }
}
function copyHex() {
  if (pixel.value?.hex) navigator.clipboard?.writeText(pixel.value.hex).catch(() => {})
}

function onClose() {
  void Window.Close()
}

onMounted(() => {
  timer = setInterval(poll, 50)
  void poll()
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>
