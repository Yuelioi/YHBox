<template>
  <HudShell
    icon="i-tabler-focus-centered"
    :title="t('calibration.title')"
    :subtitle="t('calibration.hud.subtitle')"
    :status="stageLabel"
    :status-active="stage === 'accumulating'"
    @close="onCancel"
  >
    <div class="flex min-h-0 flex-1 flex-col gap-3 p-3">
      <div class="live-hud__stage">
        <HudStatePanel
          v-if="stage === 'waiting'"
          tone="neutral"
          icon="i-tabler-keyboard"
          :eyebrow="t('calibration.hud.waiting')"
          :hint="t('calibration.hud.press_to_start', { hk })"
        >
          <UKbd :value="hk" />
        </HudStatePanel>

        <HudStatePanel
          v-else-if="stage === 'countingDown'"
          tone="warning"
          icon="i-tabler-hourglass-high"
          :eyebrow="t('calibration.hud.countdown')"
          :value="countdown"
          :hint="t('calibration.ready_status')"
        />

        <HudStatePanel
          v-else-if="stage === 'accumulating'"
          tone="success"
          active
          :eyebrow="t('calibration.recording_status')"
          :value="liveAbsDx"
          :hint="t('calibration.press_f8_stop', { hk })"
        >
          <span class="font-mono">|dy| {{ liveAbsDy }}</span>
        </HudStatePanel>

        <HudStatePanel
          v-else
          tone="primary"
          icon="i-tabler-circle-check"
          :eyebrow="t('calibration.recorded_label')"
          :value="liveAbsDx"
          :hint="t('calibration.save_or_retest', { hk })"
        />
      </div>

      <UAlert
        v-if="hotkeyWarn"
        color="warning"
        variant="subtle"
        icon="i-tabler-alert-triangle"
        :description="t('calibration.service_failed')"
        :ui="{ description: 'text-xs' }"
      />

      <div class="flex shrink-0 items-center gap-2 border-t border-default pt-3">
        <UButton
          size="sm"
          variant="ghost"
          color="neutral"
          icon="i-tabler-refresh"
          :disabled="stage === 'waiting' || stage === 'countingDown'"
          @click="resetSession"
        >
          {{ t('common.retest') }}
        </UButton>
        <span class="ml-auto" />
        <UButton size="sm" variant="ghost" color="neutral" @click="onCancel">
          {{ t('common.cancel') }}
        </UButton>
        <UButton
          size="sm"
          color="primary"
          icon="i-tabler-device-floppy"
          :disabled="stage !== 'done' || liveAbsDx === 0"
          @click="onSave"
        >
          {{ t('calibration.save_with_value', { value: liveAbsDx }) }}
        </UButton>
      </div>
    </div>
  </HudShell>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { Events } from '@wailsio/runtime'
import { backend } from '@/lib/backend'
import { useHotkeysStore } from '@/stores/hotkeys'
import { useCalibrationSession } from '@/composables/useCalibrationSession'
import HudShell from '@/components/tools/HudShell.vue'
import HudStatePanel from '@/components/tools/HudStatePanel.vue'

const { t } = useI18n()
const route = useRoute()
const hotkeys = useHotkeysStore()
const hk = hotkeys.keyFor('system.calibrate-toggle', 'F8')

const requestID = String(route.query.id ?? '')

const { stage, countdown, hotkeyWarn, liveAbsDx, liveAbsDy, open, teardown, resetSession } =
  useCalibrationSession()
const stageLabel = computed(() => t(`calibration.hud.stage.${stage.value}`))

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

<style scoped>
.live-hud__stage {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1;
}
</style>
