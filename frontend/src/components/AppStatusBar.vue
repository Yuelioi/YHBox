<template>
  <div
    class="h-7 shrink-0 flex items-center justify-between px-4 border-t border-default bg-default text-[11px] text-muted select-none"
  >
    <!-- LEFT — active status -->
    <div class="flex items-center gap-3 min-w-0 flex-1">
      <span
        class="size-1.5 rounded-full shrink-0 transition-colors duration-300"
        :class="leftDotClass"
      />
      <span class="font-medium" :class="leftLabelClass">{{ activeStatus.label }}</span>
      <template v-if="activeStatus.metrics.length">
        <span class="text-dimmed">·</span>
        <span class="tabular-nums text-toned truncate">
          {{ activeStatus.metrics.join(' · ') }}
        </span>
      </template>
      <!-- 容器跑中 → 显示当前节点 + 一键停止按钮 -->
      <template v-if="activeStatus.kind === 'container'">
        <span v-if="currentNodeLabel" class="text-dimmed">·</span>
        <span v-if="currentNodeLabel" class="text-primary truncate">
          ▶ {{ currentNodeLabel }}
        </span>
        <button
          type="button"
          class="ml-2 px-2 py-0.5 rounded text-[10px] bg-error/15 border border-error/40 text-error hover:bg-error/25 transition-colors inline-flex items-center gap-1"
          :title="
            t('status.stop_tooltip', {
              hk: hotkeys.keyFor('system.execution-stop', 'Ctrl+Shift+F9'),
            })
          "
          @click="onStopAll"
        >
          <UIcon name="i-tabler-square" class="size-2.5" /> {{ t('status.stop_button') }}
        </button>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useExecutionStore } from '@/stores/execution'
import { useContainersStore } from '@/stores/containers'
import { useHotkeysStore } from '@/stores/hotkeys'

const { t } = useI18n()
const execStore = useExecutionStore()
const containersStore = useContainersStore()
const hotkeys = useHotkeysStore()

// 显示当前正在跑的节点 (走 i18n key, KIND_LABEL_ZH[k] 值是 'node.<k>.label' 字符串)
import { KIND_LABEL_ZH } from '@/components/containers/pinSpec'
const currentNodeLabel = computed(() => {
  const k = execStore.currentNodeKind
  if (!k) return ''
  const key = KIND_LABEL_ZH[k]
  return key ? t(key) : k
})

async function onStopAll() {
  await containersStore.stopAll()
}

type Active = {
  kind: 'container' | 'idle'
  state: 'idle' | 'running'
  label: string
  metrics: string[]
}

const activeStatus = computed<Active>(() => {
  if (execStore.running) {
    const cur = containersStore.list.find((c) => c.id === execStore.currentTargetID)
    const name =
      cur?.name || execStore.currentTargetID.slice(0, 8) || t('status.container_fallback')
    const metrics: string[] = []
    if (execStore.targets.length > 1) {
      metrics.push(`target ${execStore.targetIdx + 1}/${execStore.targets.length}`)
    }
    return { kind: 'container', state: 'running', label: t('status.running', { name }), metrics }
  }
  return { kind: 'idle', state: 'idle', label: t('status.idle'), metrics: [] }
})

const leftDotClass = computed(() => {
  switch (activeStatus.value.state) {
    case 'running':
      return 'bg-primary animate-pulse'
    default:
      return 'bg-accented'
  }
})

const leftLabelClass = computed(() => {
  switch (activeStatus.value.state) {
    case 'running':
      return 'text-primary'
    default:
      return 'text-dimmed'
  }
})
</script>
