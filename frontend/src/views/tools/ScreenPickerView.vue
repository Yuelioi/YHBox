<template>
  <HudShell
    icon="i-tabler-crosshair"
    :accent="pickerAccent"
    :title="titleByMode"
    :subtitle="hint"
    :status="pickerStatus"
    :status-active="!capturing && !!dataURL"
    :close-title="t('screenPicker.cancel_close')"
    @close="cancel"
  >
    <template #actions>
      <UBadge v-if="natW && natH" size="xs" color="neutral" variant="subtle">
        {{ natW }} × {{ natH }}
      </UBadge>
    </template>

    <div class="screen-picker-layout">
      <!-- 左: 工具栏 + 图区 -->
      <div class="flex min-w-0 flex-1 flex-col">
        <div class="screen-picker-toolbar">
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-arrows-maximize"
            :title="t('screenPicker.fit')"
            :aria-label="t('screenPicker.fit')"
            @click="doFit"
          />
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            class="font-mono"
            label="1:1"
            :title="t('screenPicker.actual')"
            :aria-label="t('screenPicker.actual')"
            @click="doActual"
          />
          <USeparator orientation="vertical" class="mx-1 h-4" />
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-minus"
            :title="t('screenPicker.zoom_out')"
            :aria-label="t('screenPicker.zoom_out')"
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
            :title="t('screenPicker.zoom_in')"
            :aria-label="t('screenPicker.zoom_in')"
            @click="zoomBy(1.25)"
          />
          <USeparator orientation="vertical" class="mx-1 h-4" />
          <UButton
            size="xs"
            variant="soft"
            color="neutral"
            icon="i-tabler-refresh"
            :loading="capturing"
            :label="t('screenPicker.recapture')"
            @click="reCapture"
          />
          <span class="screen-picker-toolbar__hint">{{ t('screenPicker.gesture_hint') }}</span>
        </div>

        <!-- 图区视口 -->
        <div
          ref="viewportEl"
          class="screen-picker-canvas"
          :class="
            viewport.panning.value
              ? 'cursor-grabbing'
              : viewport.spaceHeld.value
                ? 'cursor-grab'
                : 'cursor-crosshair'
          "
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
            <p>{{ t('screenPicker.capturing') }}</p>
          </div>

          <template v-else>
            <!-- color mode 提取中遮罩 (叠在截图上, scrim — ui.md 在册例外) -->
            <div
              v-if="extracting"
              class="absolute inset-0 flex flex-col items-center justify-center bg-black/40 text-highlighted text-sm gap-2 z-20"
            >
              <UIcon name="i-tabler-loader-2" class="size-8 animate-spin" />
              <p>{{ t('screenPicker.extracting') }}</p>
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

              <!-- rect: 四块遮罩 + 高亮框 (截图上的取景 scrim — ui.md 在册例外) -->
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
                  :style="{
                    top: rectScreen.y + 'px',
                    height: rectScreen.h + 'px',
                    width: Math.max(0, rectScreen.x) + 'px',
                  }"
                />
                <div
                  class="absolute right-0 bg-black/55"
                  :style="{
                    top: rectScreen.y + 'px',
                    height: rectScreen.h + 'px',
                    left: rectScreen.x + rectScreen.w + 'px',
                  }"
                />
                <div
                  class="absolute border-2 border-primary"
                  :style="{
                    left: rectScreen.x + 'px',
                    top: rectScreen.y + 'px',
                    width: rectScreen.w + 'px',
                    height: rectScreen.h + 'px',
                  }"
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
      <aside class="screen-picker-inspector">
        <!-- 实时读数 -->
        <section class="screen-picker-section">
          <div class="screen-picker-section__heading">
            <UIcon name="i-tabler-pointer" class="size-3.5" />
            <h2>{{ t('screenPicker.cursor') }}</h2>
          </div>
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
              <div
                class="size-6 rounded border border-default shrink-0"
                :style="{ backgroundColor: cursorColor.hex }"
              />
              <div class="space-y-0.5">
                <div class="text-highlighted">{{ cursorColor.hex }}</div>
                <div class="text-[10px] text-dimmed">
                  RGB
                  <span class="text-toned"
                    >{{ cursorColor.r }} {{ cursorColor.g }} {{ cursorColor.b }}</span
                  >
                </div>
                <div class="text-[10px] text-dimmed">
                  HSV
                  <span class="text-toned"
                    >{{ cursorColor.h }} {{ cursorColor.s }} {{ cursorColor.v }}</span
                  >
                </div>
              </div>
            </div>
          </div>
          <div v-else class="text-xs text-dimmed">{{ t('screenPicker.cursor_hint') }}</div>
        </section>

        <!-- 当前选择: point -->
        <section v-if="mode === 'point'" class="screen-picker-section">
          <div class="screen-picker-section__heading">
            <UIcon name="i-tabler-focus-2" class="size-3.5" />
            <h2>{{ t('screenPicker.point') }}</h2>
          </div>
          <div v-if="!pointSelNat" class="text-xs text-dimmed">
            {{ t('screenPicker.point_hint') }}
          </div>
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
              <span class="text-primary">
                {{ pointRatio.x.toFixed(4) }}, {{ pointRatio.y.toFixed(4) }}</span
              >
            </div>
          </template>
        </section>

        <!-- 当前选择: rect / template_save / color -->
        <section v-else class="screen-picker-section">
          <div class="screen-picker-section__heading">
            <UIcon
              :name="mode === 'color' ? 'i-tabler-color-picker' : 'i-tabler-crop'"
              class="size-3.5"
            />
            <h2>{{ mode === 'color' ? t('screenPicker.color') : t('screenPicker.region') }}</h2>
          </div>
          <div v-if="!rectSelNat" class="text-xs text-dimmed">
            {{ mode === 'color' ? t('screenPicker.color_hint') : t('screenPicker.region_hint') }}
          </div>
          <template v-else>
            <div class="grid grid-cols-2 gap-1.5">
              <div class="flex items-center gap-1">
                <label class="text-[10px] text-dimmed w-3">x</label>
                <UInput
                  :model-value="Math.round(rectSelNat.x)"
                  type="number"
                  size="xs"
                  class="flex-1"
                  :ui="{ base: 'font-mono' }"
                  @update:model-value="(v: string | number) => setRect('x', v)"
                />
              </div>
              <div class="flex items-center gap-1">
                <label class="text-[10px] text-dimmed w-3">y</label>
                <UInput
                  :model-value="Math.round(rectSelNat.y)"
                  type="number"
                  size="xs"
                  class="flex-1"
                  :ui="{ base: 'font-mono' }"
                  @update:model-value="(v: string | number) => setRect('y', v)"
                />
              </div>
              <div class="flex items-center gap-1">
                <label class="text-[10px] text-dimmed w-3">w</label>
                <UInput
                  :model-value="Math.round(rectSelNat.w)"
                  type="number"
                  size="xs"
                  class="flex-1"
                  :ui="{ base: 'font-mono' }"
                  @update:model-value="(v: string | number) => setRect('w', v)"
                />
              </div>
              <div class="flex items-center gap-1">
                <label class="text-[10px] text-dimmed w-3">h</label>
                <UInput
                  :model-value="Math.round(rectSelNat.h)"
                  type="number"
                  size="xs"
                  class="flex-1"
                  :ui="{ base: 'font-mono' }"
                  @update:model-value="(v: string | number) => setRect('h', v)"
                />
              </div>
            </div>
            <div class="text-[11px] font-mono tabular-nums">
              <span class="text-dimmed">region</span>
              <span class="text-primary text-[10px]"> {{ rectRatioCSV }}</span>
            </div>
          </template>
        </section>

        <!-- New Global Asset / Workflow Resource metadata form -->
        <section
          v-if="mode === 'template_save' || mode === 'workflow_resource'"
          class="screen-picker-section"
        >
          <div class="screen-picker-section__heading">
            <UIcon name="i-tabler-photo-plus" class="size-3.5" />
            <h2>{{ t('screenPicker.template.title') }}</h2>
          </div>
          <UFormField :label="t('screenPicker.template.name')" required>
            <UInput
              v-model="tplName"
              size="sm"
              class="w-full"
              :placeholder="t('screenPicker.template.name_placeholder')"
            />
          </UFormField>
          <UFormField :label="t('screenPicker.template.category')">
            <UInputMenu
              v-model="tplCategory"
              :items="tplCategoryItems"
              :create-item="'always'"
              size="sm"
              class="w-full"
              :placeholder="t('screenPicker.template.category_placeholder')"
              @create="onCreateTplCategory"
            />
          </UFormField>
          <UFormField :label="t('screenPicker.template.tags')">
            <div v-if="tplTags.length" class="flex flex-wrap gap-1">
              <UBadge
                v-for="(tag, i) in tplTags"
                :key="tag + i"
                color="neutral"
                variant="subtle"
                class="gap-1"
              >
                {{ tag }}
                <UButton
                  size="xs"
                  color="neutral"
                  variant="link"
                  icon="i-tabler-x"
                  class="size-4 p-0"
                  :aria-label="t('screenPicker.template.remove_tag', { tag })"
                  @click="tplTags.splice(i, 1)"
                />
              </UBadge>
            </div>
            <UInput
              v-model="tplTagInput"
              size="sm"
              class="w-full"
              :placeholder="t('screenPicker.template.tags_placeholder')"
              @keyup.enter="addTplTag"
            />
          </UFormField>
          <p class="text-xs text-dimmed">{{ t('screenPicker.template.crop_hint') }}</p>
        </section>

        <div class="grow" />

        <!-- 底栏按钮 -->
        <div class="pt-2 border-t border-default flex items-center gap-2">
          <UButton size="sm" variant="ghost" color="neutral" @click="cancel">{{
            t('common.cancel')
          }}</UButton>
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
  </HudShell>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useToast } from '@/composables/useAppToast'
import { Window, Events } from '@wailsio/runtime'
import { useLocalStorage } from '@vueuse/core'
import { backend } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import { rgbToHsv, rgbToHex } from '@/lib/color'
import { usePickerViewport } from '@/composables/tools/usePickerViewport'
import PickerMagnifier from '@/components/tools/PickerMagnifier.vue'
import { addCreatedCategory, uniqueCategoryOptions } from '@/lib/categoryOptions'
import HudShell from '@/components/tools/HudShell.vue'

const route = useRoute()
const { t } = useI18n()
const toast = useToast()
const mode = computed(
  () =>
    String(route.query.mode ?? 'point') as
      | 'point'
      | 'rect'
      | 'template_save'
      | 'workflow_resource'
      | 'workflow_resource_version'
      | 'template_recapture'
      | 'color',
)
const colorSpace = computed(() => String(route.query.colorSpace ?? 'hsv') as 'hsv' | 'rgb')
const extracting = ref(false)
const requestID = computed(() => String(route.query.id ?? ''))
const targetSlot = computed(() => String(route.query.targetSlot ?? ''))
// template_recapture: 重拍目标资产 GUID (存成同 GUID 的新分辨率档).
const recaptureGUID = computed(() => String(route.query.guid ?? ''))
const pickerAccent = computed<'primary' | 'success' | 'warning'>(() => {
  if (mode.value === 'color') return 'warning'
  if (
    mode.value === 'template_save' ||
    mode.value === 'workflow_resource' ||
    mode.value === 'workflow_resource_version' ||
    mode.value === 'template_recapture'
  )
    return 'success'
  return 'primary'
})

const titleByMode = computed(() => {
  return t(`screenPicker.mode.${mode.value}`)
})

const hint = computed(() => {
  return t(`screenPicker.hint.${mode.value}`)
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
const pickerStatus = computed(() => {
  if (capturing.value) return t('screenPicker.status.capturing')
  if (saving.value) return t('screenPicker.status.saving')
  if (extracting.value) return t('screenPicker.status.extracting')
  if (pointSelNat.value || rectSelNat.value) return t('screenPicker.status.selected')
  return t('screenPicker.status.ready')
})

// 原生 → 视口容器屏幕坐标 (覆盖层用).
function natToScreen(nx: number, ny: number): Point {
  return {
    x: viewport.offset.value.x + nx * viewport.zoom.value,
    y: viewport.offset.value.y + ny * viewport.zoom.value,
  }
}
const pointScreen = computed(() =>
  pointSelNat.value ? natToScreen(pointSelNat.value.x, pointSelNat.value.y) : null,
)
const rectScreen = computed(() => {
  if (!rectSelNat.value) return null
  const tl = natToScreen(rectSelNat.value.x, rectSelNat.value.y)
  return {
    x: tl.x,
    y: tl.y,
    w: rectSelNat.value.w * viewport.zoom.value,
    h: rectSelNat.value.h * viewport.zoom.value,
  }
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

// template form — key 已移除，后端分配 GUID; 填名称 + 可选分类/标签
const tplName = ref('')
const lastTplCategory = useLocalStorage('template.capture.lastCategory', '')
const tplCategory = ref(lastTplCategory.value)
const tplKnownCategories = ref<string[]>([])
const tplCreatedCategories = ref<string[]>([])
const tplCategoryItems = computed(() =>
  uniqueCategoryOptions(tplKnownCategories.value, tplCreatedCategories.value, [tplCategory.value]),
)
const tplTags = ref<string[]>([])
const tplTagInput = ref('')
function onCreateTplCategory(item: string) {
  const result = addCreatedCategory(tplCreatedCategories.value, item)
  if (!result.value) return
  tplCreatedCategories.value = result.categories
  tplCategory.value = result.value
}
async function loadTemplateCategories() {
  if (mode.value !== 'template_save' && mode.value !== 'workflow_resource') return
  const summaries = await backend.assets.list()
  if (!summaries) return
  tplKnownCategories.value = uniqueCategoryOptions(
    (summaries as { kind: string; category?: string }[])
      .filter((item) => item.kind === 'template')
      .map((item) => item.category ?? ''),
  )
}
function addTplTag() {
  const v = tplTagInput.value.trim()
  if (v && !tplTags.value.includes(v)) tplTags.value.push(v)
  tplTagInput.value = ''
}

const canConfirm = computed(() => {
  if (!dataURL.value) return false
  if (mode.value === 'point') return !!pointSelNat.value
  if (mode.value === 'rect') return !!rectSelNat.value
  if (mode.value === 'template_save' || mode.value === 'workflow_resource')
    return !!tplName.value.trim()
  if (mode.value === 'workflow_resource_version') return !!dataURL.value
  // 重拍: 资产已有 name, 不用填; 有截图即可 (region 可选).
  if (mode.value === 'template_recapture') return !!dataURL.value
  // color mode 自动提取 (pointerup 触发), 不走 confirm 按钮
  return false
})
const confirmLabel = computed(() => {
  if (mode.value === 'template_save' || mode.value === 'workflow_resource') {
    return t(rectSelNat.value ? 'screenPicker.save_crop' : 'screenPicker.save_full')
  }
  if (mode.value === 'template_recapture' || mode.value === 'workflow_resource_version') {
    return t(rectSelNat.value ? 'screenPicker.recapture_crop' : 'screenPicker.recapture_full')
  }
  return t('common.confirm')
})

// ── 截屏 ───────────────────────────────────────────────
async function capture() {
  capturing.value = true
  rectSelNat.value = null
  pointSelNat.value = null
  cursorNat.value = null
  cursorColor.value = null
  try {
    const r = await backend.assets.capture(targetSlot.value)
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
    cursorColor.value = {
      r: d[0],
      g: d[1],
      b: d[2],
      h: hsv.h,
      s: hsv.s,
      v: hsv.v,
      hex: rgbToHex(d[0], d[1], d[2]),
    }
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
  const typing =
    !!el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable)
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
    if (
      mode.value === 'template_save' ||
      mode.value === 'workflow_resource' ||
      mode.value === 'workflow_resource_version' ||
      mode.value === 'template_recapture'
    ) {
      const png = await cropToDataURL()
      const region: [number, number, number, number] = rectSelNat.value
        ? [
            rectSelNat.value.x / natW.value,
            rectSelNat.value.y / natH.value,
            rectSelNat.value.w / natW.value,
            rectSelNat.value.h / natH.value,
          ]
        : [0, 0, 1, 1]
      if (mode.value === 'template_recapture') {
        // 重拍: 存成同 GUID 的新分辨率变体, 所有引用自动跟随新图.
        await backend.assets.addTemplateVariant(
          recaptureGUID.value,
          png,
          [natW.value, natH.value],
          region,
        )
        await emitResult({ guid: recaptureGUID.value })
      } else if (mode.value === 'template_save') {
        // SaveTemplateCapture allocates a new global asset guid.
        const guid = await backend.assets.saveTemplateCapture(
          png,
          tplName.value.trim(),
          tplCategory.value.trim(),
          tplTags.value,
          [natW.value, natH.value],
          region,
        )
        lastTplCategory.value = tplCategory.value.trim()
        await emitResult({ guid: guid as string })
      } else {
        const resource = await backend.workflowResources.createImage({
          name:
            mode.value === 'workflow_resource_version'
              ? t('screenPicker.version.default_name')
              : tplName.value.trim(),
          description: '',
          category: mode.value === 'workflow_resource_version' ? '' : tplCategory.value.trim(),
          tags: mode.value === 'workflow_resource_version' ? [] : tplTags.value,
          dataURL: png,
          resolution: [natW.value, natH.value],
          region,
        })
        lastTplCategory.value = tplCategory.value.trim()
        await emitResult({ resource })
      }
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
  } catch (error) {
    toast.add({
      title: t('toast.operation_failed'),
      description: errorMessage(error),
      color: 'error',
    })
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
  await Promise.all([capture(), loadTemplateCategories()])
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('keyup', onKeyUp)
  window.removeEventListener('resize', onResize)
})
</script>

<style scoped>
.screen-picker-layout {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1;
}

.screen-picker-toolbar {
  display: flex;
  min-height: 40px;
  flex: none;
  align-items: center;
  gap: 2px;
  padding: 5px 8px;
  border-bottom: 1px solid var(--ui-border);
  background: color-mix(in oklab, var(--ui-bg-elevated) 28%, transparent);
}

.screen-picker-toolbar__hint {
  overflow: hidden;
  margin-left: auto;
  font-size: 10px;
  line-height: 14px;
  color: var(--ui-text-dimmed);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.screen-picker-canvas {
  position: relative;
  min-width: 0;
  min-height: 0;
  flex: 1;
  overflow: hidden;
  background:
    linear-gradient(
        45deg,
        color-mix(in oklab, var(--ui-border) 35%, transparent) 25%,
        transparent 25%
      )
      0 0 / 20px 20px,
    linear-gradient(
        -45deg,
        color-mix(in oklab, var(--ui-border) 35%, transparent) 25%,
        transparent 25%
      )
      0 10px / 20px 20px,
    linear-gradient(
        45deg,
        transparent 75%,
        color-mix(in oklab, var(--ui-border) 35%, transparent) 75%
      )
      10px -10px / 20px 20px,
    linear-gradient(
        -45deg,
        transparent 75%,
        color-mix(in oklab, var(--ui-border) 35%, transparent) 75%
      ) -10px
      0 / 20px 20px,
    var(--ui-bg);
}

.screen-picker-inspector {
  display: flex;
  width: clamp(272px, 24vw, 328px);
  min-height: 0;
  flex: none;
  flex-direction: column;
  gap: 10px;
  overflow-y: auto;
  padding: 12px;
  border-left: 1px solid var(--ui-border);
  background: var(--ui-bg);
}

.screen-picker-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
  border: 1px solid var(--ui-border);
  border-radius: 12px;
  padding: 12px;
  background: color-mix(in oklab, var(--ui-bg-elevated) 24%, transparent);
}

.screen-picker-section__heading {
  display: flex;
  align-items: center;
  gap: 7px;
  color: var(--ui-text-muted);
}

.screen-picker-section__heading h2 {
  font-size: 11px;
  line-height: 14px;
  font-weight: 650;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

@media (max-width: 860px) {
  .screen-picker-inspector {
    width: 270px;
  }

  .screen-picker-toolbar__hint {
    display: none;
  }
}

@media (max-height: 560px) {
  .screen-picker-section {
    gap: 7px;
    padding: 9px;
  }
}
</style>
