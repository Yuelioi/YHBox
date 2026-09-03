<template>
  <div class="settings-page">
    <SettingsSection
      :title="t('settingsAI.roles.title')"
      :description="t('settingsAI.roles.hint')"
      icon="i-tabler-stethoscope"
    >
      <div v-if="diagnosticProfileOptions.length" class="flex flex-wrap items-center gap-2">
        <AdaptiveSelect
          data-testid="settings-ai-diagnostic-profile"
          :model-value="diagnosticSlot"
          :items="diagnosticProfileOptions"
          value-key="value"
          label-key="label"
          :placeholder="t('settingsAI.roles.diagnostics_placeholder')"
          @update:model-value="setDiagnosticSlot"
        />
        <UButton
          v-if="diagnosticSlot"
          size="xs"
          color="neutral"
          variant="ghost"
          :label="t('settingsAI.roles.clear')"
          @click="setDiagnosticSlot('')"
        />
      </div>
      <p
        v-else
        data-testid="settings-ai-diagnostic-profile-unavailable"
        class="text-xs leading-5 text-muted"
      >
        {{ t(diagnosticProfileUnavailableKey) }}
      </p>
    </SettingsSection>

    <SettingsSection
      :title="t('settingsAI.authoring.title')"
      :description="t('settingsAI.authoring.hint')"
      icon="i-tabler-adjustments"
    >
      <UFormField
        :label="t('settingsAI.authoring.max_iterations_label')"
        :description="t('settingsAI.authoring.max_iterations_hint')"
      >
        <UInputNumber
          v-model="authoringMaxIterations"
          data-testid="settings-ai-authoring-max-iterations"
          :min="8"
          :max="64"
          :step="1"
          size="sm"
          class="w-full sm:w-48"
        />
      </UFormField>
      <p class="mt-2 text-xs leading-5 text-dimmed">
        {{ t('settingsAI.authoring.not_billing_limit') }}
      </p>
    </SettingsSection>

    <SettingsSection
      :title="t('settingsAI.profiles.title')"
      :description="t('settingsAI.profiles.hint')"
      icon="i-tabler-brain"
    >
      <template #badge>
        <UBadge size="xs" color="neutral" variant="subtle">{{ draft.length }}</UBadge>
      </template>
      <template #actions>
        <UButton size="sm" color="primary" variant="soft" icon="i-tabler-plus" @click="addProfile">
          {{ t('settingsAI.profiles.add') }}
        </UButton>
      </template>

      <div v-if="draft.length" class="settings-collection">
        <article v-for="(profile, index) in draft" :key="profile.slot" class="ai-profile">
          <button
            class="settings-entity-summary cursor-pointer text-left"
            type="button"
            :aria-expanded="expandedSlot === profile.slot"
            @click="toggleExpanded(profile.slot)"
          >
            <span
              class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-default bg-elevated"
            >
              <UIcon :name="providerIcon(profile.provider)" class="size-4 text-toned" />
            </span>
            <span class="min-w-0 flex-1">
              <span class="flex flex-wrap items-center gap-2">
                <span class="truncate text-sm font-medium text-default">
                  {{ profile.label || t('settingsAI.profiles.unnamed') }}
                </span>
                <UBadge
                  v-if="secretStatus[profile.slot]"
                  size="xs"
                  color="success"
                  variant="subtle"
                  icon="i-tabler-key"
                >
                  {{ t('settingsAI.profiles.credential_saved') }}
                </UBadge>
              </span>
              <span class="mt-1 block truncate text-xs text-dimmed">
                {{ providerName(profile.provider) }} ·
                {{ profile.model || t('settingsAI.profiles.model_missing') }} ·
                <code>{{ profile.slot }}</code>
              </span>
            </span>
            <UIcon
              name="i-tabler-chevron-down"
              class="size-4 shrink-0 text-dimmed transition-transform"
              :class="expandedSlot === profile.slot ? 'rotate-180' : ''"
            />
          </button>

          <div v-if="expandedSlot === profile.slot" class="ai-profile__details">
            <div class="grid gap-4 sm:grid-cols-2">
              <UFormField :label="t('settingsAI.profiles.name_label')" required>
                <UInput
                  v-model="profile.label"
                  size="sm"
                  :placeholder="t('settingsAI.profiles.label_placeholder')"
                  @change="commit"
                />
              </UFormField>
              <UFormField
                :label="t('settingsAI.profiles.slot_label')"
                :description="t('settingsAI.profiles.slot_hint')"
              >
                <UInput :model-value="profile.slot" size="sm" disabled class="font-mono" />
              </UFormField>
              <UFormField :label="t('settingsAI.profiles.provider_label')" required>
                <AdaptiveSelect
                  :model-value="profile.provider"
                  :items="providerItems"
                  size="sm"
                  @update:model-value="(value: AIProviderKind) => onProvider(index, value)"
                />
              </UFormField>
              <UFormField
                :label="t('settingsAI.profiles.model_label')"
                :description="t('settingsAI.profiles.model_hint')"
                required
              >
                <UInput
                  v-model="profile.model"
                  size="sm"
                  :placeholder="t('settingsAI.profiles.model_placeholder')"
                  @change="commit"
                />
              </UFormField>
            </div>

            <div v-if="profile.provider !== 'codex-subscription'" class="settings-inset">
              <UFormField
                :label="t('settingsAI.profiles.endpoint_label')"
                :description="t('settingsAI.profiles.endpoint_hint')"
                :error="endpointFieldError(profile)"
                required
              >
                <div class="flex flex-col gap-2 sm:flex-row">
                  <UInput
                    v-model="profile.endpoint"
                    type="url"
                    size="sm"
                    class="min-w-0 flex-1 font-mono"
                    :placeholder="defaultProviderEndpoint(profile.provider)"
                    @change="commitEndpoint(profile)"
                  />
                  <UButton
                    size="sm"
                    color="neutral"
                    variant="soft"
                    icon="i-tabler-restore"
                    @click="restoreProviderEndpoint(profile)"
                  >
                    {{ t('settingsAI.profiles.endpoint_reset') }}
                  </UButton>
                </div>
              </UFormField>
              <div
                v-if="profile.endpoint.trim().toLowerCase().startsWith('http://')"
                class="mt-3 rounded-lg border border-error/30 bg-error/5 p-3"
              >
                <label class="flex items-start justify-between gap-3">
                  <span class="min-w-0">
                    <span class="settings-detail__label block">
                      {{ t('settingsAI.profiles.local_http_title') }}
                    </span>
                    <span class="settings-detail__hint block">
                      {{ t('settingsAI.profiles.local_http_hint') }}
                    </span>
                  </span>
                  <USwitch
                    :model-value="profile.allowLocalHttp"
                    size="sm"
                    @update:model-value="(value: boolean) => setLocalHTTP(index, value)"
                  />
                </label>
                <p class="mt-2 text-xs leading-relaxed text-error">
                  {{ t('settingsAI.profiles.local_http_warning') }}
                </p>
              </div>
            </div>

            <div v-if="profile.provider === 'codex-subscription'" class="settings-inset">
              <div class="flex items-start gap-3">
                <UIcon name="i-tabler-brand-openai" class="mt-0.5 size-4 shrink-0 text-primary" />
                <div class="min-w-0">
                  <p class="settings-detail__label">{{ t('settingsAI.codex.title') }}</p>
                  <p class="settings-detail__hint">{{ t('settingsAI.codex.hint') }}</p>
                  <code class="mt-2 block text-xs text-toned">codex login</code>
                </div>
              </div>
            </div>

            <div class="grid gap-4 sm:grid-cols-[minmax(0,1fr)_auto]">
              <UFormField
                :label="t('settingsAI.profiles.max_tokens_label')"
                :description="t('settingsAI.profiles.max_tokens_hint')"
              >
                <div class="flex flex-wrap items-center gap-3">
                  <UInput
                    v-if="profile.maxOutputTokens === 0"
                    :model-value="t('settingsAI.profiles.max_tokens_unlimited')"
                    disabled
                    size="sm"
                    class="w-full sm:w-48"
                  />
                  <UInputNumber
                    v-else
                    v-model="profile.maxOutputTokens"
                    :min="1"
                    :max="1000000"
                    :step="256"
                    :step-snapping="false"
                    size="sm"
                    class="w-full sm:w-48"
                    @change="commit"
                  />
                  <label class="flex items-center gap-2 text-xs text-toned">
                    <USwitch
                      :model-value="profile.maxOutputTokens === 0"
                      size="sm"
                      @update:model-value="
                        (value: boolean) => setUnlimitedOutputTokens(index, value)
                      "
                    />
                    <span>{{ t('settingsAI.profiles.max_tokens_unlimited') }}</span>
                  </label>
                </div>
              </UFormField>
              <div class="flex items-end pb-1">
                <UBadge size="xs" color="neutral" variant="subtle">
                  {{ t(`settingsAI.evaluation.${profile.evaluation}`) }}
                </UBadge>
              </div>
            </div>

            <div class="settings-inset">
              <p class="settings-detail__label">
                {{ t('settingsAI.capabilities.title') }}
              </p>
              <p class="settings-detail__hint">
                {{ t('settingsAI.capabilities.hint') }}
              </p>
              <div class="mt-3 grid gap-x-5 gap-y-3 sm:grid-cols-2">
                <label
                  v-for="capability in capabilityOptions"
                  :key="capability.key"
                  class="flex items-start justify-between gap-3"
                >
                  <span class="min-w-0">
                    <span class="settings-detail__label block">{{ capability.label }}</span>
                    <span class="settings-detail__hint block">{{ capability.hint }}</span>
                  </span>
                  <USwitch
                    :model-value="profile.capabilities[capability.key]"
                    size="sm"
                    :disabled="
                      profile.provider === 'openai-chat-completions' &&
                      (capability.key === 'toolCalling' || capability.key === 'parallelTools')
                    "
                    @update:model-value="
                      (value: boolean) => onCapability(index, capability.key, value)
                    "
                  />
                </label>
              </div>
            </div>

            <div v-if="profile.capabilities.toolCalling" class="settings-inset">
              <p class="settings-detail__label">{{ t('settingsAI.pricing.title') }}</p>
              <p class="settings-detail__hint">
                {{ t('settingsAI.pricing.hint') }}
              </p>
              <div class="mt-3 grid gap-3 sm:grid-cols-3">
                <UFormField :label="t('settingsAI.pricing.input')">
                  <UInputNumber
                    v-model="profile.pricing.inputMicrounitsPerMillion"
                    :min="0"
                    :max="1000000000000"
                    :step="1"
                    size="sm"
                    class="w-full"
                    @change="commit"
                  />
                </UFormField>
                <UFormField :label="t('settingsAI.pricing.cache_read')">
                  <UInputNumber
                    v-model="profile.pricing.cacheReadMicrounitsPerMillion"
                    :min="0"
                    :max="1000000000000"
                    :step="1"
                    size="sm"
                    class="w-full"
                    @change="commit"
                  />
                </UFormField>
                <UFormField :label="t('settingsAI.pricing.output')">
                  <UInputNumber
                    v-model="profile.pricing.outputMicrounitsPerMillion"
                    :min="0"
                    :max="1000000000000"
                    :step="1"
                    size="sm"
                    class="w-full"
                    @change="commit"
                  />
                </UFormField>
              </div>
            </div>

            <UFormField
              v-if="profile.provider !== 'codex-subscription'"
              :label="t('settingsAI.profiles.apikey_label')"
              :description="t('settingsAI.profiles.apikey_hint')"
            >
              <UInput
                v-model="apiKeys[profile.slot]"
                :type="revealed[profile.slot] ? 'text' : 'password'"
                size="sm"
                :placeholder="
                  secretStatus[profile.slot]
                    ? t('settingsAI.profiles.apikey_replace_placeholder')
                    : t('settingsAI.profiles.apikey_placeholder')
                "
                @change="saveAPIKey(profile)"
              >
                <template #trailing>
                  <UButton
                    :icon="revealed[profile.slot] ? 'i-tabler-eye-off' : 'i-tabler-eye'"
                    variant="link"
                    color="neutral"
                    size="xs"
                    :aria-label="t('settingsAI.profiles.reveal')"
                    @click="revealed[profile.slot] = !revealed[profile.slot]"
                  />
                </template>
              </UInput>
              <template #help>
                <div class="flex flex-wrap items-center gap-2">
                  <span>{{ t('settingsAI.profiles.apikey_secure_hint') }}</span>
                  <UButton
                    v-if="secretStatus[profile.slot]"
                    size="xs"
                    variant="link"
                    color="error"
                    icon="i-tabler-key-off"
                    @click="deleteAPIKey(profile.slot)"
                  >
                    {{ t('settingsAI.profiles.apikey_remove') }}
                  </UButton>
                </div>
              </template>
            </UFormField>

            <div class="flex flex-wrap items-center gap-2 border-t border-default/60 pt-4">
              <UButton
                size="sm"
                variant="soft"
                color="primary"
                icon="i-tabler-bolt"
                :loading="testing[profile.slot]"
                :disabled="!profile.persisted"
                @click="testProfile(profile)"
              >
                {{ t('settingsAI.profiles.test') }}
              </UButton>
              <UButton
                class="ml-auto"
                size="sm"
                variant="ghost"
                color="error"
                icon="i-tabler-trash"
                @click="removeProfile(profile)"
              >
                {{ t('settingsAI.profiles.delete') }}
              </UButton>
            </div>

            <div
              v-if="results[profile.slot]"
              class="flex items-start gap-2 rounded-lg border p-3 text-xs"
              :class="
                results[profile.slot]!.ok
                  ? 'border-success/30 bg-success/5 text-success'
                  : 'border-error/30 bg-error/5 text-error'
              "
              role="status"
            >
              <UIcon
                :name="results[profile.slot]!.ok ? 'i-tabler-circle-check' : 'i-tabler-circle-x'"
                class="mt-0.5 size-4 shrink-0"
              />
              <span v-if="results[profile.slot]!.ok">
                {{
                  t('settingsAI.profiles.test_ok', {
                    model: results[profile.slot]!.resolvedModel,
                  })
                }}
              </span>
              <span v-else>{{ testFailureMessage(profile, results[profile.slot]!) }}</span>
            </div>
          </div>
        </article>
      </div>

      <div v-else class="settings-empty-state">
        <UIcon name="i-tabler-brain" class="size-6 text-dimmed" />
        <p class="text-sm font-medium text-default">{{ t('settingsAI.profiles.empty') }}</p>
        <p class="max-w-md text-center text-xs leading-relaxed text-dimmed">
          {{ t('settingsAI.profiles.empty_hint') }}
        </p>
        <UButton size="sm" color="primary" variant="soft" icon="i-tabler-plus" @click="addProfile">
          {{ t('settingsAI.profiles.add') }}
        </UButton>
      </div>
    </SettingsSection>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@/composables/useAppToast'
import {
  backend,
  type AIModelProfile,
  type AIProfileCapabilities,
  type AIProfileTestResult,
  type AIProviderKind,
} from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import SettingsSection from '@/components/settings/SettingsSection.vue'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'
import { errorMessage } from '@/lib/invoke'
import { aiProviderEndpointIssue } from '@/settings/aiProviderEndpoint'
import {
  eligibleDiagnosticProfiles,
  explainAIProfileEligibility,
} from '@/app/editor/aiProfileEligibility'

interface AIModelProfileDraft extends AIModelProfile {
  persisted: boolean
}

type CapabilityKey = keyof AIProfileCapabilities

const { t } = useI18n()
const toast = useToast()
const { confirm } = useConfirm()
const store = useSettingsStore()
const draft = ref<AIModelProfileDraft[]>([])
const profiles = computed<AIModelProfile[]>(() => store.data?.ai.profiles ?? [])
const diagnosticSlot = computed(() => store.data?.ai.roles?.diagnostics ?? '')
const authoringMaxIterations = computed({
  get: () => store.data?.ai.authoring?.maxIterations ?? 24,
  set: (maxIterations: number) => {
    void store.patchAIAuthoring({ maxIterations })
  },
})
const diagnosticProfileOptions = computed(() =>
  eligibleDiagnosticProfiles(draft.value).map((profile) => ({
    label: `${profile.label} (${profile.model})`,
    value: profile.slot,
  })),
)
const diagnosticProfileUnavailableKey = computed(
  () => `settingsAI.roles.unavailable.${explainAIProfileEligibility(draft.value)}`,
)
const expandedSlot = ref('')
const testing = reactive<Record<string, boolean>>({})
const results = reactive<Record<string, AIProfileTestResult>>({})
const revealed = reactive<Record<string, boolean>>({})
const secretStatus = reactive<Record<string, boolean>>({})
const apiKeys = reactive<Record<string, string>>({})
const endpointValidationVisible = reactive<Record<string, boolean>>({})
const previousOutputTokenLimits = reactive<Record<string, number>>({})
const providerItems = computed(() => [
  { label: t('settingsAI.provider.codex_subscription'), value: 'codex-subscription' },
  { label: t('settingsAI.provider.openai_responses'), value: 'openai-responses' },
  { label: t('settingsAI.provider.openai_chat_completions'), value: 'openai-chat-completions' },
  { label: t('settingsAI.provider.anthropic_messages'), value: 'anthropic-messages' },
])
const capabilityOptions = computed<Array<{ key: CapabilityKey; label: string; hint: string }>>(
  () => [
    {
      key: 'structuredOutput',
      label: t('settingsAI.capabilities.structured_output'),
      hint: t('settingsAI.capabilities.structured_output_hint'),
    },
    {
      key: 'toolCalling',
      label: t('settingsAI.capabilities.tool_calling'),
      hint: t('settingsAI.capabilities.tool_calling_hint'),
    },
    {
      key: 'parallelTools',
      label: t('settingsAI.capabilities.parallel_tools'),
      hint: t('settingsAI.capabilities.parallel_tools_hint'),
    },
    {
      key: 'background',
      label: t('settingsAI.capabilities.background'),
      hint: t('settingsAI.capabilities.background_hint'),
    },
    {
      key: 'zeroRetention',
      label: t('settingsAI.capabilities.zero_retention'),
      hint: t('settingsAI.capabilities.zero_retention_hint'),
    },
  ],
)

watch(
  profiles,
  () => {
    const localProfiles = draft.value.filter((profile) => !profile.persisted)
    const persistedProfiles = profiles.value.map((profile) => ({
      ...profile,
      endpoint: profile.endpoint || defaultProviderEndpoint(profile.provider),
      capabilities: { ...profile.capabilities },
      pricing: { ...profile.pricing },
      evaluationReport: profile.evaluationReport ? { ...profile.evaluationReport } : undefined,
      persisted: true,
    }))
    const persistedSlots = new Set(persistedProfiles.map((profile) => profile.slot))
    draft.value = [
      ...persistedProfiles,
      ...localProfiles.filter((profile) => !persistedSlots.has(profile.slot)),
    ]
    void refreshSecretStatus()
  },
  { immediate: true },
)

const providerName = (provider: AIProviderKind) => {
  switch (provider) {
    case 'openai-responses':
      return t('settingsAI.provider.openai_responses')
    case 'openai-chat-completions':
      return t('settingsAI.provider.openai_chat_completions')
    case 'anthropic-messages':
      return t('settingsAI.provider.anthropic_messages')
    case 'codex-subscription':
      return t('settingsAI.provider.codex_subscription')
  }
}
const providerIcon = (provider: AIProviderKind) =>
  provider === 'anthropic-messages' ? 'i-tabler-letter-a' : 'i-tabler-brand-openai'
const toggleExpanded = (slot: string) =>
  (expandedSlot.value = expandedSlot.value === slot ? '' : slot)

function setDiagnosticSlot(value: string): void {
  void store.patchAIRoles({ diagnostics: value })
}

function uniqueSlot(): string {
  const taken = new Set([
    ...draft.value.map((profile) => profile.slot),
    ...(store.data?.network.httpOrigins ?? []).map((origin) => origin.slot),
    ...(store.data?.applications.profiles ?? []).map((profile) => profile.slot),
    ...(store.data?.automation.targets ?? []).map((target) => target.slot),
  ])
  if (!taken.has('model')) return 'model'
  for (let index = 2; ; index++) {
    const candidate = `model-${index}`
    if (!taken.has(candidate)) return candidate
  }
}

function uniqueLabel(base: string): string {
  const taken = new Set(draft.value.map((profile) => profile.label))
  if (!taken.has(base)) return base
  for (let index = 2; ; index++) {
    const candidate = `${base} ${index}`
    if (!taken.has(candidate)) return candidate
  }
}

function addProfile(): void {
  const slot = uniqueSlot()
  draft.value.push({
    slot,
    label: uniqueLabel(t('settingsAI.profiles.new_label')),
    provider: 'openai-responses',
    endpoint: defaultProviderEndpoint('openai-responses'),
    allowLocalHttp: false,
    model: '',
    maxOutputTokens: 4096,
    capabilities: {
      structuredOutput: false,
      toolCalling: false,
      parallelTools: false,
      background: false,
      zeroRetention: false,
    },
    pricing: {
      inputMicrounitsPerMillion: 0,
      cacheReadMicrounitsPerMillion: 0,
      outputMicrounitsPerMillion: 0,
    },
    evaluation: 'unverified',
    persisted: false,
  })
  expandedSlot.value = slot
}

async function commit(): Promise<boolean> {
  const invalidProfiles = draft.value.filter(
    (profile) => profileRequiredFieldsComplete(profile) && endpointIssue(profile),
  )
  if (invalidProfiles.length) {
    for (const profile of invalidProfiles) endpointValidationVisible[profile.slot] = true
    return false
  }
  if (draft.value.some((profile) => profile.persisted && !profileComplete(profile))) {
    return false
  }
  const savable = draft.value.filter(profileComplete)
  const ok = await store.patchAIProfiles(savable.map(profileMetadata))
  if (ok) {
    for (const profile of savable) profile.persisted = true
    await savePendingAPIKeys(savable)
  }
  return ok
}

function profileComplete(profile: AIModelProfileDraft): boolean {
  return Boolean(profileRequiredFieldsComplete(profile) && !endpointIssue(profile))
}

function profileRequiredFieldsComplete(profile: AIModelProfileDraft): boolean {
  return Boolean(
    profile.slot && profile.label.trim() && profile.model.trim() && profile.endpoint.trim(),
  )
}

function endpointIssue(profile: AIModelProfileDraft) {
  if (profile.provider === 'codex-subscription') return undefined
  return aiProviderEndpointIssue(profile.endpoint, profile.allowLocalHttp)
}

function endpointFieldError(profile: AIModelProfileDraft): string | undefined {
  if (!endpointValidationVisible[profile.slot]) return undefined
  const issue = endpointIssue(profile)
  return issue ? t(`settingsAI.profiles.endpoint_${issue}`) : undefined
}

async function commitEndpoint(profile: AIModelProfileDraft): Promise<void> {
  endpointValidationVisible[profile.slot] = true
  await commit()
}

async function onProvider(index: number, provider: AIProviderKind): Promise<void> {
  const profile = draft.value[index]
  const priorDefault = defaultProviderEndpoint(profile.provider)
  if (!profile.endpoint.trim() || profile.endpoint.trim() === priorDefault) {
    profile.endpoint = defaultProviderEndpoint(provider)
    profile.allowLocalHttp = false
  }
  profile.provider = provider
  if (provider === 'openai-chat-completions') {
    profile.capabilities.toolCalling = false
    profile.capabilities.parallelTools = false
  }
  if (provider === 'codex-subscription') {
    profile.endpoint = defaultProviderEndpoint(provider)
    profile.allowLocalHttp = false
    profile.capabilities.structuredOutput = true
    profile.capabilities.toolCalling = true
    profile.capabilities.parallelTools = true
    profile.capabilities.background = false
    profile.capabilities.zeroRetention = false
  }
  await commit()
}

async function setLocalHTTP(index: number, enabled: boolean): Promise<void> {
  draft.value[index].allowLocalHttp = enabled
  await commit()
}

async function setUnlimitedOutputTokens(index: number, enabled: boolean): Promise<void> {
  const profile = draft.value[index]
  if (enabled) {
    if (profile.maxOutputTokens > 0) {
      previousOutputTokenLimits[profile.slot] = profile.maxOutputTokens
    }
    profile.maxOutputTokens = 0
  } else {
    profile.maxOutputTokens = previousOutputTokenLimits[profile.slot] || 4096
  }
  await commit()
}

async function restoreProviderEndpoint(profile: AIModelProfileDraft): Promise<void> {
  profile.endpoint = defaultProviderEndpoint(profile.provider)
  profile.allowLocalHttp = false
  await commit()
}

function defaultProviderEndpoint(provider: AIProviderKind): string {
  switch (provider) {
    case 'openai-responses':
      return 'https://api.openai.com/v1/responses'
    case 'openai-chat-completions':
      return 'https://api.openai.com/v1'
    case 'anthropic-messages':
      return 'https://api.anthropic.com/v1/messages'
    case 'codex-subscription':
      return 'codex://subscription'
  }
}

async function onCapability(
  index: number,
  capability: CapabilityKey,
  enabled: boolean,
): Promise<void> {
  const profile = draft.value[index]
  profile.capabilities[capability] = enabled
  if (capability === 'toolCalling' && !enabled) profile.capabilities.parallelTools = false
  if (capability === 'parallelTools' && enabled) profile.capabilities.toolCalling = true
  await commit()
}

function profileMetadata(profile: AIModelProfileDraft): AIModelProfile {
  return {
    slot: profile.slot,
    label: profile.label.trim(),
    provider: profile.provider,
    endpoint: profile.endpoint.trim(),
    allowLocalHttp: profile.allowLocalHttp,
    model: profile.model.trim(),
    maxOutputTokens: profile.maxOutputTokens,
    capabilities: { ...profile.capabilities },
    pricing: { ...profile.pricing },
    evaluation: profile.evaluation,
    ...(profile.evaluationSuite ? { evaluationSuite: profile.evaluationSuite } : {}),
    ...(profile.evaluationReport ? { evaluationReport: { ...profile.evaluationReport } } : {}),
  }
}

async function removeProfile(profile: AIModelProfileDraft): Promise<void> {
  if (!profile.persisted) {
    draft.value = draft.value.filter((candidate) => candidate.slot !== profile.slot)
    return
  }
  const ok = await confirm({
    title: t('settingsAI.confirm.delete_title', { name: profile.label }),
    description: t('settingsAI.confirm.delete_profile'),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    color: 'error',
  })
  if (ok !== true) return
  await store.patchAIProfiles(
    draft.value.filter((candidate) => candidate.slot !== profile.slot).map(profileMetadata),
  )
}

async function refreshSecretStatus(): Promise<void> {
  try {
    const slots = profiles.value.map((profile) => profile.slot)
    const status = (await backend.ai.secretStatus(slots)) ?? {}
    for (const key of Object.keys(secretStatus)) delete secretStatus[key]
    Object.assign(secretStatus, status)
  } catch (error) {
    showActionError(error)
  }
}

async function saveAPIKey(profile: AIModelProfileDraft): Promise<void> {
  const apiKey = apiKeys[profile.slot]
  if (!apiKey || !profileComplete(profile)) return
  await commit()
}

async function savePendingAPIKeys(profilesToSave: AIModelProfileDraft[]): Promise<void> {
  let saved = false
  for (const profile of profilesToSave) {
    const apiKey = apiKeys[profile.slot]
    if (!apiKey) continue
    try {
      await backend.ai.setAPIKey(profile.slot, apiKey)
      apiKeys[profile.slot] = ''
      revealed[profile.slot] = false
      saved = true
    } catch (error) {
      showActionError(error)
    }
  }
  if (saved) await refreshSecretStatus()
}

async function deleteAPIKey(slot: string): Promise<void> {
  const ok = await confirm({
    title: t('settingsAI.confirm.delete_key_title'),
    description: t('settingsAI.confirm.delete_key_hint'),
    confirmText: t('settingsAI.profiles.apikey_remove'),
    cancelText: t('common.cancel'),
    color: 'error',
  })
  if (ok !== true) return
  try {
    await backend.ai.deleteAPIKey(slot)
    await refreshSecretStatus()
  } catch (error) {
    showActionError(error)
  }
}

async function testProfile(profile: AIModelProfileDraft): Promise<void> {
  if (!(await commit())) return
  const apiKey = apiKeys[profile.slot]
  if (apiKey) {
    try {
      await backend.ai.setAPIKey(profile.slot, apiKey)
      apiKeys[profile.slot] = ''
      revealed[profile.slot] = false
      await refreshSecretStatus()
    } catch (error) {
      showActionError(error)
      return
    }
  }
  testing[profile.slot] = true
  delete results[profile.slot]
  try {
    const result = await backend.ai.testProfile(profileMetadata(profile))
    if (result) results[profile.slot] = result
  } catch (error) {
    showActionError(error)
  } finally {
    testing[profile.slot] = false
  }
}

function testFailureMessage(profile: AIModelProfileDraft, result: AIProfileTestResult): string {
  const status = result.httpStatus ? `HTTP ${result.httpStatus}` : ''
  switch (result.failureClass) {
    case 'not-found':
      return t(
        profile.provider === 'openai-responses'
          ? 'settingsAI.test_errors.not_found_responses'
          : 'settingsAI.test_errors.not_found',
        { status },
      )
    case 'authentication':
      return t('settingsAI.test_errors.authentication', { status })
    case 'permission':
      return t('settingsAI.test_errors.permission', { status })
    case 'invalid-request':
      return t('settingsAI.test_errors.invalid_request', { status })
    case 'invalid-response':
      return t('settingsAI.test_errors.invalid_response', { status })
    case 'rate-limit':
      return t('settingsAI.test_errors.rate_limit', { status })
    case 'timeout':
      return t('settingsAI.test_errors.timeout', { status })
    case 'server':
    case 'overloaded':
      return t('settingsAI.test_errors.server', { status })
    case 'conflict':
      return t('settingsAI.test_errors.conflict', { status })
    case 'cancelled':
      return t('settingsAI.test_errors.cancelled')
    default:
      return t('settingsAI.test_errors.unknown')
  }
}

function showActionError(error: unknown): void {
  toast.add({
    title: t('toast.operation_failed'),
    description: errorMessage(error),
    color: 'error',
  })
}
</script>
