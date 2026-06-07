<template>
  <div class="h-screen flex flex-col bg-default text-default select-none">
    <!-- Header (跟录屏家族 HUD 同风格: 主题底色 + 下边框 + UButton 关闭) -->
    <header
      class="shrink-0 flex items-center gap-2 px-3 py-2 border-b border-default"
      style="--wails-draggable: drag"
    >
      <UIcon name="i-tabler-crosshair" class="size-3.5 text-primary" />
      <span class="text-xs font-medium text-highlighted">{{ titleByMode }}</span>
      <span class="ml-auto text-[10px] text-dimmed shrink-0" style="--wails-draggable: no-drag">
        {{ hint }}
      </span>
      <UButton
        size="xs"
        variant="ghost"
        color="neutral"
        icon="i-tabler-x"
        style="--wails-draggable: no-drag"
        title="取消并关闭"
        @click="cancel"
      />
    </header>

    <!-- Body -->
    <div class="flex-1 flex min-h-0">
      <!-- 左: 工具栏 + 图区 -->
      <div class="flex-1 min-w-0 flex flex-col">
        <!-- 工具栏 (底部状态条) -->
        <div class="order-last shrink-0 h-9 flex items-center gap-1 px-2 border-t border-default">
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-arrows-maximize"
            title="适应窗口"
            @click="doFit"
          />
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            label="1:1"
            :ui="{ label: 'text-[11px] font-mono' }"
            title="原始像素 1:1"
            @click="doActual"
          />
          <div class="mx-1 h-4 w-px bg-default" />
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-minus"
            title="缩小"
            @click="zoomBy(1 / 1.25)"
          />
          <span class="text-[11px] font-mono tabular-nums text-toned w-12 text-center">
            {{ Math.round(viewport.zoom.value * 100) }}%
          </span>
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-plus"
            title="放大"
            @click="zoomBy(1.25)"
          />
          <div class="mx-1 h-4 w-px bg-default" />
          <UButton
            size="xs"
            variant="soft"
            color="neutral"
            icon="i-tabler-refresh"
            :loading="capturing"
            label="重新截屏"
            @click="reCapture"
          />
          <span class="ml-auto text-[10px] text-dimmed">滚轮缩放 · 右键/空格拖动平移 · 方向键微调</span>
        </div>

        <!-- 图区视口 -->
        <div
          ref="viewportEl"
          class="flex-1 min-h-0 relative overflow-hidden bg-zinc-900"
          :class="viewport.panning.value ? 'cursor-grabbing' : viewport.spaceHeld.value ? 'cursor-grab' : 'cursor-crosshair'"
          @wheel="viewport.onWheel"
          @pointerdown="onViewportPointerDown"
          @mousemove="onViewportMouseMove"
          @mouseleave="onViewportLeave"
          @contextmenu.prevent
        >
          <!-- 加载态 -->
          <div
            v-if="!dataURL"
            class="absolute inset-0 flex flex-col items-center justify-center text-dimmed text-sm gap-3"
          >
            <UIcon name="i-tabler-camera" class="size-10 animate-pulse" />
            <p>正在截屏...</p>
          </div>

          <template v-else>
            <!-- color mode 提取中遮罩 (叠在图上方) -->
            <div
              v-if="extracting"
              class="absolute inset-0 flex flex-col items-center justify-center bg-black/40 text-white text-sm gap-2 z-20"
            >
              <UIcon name="i-tabler-loader-2" class="size-8 animate-spin" />
              <p>正在提取颜色范围...</p>
            </div>
            <!-- 图 (transform 缩放/平移) -->
            <img
              ref="imgRef"
              :src="dataURL"
              class="absolute top-0 left-0 max-w-none pointer-events-none"
              :style="{ ...viewport.transformStyle.value, width: natW + 'px', height: natH + 'px' }"
              @load="onImgLoad"
            />

            <!-- 覆盖层 (屏幕空间, 边框不随缩放变粗) -->
            <div class="absolute inset-0 pointer-events-none">
              <!-- point: 十字线 + 圆点 -->
              <template v-if="mode === 'point' && pointScreen">
                <div
                  class="absolute left-0 right-0 h-px bg-primary/60"
                  :style="{ top: pointScreen.y + 'px' }"
                />
                <div
                  class="absolute top-0 bottom-0 w-px bg-primary/60"
                  :style="{ left: pointScreen.x + 'px' }"
                />
                <div
                  class="absolute size-3 -translate-x-1.5 -translate-y-1.5 rounded-full border-2 border-primary bg-primary/30"
                  :style="{ left: pointScreen.x + 'px', top: pointScreen.y + 'px' }"
                />
              </template>

              <!-- rect: 四块遮罩 + 高亮框 -->
              <template v-if="rectScreen">
                <div
                  class="absolute inset-x-0 top-0 bg-black/55"
                  :style="{ height: Math.max(0, rectScreen.y) + 'px' }"
                />
                <div
                  class="absolute inset-x-0 bottom-0 bg-black/55"
                  :style="{ top: rectScreen.y + rectScreen.h + 'px' }"
                />
                <div
                  class="absolute left-0 bg-black/55"
                  :style="{ top: rectScreen.y + 'px', height: rectScreen.h + 'px', width: Math.max(0, rectScreen.x) + 'px' }"
                />
                <div
                  class="absolute right-0 bg-black/55"
                  :style="{ top: rectScreen.y + 'px', height: rectScreen.h + 'px', left: rectScreen.x + rectScreen.w + 'px' }"
                />
                <div
                  class="absolute border-2 border-primary"
                  :style="{ left: rectScreen.x + 'px', top: rectScreen.y + 'px', width: rectScreen.w + 'px', height: rectScreen.h + 'px' }"
                />
              </template>
            </div>

            <!-- 放大镜 (跟随光标) -->
            <PickerMagnifier
              v-if="cursorNat && !viewport.panning.value"
              class="absolute z-10"
              :style="loupeStyle"
              :source="sampleCanvas"
              :nx="cursorNat.x"
              :ny="cursorNat.y"
            />
          </template>
        </div>
      </div>

      <!-- 右侧侧栏 -->
      <aside class="w-72 shrink-0 border-l border-default bg-default p-3 overflow-y-auto flex flex-col gap-3">
        <!-- 实时读数 -->
        <section class="rounded-lg border border-default/60 bg-elevated/40 p-2.5 space-y-2">
          <h4 class="text-[10px] uppercase tracking-wider text-dimmed">光标</h4>
          <div v-if="cursorNat" class="space-y-1.5 text-[11px] font-mono tabular-nums">
            <div>
              <span class="text-dimmed">px</span> {{ Math.round(cursorNat.x) }},
              {{ Math.round(cursorNat.y) }}
            </div>
            <div>
              <span class="text-dimmed">ratio</span>
              <span class="text-primary">
                {{ (cursorNat.x / natW).toFixed(4) }}, {{ (cursorNat.y / natH).toFixed(4) }}
              </span>
            </div>
            <div v-if="cursorColor" class="flex items-center gap-2 pt-1">
              <div class="size-6 rounded border border-default shrink-0" :style="{ backgroundColor: cursorColor.hex }" />
              <div class="space-y-0.5">
                <div class="text-highlighted">{{ cursorColor.hex }}</div>
                <div class="text-[10px] text-dimmed">
                  RGB <span class="text-toned">{{ cursorColor.r }} {{ cursorColor.g }} {{ cursorColor.b }}</span>
                </div>
                <div class="text-[10px] text-dimmed">
                  HSV <span class="text-toned">{{ cursorColor.h }} {{ cursorColor.s }} {{ cursorColor.v }}</span>
                </div>
              </div>
            </div>
          </div>
          <div v-else class="text-[11px] text-dimmed">移到图上看坐标/取色</div>
        </section>

        <!-- 当前选择: point -->
        <section
          v-if="mode === 'point'"
          class="rounded-lg border border-default/60 bg-elevated/40 p-2.5 space-y-2"
        >
          <h4 class="text-[10px] uppercase tracking-wider text-dimmed">选点</h4>
          <div v-if="!pointSelNat" class="text-[11px] text-dimmed">在图上点一下</div>
          <template v-else>
            <div class="flex items-center gap-2">
              <label class="text-[10px] text-dimmed w-3">x</label>
              <UInput
                :model-value="Math.round(pointSelNat.x)"
                type="number"
                size="xs"
                class="flex-1"
                :ui="{ base: 'font-mono' }"
                @update:model-value="(v: string | number) => setPoint('x', v)"
              />
              <label class="text-[10px] text-dimmed w-3">y</label>
              <UInput
                :model-value="Math.round(pointSelNat.y)"
                type="number"
                size="xs"
                class="flex-1"
                :ui="{ base: 'font-mono' }"
                @update:model-value="(v: string | number) => setPoint('y', v)"
              />
            </div>
            <div class="text-[11px] font-mono tabular-nums">
              <span class="text-dimmed">ratio</span>
              <span class="text-primary"> {{ pointRatio.x.toFixed(4) }}, {{ pointRatio.y.toFixed(4) }}</span>
            </div>
          </template>
        </section>

        <!-- 当前选择: rect / template_save / color -->
        <section v-else class="rounded-lg border border-default/60 bg-elevated/40 p-2.5 space-y-2">
          <h4 class="text-[10px] uppercase tracking-wider text-dimmed">{{ mode === 'color' ? '取色' : '框选' }}</h4>
          <div v-if="!rectSelNat" class="text-[11px] text-dimmed">{{ mode === 'color' ? '框选颜色区域或点击单点取色' : '在图上拖一个矩形' }}</div>
          <template v-else>
            <div class="grid grid-cols-2 gap-1.5">
              <div class="flex items-center gap-1">
                <label class="text-[10px] text-dimmed w-3">x</label>
                <UInput :model-value="Math.round(rectSelNat.x)" type="number" size="xs" class="flex-1" :ui="{ base: 'font-mono' }" @update:model-value="(v: string | number) => setRect('x', v)" />
              </div>
              <div class="flex items-center gap-1">
                <label class="text-[10px] text-dimmed w-3">y</label>
                <UInput :model-value="Math.round(rectSelNat.y)" type="number" size="xs" class="flex-1" :ui="{ base: 'font-mono' }" @update:model-value="(v: string | number) => setRect('y', v)" />
              </div>
              <div class="flex items-center gap-1">
                <label class="text-[10px] text-dimmed w-3">w</label>
                <UInput :model-value="Math.round(rectSelNat.w)" type="number" size="xs" class="flex-1" :ui="{ base: 'font-mono' }" @update:model-value="(v: string | number) => setRect('w', v)" />
              </div>
              <div class="flex items-center gap-1">
                <label class="text-[10px] text-dimmed w-3">h</label>
                <UInput :model-value="Math.round(rectSelNat.h)" type="number" size="xs" class="flex-1" :ui="{ base: 'font-mono' }" @update:model-value="(v: string | number) => setRect('h', v)" />
              </div>
            </div>
            <div class="text-[11px] font-mono tabular-nums">
              <span class="text-dimmed">region</span>
              <span class="text-primary text-[10px]"> {{ rectRatioCSV }}</span>
            </div>
          </template>
        </section>

        <!-- template_save 表单 -->
        <section
          v-if="mode === 'template_save'"
          class="rounded-lg border border-default/60 bg-elevated/40 p-2.5 space-y-3"
        >
          <h4 class="text-[10px] uppercase tracking-wider text-dimmed">保存为模板</h4>
          <UFormField :error="keyError" label="key (namespace.name, 必填)">
            <UInput v-model="tplKey" size="sm" class="w-full" placeholder="fishing.hook_icon" :ui="{ base: 'font-mono' }" />
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
        <div class="pt-2 border-t border-default flex items-center gap-2">
          <UButton size="sm" variant="ghost" color="neutral" @click="cancel">取消</UButton>
          <span class="grow" />
          <UButton
            v-if="mode !== 'color'"
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
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { Window, Events } from '@wailsio/runtime'
import { backend } from '@/lib/backend'
import { rgbToHsv, rgbToHex } from '@/lib/color'
import { usePickerViewport } from '@/composables/tools/usePickerViewport'
import PickerMagnifier from '@/components/tools/PickerMagnifier.vue'

const route = useRoute()
const mode = computed(
  () => String(route.query.mode ?? 'point') as 'point' | 'rect' | 'template_save' | 'color',
)
const colorSpace = computed(() => String(route.query.colorSpace ?? 'hsv') as 'hsv' | 'rgb')
const extracting = ref(false)
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
    case 'color':
      return '屏幕取色'
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
    case 'color':
      return '框选目标颜色区域（点一下取单点色）'
  }
  return ''
})

const dataURL = ref('')
const capturing = ref(true)
const saving = ref(false)
const natW = ref(0)
const natH = ref(0)
const imgRef = ref<HTMLImageElement | null>(null)
const viewportEl = ref<HTMLElement | null>(null)
const viewport = usePickerViewport(() => viewportEl.value)

// 选区: 以原生像素为唯一真相.
type Rect = { x: number; y: number; w: number; h: number }
type Point = { x: number; y: number }
const pointSelNat = ref<Point | null>(null)
const rectSelNat = ref<Rect | null>(null)

// 光标实时态 (原生坐标 + 取色) + 屏幕坐标 (放大镜定位).
const cursorNat = ref<Point | null>(null)
const mouseContainer = ref<Point>({ x: 0, y: 0 })

// 离屏 canvas: 整图 1:1, 供取色 getImageData + 放大镜 drawImage 源.
const sampleCanvas = ref<HTMLCanvasElement | null>(null)
let sampleCtx: CanvasRenderingContext2D | null = null

interface CursorColor {
  r: number
  g: number
  b: number
  h: number
  s: number
  v: number
  hex: string
}
const cursorColor = ref<CursorColor | null>(null)

// 原生 → 视口容器屏幕坐标 (覆盖层用).
function natToScreen(nx: number, ny: number): Point {
  return { x: viewport.offset.value.x + nx * viewport.zoom.value, y: viewport.offset.value.y + ny * viewport.zoom.value }
}
const pointScreen = computed(() => (pointSelNat.value ? natToScreen(pointSelNat.value.x, pointSelNat.value.y) : null))
const rectScreen = computed(() => {
  if (!rectSelNat.value) return null
  const tl = natToScreen(rectSelNat.value.x, rectSelNat.value.y)
  return { x: tl.x, y: tl.y, w: rectSelNat.value.w * viewport.zoom.value, h: rectSelNat.value.h * viewport.zoom.value }
})

const pointRatio = computed<Point>(() => ({
  x: natW.value ? (pointSelNat.value?.x ?? 0) / natW.value : 0,
  y: natH.value ? (pointSelNat.value?.y ?? 0) / natH.value : 0,
}))
const rectRatioCSV = computed(() => {
  if (!rectSelNat.value || !natW.value || !natH.value) return ''
  const r = rectSelNat.value
  return `${(r.x / natW.value).toFixed(3)}, ${(r.y / natH.value).toFixed(3)}, ${(r.w / natW.value).toFixed(3)}, ${(r.h / natH.value).toFixed(3)}`
})

// 放大镜跟随光标, 贴边自动翻转.
const loupeStyle = computed(() => {
  const off = 18
  const sz = 136
  const el = viewportEl.value
  const vw = el?.clientWidth ?? 0
  const vh = el?.clientHeight ?? 0
  let left = mouseContainer.value.x + off
  let top = mouseContainer.value.y + off
  if (left + sz > vw) left = mouseContainer.value.x - off - sz
  if (top + sz + 20 > vh) top = mouseContainer.value.y - off - (sz + 20)
  return { left: `${left}px`, top: `${top}px` }
})

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
  if (mode.value === 'point') return !!pointSelNat.value
  if (mode.value === 'rect') return !!rectSelNat.value
  if (mode.value === 'template_save') return keyValid.value && !!tplName.value.trim()
  // color mode 自动提取 (pointerup 触发), 不走 confirm 按钮
  return false
})
const confirmLabel = computed(() => {
  if (mode.value === 'template_save') return rectSelNat.value ? '保存（裁剪）' : '保存（全图）'
  return '确定'
})

// ── 截屏 ───────────────────────────────────────────────
async function capture() {
  capturing.value = true
  rectSelNat.value = null
  pointSelNat.value = null
  cursorNat.value = null
  cursorColor.value = null
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
  // 建离屏 canvas 供取色 + 放大镜.
  const c = document.createElement('canvas')
  c.width = natW.value
  c.height = natH.value
  const ctx = c.getContext('2d', { willReadFrequently: true })
  if (ctx) {
    ctx.drawImage(img, 0, 0)
    sampleCtx = ctx
    sampleCanvas.value = c
  }
  nextTick(doFit)
}

// ── 缩放/平移工具栏 ────────────────────────────────────
// 工具栏按钮点完立即失焦: 否则按钮留焦 → 之后按空格会"点击"它复位视图 (空格拖不动的根因).
function blurActive() {
  ;(document.activeElement as HTMLElement | null)?.blur?.()
}
function doFit() {
  viewport.fit(natW.value, natH.value)
  blurActive()
}
function doActual() {
  viewport.actualSize(natW.value, natH.value)
  blurActive()
}
function zoomBy(factor: number) {
  const el = viewportEl.value
  if (!el) return
  const r = el.getBoundingClientRect()
  viewport.zoomAt(r.left + el.clientWidth / 2, r.top + el.clientHeight / 2, factor)
  blurActive()
}

// ── 鼠标交互 ───────────────────────────────────────────
const clampX = (x: number) => Math.min(natW.value - 1, Math.max(0, x))
const clampY = (y: number) => Math.min(natH.value - 1, Math.max(0, y))
function clampNative(p: Point): Point {
  return { x: clampX(p.x), y: clampY(p.y) }
}

const interacting = ref<'pan' | 'rect' | null>(null)
let dragStart: Point = { x: 0, y: 0 }

function updateCursor(clientX: number, clientY: number) {
  const el = viewportEl.value
  if (!el || !natW.value) return
  const r = el.getBoundingClientRect()
  mouseContainer.value = { x: clientX - r.left, y: clientY - r.top }
  const raw = viewport.screenToNative(clientX, clientY)
  if (raw.x < 0 || raw.y < 0 || raw.x >= natW.value || raw.y >= natH.value) {
    cursorNat.value = null
    cursorColor.value = null
    return
  }
  cursorNat.value = { x: raw.x, y: raw.y }
  if (sampleCtx) {
    const px = Math.min(natW.value - 1, Math.round(raw.x))
    const py = Math.min(natH.value - 1, Math.round(raw.y))
    const d = sampleCtx.getImageData(px, py, 1, 1).data
    const hsv = rgbToHsv(d[0], d[1], d[2])
    cursorColor.value = { r: d[0], g: d[1], b: d[2], h: hsv.h, s: hsv.s, v: hsv.v, hex: rgbToHex(d[0], d[1], d[2]) }
  }
}

function rectFrom(a: Point, b: Point): Rect {
  const x0 = Math.min(a.x, b.x)
  const y0 = Math.min(a.y, b.y)
  return { x: x0, y: y0, w: Math.abs(a.x - b.x), h: Math.abs(a.y - b.y) }
}

function onViewportPointerDown(e: PointerEvent) {
  if (!dataURL.value) return
  // 平移: 空格+左键 (Photoshop 手型) 或 右键拖动. 不用中键 — 滚轮型中键拖动会误触发缩放.
  const wantPan = (viewport.spaceHeld.value && e.button === 0) || e.button === 2
  if (wantPan) {
    interacting.value = 'pan'
    viewport.beginPan(e.clientX, e.clientY)
    window.addEventListener('pointermove', onWinPointerMove)
    window.addEventListener('pointerup', onWinPointerUp)
    e.preventDefault()
    return
  }
  if (e.button !== 0) return
  const p = clampNative(viewport.screenToNative(e.clientX, e.clientY))
  if (mode.value === 'point') {
    pointSelNat.value = p
    return
  }
  interacting.value = 'rect'
  dragStart = p
  rectSelNat.value = { x: p.x, y: p.y, w: 0, h: 0 }
  window.addEventListener('pointermove', onWinPointerMove)
  window.addEventListener('pointerup', onWinPointerUp)
}

function onViewportMouseMove(e: MouseEvent) {
  updateCursor(e.clientX, e.clientY)
}
function onViewportLeave() {
  if (interacting.value) return
  cursorNat.value = null
  cursorColor.value = null
}

function onWinPointerMove(e: PointerEvent) {
  if (interacting.value === 'pan') viewport.movePan(e.clientX, e.clientY)
  else if (interacting.value === 'rect') {
    const p = clampNative(viewport.screenToNative(e.clientX, e.clientY))
    rectSelNat.value = rectFrom(dragStart, p)
  }
  updateCursor(e.clientX, e.clientY)
}
function onWinPointerUp() {
  if (interacting.value === 'rect' && rectSelNat.value) {
    if (mode.value === 'color') {
      void extractColorAt(rectSelNat.value)
    } else if (rectSelNat.value.w < 2 || rectSelNat.value.h < 2) {
      rectSelNat.value = null
    }
  }
  if (interacting.value === 'pan') viewport.endPan()
  interacting.value = null
  window.removeEventListener('pointermove', onWinPointerMove)
  window.removeEventListener('pointerup', onWinPointerUp)
}

// ── 数值编辑 ───────────────────────────────────────────
function setPoint(axis: 'x' | 'y', v: string | number) {
  if (!pointSelNat.value) return
  const n = Math.round(Number(v) || 0)
  pointSelNat.value = {
    ...pointSelNat.value,
    [axis]: axis === 'x' ? clampX(n) : clampY(n),
  }
}
function setRect(field: 'x' | 'y' | 'w' | 'h', v: string | number) {
  if (!rectSelNat.value) return
  const n = Math.max(0, Math.round(Number(v) || 0))
  const r = { ...rectSelNat.value }
  if (field === 'x') r.x = Math.min(natW.value - 1, n)
  else if (field === 'y') r.y = Math.min(natH.value - 1, n)
  else if (field === 'w') r.w = Math.min(natW.value - r.x, n)
  else r.h = Math.min(natH.value - r.y, n)
  rectSelNat.value = r
}

// ── 方向键微调 ─────────────────────────────────────────
function onKeyDown(e: KeyboardEvent) {
  const el = e.target as HTMLElement | null
  const typing = !!el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable)
  // 空格 = 临时手型平移; preventDefault 防止聚焦的工具栏按钮被空格"点击"复位视图 + 防页面滚动.
  if (e.code === 'Space' && !typing) e.preventDefault()
  viewport.onKeyDown(e)
  if (capturing.value || !natW.value || typing) return
  const arrows = ['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown']
  if (!arrows.includes(e.key)) return
  const step = e.shiftKey ? 10 : 1
  let dx = 0
  let dy = 0
  if (e.key === 'ArrowLeft') dx = -step
  else if (e.key === 'ArrowRight') dx = step
  else if (e.key === 'ArrowUp') dy = -step
  else dy = step
  if (mode.value === 'point' && pointSelNat.value) {
    e.preventDefault()
    pointSelNat.value = { x: clampX(pointSelNat.value.x + dx), y: clampY(pointSelNat.value.y + dy) }
  } else if (rectSelNat.value) {
    e.preventDefault()
    const r = { ...rectSelNat.value }
    r.x = Math.min(natW.value - r.w, Math.max(0, r.x + dx))
    r.y = Math.min(natH.value - r.h, Math.max(0, r.y + dy))
    rectSelNat.value = r
  }
}
function onKeyUp(e: KeyboardEvent) {
  viewport.onKeyUp(e)
}

// ── 颜色范围提取 (color mode) ──────────────────────────
async function extractColorAt(region: { x: number; y: number; w: number; h: number }) {
  if (!sampleCtx || extracting.value) return
  extracting.value = true
  try {
    const x = Math.max(0, Math.round(region.x))
    const y = Math.max(0, Math.round(region.y))
    const w = Math.min(natW.value - x, Math.max(1, Math.round(region.w)))
    const h = Math.min(natH.value - y, Math.max(1, Math.round(region.h)))
    const img = sampleCtx.getImageData(x, y, w, h).data
    const total = w * h
    const stride = Math.max(1, Math.ceil(total / 4096))
    const samples: { R: number; G: number; B: number }[] = []
    for (let i = 0; i < total; i += stride) {
      const o = i * 4
      samples.push({ R: img[o], G: img[o + 1], B: img[o + 2] })
    }
    const res = await backend.tools.extractColorRange(samples, colorSpace.value)
    if (!res) return
    await emitResult({ range: res.range, hueWrap: res.hueWrap })
    await closeWindow()
  } finally {
    extracting.value = false
  }
}

// ── 确定 / 取消 ────────────────────────────────────────
async function cropToDataURL(): Promise<string> {
  if (!rectSelNat.value) return dataURL.value
  const r = rectSelNat.value
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
      const region: [number, number, number, number] = rectSelNat.value
        ? [
            rectSelNat.value.x / natW.value,
            rectSelNat.value.y / natH.value,
            rectSelNat.value.w / natW.value,
            rectSelNat.value.h / natH.value,
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
    } else if (mode.value === 'point' && pointSelNat.value) {
      await emitResult({
        x: Math.round(pointSelNat.value.x),
        y: Math.round(pointSelNat.value.y),
        xRatio: pointRatio.value.x,
        yRatio: pointRatio.value.y,
        screenW: natW.value,
        screenH: natH.value,
      })
    } else if (mode.value === 'rect' && rectSelNat.value) {
      const r = rectSelNat.value
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
  // wails3 Events.Emit 签名是 (name, data); 必须 await, 否则窗口先关对面收不到.
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

// ── 生命周期 ───────────────────────────────────────────
const onResize = () => {
  if (natW.value) doFit()
}
onMounted(async () => {
  window.addEventListener('keydown', onKeyDown)
  window.addEventListener('keyup', onKeyUp)
  window.addEventListener('resize', onResize)
  await capture()
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('keyup', onKeyUp)
  window.removeEventListener('resize', onResize)
})
</script>
