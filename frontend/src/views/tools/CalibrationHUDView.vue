<template>
  <!-- 独立置顶校准 HUD (360×220 Frameless + AlwaysOnTop). 开窗时后端已把游戏置前,
       用户不用自己切屏; F8 走后端自治 LL hook, 切到游戏也收得到。 -->
  <div class="h-screen w-screen bg-default flex flex-col select-none">
    <header
      class="flex items-center gap-2 px-3 py-2 border-b border-default shrink-0"
      style="--wails-draggable: drag"
    >
      <UIcon name="i-tabler-target" class="size-4 text-primary" />
      <h3 class="text-xs font-medium text-highlighted">{{ t('calibration.title') }}</h3>
      <span class="ml-auto" />
      <UButton size="xs" variant="ghost" color="neutral" icon="i-tabler-x" @click="onCancel" />
    </header>

    <div class="flex-1 min-h-0 flex flex-col items-center justify-center px-3 py-3 gap-2">
      <!-- 状态面板: 深底 + 语义实彩描边 (还原旧 CalibratorModal 风格, 适配小窗) -->
      <div
        v-if="stage === 'waiting'"
        class="w-full rounded-lg border border-dashed border-default/60 bg-elevated/40 px-4 py-3 text-center space-y-1"
      >
        <UIcon name="i-tabler-keyboard" class="size-7 text-primary mx-auto" />
        <p class="text-sm text-highlighted">{{ t('calibration.hud.press_to_start', { hk }) }}</p>
        <p class="text-[10px] text-dimmed">{{ t('calibration.countdown_desc') }}</p>
      </div>

      <div
        v-else-if="stage === 'countingDown'"
        class="w-full rounded-lg border border-warning/40 bg-warning/10 px-4 py-3 text-center space-y-1"
      >
        <div class="text-5xl font-mono tabular-nums text-warning">{{ countdown }}</div>
        <p class="text-xs text-warning">{{ t('calibration.ready_status') }}</p>
      </div>

      <div
        v-else-if="stage === 'accumulating'"
        class="w-full rounded-lg border border-success/40 bg-success/10 px-4 py-3 text-center space-y-1"
      >
        <div class="flex items-center justify-center gap-2 text-success">
          <span class="size-2 rounded-full bg-success animate-pulse" />
          <span class="text-xs">{{ t('calibration.recording_status') }}</span>
        </div>
        <div class="text-4xl font-mono tabular-nums text-success">{{ liveAbsDx }}</div>
        <p class="text-[10px] text-dimmed font-mono">|dy| {{ liveAbsDy }}</p>
        <p class="text-[10px] text-success/80">{{ t('calibration.press_f8_stop', { hk }) }}</p>
      </div>

      <div
        v-else
        class="w-full rounded-lg border border-primary/40 bg-primary/10 px-4 py-3 text-center space-y-1"
      >
        <UIcon name="i-tabler-circle-check" class="size-7 text-primary mx-auto" />
        <div class="text-4xl font-mono tabular-nums text-primary">{{ liveAbsDx }}</div>
        <p class="text-[10px] text-dimmed">{{ t('calibration.save_or_retest', { hk }) }}</p>
      </div>

      <p v-if="hotkeyWarn" class="text-[10px] text-warning">
        <UIcon name="i-tabler-alert-triangle" class="size-3 inline" />
        {{ t('calibration.service_failed') }}
      </p>
    </div>

    <footer class="flex items-center gap-2 px-3 py-2 border-t border-default shrink-0">
      <UButton
        size="xs"
        variant="ghost"
        color="neutral"
        icon="i-tabler-refresh"
        :disabled="stage === 'waiting' || stage === 'countingDown'"
        @click="resetSession"
        >{{ t('common.retest') }}</UButton
      >
      <span class="ml-auto" />
      <UButton size="xs" variant="ghost" color="neutral" @click="onCancel">{{
        t('common.cancel')
      }}</UButton>
      <UButton
        size="xs"
        color="primary"
        icon="i-tabler-device-floppy"
        :disabled="stage !== 'done' || liveAbsDx === 0"
        @click="onSave"
        >{{ t('calibration.save_with_value', { value: liveAbsDx }) }}</UButton
      >
    </footer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { Events } from '@wailsio/runtime'
import { backend } from '@/lib/backend'
import { useHotkeysStore } from '@/stores/hotkeys'
import { useCalibrationSession } from '@/composables/useCalibrationSession'

const { t } = useI18n()
const route = useRoute()
const hotkeys = useHotkeysStore()
const hk = hotkeys.keyFor('system.calibrate-toggle', 'F8')

const requestID = String(route.query.id ?? '')

const { stage, countdown, hotkeyWarn, liveAbsDx, liveAbsDy, open, teardown, resetSession } =
  useCalibrationSession()

onMounted(() => {
  void open()
})
onUnmounted(() => {
  void teardown()
})

// 保存: 把累计值 emit 给开窗方 (按 requestID 匹配), 后端关窗 (触发兜底卸钩)。
async function onSave() {
  const counts = liveAbsDx.value
  if (counts > 0) {
    await Events.Emit('calibration:result', { id: requestID, counts })
  }
  await closeWindow()
}

async function onCancel() {
  await Events.Emit('calibration:result', { id: requestID, cancelled: true })
  await closeWindow()
}

async function closeWindow() {
  await teardown()
  try {
    await backend.tools.closeCalibratorHUD()
  } catch {}
}
</script>
