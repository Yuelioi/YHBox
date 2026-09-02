<template>
  <!-- THESIS: AI authoring is a workflow-scoped conversation, not a disposable form. -->
  <aside
    data-testid="ai-workflow-review-panel"
    class="flex h-full w-full min-w-0 flex-col border-l border-default bg-default"
  >
    <header class="shrink-0 border-b border-default px-3 py-2.5">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-sparkles" class="size-4 shrink-0 text-primary" />
        <div class="min-w-0 flex-1">
          <h2 class="truncate text-sm font-semibold text-highlighted">
            {{ t('workflow.ai.conversation_title') }}
          </h2>
          <p class="truncate text-[10px] text-dimmed">{{ t('workflow.ai.workflow_isolation') }}</p>
        </div>
        <UButton
          data-testid="ai-conversation-new"
          icon="i-tabler-message-plus"
          color="neutral"
          variant="ghost"
          size="xs"
          :disabled="busy"
          :aria-label="t('workflow.ai.new_conversation')"
          :title="t('workflow.ai.new_conversation')"
          @click="createConversation"
        />
        <UButton
          data-testid="ai-conversation-delete"
          icon="i-tabler-trash"
          color="error"
          variant="ghost"
          size="xs"
          :disabled="busy || !activeConversationId"
          :aria-label="t('workflow.ai.delete_conversation')"
          :title="t('workflow.ai.delete_conversation')"
          @click="deleteConversation"
        />
        <UButton
          icon="i-tabler-x"
          color="neutral"
          variant="ghost"
          size="xs"
          :aria-label="t('workflow.ai.close')"
          @click="emit('close')"
        />
      </div>
      <div class="mt-2 grid grid-cols-2 gap-2">
        <AdaptiveSelect
          data-testid="ai-conversation-select"
          :model-value="activeConversationId"
          :items="conversationOptions"
          value-key="value"
          label-key="label"
          width-mode="fill"
          :placeholder="t('workflow.ai.conversation_history')"
          :disabled="loading || busy"
          @update:model-value="selectConversation"
        />
        <AdaptiveSelect
          v-if="profileOptions.length"
          data-testid="ai-workflow-profile"
          v-model="slot"
          :items="profileOptions"
          value-key="value"
          label-key="label"
          width-mode="fill"
          :disabled="busy"
        />
        <UButton
          v-else
          size="xs"
          color="neutral"
          variant="soft"
          icon="i-tabler-settings"
          :label="t('workflow.ai.configure_profile')"
          @click="openAISettings"
        />
      </div>
    </header>

    <div ref="transcript" class="min-h-0 flex-1 overflow-y-auto px-3 py-4" aria-live="polite">
      <div
        v-if="loading"
        class="flex h-full items-center justify-center gap-2 text-xs text-muted"
        role="status"
      >
        <UIcon name="i-tabler-loader-2" class="size-4 animate-spin motion-reduce:animate-none" />
        {{ t('workflow.ai.loading_conversation') }}
      </div>
      <div
        v-else-if="!visibleMessages.length"
        class="flex h-full flex-col items-center justify-center px-6 text-center"
      >
        <span class="grid size-10 place-items-center rounded-xl bg-primary/10 text-primary"
          ><UIcon name="i-tabler-message-chatbot" class="size-5"
        /></span>
        <p class="mt-3 text-sm font-medium text-highlighted">{{ t('workflow.ai.empty_title') }}</p>
        <p class="mt-1 max-w-64 text-[11px] leading-5 text-muted">
          {{ t('workflow.ai.empty_hint') }}
        </p>
      </div>
      <div v-else class="space-y-5">
        <article
          v-for="message in visibleMessages"
          :key="message.id"
          class="flex min-w-0 items-start gap-2.5"
          :class="message.role === 'user' ? 'pl-5' : 'pr-2'"
        >
          <span
            class="mt-0.5 grid size-7 shrink-0 place-items-center rounded-lg text-[10px] font-semibold"
            :class="
              message.role === 'user'
                ? 'order-2 bg-elevated text-toned'
                : 'bg-primary/12 text-primary'
            "
            aria-hidden="true"
          >
            <UIcon
              v-if="message.role === 'assistant'"
              name="i-tabler-sparkles"
              class="size-3.5"
            /><template v-else>{{ t('workflow.ai.you') }}</template>
          </span>
          <div
            class="min-w-0 flex-1"
            :class="message.role === 'user' ? 'rounded-xl bg-elevated/70 px-3 py-2.5' : 'pt-1'"
          >
            <p class="whitespace-pre-wrap break-words text-xs leading-5 text-toned">
              {{ messageText(message) }}
            </p>
            <div
              v-if="message.review"
              class="mt-3 overflow-hidden rounded-xl border border-default bg-elevated/25"
            >
              <div class="flex items-center justify-between gap-2 px-3 py-2.5">
                <div class="min-w-0">
                  <p class="truncate text-[11px] font-semibold text-highlighted">
                    {{ t('workflow.ai.review') }}
                  </p>
                  <p class="mt-0.5 text-[10px] text-muted">
                    {{ t('workflow.ai.change_count', { n: message.review.changes.length }) }} ·
                    {{ message.review.baseRevision }} → {{ message.review.newRevision }}
                  </p>
                </div>
                <UBadge :color="reviewStatusColor(message.review)" variant="soft" size="xs">{{
                  t(`workflow.ai.status.${message.review.status}`)
                }}</UBadge>
              </div>
              <details class="border-t border-default">
                <summary class="cursor-pointer px-3 py-2 text-[10px] font-medium text-muted">
                  {{ t('workflow.ai.review_details') }}
                </summary>
                <div class="space-y-2 border-t border-default px-3 py-2.5">
                  <div
                    v-for="change in message.review.changes"
                    :key="change.index"
                    class="flex items-start gap-2 text-[10px]"
                  >
                    <span class="shrink-0 font-mono text-primary">{{ change.kind }}</span
                    ><span class="min-w-0 break-all text-muted">{{ change.target }}</span>
                  </div>
                  <p v-if="!message.review.changes.length" class="text-[10px] text-muted">
                    {{ t('workflow.ai.no_changes') }}
                  </p>
                  <p class="text-[10px] text-dimmed">
                    {{ message.review.usage.toolCalls }} {{ t('workflow.ai.tool_calls') }} ·
                    {{ message.review.usage.wallTimeMillis }} ms
                  </p>
                </div>
              </details>
              <div
                v-if="message.review.status === 'proposed'"
                class="grid grid-cols-2 gap-2 border-t border-default p-2"
              >
                <UButton
                  size="xs"
                  color="neutral"
                  variant="soft"
                  :label="t('workflow.ai.reject')"
                  :disabled="busy"
                  @click="reject(message.review)"
                />
                <UButton
                  size="xs"
                  icon="i-tabler-check"
                  :label="t('workflow.ai.accept')"
                  :loading="busyReviewId === message.review.reviewId"
                  @click="accept(message.review)"
                />
              </div>
            </div>
          </div>
        </article>
        <article v-if="busy" class="flex items-start gap-2.5 pr-2">
          <span
            class="mt-0.5 grid size-7 shrink-0 place-items-center rounded-lg bg-primary/12 text-primary"
            ><UIcon name="i-tabler-sparkles" class="size-3.5"
          /></span>
          <div class="min-w-0 flex-1 pt-1">
            <div class="flex items-center gap-2 text-xs text-muted" role="status">
              <UIcon
                name="i-tabler-loader-2"
                class="size-3.5 animate-spin motion-reduce:animate-none"
              />{{ progressLabel }}
            </div>
            <ol v-if="progressEvents.length" class="mt-2 space-y-1 text-[10px] text-dimmed">
              <li
                v-for="(event, index) in progressEvents.slice(-4)"
                :key="`${event.kind}:${index}`"
              >
                {{ progressEventLabel(event) }}
              </li>
            </ol>
          </div>
        </article>
      </div>
    </div>

    <footer class="shrink-0 border-t border-default p-3">
      <div
        v-if="failure"
        class="mb-2 rounded-lg border border-error/35 bg-error/10 px-3 py-2 text-[11px] leading-5 text-error"
        role="alert"
      >
        {{ failure }}
      </div>
      <div
        v-if="dirty"
        class="mb-2 rounded-lg border border-warning/35 bg-warning/10 px-3 py-2 text-[11px] text-warning"
      >
        {{ t('workflow.ai.save_first') }}
      </div>
      <div
        class="rounded-xl border border-default bg-elevated/35 p-2 focus-within:border-primary/55"
      >
        <UTextarea
          v-model="instruction"
          data-testid="ai-conversation-composer"
          :placeholder="t('workflow.ai.conversation_placeholder')"
          :rows="3"
          autoresize
          :disabled="busy"
          variant="none"
          @keydown.enter.exact.prevent="send"
        />
        <div class="mt-1 flex items-center justify-between gap-2 px-1">
          <span class="text-[10px] text-dimmed">{{ t('workflow.ai.send_hint') }}</span
          ><UButton
            data-testid="ai-conversation-send"
            size="xs"
            icon="i-tabler-arrow-up"
            :label="t('workflow.ai.send')"
            :loading="busy"
            :disabled="!canSend"
            @click="send"
          />
        </div>
      </div>
    </footer>
  </aside>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import {
  backend,
  type AIConversation,
  type AIConversationMessage,
  type AIConversationProgress,
  type AIConversationSummary,
  type AIWorkflowReview,
} from '@/lib/backend'
import { errorMessage, normalizeError } from '@/lib/invoke'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'
import { eligibleDiagnosticProfiles } from './aiProfileEligibility'

const props = defineProps<{
  workflowId: string
  baseRevision: number
  dirty: boolean
  runId?: string
}>()
const emit = defineEmits<{ close: []; accepted: [review: AIWorkflowReview] }>()
const { t, te } = useI18n()
const router = useRouter()
const { confirm } = useConfirm()
const settings = useSettingsStore()
const transcript = ref<HTMLElement | null>(null)
const summaries = ref<AIConversationSummary[]>([])
const conversation = ref<AIConversation | null>(null)
const activeConversationId = ref('')
const instruction = ref('')
const slot = ref('')
const loading = ref(true)
const busy = ref(false)
const busyReviewId = ref('')
const failure = ref('')
const optimisticMessage = ref<AIConversationMessage | null>(null)
const progressEvents = ref<AIConversationProgress[]>([])
let stopProgress: (() => void) | undefined

const profiles = computed(() => eligibleDiagnosticProfiles(settings.data?.ai.profiles ?? []))
const profileOptions = computed(() =>
  profiles.value.map((profile) => ({
    label: `${profile.label} (${profile.model})`,
    value: profile.slot,
  })),
)
const conversationOptions = computed(() =>
  summaries.value.map((item) => ({
    label: item.title || t('workflow.ai.new_conversation'),
    value: item.id,
  })),
)
const visibleMessages = computed(() =>
  optimisticMessage.value
    ? [...(conversation.value?.messages ?? []), optimisticMessage.value]
    : (conversation.value?.messages ?? []),
)
const canSend = computed(
  () =>
    !loading.value &&
    !busy.value &&
    !props.dirty &&
    Boolean(slot.value && activeConversationId.value && instruction.value.trim()),
)
const progressLabel = computed(() => {
  const key = `workflow.ai.progress.${progressEvents.value.at(-1)?.kind ?? 'started'}`
  return te(key) ? t(key) : t('workflow.ai.progress.working')
})

onMounted(async () => {
  if (!settings.loaded) await settings.load()
  const preferred = settings.data?.ai.roles?.diagnostics ?? ''
  slot.value = profiles.value.some((profile) => profile.slot === preferred)
    ? preferred
    : (profiles.value[0]?.slot ?? '')
  stopProgress = backend.events.onAIConversationProgress((event) => {
    if (event?.conversationId === activeConversationId.value) {
      progressEvents.value.push(event)
      void scrollToBottom()
    }
  })
  await loadConversations()
  if (props.runId) instruction.value = t('workflow.ai.diagnose_instruction')
})
onBeforeUnmount(() => stopProgress?.())

async function loadConversations() {
  loading.value = true
  try {
    summaries.value = await backend.ai.listConversations(props.workflowId)
    if (!summaries.value.length) {
      const created = await backend.ai.createConversation(props.workflowId)
      summaries.value = [toSummary(created)]
    }
    await selectConversation(summaries.value[0]!.id)
  } catch (error) {
    failure.value = stageError(error, 'load')
  } finally {
    loading.value = false
  }
}
async function createConversation() {
  if (busy.value) return
  try {
    const created = await backend.ai.createConversation(props.workflowId)
    summaries.value = [
      toSummary(created),
      ...summaries.value.filter((item) => item.id !== created.id),
    ]
    activeConversationId.value = created.id
    conversation.value = created
    instruction.value = ''
    progressEvents.value = []
  } catch (error) {
    failure.value = stageError(error, 'create')
  }
}
async function selectConversation(value: string) {
  if (!value) return
  activeConversationId.value = value
  conversation.value = await backend.ai.getConversation(props.workflowId, value)
  progressEvents.value = []
  await scrollToBottom()
}
async function deleteConversation() {
  if (!activeConversationId.value || busy.value) return
  const currentTitle = conversation.value?.title || t('workflow.ai.new_conversation')
  if (
    (await confirm({
      title: t('workflow.ai.delete_conversation_title', { title: currentTitle }),
      description: t('workflow.ai.delete_conversation_hint'),
      confirmText: t('workflow.ai.delete_conversation'),
      color: 'error',
    })) !== true
  )
    return
  try {
    await backend.ai.deleteConversation(props.workflowId, activeConversationId.value)
    summaries.value = await backend.ai.listConversations(props.workflowId)
    conversation.value = null
    activeConversationId.value = ''
    if (summaries.value.length) await selectConversation(summaries.value[0]!.id)
    else await createConversation()
  } catch (error) {
    failure.value = stageError(error, 'delete')
  }
}
async function send() {
  if (!canSend.value) return
  const content = instruction.value.trim()
  instruction.value = ''
  failure.value = ''
  progressEvents.value = []
  optimisticMessage.value = {
    id: `pending:${Date.now()}`,
    role: 'user',
    content,
    createdAt: new Date().toISOString(),
  }
  busy.value = true
  await scrollToBottom()
  try {
    conversation.value = await backend.ai.sendConversationMessage(
      slot.value,
      props.workflowId,
      activeConversationId.value,
      props.baseRevision,
      content,
      props.runId ?? '',
    )
    summaries.value = await backend.ai.listConversations(props.workflowId)
  } catch (error) {
    failure.value = stageError(error, 'send')
    const refreshed = await backend.ai
      .getConversation(props.workflowId, activeConversationId.value)
      .catch(() => conversation.value)
    conversation.value = refreshed
    const persistedFailure = refreshed?.messages.at(-1)
    if (persistedFailure?.problemId) failure.value = messageText(persistedFailure)
  } finally {
    optimisticMessage.value = null
    busy.value = false
    await scrollToBottom()
  }
}
async function accept(review: AIWorkflowReview) {
  if (
    review.permissions.added.length &&
    (await confirm({
      title: t('workflow.ai.permission_confirm_title'),
      description: t('workflow.ai.permission_confirm', { n: review.permissions.added.length }),
      confirmText: t('workflow.ai.accept'),
      color: 'warning',
    })) !== true
  )
    return
  busyReviewId.value = review.reviewId
  try {
    const accepted = await backend.ai.acceptWorkflowProposal(review.reviewId)
    replaceReview(accepted)
    emit('accepted', accepted)
  } catch (error) {
    failure.value = stageError(error, 'accept')
  } finally {
    busyReviewId.value = ''
  }
}
async function reject(review: AIWorkflowReview) {
  busyReviewId.value = review.reviewId
  try {
    replaceReview(await backend.ai.rejectWorkflowProposal(review.reviewId))
  } catch (error) {
    failure.value = stageError(error, 'reject')
  } finally {
    busyReviewId.value = ''
  }
}
function replaceReview(review: AIWorkflowReview) {
  for (const message of conversation.value?.messages ?? [])
    if (message.reviewId === review.reviewId) message.review = review
}
function messageText(message: AIConversationMessage) {
  if (!message.problemId) return message.content
  const key = `error.${message.problemId}`
  return te(key) ? t(key) : t('error.UNKNOWN_ERROR')
}
function stageError(
  error: unknown,
  stage: 'load' | 'create' | 'send' | 'accept' | 'reject' | 'delete',
) {
  return normalizeError(error).id ? errorMessage(error) : t(`workflow.ai.errors.${stage}`)
}
function reviewStatusColor(review: AIWorkflowReview): 'primary' | 'success' | 'error' {
  return review.status === 'accepted'
    ? 'success'
    : review.status === 'proposed'
      ? 'primary'
      : 'error'
}
function progressEventLabel(event: AIConversationProgress) {
  if (event.kind === 'tool' && event.facts?.name)
    return t('workflow.ai.progress.tool', { name: event.facts.name })
  const key = `workflow.ai.progress.${event.kind}`
  return te(key) ? t(key) : t('workflow.ai.progress.working')
}
function toSummary(value: AIConversation): AIConversationSummary {
  return {
    id: value.id,
    workflowId: value.workflowId,
    title: value.title,
    createdAt: value.createdAt,
    updatedAt: value.updatedAt,
    messageCount: value.messages.length,
  }
}
async function scrollToBottom() {
  await nextTick()
  if (transcript.value) transcript.value.scrollTop = transcript.value.scrollHeight
}
function openAISettings() {
  void router.push({ path: '/settings', query: { section: 'ai' } })
}
</script>
