<template>
  <div class="space-y-3">
    <!-- 默认段: 百分比 -->
    <div class="space-y-2">
      <div class="flex items-center gap-1.5">
        <span class="text-[11px] text-toned font-medium">默认区域</span>
        <span v-if="isFullFrame" class="text-[10px] text-dimmed">(全帧)</span>
      </div>

      <div class="grid grid-cols-2 gap-1.5">
        <div class="space-y-0.5">
          <label class="text-[10px] text-dimmed">X %</label>
          <UInputNumber
            :model-value="pctDisplay.x"
            size="xs"
            class="w-full"
            :min="0"
            :max="100"
            :step="0.1"
            :placeholder="isFullFrame ? '留空' : undefined"
            @update:model-value="(v: number) => onPctChange('x', v)"
          />
        </div>
        <div class="space-y-0.5">
          <label class="text-[10px] text-dimmed">Y %</label>
          <UInputNumber
            :model-value="pctDisplay.y"
            size="xs"
            class="w-full"
            :min="0"
            :max="100"
            :step="0.1"
            :placeholder="isFullFrame ? '留空' : undefined"
            @update:model-value="(v: number) => onPctChange('y', v)"
          />
        </div>
        <div class="space-y-0.5">
          <label class="text-[10px] text-dimmed">W %</label>
          <UInputNumber
            :model-value="pctDisplay.w"
            size="xs"
            class="w-full"
            :min="0"
            :max="100"
            :step="0.1"
            :placeholder="isFullFrame ? '留空' : undefined"
            @update:model-value="(v: number) => onPctChange('w', v)"
          />
        </div>
        <div class="space-y-0.5">
          <label class="text-[10px] text-dimmed">H %</label>
          <UInputNumber
            :model-value="pctDisplay.h"
            size="xs"
            class="w-full"
            :min="0"
            :max="100"
            :step="0.1"
            :placeholder="isFullFrame ? '留空' : undefined"
            @update:model-value="(v: number) => onPctChange('h', v)"
          />
        </div>
      </div>

      <!-- 超 100% 提示 -->
      <p v-if="hasPctOver100" class="text-[10px] text-error leading-snug">
        部分值超过 100%，请确认是否正确
      </p>

      <!-- 截图框选按钮 -->
      <UButton
        size="xs"
        variant="soft"
        color="primary"
        icon="i-tabler-camera"
        :loading="picking"
        @click="onPickScreenRect"
      >
        截图框选
      </UButton>
    </div>

    <!-- 高级: 分辨率覆盖 (collapsible) -->
    <UCollapsible v-model:open="overridesOpen">
      <UButton
        size="xs"
        variant="ghost"
        color="neutral"
        :icon="overridesOpen ? 'i-tabler-chevron-down' : 'i-tabler-chevron-right'"
        class="w-full justify-start text-dimmed"
      >
        高级: 分辨率覆盖
        <UBadge
          v-if="safeValue.overrides && safeValue.overrides.length > 0"
          size="xs"
          variant="soft"
          color="neutral"
          class="ml-1"
        >
          {{ safeValue.overrides.length }}
        </UBadge>
      </UButton>

      <template #content>
        <div class="mt-2 space-y-3 pl-1">
          <!-- 已有 overrides 列表 -->
          <div
            v-for="(ov, idx) in safeValue.overrides ?? []"
            :key="idx"
            class="rounded-md border border-default/60 bg-elevated/20 p-2 space-y-2"
          >
            <div class="flex items-center gap-2">
              <span class="text-[11px] text-toned font-mono">
                {{ ov.resolution.w }}×{{ ov.resolution.h }}
              </span>
              <span class="grow" />
              <UButton
                size="xs"
                variant="ghost"
                color="error"
                icon="i-tabler-trash"
                @click="removeOverride(idx)"
              />
            </div>

            <!-- Override px 输入 -->
            <div class="grid grid-cols-2 gap-1">
              <div class="space-y-0.5">
                <label class="text-[10px] text-dimmed">X px</label>
                <UInputNumber
                  :model-value="ov.px.x"
                  size="xs"
                  class="w-full"
                  :min="0"
                  :step="1"
                  @update:model-value="(v: number) => updateOverridePx(idx, 'x', v)"
                />
              </div>
              <div class="space-y-0.5">
                <label class="text-[10px] text-dimmed">Y px</label>
                <UInputNumber
                  :model-value="ov.px.y"
                  size="xs"
                  class="w-full"
                  :min="0"
                  :step="1"
                  @update:model-value="(v: number) => updateOverridePx(idx, 'y', v)"
                />
              </div>
              <div class="space-y-0.5">
                <label class="text-[10px] text-dimmed">W px</label>
                <UInputNumber
                  :model-value="ov.px.w"
                  size="xs"
                  class="w-full"
                  :min="0"
                  :step="1"
                  @update:model-value="(v: number) => updateOverridePx(idx, 'w', v)"
                />
              </div>
              <div class="space-y-0.5">
                <label class="text-[10px] text-dimmed">H px</label>
                <UInputNumber
                  :model-value="ov.px.h"
                  size="xs"
                  class="w-full"
                  :min="0"
                  :step="1"
                  @update:model-value="(v: number) => updateOverridePx(idx, 'h', v)"
                />
              </div>
            </div>

            <!-- 截图框选 (仅当前屏幕分辨率匹配时可用) -->
            <UTooltip
              :text="screenResMatches(ov.resolution) ? '' : '仅屏幕分辨率匹配时可框选'"
              :disabled="screenResMatches(ov.resolution)"
            >
              <UButton
                size="xs"
                variant="ghost"
                color="primary"
                icon="i-tabler-camera"
                :disabled="!screenResMatches(ov.resolution) || picking"
                :loading="picking"
                @click="onPickOverrideRect(idx)"
              >
                截图框选 (px)
              </UButton>
            </UTooltip>
          </div>

          <!-- 添加覆盖面板 -->
          <div class="space-y-2 pt-1 border-t border-default/40">
            <p class="text-[10px] text-dimmed">添加分辨率覆盖</p>

            <!-- 分辨率预设选择 -->
            <USelect
              v-model="addResPreset"
              :items="resPresetItems"
              size="xs"
              class="w-full"
            />

            <!-- 自定义分辨率输入 -->
            <div v-if="addResPreset === 'custom'" class="flex items-center gap-1.5">
              <UInputNumber
                v-model="addCustomW"
                :min="1"
                size="xs"
                class="flex-1"
                placeholder="W"
              />
              <span class="text-[10px] text-dimmed">×</span>
              <UInputNumber
                v-model="addCustomH"
                :min="1"
                size="xs"
                class="flex-1"
                placeholder="H"
              />
            </div>

            <!-- 重复分辨率警告 -->
            <p v-if="isDupResolution" class="text-[10px] text-error">
              此分辨率已存在覆盖
            </p>

            <UButton
              size="xs"
              variant="soft"
              color="neutral"
              icon="i-tabler-plus"
              :disabled="isDupResolution || !canAddResolution"
              @click="addOverride"
            >
              添加
            </UButton>
          </div>
        </div>
      </template>
    </UCollapsible>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { backend } from '@/lib/backend'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import type { GeometryValue } from '@/components/containers/nodeRegistry/index'

type RectPayload = {
  region: [number, number, number, number]
  screenW?: number
  screenH?: number
  cancelled?: boolean
}
interface PickerResult {
  id: string
  mode: string
  payload: RectPayload
}

const props = defineProps<{
  modelValue: GeometryValue | null
  fieldPath: string
}>()
const emit = defineEmits<{ (e: 'update:modelValue', v: GeometryValue): void }>()

// ─── 归一 ──────────────────────────────────────────────────────────────────
const safeValue = computed<GeometryValue>(() => {
  const v = props.modelValue
  if (!v || !v.pct) return { pct: { x: 0, y: 0, w: 0, h: 0 }, overrides: [] }
  return { pct: v.pct, overrides: v.overrides ?? [] }
})

/** w===0 && h===0 → 全帧语义 */
const isFullFrame = computed(() => {
  const p = safeValue.value.pct
  return p.w === 0 && p.h === 0
})

/** 0-1 → 0-100 display */
const pctDisplay = computed(() => {
  const p = safeValue.value.pct
  return {
    x: round4(p.x * 100),
    y: round4(p.y * 100),
    w: round4(p.w * 100),
    h: round4(p.h * 100),
  }
})

const hasPctOver100 = computed(() => {
  const d = pctDisplay.value
  return d.x > 100 || d.y > 100 || d.w > 100 || d.h > 100
})

// ─── 工具 ──────────────────────────────────────────────────────────────────
function round4(n: number): number {
  return Math.round(n * 1e4) / 1e4
}

function cloneValue(): GeometryValue {
  const v = safeValue.value
  return {
    pct: { ...v.pct },
    overrides: (v.overrides ?? []).map((o) => ({
      resolution: { ...o.resolution },
      px: { ...o.px },
    })),
  }
}

// ─── 百分比编辑 ────────────────────────────────────────────────────────────
function onPctChange(field: 'x' | 'y' | 'w' | 'h', displayVal: number) {
  const next = cloneValue()
  next.pct[field] = round4(displayVal / 100)
  emit('update:modelValue', next)
}

// ─── 截图框选 (pct) ────────────────────────────────────────────────────────
const picking = ref(false)

function genPickID(): string {
  return 'pick-geo-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now()
}

async function openRectPicker(): Promise<RectPayload | null> {
  const id = genPickID()
  picking.value = true
  try {
    const waiter = awaitWailsEvent<PickerResult>('tools:picker-result', (p) => p?.id === id)
    const r = await backend.tools.openScreenPicker('rect', id)
    if (r === undefined) return null
    const result = await waiter
    if (result.payload?.cancelled) return null
    return result.payload
  } finally {
    picking.value = false
  }
}

async function onPickScreenRect() {
  const p = await openRectPicker()
  if (!p) return
  const [rx, ry, rw, rh] = p.region
  const next = cloneValue()
  next.pct = { x: round4(rx), y: round4(ry), w: round4(rw), h: round4(rh) }
  emit('update:modelValue', next)
}

// ─── Override: 截图框选 (px) ───────────────────────────────────────────────
function screenResMatches(res: { w: number; h: number }): boolean {
  return window.screen.width === res.w && window.screen.height === res.h
}

async function onPickOverrideRect(idx: number) {
  const ov = safeValue.value.overrides?.[idx]
  if (!ov) return
  const p = await openRectPicker()
  if (!p) return
  // region ratio → px (乘以 override 分辨率)
  const [rx, ry, rw, rh] = p.region
  const resW = ov.resolution.w
  const resH = ov.resolution.h
  const next = cloneValue()
  next.overrides![idx].px = {
    x: Math.round(rx * resW),
    y: Math.round(ry * resH),
    w: Math.round(rw * resW),
    h: Math.round(rh * resH),
  }
  emit('update:modelValue', next)
}

// ─── Override CRUD ─────────────────────────────────────────────────────────
const overridesOpen = ref(false)

const PRESET_RESOLUTIONS = [
  { label: '当前屏幕', value: 'screen' },
  { label: '1920×1080', value: '1920x1080' },
  { label: '2560×1440', value: '2560x1440' },
  { label: '3840×2160', value: '3840x2160' },
  { label: '自定义', value: 'custom' },
] as const

const resPresetItems = PRESET_RESOLUTIONS.map((p) => ({ label: p.label, value: p.value }))

const addResPreset = ref<string>('1920x1080')
const addCustomW = ref<number>(1920)
const addCustomH = ref<number>(1080)

const resolvedAddRes = computed<{ w: number; h: number }>(() => {
  if (addResPreset.value === 'screen') {
    return { w: window.screen.width, h: window.screen.height }
  }
  if (addResPreset.value === 'custom') {
    return { w: addCustomW.value || 1920, h: addCustomH.value || 1080 }
  }
  const [wStr, hStr] = addResPreset.value.split('x')
  return { w: parseInt(wStr, 10) || 1920, h: parseInt(hStr, 10) || 1080 }
})

const isDupResolution = computed(() => {
  const res = resolvedAddRes.value
  return (safeValue.value.overrides ?? []).some(
    (o) => o.resolution.w === res.w && o.resolution.h === res.h,
  )
})

const canAddResolution = computed(() => {
  const res = resolvedAddRes.value
  return res.w > 0 && res.h > 0
})

function addOverride() {
  if (isDupResolution.value || !canAddResolution.value) return
  const res = resolvedAddRes.value
  const next = cloneValue()
  next.overrides!.push({ resolution: { ...res }, px: { x: 0, y: 0, w: 0, h: 0 } })
  emit('update:modelValue', next)
}

function removeOverride(idx: number) {
  const next = cloneValue()
  next.overrides!.splice(idx, 1)
  emit('update:modelValue', next)
}

function updateOverridePx(idx: number, field: 'x' | 'y' | 'w' | 'h', v: number) {
  const next = cloneValue()
  next.overrides![idx].px[field] = Math.max(0, Math.round(v))
  emit('update:modelValue', next)
}
</script>
