<template>
  <section
    class="flex max-h-80 shrink-0 flex-col border-t border-default bg-default"
    data-testid="workflow-debugger"
  >
    <header class="flex items-center gap-3 border-b border-default px-4 py-2.5">
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <span class="size-2 rounded-full" :class="statusDot" aria-hidden="true" />
          <h2 class="text-xs font-semibold text-highlighted">{{ t('workflow.debug.title') }}</h2>
          <UBadge color="neutral" variant="soft" size="xs">{{
            t(`workflow.debug.status_${snapshot.status.replace('-', '_')}`)
          }}</UBadge>
        </div>
        <p class="mt-0.5 truncate font-mono text-[10px] text-dimmed">
          {{ snapshot.graphId || '—' }} / {{ snapshot.nodeId || t('workflow.debug.waiting') }}
        </p>
      </div>
      <UButton
        v-if="paused"
        data-testid="workflow-debug-step"
        :label="t('workflow.debug.step')"
        icon="i-tabler-player-track-next"
        size="xs"
        @click="emit('step')"
      />
      <UButton
        v-if="paused"
        data-testid="workflow-debug-continue"
        :label="t('workflow.debug.continue')"
        icon="i-tabler-player-play"
        color="neutral"
        variant="soft"
        size="xs"
        @click="emit('continue')"
      />
      <UButton
        v-else-if="!completed"
        data-testid="workflow-debug-pause"
        :label="t('workflow.debug.pause')"
        icon="i-tabler-player-pause"
        color="neutral"
        variant="soft"
        size="xs"
        @click="emit('pause')"
      />
      <UButton
        v-if="!completed"
        data-testid="workflow-debug-stop"
        :label="t('workflow.action.stop')"
        icon="i-tabler-square"
        color="error"
        variant="soft"
        size="xs"
        @click="emit('stop')"
      />
      <UButton
        icon="i-tabler-x"
        color="neutral"
        variant="ghost"
        size="xs"
        :aria-label="t('workflow.debug.close')"
        @click="emit('close')"
      />
    </header>

    <div class="grid min-h-0 flex-1 gap-4 overflow-y-auto px-4 py-3 lg:grid-cols-4">
      <DebugFactList :title="t('workflow.debug.inputs')" :values="snapshot.inputs" />
      <DebugFactList :title="t('workflow.debug.state')" :values="stateValues" />
      <section class="min-w-0">
        <h3 class="mb-2 text-[10px] font-semibold uppercase tracking-wide text-dimmed">
          {{ t('workflow.debug.queue') }}
        </h3>
        <p v-if="!snapshot.queue.length" class="text-[11px] text-muted">
          {{ t('workflow.debug.empty') }}
        </p>
        <ol v-else class="space-y-1">
          <li
            v-for="(entry, index) in snapshot.queue"
            :key="`${entry.graphId}:${entry.nodeId}:${index}`"
            class="truncate rounded bg-elevated/50 px-2 py-1 font-mono text-[10px] text-toned"
          >
            {{ entry.nodeId }}
          </li>
        </ol>
      </section>
      <section class="min-w-0">
        <h3 class="mb-2 text-[10px] font-semibold uppercase tracking-wide text-dimmed">
          {{ t('workflow.debug.outputs') }}
        </h3>
        <p v-if="!outputFacts.length" class="text-[11px] text-muted">
          {{ t('workflow.debug.empty') }}
        </p>
        <div v-else class="space-y-1">
          <div
            v-for="fact in outputFacts"
            :key="fact.key"
            class="rounded bg-elevated/50 px-2 py-1.5"
          >
            <p class="truncate font-mono text-[10px] text-toned">{{ fact.key }}</p>
            <p class="truncate text-[9px] text-dimmed">
              {{ fact.value.representation }} · {{ fact.value.size }} B
            </p>
          </div>
        </div>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DebugSnapshot } from '@/app/transport/workflow'
import type { DebugValueView } from '@bindings/github.com/yottaapp/yotta/internal/workflow/compiler/models.js'

const props = defineProps<{ snapshot: DebugSnapshot }>()
const emit = defineEmits<{ continue: []; pause: []; step: []; stop: []; close: [] }>()
const { t } = useI18n()

const paused = computed(() => props.snapshot.status === 'paused')
const completed = computed(() => props.snapshot.status === 'completed')
const statusDot = computed(() => {
  if (completed.value) return 'bg-dimmed'
  if (paused.value) return 'bg-warning'
  return 'bg-primary animate-pulse motion-reduce:animate-none'
})
const stateValues = computed<Record<string, DebugValueView>>(() =>
  Object.fromEntries(
    Object.entries(props.snapshot.state).flatMap(([name, state]) =>
      state ? [[name, state.value] as const] : [],
    ),
  ),
)
const outputFacts = computed(() =>
  Object.entries(props.snapshot.outputs).flatMap(([nodeId, outputs]) =>
    Object.entries(outputs ?? {}).flatMap(([portId, value]) =>
      value ? [{ key: `${nodeId}.${portId}`, value }] : [],
    ),
  ),
)

const DebugFactList = defineComponent({
  props: {
    title: { type: String, required: true },
    values: {
      type: Object as PropType<Record<string, DebugValueView | undefined>>,
      required: true,
    },
  },
  setup(listProps) {
    return () =>
      h('section', { class: 'min-w-0' }, [
        h(
          'h3',
          { class: 'mb-2 text-[10px] font-semibold uppercase tracking-wide text-dimmed' },
          listProps.title,
        ),
        ...(!Object.keys(listProps.values).length
          ? [h('p', { class: 'text-[11px] text-muted' }, t('workflow.debug.empty'))]
          : Object.entries(listProps.values).flatMap(([name, value]) =>
              value
                ? [
                    h('div', { class: 'mb-1 rounded bg-elevated/50 px-2 py-1.5' }, [
                      h('p', { class: 'truncate font-mono text-[10px] text-toned' }, name),
                      h(
                        'p',
                        { class: 'truncate text-[9px] text-dimmed' },
                        value.digest
                          ? `${value.representation} · ${value.size} B · ${value.digest.slice(0, 18)}`
                          : `${value.representation} · ${value.size} B · ${t('workflow.debug.redacted')}`,
                      ),
                    ]),
                  ]
                : [],
            )),
      ])
  },
})
</script>
