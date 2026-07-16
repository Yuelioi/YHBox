<template>
  <aside
    data-testid="ai-workflow-review-panel"
    class="flex h-full w-[380px] shrink-0 flex-col border-l border-default bg-default"
  >
    <header class="border-b border-default px-4 py-3">
      <div class="flex items-center justify-between gap-3">
        <div class="min-w-0">
          <h2 class="text-sm font-semibold text-highlighted">{{ t('workflow31.ai.title') }}</h2>
          <p class="mt-0.5 text-[10px] leading-4 text-dimmed">{{ t('workflow31.ai.hint') }}</p>
        </div>
        <UButton
          icon="i-tabler-x"
          color="neutral"
          variant="ghost"
          size="xs"
          :aria-label="t('workflow31.ai.close')"
          @click="emit('close')"
        />
      </div>
    </header>

    <div class="flex-1 overflow-y-auto">
      <section class="space-y-3 border-b border-default p-4">
        <UFormField :label="t('workflow31.ai.profile')">
          <USelect
            v-model="slot"
            :items="profileOptions"
            value-key="value"
            label-key="label"
            :disabled="busy || review?.status === 'proposed'"
          />
        </UFormField>
        <UFormField :label="t('workflow31.ai.request')" :help="t('workflow31.ai.request_help')">
          <UTextarea
            v-model="instruction"
            :placeholder="t('workflow31.ai.request_placeholder')"
            :rows="4"
            autoresize
            :disabled="busy || review?.status === 'proposed'"
          />
        </UFormField>
        <div
          v-if="dirty"
          class="rounded-lg border border-warning/35 bg-warning/10 px-3 py-2 text-xs text-warning"
        >
          {{ t('workflow31.ai.save_first') }}
        </div>
        <div
          v-else-if="profileOptions.length === 0"
          class="rounded-lg border border-default bg-elevated/35 px-3 py-2 text-xs text-muted"
        >
          {{ t('workflow31.ai.no_profile') }}
        </div>
        <UButton
          class="w-full justify-center"
          :label="review ? t('workflow31.ai.retry') : t('workflow31.ai.propose')"
          icon="i-tabler-sparkles"
          :loading="busy"
          :disabled="!canPropose"
          @click="propose"
        />
      </section>

      <div
        v-if="failure"
        class="m-4 rounded-lg border border-error/35 bg-error/10 px-3 py-2 text-xs leading-5 text-error"
        role="alert"
      >
        {{ failure }}
      </div>

      <section v-if="review" class="space-y-4 p-4" aria-live="polite">
        <div class="flex items-start justify-between gap-3">
          <div>
            <p class="text-xs font-semibold text-highlighted">{{ t('workflow31.ai.review') }}</p>
            <p class="mt-1 text-[11px] leading-5 text-muted">{{ review.summary }}</p>
          </div>
          <UBadge :color="statusColor" variant="soft">{{
            t(`workflow31.ai.status.${review.status}`)
          }}</UBadge>
        </div>

        <dl class="grid grid-cols-2 gap-2 text-[10px]">
          <div class="rounded-lg bg-elevated/45 p-2.5">
            <dt class="text-dimmed">{{ t('workflow31.ai.revision') }}</dt>
            <dd class="mt-1 font-mono text-toned">
              {{ review.baseRevision }} -> {{ review.newRevision }}
            </dd>
          </div>
          <div class="rounded-lg bg-elevated/45 p-2.5">
            <dt class="text-dimmed">{{ t('workflow31.ai.candidate') }}</dt>
            <dd class="mt-1 truncate font-mono text-toned" :title="review.candidateHash">
              {{ shortHash(review.candidateHash) }}
            </dd>
          </div>
        </dl>

        <section>
          <h3 class="text-xs font-semibold text-highlighted">{{ t('workflow31.ai.changes') }}</h3>
          <div class="mt-2 space-y-1.5">
            <div
              v-for="change in review.changes"
              :key="change.index"
              class="rounded-lg bg-elevated/35 px-3 py-2"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="font-mono text-[10px] text-primary">{{ change.kind }}</span>
                <UIcon
                  v-if="change.sensitive"
                  name="i-tabler-eye-off"
                  class="size-3.5 text-warning"
                  :aria-label="t('workflow31.ai.redacted')"
                />
              </div>
              <p class="mt-1 break-all text-[10px] text-muted">{{ change.target }}</p>
            </div>
          </div>
        </section>

        <section>
          <div class="flex items-center justify-between">
            <h3 class="text-xs font-semibold text-highlighted">
              {{ t('workflow31.ai.diagnostics') }}
            </h3>
            <span class="font-mono text-[10px] text-dimmed">{{ review.diagnostics.length }}</span>
          </div>
          <p v-if="review.diagnostics.length === 0" class="mt-2 text-[11px] text-success">
            {{ t('workflow31.ai.no_diagnostics') }}
          </p>
          <div v-else class="mt-2 space-y-1.5">
            <div
              v-for="(diagnostic, index) in review.diagnostics"
              :key="`${diagnostic.code}:${index}`"
              class="rounded-lg border border-warning/30 bg-warning/10 px-3 py-2 text-[10px] text-warning"
            >
              <span class="font-mono">{{ diagnostic.code }}</span>
              <span v-if="diagnostic.nodeId" class="ml-2 text-muted">{{ diagnostic.nodeId }}</span>
            </div>
          </div>
        </section>

        <section>
          <div class="flex items-center justify-between">
            <h3 class="text-xs font-semibold text-highlighted">
              {{ t('workflow31.ai.permissions') }}
            </h3>
            <span
              class="font-mono text-[10px]"
              :class="review.permissions.added.length ? 'text-warning' : 'text-dimmed'"
              >+{{ review.permissions.added.length }} / -{{
                review.permissions.removed.length
              }}</span
            >
          </div>
          <p
            v-if="review.permissions.added.length === 0 && review.permissions.removed.length === 0"
            class="mt-2 text-[11px] text-success"
          >
            {{ t('workflow31.ai.no_permission_change') }}
          </p>
          <div v-else class="mt-2 space-y-1.5">
            <div
              v-for="permission in review.permissions.added"
              :key="`add:${permission.capabilityId}:${permission.targetSlot}`"
              class="rounded-lg border border-warning/30 bg-warning/10 px-3 py-2 text-[10px]"
            >
              <p class="break-all font-mono text-warning">+ {{ permission.capabilityId }}</p>
              <p class="mt-1 text-muted">
                {{ permission.targetSlot
                }}<template v-if="permission.credentialSlot">
                  / {{ permission.credentialSlot }}</template
                >
              </p>
            </div>
            <div
              v-for="permission in review.permissions.removed"
              :key="`remove:${permission.capabilityId}:${permission.targetSlot}`"
              class="rounded-lg bg-elevated/35 px-3 py-2 text-[10px]"
            >
              <p class="break-all font-mono text-muted">- {{ permission.capabilityId }}</p>
            </div>
          </div>
        </section>

        <details class="rounded-lg border border-default bg-elevated/20">
          <summary class="cursor-pointer px-3 py-2 text-xs font-medium text-toned">
            {{ t('workflow31.ai.audit') }}
          </summary>
          <div class="space-y-2 border-t border-default p-3">
            <div class="grid grid-cols-3 gap-2 text-[10px] text-muted">
              <span>{{ review.usage.iterations }} {{ t('workflow31.ai.turns') }}</span>
              <span>{{ review.usage.toolCalls }} {{ t('workflow31.ai.tool_calls') }}</span>
              <span>{{ review.usage.wallTimeMillis }} ms</span>
            </div>
            <ol class="space-y-1.5">
              <li
                v-for="event in review.trace"
                :key="event.sequence"
                class="grid grid-cols-[24px_1fr] gap-2 text-[10px]"
              >
                <span class="font-mono text-dimmed">{{ event.sequence }}</span>
                <span class="font-mono text-muted">{{ event.kind }}</span>
              </li>
            </ol>
            <p class="text-[10px] leading-4 text-dimmed">
              {{ t('workflow31.ai.audit_redaction') }}
            </p>
          </div>
        </details>

        <div
          v-if="review.status === 'proposed'"
          class="grid grid-cols-2 gap-2 border-t border-default pt-4"
        >
          <UButton
            :label="t('workflow31.ai.reject')"
            color="neutral"
            variant="soft"
            :disabled="busy"
            @click="reject"
          />
          <UButton
            :label="t('workflow31.ai.accept')"
            icon="i-tabler-check"
            :loading="busy"
            @click="accept"
          />
        </div>
      </section>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type AIWorkflowReview } from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'

const props = defineProps<{ workflowId: string; baseRevision: number; dirty: boolean }>()
const emit = defineEmits<{ close: []; accepted: [review: AIWorkflowReview] }>()
const { t } = useI18n()
const settings = useSettingsStore()
const instruction = ref('')
const slot = ref('')
const busy = ref(false)
const failure = ref('')
const review = ref<AIWorkflowReview | null>(null)

const profiles = computed(() =>
  (settings.data?.ai.profiles ?? []).filter(
    (profile) => profile.evaluation === 'approved' && profile.capabilities.toolCalling,
  ),
)
const profileOptions = computed(() =>
  profiles.value.map((profile) => ({
    label: `${profile.label} (${profile.model})`,
    value: profile.slot,
  })),
)
const canPropose = computed(
  () => !busy.value && !props.dirty && Boolean(slot.value && instruction.value.trim()),
)
const statusColor = computed(() => {
  if (review.value?.status === 'accepted') return 'success' as const
  if (review.value?.status === 'stale') return 'warning' as const
  if (review.value?.status === 'rejected') return 'neutral' as const
  return 'primary' as const
})

onMounted(async () => {
  if (!settings.loaded) await settings.load()
  slot.value = profiles.value[0]?.slot ?? ''
})

async function propose(): Promise<void> {
  if (!canPropose.value) return
  busy.value = true
  failure.value = ''
  try {
    review.value =
      (await backend.ai.proposeWorkflow(
        slot.value,
        props.workflowId,
        props.baseRevision,
        instruction.value.trim(),
      )) ?? null
  } catch (error) {
    failure.value = errorText(error)
  } finally {
    busy.value = false
  }
}

async function accept(): Promise<void> {
  if (!review.value || busy.value) return
  if (
    review.value.permissions.added.length > 0 &&
    !window.confirm(
      t('workflow31.ai.permission_confirm', { n: review.value.permissions.added.length }),
    )
  )
    return
  busy.value = true
  failure.value = ''
  try {
    const accepted = await backend.ai.acceptWorkflowProposal(review.value.reviewId)
    if (accepted) {
      review.value = accepted
      emit('accepted', accepted)
    }
  } catch (error) {
    failure.value = errorText(error)
    const refreshed = await backend.ai
      .getWorkflowProposal(review.value.reviewId)
      .catch(() => undefined)
    if (refreshed) review.value = refreshed
  } finally {
    busy.value = false
  }
}

async function reject(): Promise<void> {
  if (!review.value || busy.value) return
  busy.value = true
  failure.value = ''
  try {
    review.value = (await backend.ai.rejectWorkflowProposal(review.value.reviewId)) ?? review.value
  } catch (error) {
    failure.value = errorText(error)
  } finally {
    busy.value = false
  }
}

function shortHash(value: string): string {
  return value.length > 18 ? `${value.slice(0, 18)}...` : value
}

function errorText(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
</script>
