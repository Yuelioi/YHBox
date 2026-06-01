<template>
  <div class="h-screen flex flex-col bg-zinc-950 text-default select-none">
    <!-- Header -->
    <header
      class="h-9 shrink-0 flex items-center gap-2 px-3 border-b border-default bg-default"
      style="--wails-draggable: drag"
    >
      <UIcon name="i-tabler-crosshair" class="size-3.5 text-primary" />
      <span class="text-xs text-toned">{{ titleByMode }}</span>
      <span class="ml-auto text-[10px] text-dimmed shrink-0" style="--wails-draggable: no-drag">
        {{ hint }}
      </span>
      <button
        type="button"
        class="size-6 ml-2 flex items-center justify-center text-muted hover:bg-error hover:text-highlighted rounded transition-colors"
        style="--wails-draggable: no-drag"
        title="取消并关闭"
        @click="cancel"
      >
        <UIcon name="i-tabler-x" class="size-4" />
      </button>
    </header>

    <!-- Body -->
    <div class="flex-1 flex min-h-0">
      <!-- 图区 -->
      <div class="flex-1 min-w-0 relative bg-zinc-900/60 overflow-auto" ref="canvasWrap">
        <div
          v-if="!dataURL"
          class="absolute inset-0 flex flex-col items-center justify-center text-dimmed text-sm gap-3"
        >
          <UIcon name="i-tabler-camera" class="size-10" />
          <p>正在截屏...</p>
        </div>

        <div v-else class="relative inline-block" :style="{ minWidth: '100%', minHeight: '100%' }">
          <img
            ref="imgRef"
            :src="dataURL"
            class="block max-w-none pointer-events-none"
            :style="{ width: dispW + 'px', height: dispH + 'px' }"
            @load="onImgLoad"
          />

          <!-- 交互层 -->
          <div
            class="absolute inset-0"
            :class="mode === 'point' ? 'cursor-crosshair' : 'cursor-crosshair'"
            @mousedown.prevent="onMouseDown"
            @mousemove="onMouseMove"
            @mouseup="onMouseUp"
            @mouseleave="onMouseUp"
          >
            <!-- point 模式：单击后显示一个十字 + 圆 -->
            <template v-if="mode === 'point' && pointSel">
              <!-- 水平/竖直十字线 -->
              <div
                class="absolute left-0 right-0 h-px bg-primary/60 pointer-events-none"
                :style="{ top: pointSel.y + 'px' }"
              />
              <div
                class="absolute top-0 bottom-0 w-px bg-primary/60 pointer-events-none"
                :style="{ left: pointSel.x + 'px' }"
              />
              <div
                class="absolute size-3 -translate-x-1.5 -translate-y-1.5 rounded-full border-2 border-primary bg-primary/30 pointer-events-none"
                :style="{ left: pointSel.x + 'px', top: pointSel.y + 'px' }"
              />
            </template>

            <!-- rect / template_save 模式：四块遮罩 + 高亮框 -->
            <template v-if="rectSel">
              <div
                class="absolute inset-x-0 top-0 bg-black/55 pointer-events-none"
                :style="{ height: rectSel.y + 'px' }"
              />
              <div
                class="absolute left-0 bg-black/55 pointer-events-none"
                :style="{
                  top: rectSel.y + 'px',
                  height: rectSel.h + 'px',
                  width: rectSel.x + 'px',
                }"
              />
              <div
                class="absolute right-0 bg-black/55 pointer-events-none"
                :style="{
                  top: rectSel.y + 'px',
                  height: rectSel.h + 'px',
                  width: dispW - rectSel.x - rectSel.w + 'px',
                }"
              />
              <div
                class="absolute inset-x-0 bottom-0 bg-black/55 pointer-events-none"
                :style="{ top: rectSel.y + rectSel.h + 'px' }"
              />
              <div
                class="absolute border-2 border-primary pointer-events-none"
                :style="{
                  left: rectSel.x + 'px',
                  top: rectSel.y + 'px',
                  width: rectSel.w + 'px',
                  height: rectSel.h + 'px',
                }"
              />
            </template>
          </div>
        </div>
      </div>

      <!-- 右侧侧栏 -->
      <aside class="w-72 shrink-0 border-l border-default bg-default p-4 overflow-y-auto space-y-4">
        <!-- 通用工具 -->
        <section class="space-y-2">
          <div class="flex items-center gap-2">
            <UButton
              size="xs"
              variant="soft"
              color="neutral"
              icon="i-tabler-refresh"
              :loading="capturing"
              @click="reCapture"
              >重新截屏</UButton
            >
            <UButton
              size="xs"
              variant="ghost"
              color="neutral"
              :icon="zoomFit ? 'i-tabler-zoom-out-area' : 'i-tabler-zoom-in-area'"
              :title="zoomFit ? '切到 1:1（原始像素）' : '切到适应窗口'"
              @click="zoomFit = !zoomFit"
            >
              {{ zoomFit ? '适应窗口' : '原始 1:1' }}
            </UButton>
          </div>
          <p class="text-[10px] text-dimmed">
            原图 {{ natW }} × {{ natH }} · 缩放 {{ Math.round(scale * 100) }}%
          </p>
        </section>

        <!-- 当前选择 -->
        <section class="space-y-2" v-if="mode === 'point'">
          <h4 class="text-[10px] uppercase tracking-wider text-dimmed">选点</h4>
          <div v-if="!pointSel" class="text-[11px] text-dimmed">在图上点一下</div>
          <div v-else class="text-[11px] space-y-1 font-mono tabular-nums">
            <div>
              <span class="text-dimmed">px</span> {{ Math.round(pointNat.x) }},
              {{ Math.round(pointNat.y) }}
            </div>
            <div>
              <span class="text-dimmed">ratio</span>
              <span class="text-primary"
                >{{ pointRatio.x.toFixed(4) }}, {{ pointRatio.y.toFixed(4) }}</span
              >
            </div>
          </div>
        </section>

        <section class="space-y-2" v-else>
          <h4 class="text-[10px] uppercase tracking-wider text-dimmed">框选</h4>
          <div v-if="!rectSel" class="text-[11px] text-dimmed">拖一个矩形</div>
          <div v-else class="text-[11px] space-y-1 font-mono tabular-nums">
            <div>
              <span class="text-dimmed">起点</span> {{ Math.round(rectNat.x) }},
              {{ Math.round(rectNat.y) }}
            </div>
            <div>
              <span class="text-dimmed">尺寸</span> {{ Math.round(rectNat.w) }} ×
              {{ Math.round(rectNat.h) }}
            </div>
            <div>
              <span class="text-dimmed">region</span>
              <span class="text-primary text-[10px]">{{ rectRatioCSV }}</span>
            </div>
          </div>
        </section>

        <!-- template_save 模式专属表单 -->
        <section v-if="mode === 'template_save'" class="space-y-3 pt-2 border-t border-default/60">
          <h4 class="text-[10px] uppercase tracking-wider text-dimmed">保存为模板</h4>
          <UFormField :error="keyError" label="key (namespace.name, 必填)">
            <UInput
              v-model="tplKey"
              size="sm"
              class="w-full"
              placeholder="fishing.hook_icon"
              :ui="{ base: 'font-mono' }"
            />
          </UFormField>
          <div class="space-y-1.5">
            <label class="block text-[11px] text-toned">显示名 (必填)</label>
            <UInput v-model="tplName" size="sm" class="w-full" placeholder="例：上钩图标" />
          </div>
          <div class="space-y-1.5">
            <label class="block text-[11px] text-toned">描述</label>
            <UTextarea v-model="tplDescription" :rows="2" class="w-full" placeholder="可选" />
          </div>
          <p class="text-[10px] text-dimmed">不框选 = 保存全图；框选 = 自动裁剪</p>
        </section>

        <div class="grow" />

        <!-- 底栏按钮 -->
        <div class="pt-3 border-t border-default flex items-center gap-2">
          <UButton size="sm" variant="ghost" color="neutral" @click="cancel">取消</UButton>
          <span class="grow" />
          <UButton
            size="sm"
            color="primary"
            icon="i-tabler-check"
            :disabled="!canConfirm"
            :loading="saving"
            @click="confirm"
          >
            {{ confirmLabel }}
          </UButton>
        </div>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Window, Events } from '@wailsio/runtime'
import { backend } from '@/lib/backend'

const route = useRoute()
const mode = computed(
  () => String(route.query.mode ?? 'point') as 'point' | 'rect' | 'template_save',
)
const requestID = computed(() => String(route.query.id ?? ''))
const containerID = computed(() => String(route.query.containerID ?? ''))

const titleByMode = computed(() => {
  switch (mode.value) {
    case 'point':
      return '屏幕选点'
    case 'rect':
      return '屏幕框选'
    case 'template_save':
      return '截图新模板'
  }
  return '屏幕选择器'
})

const hint = computed(() => {
  switch (mode.value) {
    case 'point':
      return '在图上点一下选取位置'
    case 'rect':
      return '在图上拖一个矩形选取区域'
    case 'template_save':
      return '可选拖一个矩形裁剪；不拖则保存全图'
  }
  return ''
})

const dataURL = ref('')
const capturing = ref(true)
const saving = ref(false)
const natW = ref(0)
const natH = ref(0)
const dispW = ref(0)
const dispH = ref(0)
const zoomFit = ref(true)
const canvasWrap = ref<HTMLElement | null>(null)
const imgRef = ref<HTMLImageElement | null>(null)

const scale = computed(() => (natW.value === 0 ? 1 : dispW.value / natW.value))

// 框/点状态：图上 **displayed** 像素
type Rect = { x: number; y: number; w: number; h: number }
type Point = { x: number; y: number }
const rectSel = ref<Rect | null>(null)
const pointSel = ref<Point | null>(null)
const dragging = ref(false)
const dragStart = ref<Point>({ x: 0, y: 0 })

const rectNat = computed<Rect>(() => {
  if (!rectSel.value || dispW.value === 0) return { x: 0, y: 0, w: 0, h: 0 }
  const sx = natW.value / dispW.value
  const sy = natH.value / dispH.value
  return {
    x: rectSel.value.x * sx,
    y: rectSel.value.y * sy,
    w: rectSel.value.w * sx,
    h: rectSel.value.h * sy,
  }
})
const rectRatioCSV = computed(() => {
  if (!rectSel.value || !natW.value || !natH.value) return ''
  const r = rectNat.value
  return `${(r.x / natW.value).toFixed(3)}, ${(r.y / natH.value).toFixed(3)}, ${(r.w / natW.value).toFixed(3)}, ${(r.h / natH.value).toFixed(3)}`
})

const pointNat = computed<Point>(() => {
  if (!pointSel.value || dispW.value === 0) return { x: 0, y: 0 }
  return {
    x: pointSel.value.x * (natW.value / dispW.value),
    y: pointSel.value.y * (natH.value / dispH.value),
  }
})
const pointRatio = computed<Point>(() => ({
  x: natW.value ? pointNat.value.x / natW.value : 0,
  y: natH.value ? pointNat.value.y / natH.value : 0,
}))

// template form
const tplKey = ref('')
const tplName = ref('')
const tplDescription = ref('')

const keyPattern = /^[a-z0-9_]+(\.[a-z0-9_]+)+$/
const keyValid = computed(() => keyPattern.test(tplKey.value))
const keyError = computed(() => {
  if (!tplKey.value) return ''
  if (keyValid.value) return ''
  return 'key 必须形如 fishing.hook_icon（字母数字下划线 + 至少 1 个点）'
})

const canConfirm = computed(() => {
  if (!dataURL.value) return false
  if (mode.value === 'point') return !!pointSel.value
  if (mode.value === 'rect') return !!rectSel.value
  if (mode.value === 'template_save') return keyValid.value && !!tplName.value.trim()
  return false
})

const confirmLabel = computed(() => {
  if (mode.value === 'template_save') return rectSel.value ? '保存（裁剪）' : '保存（全图）'
  return '确定'
})

async function capture() {
  capturing.value = true
  rectSel.value = null
  pointSel.value = null
  try {
    const r = await backend.templates.capture(containerID.value)
    if (r) dataURL.value = r as string
  } finally {
    capturing.value = false
  }
}
async function reCapture() {
  dataURL.value = ''
  await capture()
}

function onImgLoad() {
  const img = imgRef.value
  if (!img) return
  natW.value = img.naturalWidth
  natH.value = img.naturalHeight
  recomputeDispSize()
}

function recomputeDispSize() {
  if (!natW.value || !natH.value) return
  if (zoomFit.value) {
    const wrap = canvasWrap.value
    const padW = wrap?.clientWidth ?? 800
    const padH = wrap?.clientHeight ?? 600
    const scaleW = padW / natW.value
    const scaleH = padH / natH.value
    const s = Math.min(scaleW, scaleH, 1)
    dispW.value = natW.value * s
    dispH.value = natH.value * s
  } else {
    dispW.value = natW.value
    dispH.value = natH.value
  }
}

watch(zoomFit, () => {
  nextTick(recomputeDispSize)
})

// 鼠标事件
function getRelXY(e: MouseEvent): Point | null {
  const img = imgRef.value
  if (!img) return null
  const rect = img.getBoundingClientRect()
  let x = e.clientX - rect.left
  let y = e.clientY - rect.top
  x = Math.max(0, Math.min(rect.width, x))
  y = Math.max(0, Math.min(rect.height, y))
  return { x, y }
}

function onMouseDown(e: MouseEvent) {
  const p = getRelXY(e)
  if (!p) return
  if (mode.value === 'point') {
    pointSel.value = p
    return
  }
  dragging.value = true
  dragStart.value = p
  rectSel.value = { x: p.x, y: p.y, w: 0, h: 0 }
}

function onMouseMove(e: MouseEvent) {
  if (!dragging.value) return
  const p = getRelXY(e)
  if (!p) return
  const x0 = Math.min(p.x, dragStart.value.x)
  const y0 = Math.min(p.y, dragStart.value.y)
  const x1 = Math.max(p.x, dragStart.value.x)
  const y1 = Math.max(p.y, dragStart.value.y)
  rectSel.value = { x: x0, y: y0, w: x1 - x0, h: y1 - y0 }
}

function onMouseUp() {
  if (!dragging.value) return
  dragging.value = false
  if (rectSel.value && (rectSel.value.w < 4 || rectSel.value.h < 4)) {
    rectSel.value = null
  }
}

async function cropToDataURL(): Promise<string> {
  if (!rectSel.value) return dataURL.value
  const r = rectNat.value
  if (r.w < 1 || r.h < 1) return dataURL.value
  const img = new Image()
  img.src = dataURL.value
  await new Promise<void>((res, rej) => {
    img.onload = () => res()
    img.onerror = () => rej(new Error('load'))
  })
  const canvas = document.createElement('canvas')
  canvas.width = Math.round(r.w)
  canvas.height = Math.round(r.h)
  const ctx = canvas.getContext('2d')
  if (!ctx) return dataURL.value
  ctx.drawImage(img, r.x, r.y, r.w, r.h, 0, 0, canvas.width, canvas.height)
  return canvas.toDataURL('image/png')
}

async function confirm() {
  if (!canConfirm.value) return
  saving.value = true
  try {
    if (mode.value === 'template_save') {
      const png = await cropToDataURL()
      const region: [number, number, number, number] = rectSel.value
        ? [
            rectNat.value.x / natW.value,
            rectNat.value.y / natH.value,
            rectNat.value.w / natW.value,
            rectNat.value.h / natH.value,
          ]
        : [0, 0, 1, 1]
      const ok = await backend.templates.save(
        containerID.value,
        tplKey.value.trim(),
        png,
        tplName.value.trim(),
        tplDescription.value.trim(),
        [natW.value, natH.value],
        region,
      )
      if (!ok) {
        saving.value = false
        return
      }
      await emitResult({ key: tplKey.value.trim() })
    } else if (mode.value === 'point') {
      await emitResult({
        x: Math.round(pointNat.value.x),
        y: Math.round(pointNat.value.y),
        xRatio: pointRatio.value.x,
        yRatio: pointRatio.value.y,
        screenW: natW.value,
        screenH: natH.value,
      })
    } else if (mode.value === 'rect') {
      const r = rectNat.value
      await emitResult({
        x: Math.round(r.x),
        y: Math.round(r.y),
        w: Math.round(r.w),
        h: Math.round(r.h),
        region: [r.x / natW.value, r.y / natH.value, r.w / natW.value, r.h / natH.value],
        screenW: natW.value,
        screenH: natH.value,
      })
    }
    await closeWindow()
  } finally {
    saving.value = false
  }
}

async function emitResult(payload: any) {
  // 注意 wails3 Events.Emit 签名是 (name: string, data: any)，不是 {name, data}。
  // 必须 await：否则 emit Promise 还没 resolve 窗口就被 close 了，对面收不到。
  await Events.Emit('tools:picker-result', {
    id: requestID.value,
    mode: mode.value,
    payload,
  } as any)
}

async function closeWindow() {
  try {
    await backend.tools.closePicker(requestID.value)
  } catch {}
  try {
    await Window.Close()
  } catch {}
}

async function cancel() {
  await emitResult({ cancelled: true })
  await closeWindow()
}

onMounted(async () => {
  await capture()
})

// 窗口尺寸变化时重算适应窗口缩放
const onResize = () => recomputeDispSize()
onMounted(() => window.addEventListener('resize', onResize))
</script>
