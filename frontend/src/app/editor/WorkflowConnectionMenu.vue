<template>
  <div
    data-testid="workflow-connection-menu"
    class="nowheel nodrag nopan absolute z-30 flex w-80 max-h-[26rem] flex-col overflow-hidden rounded-xl border border-default bg-default shadow-2xl"
    :style="{ left: `${position.x}px`, top: `${position.y}px` }"
    @mousedown.stop
    @click.stop
    @wheel.stop
    @keydown="onListKeydown"
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

    <div ref="resultsElement" class="min-h-0 flex-1 overflow-y-auto p-2">
      <div v-if="visibleCandidates.length" class="space-y-1">
        <button
          v-for="(candidate, index) in visibleCandidates"
          :key="candidate.key"
          type="button"
          data-testid="workflow-connection-candidate"
          :data-node-type-id="candidate.nodeTypeId"
          :data-port-id="candidate.handle?.portId"
          class="flex h-auto w-full items-center justify-start gap-2 rounded-md px-2.5 py-2 text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
          :class="index === activeIndex ? 'bg-primary/10' : 'hover:bg-elevated'"
          @mouseenter="activeIndex = index"
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
          <UKbd v-if="index < NUMBERED_SELECTION_LIMIT" size="xs">{{ index + 1 }}</UKbd>
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
      <UButton
        v-if="visibleCandidates.length < matchingCandidates.length"
        color="neutral"
        variant="soft"
        size="sm"
        class="mt-2 w-full justify-center"
        :label="
          t('workflow.connection.show_more', {
            remaining: matchingCandidates.length - visibleCandidates.length,
          })
        "
        @click="visibleLimit += CANDIDATE_PAGE_SIZE"
      />
    </div>

    <div class="border-t border-default p-2">
      <UButton
        color="neutral"
        variant="ghost"
        size="sm"
        class="w-full justify-center"
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
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ParsedHandle } from './graphHandles'
import {
  NUMBERED_SELECTION_LIMIT,
  moveListSelection,
  numberedSelectionIndex,
} from './listKeyboardSelection'

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
const CANDIDATE_PAGE_SIZE = 80
const visibleLimit = ref(CANDIDATE_PAGE_SIZE)
const searchInput = ref<{ inputRef?: HTMLInputElement } | null>(null)
const resultsElement = ref<HTMLElement | null>(null)
const activeIndex = ref(0)

const matchingCandidates = computed(() => {
  const normalized = query.value.trim().toLocaleLowerCase()
  const candidates = showAll.value ? props.allCandidates : props.compatibleCandidates
  return normalized
    ? candidates.filter((candidate) => candidate.searchText.includes(normalized))
    : candidates
})
const visibleCandidates = computed(() => matchingCandidates.value.slice(0, visibleLimit.value))

watch([query, showAll], () => {
  visibleLimit.value = CANDIDATE_PAGE_SIZE
  activeIndex.value = 0
})

onMounted(() => {
  void nextTick(() => searchInput.value?.inputRef?.focus())
})

function move(delta: number): void {
  activeIndex.value = moveListSelection(activeIndex.value, delta, visibleCandidates.value.length)
  void nextTick(() => {
    resultsElement.value
      ?.querySelectorAll<HTMLElement>('[data-testid="workflow-connection-candidate"]')
      [activeIndex.value]?.scrollIntoView({ block: 'nearest' })
  })
}

function selectActive(): void {
  const candidate = visibleCandidates.value[activeIndex.value]
  if (candidate) emit('select', candidate)
}

function onListKeydown(event: KeyboardEvent): void {
  if (event.isComposing) return
  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    move(event.key === 'ArrowDown' ? 1 : -1)
    return
  }
  if (event.key === 'Enter') {
    event.preventDefault()
    selectActive()
    return
  }
  const index = numberedSelectionIndex(event, visibleCandidates.value.length)
  if (index === undefined) return
  event.preventDefault()
  emit('select', visibleCandidates.value[index]!)
}
</script>
