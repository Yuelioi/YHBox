<template>
  <!-- 倒计时阶段：modal（大圆环明显，引导用户切到游戏窗口） -->
  <UModal
    v-if="store.countdown !== null"
    :open="true"
    :dismissible="false"
    :close="false"
    :ui="{ content: 'w-[520px] sm:max-w-[520px]' }"
  >
    <template #content>
      <div class="flex flex-col items-center gap-6 bg-default" style="padding: 36px 32px">
        <div class="text-center space-y-1">
          <p class="text-sm font-medium text-highlighted">准备开始录制</p>
          <p class="text-xs text-dimmed">请切换到游戏窗口，倒计时结束自动开录</p>
        </div>

        <div class="relative" style="width: 200px; height: 200px">
          <svg viewBox="0 0 100 100" class="w-full h-full -rotate-90">
            <circle cx="50" cy="50" r="46" fill="none" stroke="var(--ui-border)" stroke-width="3" />
            <circle
              :key="store.countdown"
              cx="50"
              cy="50"
              r="46"
              fill="none"
              stroke="var(--ui-color-primary-500)"
              stroke-width="3"
              stroke-linecap="round"
              :stroke-dasharray="ringCirc"
              :stroke-dashoffset="ringCirc"
              class="count-ring"
            />
          </svg>
          <div class="absolute inset-0 flex items-center justify-center">
            <div
              :key="store.countdown"
              class="font-semibold tabular-nums text-primary count-pop"
              style="font-size: 96px; line-height: 1"
            >
              {{ store.countdown }}
            </div>
          </div>
        </div>

        <UButton variant="soft" color="neutral" size="md" @click="onCancel">取消</UButton>
      </div>
    </template>
  </UModal>

  <!-- 录制阶段：右下角浮动 pill —— 不阻挡 canvas，让用户能看自己当前页面 -->
  <Teleport to="body">
    <div
      v-if="store.recording"
      class="fixed bottom-12 right-4 z-[60] flex items-center gap-3 px-4 py-3 rounded-xl bg-error/15 border border-error/40 backdrop-blur-md shadow-2xl animate-in fade-in slide-in-from-bottom-2 duration-200"
    >
      <span class="size-2 rounded-full bg-error animate-pulse shrink-0" />
      <div class="text-xs">
        <div class="font-medium text-highlighted leading-tight">正在录制</div>
        <div class="text-dimmed mt-0.5 tabular-nums">
          {{ store.recordingStepCount }} 个事件 · {{ duration.toFixed(1) }}s
        </div>
      </div>
      <UButton size="xs" color="error" variant="solid" icon="i-tabler-square" @click="onSave">
        停止
      </UButton>
      <UButton
        size="xs"
        variant="ghost"
        color="neutral"
        icon="i-tabler-x"
        title="取消（不保存）"
        @click="onCancel"
      />
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import { useActionsStore } from '@/stores/actions'

const store = useActionsStore()
const duration = ref(0)

const ringCirc = 2 * Math.PI * 46

let startTime = 0
let tick: number | undefined

watch(
  () => store.recording,
  (v) => {
    if (v) {
      startTime = Date.now()
      duration.value = 0
      tick = window.setInterval(() => {
        duration.value = (Date.now() - startTime) / 1000
      }, 100)
    } else if (tick !== undefined) {
      clearInterval(tick)
      tick = undefined
    }
  },
  { immediate: true },
)

onUnmounted(() => {
  if (tick !== undefined) clearInterval(tick)
})

async function onSave() {
  await store.stopRecording()
}
async function onCancel() {
  await store.cancelRecording()
}
</script>

<style scoped>
.count-pop {
  animation: count-pop 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
}
@keyframes count-pop {
  0% {
    transform: scale(0.6);
    opacity: 0;
  }
  60% {
    transform: scale(1.12);
    opacity: 1;
  }
  100% {
    transform: scale(1);
    opacity: 1;
  }
}

.count-ring {
  animation: count-ring 1s linear forwards;
}
@keyframes count-ring {
  from {
    stroke-dashoffset: 289;
  }
  to {
    stroke-dashoffset: 0;
  }
}

/* fade + slide-up entrance */
.animate-in.fade-in.slide-in-from-bottom-2 {
  animation: fade-slide 0.2s ease-out;
}
@keyframes fade-slide {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
