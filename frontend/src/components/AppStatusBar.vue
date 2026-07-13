<template>
  <div
    class="flex h-8 shrink-0 select-none items-center justify-between border-t border-default bg-default px-4 text-xs text-muted"
  >
    <!-- LEFT — active status -->
    <div class="flex items-center gap-3 min-w-0 flex-1">
      <span
        class="size-1.5 shrink-0 rounded-full transition-colors duration-300 motion-reduce:transition-none"
        :class="leftDotClass"
        aria-hidden="true"
      />
      <span
        role="status"
        aria-live="polite"
        class="min-w-0 truncate font-medium"
        :class="leftLabelClass"
        >{{ activeStatus.label }}</span
      >
      <template v-if="activeStatus.metrics.length">
        <span class="text-dimmed">·</span>
        <span class="tabular-nums text-toned truncate">
          {{ activeStatus.metrics.join(' · ') }}
        </span>
      </template>
      <!-- 容器跑中 → 显示当前节点 + 一键停止按钮 -->
      <template v-if="activeStatus.kind === 'container'">
        <span v-if="currentNodeLabel" class="text-dimmed">·</span>
        <span v-if="currentNodeLabel" class="min-w-0 truncate text-primary">
          ▶ {{ currentNodeLabel }}
        </span>
        <button
          type="button"
          class="ml-2 inline-flex h-6 shrink-0 items-center gap-1 rounded border border-error/40 bg-error/15 px-2 text-xs text-error transition-colors hover:bg-error/25"
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
      return 'bg-primary animate-pulse motion-reduce:animate-none'
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
