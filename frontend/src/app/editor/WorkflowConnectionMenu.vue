<template>
  <div
    data-testid="workflow-connection-menu"
    class="absolute z-30 flex w-80 max-h-[26rem] flex-col overflow-hidden rounded-xl border border-default bg-default shadow-2xl"
    :style="{ left: `${position.x}px`, top: `${position.y}px` }"
    @mousedown.stop
    @click.stop
  >
    <div class="border-b border-default px-3 py-3">
      <div class="flex items-start gap-2">
        <div class="min-w-0 flex-1">
          <h3 class="text-xs font-semibold text-highlighted">
            {{ t('workflow.connection.title') }}
          </h3>
          <p class="mt-1 text-[11px] leading-4 text-muted">
            {{
              showAll ? t('workflow.connection.all_hint') : t('workflow.connection.compatible_hint')
            }}
          </p>
        </div>
        <UButton
          icon="i-tabler-x"
          color="neutral"
          variant="ghost"
          size="xs"
          :aria-label="t('common.close')"
          @click="emit('close')"
        />
      </div>
      <UInput
        ref="searchInput"
        v-model="query"
        data-testid="workflow-connection-search"
        icon="i-tabler-search"
        size="sm"
        class="mt-3"
        :placeholder="t('workflow.connection.search')"
      />
      <p
        v-if="error"
        data-testid="workflow-connection-error"
        class="mt-2 text-[11px] leading-4 text-error"
      >
        {{ error }}
      </p>
    </div>

    <div class="min-h-0 flex-1 overflow-y-auto p-2">
      <div v-if="visibleCandidates.length" class="space-y-1">
        <button
          v-for="candidate in visibleCandidates"
          :key="candidate.key"
          type="button"
          data-testid="workflow-connection-candidate"
          :data-node-type-id="candidate.nodeTypeId"
          :data-port-id="candidate.handle?.portId"
          class="flex h-auto w-full items-center justify-start gap-2 rounded-md px-2.5 py-2 text-left hover:bg-elevated focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
          @click.stop="emit('select', candidate)"
        >
          <UIcon
            :name="`i-tabler-${candidate.icon || 'box'}`"
            class="size-4 shrink-0 text-primary"
          />
          <span class="min-w-0 flex-1">
            <span class="block truncate text-xs font-medium text-toned">{{ candidate.title }}</span>
            <span class="block truncate font-mono text-[10px] text-dimmed">
              {{
                candidate.actionHint
                  ? candidate.actionHint
                  : candidate.handle
                    ? t('workflow.connection.via_port', { port: candidate.handle.portId })
                    : t('workflow.connection.add_only')
              }}
            </span>
          </span>
          <UBadge v-if="candidate.conversionKind" color="warning" variant="soft" size="sm">
            {{ t(`workflow.connection.conversion_${candidate.conversionKind}`) }}
          </UBadge>
          <UBadge v-else-if="candidate.match" color="neutral" variant="soft" size="sm">
            {{ t(`workflow.connection.match_${candidate.match.replace('-', '_')}`) }}
          </UBadge>
        </button>
      </div>
      <div v-else class="px-3 py-8 text-center text-xs text-muted">
        {{ t('workflow.connection.no_results') }}
      </div>
    </div>

    <div class="border-t border-default p-2">
      <UButton
        color="neutral"
        variant="ghost"
        size="sm"
        class="w-full"
        :icon="showAll ? 'i-tabler-filter' : 'i-tabler-list'"
        :label="
          showAll ? t('workflow.connection.show_compatible') : t('workflow.connection.show_all')
        "
        @click="showAll = !showAll"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ParsedHandle } from './graphHandles'

export interface WorkflowConnectionCandidate {
  key: string
  nodeTypeId: string
  title: string
  icon?: string
  searchText: string
  handle?: ParsedHandle
  match?: 'exact' | 'assignable' | 'generic-bind'
  conversionKind?: 'lossless' | 'lossy' | 'parser'
  promoteState?: boolean
  actionHint?: string
}

const props = defineProps<{
  position: { x: number; y: number }
  compatibleCandidates: WorkflowConnectionCandidate[]
  allCandidates: WorkflowConnectionCandidate[]
  error?: string
}>()
const emit = defineEmits<{
  select: [candidate: WorkflowConnectionCandidate]
  close: []
}>()
const { t } = useI18n()
const query = ref('')
const showAll = ref(false)
const searchInput = ref<{ inputRef?: HTMLInputElement } | null>(null)

const visibleCandidates = computed(() => {
  const normalized = query.value.trim().toLocaleLowerCase()
  const candidates = showAll.value ? props.allCandidates : props.compatibleCandidates
  return normalized
    ? candidates.filter((candidate) => candidate.searchText.includes(normalized))
    : candidates
})

onMounted(() => {
  void nextTick(() => searchInput.value?.inputRef?.focus())
})
</script>
