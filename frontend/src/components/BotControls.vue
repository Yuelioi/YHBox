<template>
  <div class="flex items-center gap-3">
    <!-- State badge -->
    <div
      class="inline-flex items-center gap-2 px-3 py-1.5 rounded-md bg-default border border-default text-xs font-medium"
    >
      <span
        :class="[
          'size-1.5 rounded-full',
          stopping
            ? 'bg-error animate-pulse'
            : state === 'running'
              ? 'bg-primary animate-pulse'
              : state === 'paused'
                ? 'bg-warning'
                : 'bg-accented',
        ]"
      />
      <span
        :class="[
          stopping
            ? 'text-error'
            : state === 'running'
              ? 'text-primary'
              : state === 'paused'
                ? 'text-warning'
                : 'text-muted',
        ]"
      >
        {{ stateLabel }}
      </span>
    </div>

    <!-- Buttons -->
    <!-- Start: only when idle.
         disabledReason 通过 title 属性显示 (hover tooltip)，不再用单独 info icon
         避免徽章和按钮之间出现突兀的 ⓘ。 -->
    <UButton
      v-if="state === 'idle'"
      color="primary"
      size="md"
      icon="i-tabler-player-play"
      :disabled="!canStart"
      :title="!canStart ? disabledReason : undefined"
      @click="emit('start')"
    >
      {{ t('controls.start') }}
    </UButton>

    <!-- Pause: only when running -->
    <UButton
      v-if="state === 'running'"
      color="neutral"
      size="md"
      icon="i-tabler-player-pause"
      :disabled="stopping"
      @click="emit('pause')"
    >
      {{ t('controls.pause') }}
    </UButton>

    <!-- Resume: only when paused -->
    <UButton
      v-if="state === 'paused'"
      color="primary"
      size="md"
      icon="i-tabler-player-play"
      :disabled="stopping"
      @click="emit('resume')"
    >
      {{ t('controls.resume') }}
    </UButton>

    <!-- Stop: visible when running or paused; disabled + spinner during stopping -->
    <UButton
      v-if="state !== 'idle'"
      color="error"
      variant="soft"
      size="md"
      :icon="stopping ? 'i-tabler-loader-2' : 'i-tabler-square'"
      :loading="stopping"
      :disabled="stopping"
      @click="onStop"
    >
      {{ stopping ? t('controls.stopping') : t('controls.stop') }}
    </UButton>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  state: 'idle' | 'running' | 'paused'
  canStart: boolean
  disabledReason?: string
}>()

const emit = defineEmits<{
  start: []
  pause: []
  resume: []
  stop: []
}>()

// 用户点 Stop 后立刻进入 stopping 态。worker goroutine 实际退出可能延迟数百 ms 到几秒
// （bot 的 sleep 周期 / cleanup），期间 backend 还会 emit running/paused，让 stop 按钮
// 给到清晰反馈：badge 变红脉冲 + 按钮显示 "停止中..." + 禁用所有按钮防误点。
// state 真正变成 idle 时 stopping 自动 reset。
const stopping = ref(false)

function onStop() {
  stopping.value = true
  emit('stop')
}

watch(
  () => props.state,
  (s) => {
    if (s === 'idle') stopping.value = false
  },
)

const stateLabel = computed(() => {
  if (stopping.value) return t('controls.stopping')
  switch (props.state) {
    case 'running':
      return t('controls.state.running')
    case 'paused':
      return t('controls.state.paused')
    default:
      return t('controls.state.idle')
  }
})
</script>
