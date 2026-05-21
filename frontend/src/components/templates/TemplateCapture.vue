<template>
  <UModal :open="open" @update:open="onUpdateOpen" :ui="{ content: 'sm:max-w-[760px]' }">
    <template #content>
      <div class="bg-default flex flex-col">
        <!-- Header -->
        <header class="flex items-center gap-2 px-5 py-3 border-b border-default">
          <UIcon name="i-tabler-photo-plus" class="size-4 text-primary" />
          <h3 class="text-sm font-medium text-highlighted">截图新模板</h3>
          <span class="ml-auto text-[11px] text-dimmed">截屏 → 拖拽框选目标区域 → 保存</span>
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-x"
            :padded="false"
            class="ml-2"
            @click="$emit('close')"
          />
        </header>

        <!-- Body -->
        <div class="p-5 space-y-4">
          <!-- 空态：还没截图 -->
          <div
            v-if="!dataURL"
            class="rounded-md border border-dashed border-default/60 bg-elevated/40 py-10 px-6 flex flex-col items-center gap-3 text-center"
          >
            <UIcon name="i-tabler-camera" class="size-8 text-dimmed" />
            <p class="text-xs text-dimmed">截屏游戏窗口，把要识别的目标拍下来作为模板。</p>
            <UButton color="primary" icon="i-tabler-camera" :loading="capturing" @click="onCapture">
              {{ capturing ? '正在截屏...' : '截屏' }}
            </UButton>
          </div>

          <!-- 有截图：预览 + 框选 + 表单 -->
          <div v-else class="space-y-3">
            <div
              class="relative rounded-md overflow-hidden border border-default bg-zinc-900/60 select-none"
              ref="frameRef"
            >
              <img
                ref="imgRef"
                :src="dataURL"
                class="block w-full max-h-[55vh] object-contain pointer-events-none"
                alt="模板预览"
                @load="onImgLoad"
              />

              <!-- 框选 overlay：覆盖整个 image 区域，处理 mouse 事件 -->
              <div
                class="absolute inset-0 cursor-crosshair"
                @mousedown.prevent="onMouseDown"
                @mousemove="onMouseMove"
                @mouseup="onMouseUp"
                @mouseleave="onMouseUp"
              >
                <!-- ROI 暗化区 + 高亮框（仅当有选区时） -->
                <template v-if="hasSelection">
                  <!-- 四块半透明遮罩（上下左右） -->
                  <div
                    class="absolute inset-x-0 top-0 bg-black/55"
                    :style="{ height: selDisp.y + 'px' }"
                  />
                  <div
                    class="absolute left-0 bg-black/55"
                    :style="{
                      top: selDisp.y + 'px',
                      height: selDisp.h + 'px',
                      width: selDisp.x + 'px',
                    }"
                  />
                  <div
                    class="absolute right-0 bg-black/55"
                    :style="{
                      top: selDisp.y + 'px',
                      height: selDisp.h + 'px',
                      width: imgDispW - selDisp.x - selDisp.w + 'px',
                    }"
                  />
                  <div
                    class="absolute inset-x-0 bottom-0 bg-black/55"
                    :style="{ top: selDisp.y + selDisp.h + 'px' }"
                  />
                  <!-- 高亮边框 -->
                  <div
                    class="absolute border-2 border-primary pointer-events-none"
                    :style="{
                      left: selDisp.x + 'px',
                      top: selDisp.y + 'px',
                      width: selDisp.w + 'px',
                      height: selDisp.h + 'px',
                    }"
                  />
                </template>
              </div>

              <!-- 工具栏：右上角 -->
              <div class="absolute top-2 right-2 flex gap-1">
                <UButton
                  v-if="hasSelection"
                  size="xs"
                  variant="solid"
                  color="neutral"
                  icon="i-tabler-square-x"
                  @click="clearSelection"
                >
                  清除框选
                </UButton>
                <UButton
                  size="xs"
                  variant="solid"
                  color="neutral"
                  icon="i-tabler-refresh"
                  :loading="capturing"
                  @click="onCapture"
                >
                  重新截屏
                </UButton>
              </div>
            </div>

            <p class="text-[11px] text-dimmed">
              <UIcon name="i-tabler-info-circle" class="size-3 inline" />
              在图上拖一个矩形框定要识别的目标。不框选则保存整张截图。
            </p>

            <div class="grid grid-cols-2 gap-3">
              <div class="space-y-1.5">
                <UFormField :error="keyError" label="key (namespace.name, 必填)">
                  <UInput
                    v-model="keyPath"
                    size="md"
                    class="w-full"
                    placeholder="fishing.hook_icon"
                    :ui="{ base: 'font-mono' }"
                  />
                </UFormField>
              </div>
              <div class="space-y-1.5">
                <label class="block text-[11px] text-toned">显示名 (必填)</label>
                <UInput v-model="name" size="md" class="w-full" placeholder="例：上钩图标" />
              </div>
            </div>

            <div class="space-y-1.5">
              <label class="block text-[11px] text-toned">描述（可选）</label>
              <UTextarea
                v-model="description"
                :rows="2"
                class="w-full"
                placeholder="什么场景下匹配？阈值有什么注意事项？"
              />
            </div>

            <div class="flex items-center gap-3 text-[11px] text-dimmed flex-wrap">
              <span class="inline-flex items-center gap-1">
                <UIcon name="i-tabler-aspect-ratio" class="size-3" />
                原图 {{ imgNatW }} × {{ imgNatH }}
              </span>
              <span v-if="hasSelection" class="inline-flex items-center gap-1 text-primary">
                <UIcon name="i-tabler-crop" class="size-3" />
                框选 {{ Math.round(selNat.w) }} × {{ Math.round(selNat.h) }}
              </span>
              <span class="inline-flex items-center gap-1">
                <UIcon name="i-tabler-device-desktop" class="size-3" />
                录制分辨率 {{ resolutionLabel }}
              </span>
            </div>
          </div>
        </div>

        <!-- Footer -->
        <footer class="px-5 py-3 border-t border-default flex items-center justify-end gap-2">
          <UButton variant="ghost" color="neutral" @click="$emit('close')">取消</UButton>
          <UButton
            color="primary"
            icon="i-tabler-device-floppy"
            :disabled="!canSave"
            :loading="saving"
            @click="onSave"
          >
            保存模板{{ hasSelection ? '（裁剪）' : '（全图）' }}
          </UButton>
        </footer>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useTemplatesStore } from '@/stores/templates'
import { useGameStore } from '@/stores/game'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()

const templates = useTemplatesStore()
const gameStore = useGameStore()

const dataURL = ref('')
const keyPath = ref('')
const name = ref('')
const description = ref('')
const capturing = ref(false)
const saving = ref(false)

const imgRef = ref<HTMLImageElement | null>(null)
const frameRef = ref<HTMLElement | null>(null)
const imgNatW = ref(0)
const imgNatH = ref(0)
const imgDispW = ref(0)
const imgDispH = ref(0)

// 框选状态：用 image **显示坐标**（rendered px），转换时再映射到自然像素
type Rect = { x: number; y: number; w: number; h: number }
const selDisp = ref<Rect>({ x: 0, y: 0, w: 0, h: 0 })
const hasSelection = ref(false)
const dragging = ref(false)
const dragStart = ref<{ x: number; y: number }>({ x: 0, y: 0 })

// 框选自然像素 = 显示像素 × (natW/dispW)
const selNat = computed<Rect>(() => {
  if (!hasSelection.value || imgDispW.value === 0) return { x: 0, y: 0, w: 0, h: 0 }
  const sx = imgNatW.value / imgDispW.value
  const sy = imgNatH.value / imgDispH.value
  return {
    x: selDisp.value.x * sx,
    y: selDisp.value.y * sy,
    w: selDisp.value.w * sx,
    h: selDisp.value.h * sy,
  }
})

const resolutionLabel = computed(() => {
  const s = gameStore.status
  if (!s?.ok || !s.w) return '未检测到游戏窗口'
  return `${s.w} × ${s.h}`
})

const keyPattern = /^[a-z0-9_]+(\.[a-z0-9_]+)+$/
const keyValid = computed(() => keyPattern.test(keyPath.value))
const keyError = computed(() => {
  if (!keyPath.value) return ''
  if (keyValid.value) return ''
  return 'key 必须形如 fishing.hook_icon（字母数字下划线 + 至少 1 个点）'
})

const canSave = computed(() => !!dataURL.value && keyValid.value && !!name.value.trim())

watch(
  () => props.open,
  (v) => {
    if (v) reset()
  },
)

function reset() {
  dataURL.value = ''
  keyPath.value = ''
  name.value = ''
  description.value = ''
  imgNatW.value = 0
  imgNatH.value = 0
  imgDispW.value = 0
  imgDispH.value = 0
  clearSelection()
}

function clearSelection() {
  hasSelection.value = false
  selDisp.value = { x: 0, y: 0, w: 0, h: 0 }
  dragging.value = false
}

function onUpdateOpen(v: boolean) {
  if (!v) emit('close')
}

async function onCapture() {
  capturing.value = true
  clearSelection()
  try {
    const r = await templates.capture()
    if (r) {
      dataURL.value = r
    }
  } finally {
    capturing.value = false
  }
}

function onImgLoad() {
  const img = imgRef.value
  if (!img) return
  imgNatW.value = img.naturalWidth
  imgNatH.value = img.naturalHeight
  // 等下一帧再测量 displayed size（object-contain 已布局完）
  nextTick(() => {
    const rect = img.getBoundingClientRect()
    imgDispW.value = rect.width
    imgDispH.value = rect.height
  })
}

// 鼠标事件：相对 image 显示区域取坐标
function getRelXY(e: MouseEvent): { x: number; y: number } | null {
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
  dragging.value = true
  dragStart.value = p
  selDisp.value = { x: p.x, y: p.y, w: 0, h: 0 }
  hasSelection.value = true
}

function onMouseMove(e: MouseEvent) {
  if (!dragging.value) return
  const p = getRelXY(e)
  if (!p) return
  const x0 = Math.min(p.x, dragStart.value.x)
  const y0 = Math.min(p.y, dragStart.value.y)
  const x1 = Math.max(p.x, dragStart.value.x)
  const y1 = Math.max(p.y, dragStart.value.y)
  selDisp.value = { x: x0, y: y0, w: x1 - x0, h: y1 - y0 }
}

function onMouseUp() {
  if (!dragging.value) return
  dragging.value = false
  // 框太小（< 4px）算作没框
  if (selDisp.value.w < 4 || selDisp.value.h < 4) {
    clearSelection()
  }
}

// 裁剪：用 canvas 把选定的 selNat 区域从原 PNG 抠出来
async function cropToDataURL(): Promise<string> {
  if (!hasSelection.value) return dataURL.value
  const r = selNat.value
  if (r.w < 1 || r.h < 1) return dataURL.value

  // 重新加载原图到 image 元素（不能复用 imgRef，因为它带 css 缩放）
  const img = new Image()
  img.src = dataURL.value
  await new Promise<void>((res, rej) => {
    img.onload = () => res()
    img.onerror = () => rej(new Error('image load failed'))
  })
  const canvas = document.createElement('canvas')
  canvas.width = Math.round(r.w)
  canvas.height = Math.round(r.h)
  const ctx = canvas.getContext('2d')
  if (!ctx) return dataURL.value
  ctx.drawImage(img, r.x, r.y, r.w, r.h, 0, 0, canvas.width, canvas.height)
  return canvas.toDataURL('image/png')
}

async function onSave() {
  if (!canSave.value) return
  saving.value = true
  try {
    const png = await cropToDataURL()
    const s = gameStore.status
    const screenW = s?.w ?? 0
    const screenH = s?.h ?? 0
    // region: 框选在原 PNG 中的 ratio xywh；没框选 → [0,0,1,1] 全图
    let region: [number, number, number, number] = [0, 0, 1, 1]
    if (hasSelection.value && imgNatW.value && imgNatH.value) {
      region = [
        selNat.value.x / imgNatW.value,
        selNat.value.y / imgNatH.value,
        selNat.value.w / imgNatW.value,
        selNat.value.h / imgNatH.value,
      ]
    }
    const ok = await templates.save(
      keyPath.value.trim(),
      png,
      name.value.trim(),
      description.value.trim(),
      [screenW, screenH],
      region,
    )
    if (ok) emit('close')
  } finally {
    saving.value = false
  }
}
</script>
