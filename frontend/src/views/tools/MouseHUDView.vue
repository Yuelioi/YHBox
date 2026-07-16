<template>
  <HudShell
    dense
    icon="i-tabler-pointer"
    :title="t('mouseHud.title')"
    :subtitle="t('mouseHud.subtitle')"
    :status="pos.hasGame ? t('mouseHud.target_ready') : t('mouseHud.screen_only')"
    :status-active="pos.hasGame"
    @close="onClose"
  >
    <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-3">
      <UAlert
        v-if="toolError"
        color="error"
        variant="subtle"
        icon="i-tabler-alert-circle"
        :description="toolError"
        :ui="{ description: 'break-words text-xs' }"
      />

      <section class="rounded-xl border border-default bg-elevated/25 px-3 py-2.5">
        <dl
          class="grid grid-cols-[72px_minmax(0,1fr)] gap-x-3 gap-y-2 font-mono text-xs tabular-nums"
        >
          <dt class="text-dimmed">{{ t('mouseHud.screen') }}</dt>
          <dd class="text-highlighted">{{ pos.screenX }}, {{ pos.screenY }}</dd>
          <template v-if="pos.hasGame">
            <dt class="text-dimmed">{{ t('mouseHud.client') }}</dt>
            <dd class="flex min-w-0 items-center gap-2 text-highlighted">
              <span>{{ pos.clientX }}, {{ pos.clientY }}</span>
              <span class="ml-auto text-xs text-dimmed">{{ pos.clientW }}×{{ pos.clientH }}</span>
            </dd>
            <dt class="text-dimmed">{{ t('mouseHud.ratio') }}</dt>
            <dd class="text-primary">{{ pos.xRatio.toFixed(4) }}, {{ pos.yRatio.toFixed(4) }}</dd>
          </template>
        </dl>
      </section>

      <template v-if="pos.hasGame">
        <div class="grid grid-cols-2 gap-2">
          <UButton
            size="sm"
            color="neutral"
            variant="soft"
            :icon="copiedRatio ? 'i-tabler-check' : 'i-tabler-copy'"
            @click="copyRatio"
          >
            {{ copiedRatio ? t('common.copied') : t('mouseHud.copy_ratio') }}
          </UButton>
          <UButton
            size="sm"
            color="primary"
            variant="soft"
            icon="i-tabler-color-picker"
            @click="pickPixel"
          >
            {{ t('mouseHud.pick_color') }}
          </UButton>
        </div>

        <section v-if="pixel?.ok" class="rounded-xl border border-default bg-elevated/25 p-3">
          <div class="flex items-center gap-3">
            <span
              class="size-9 shrink-0 rounded-lg border border-default"
              :style="{ backgroundColor: pixel.hex }"
            />
            <div class="min-w-0 flex-1">
              <code class="text-sm text-highlighted">{{ pixel.hex }}</code>
              <p class="mt-1 font-mono text-xs text-dimmed">
                RGB <span class="text-toned">{{ pixel.r }} {{ pixel.g }} {{ pixel.b }}</span> · HSV
                <span class="text-toned">{{ pixel.h }} {{ pixel.s }} {{ pixel.v }}</span>
              </p>
            </div>
            <UButton
              size="xs"
              variant="ghost"
              color="neutral"
              :icon="copiedHex ? 'i-tabler-check' : 'i-tabler-copy'"
              :aria-label="t('mouseHud.copy_hex')"
              :title="t('mouseHud.copy_hex')"
              @click="copyHex"
            />
          </div>
        </section>
        <UAlert
          v-else-if="pixel && !pixel.ok"
          color="warning"
          variant="subtle"
          icon="i-tabler-focus-auto"
          :description="t('mouseHud.outside_target')"
        />
      </template>

      <UAlert
        v-else
        color="warning"
        variant="subtle"
        icon="i-tabler-window-off"
        :description="t('mouseHud.no_target')"
      />
    </div>
  </HudShell>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Window } from '@wailsio/runtime'
import { backend } from '@/lib/backend'
import HudShell from '@/components/tools/HudShell.vue'

const route = useRoute()
const { t } = useI18n()
const targetSlot = String(route.query.targetSlot ?? '')

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
const copiedRatio = ref(false)
const copiedHex = ref(false)

let timer: ReturnType<typeof setInterval> | null = null

async function poll() {
  try {
    const r = await backend.tools.mousePos(targetSlot)
    if (r) pos.value = r as any
    toolError.value = ''
  } catch (error) {
    toolError.value = error instanceof Error ? error.message : String(error)
  }
}

async function copyRatio() {
  const v = `${pos.value.xRatio.toFixed(4)}, ${pos.value.yRatio.toFixed(4)}`
  await navigator.clipboard?.writeText(v).catch(() => undefined)
  copiedRatio.value = true
  window.setTimeout(() => (copiedRatio.value = false), 1400)
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
    const r = await backend.tools.pixelAt(targetSlot)
    if (r) pixel.value = r as any
    toolError.value = ''
  } catch (error) {
    toolError.value = error instanceof Error ? error.message : String(error)
  }
}
async function copyHex() {
  if (!pixel.value?.hex) return
  await navigator.clipboard?.writeText(pixel.value.hex).catch(() => undefined)
  copiedHex.value = true
  window.setTimeout(() => (copiedHex.value = false), 1400)
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
