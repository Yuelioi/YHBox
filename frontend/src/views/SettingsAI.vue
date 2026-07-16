<template>
  <div class="settings-page">
    <div class="flex items-start gap-3 rounded-xl border border-warning/35 bg-warning/5 p-4">
      <UIcon name="i-tabler-shield-lock" class="mt-0.5 size-5 shrink-0 text-warning" />
      <div class="min-w-0">
        <p class="text-sm font-medium text-default">{{ t('settingsAI.security.title') }}</p>
        <p class="mt-1 text-xs leading-relaxed text-dimmed">{{ t('settingsAI.security.hint') }}</p>
      </div>
    </div>

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

      <div v-if="draft.length" class="space-y-3">
        <article v-for="(profile, index) in draft" :key="profile.slot" class="ai-profile">
          <button
            class="flex min-w-0 flex-1 items-center gap-3 text-left"
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
                <UBadge
                  v-if="profile.workflowConsent"
                  size="xs"
                  color="success"
                  variant="subtle"
                  icon="i-tabler-shield-check"
                >
                  {{ t('settingsAI.profiles.workflow_allowed') }}
                </UBadge>
                <UBadge v-else size="xs" color="warning" variant="subtle">
                  {{ t('settingsAI.profiles.consent_required') }}
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
                :hint="t('settingsAI.profiles.slot_hint')"
              >
                <UInput :model-value="profile.slot" size="sm" disabled class="font-mono" />
              </UFormField>
              <UFormField :label="t('settingsAI.profiles.provider_label')" required>
                <USelect
                  :model-value="profile.provider"
                  :items="providerItems"
                  size="sm"
                  @update:model-value="(value: AIProviderKind) => onProvider(index, value)"
                />
              </UFormField>
              <UFormField
                :label="t('settingsAI.profiles.model_label')"
                :hint="t('settingsAI.profiles.model_hint')"
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

            <div class="grid gap-4 sm:grid-cols-[minmax(0,1fr)_auto]">
              <UFormField
                :label="t('settingsAI.profiles.max_tokens_label')"
                :hint="t('settingsAI.profiles.max_tokens_hint')"
              >
                <UInputNumber
                  v-model="profile.maxOutputTokens"
                  :min="1"
                  :max="1000000"
                  :step="256"
                  size="sm"
                  class="w-full sm:w-48"
                  @change="commit"
                />
              </UFormField>
              <div class="flex items-end pb-1">
                <UBadge size="xs" color="neutral" variant="subtle">
                  {{ t(`settingsAI.evaluation.${profile.evaluation}`) }}
                </UBadge>
              </div>
            </div>

            <div class="rounded-lg border border-default/70 bg-elevated/35 p-3">
              <p class="text-xs font-medium text-default">
                {{ t('settingsAI.capabilities.title') }}
              </p>
              <p class="mt-1 text-xs leading-relaxed text-dimmed">
                {{ t('settingsAI.capabilities.hint') }}
              </p>
              <div class="mt-3 grid gap-x-5 gap-y-3 sm:grid-cols-2">
                <label
                  v-for="capability in capabilityOptions"
                  :key="capability.key"
                  class="flex items-start justify-between gap-3"
                >
                  <span class="min-w-0">
                    <span class="block text-xs font-medium text-default">{{
                      capability.label
                    }}</span>
                    <span class="mt-0.5 block text-xs leading-relaxed text-dimmed">{{
                      capability.hint
                    }}</span>
                  </span>
                  <USwitch
                    :model-value="profile.capabilities[capability.key]"
                    size="sm"
                    @update:model-value="
                      (value: boolean) => onCapability(index, capability.key, value)
                    "
                  />
                </label>
              </div>
            </div>

            <UFormField
              :label="t('settingsAI.profiles.apikey_label')"
              :hint="t('settingsAI.profiles.apikey_hint')"
            >
              <UInput
                v-model="profile.apiKey"
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

            <div class="rounded-lg border border-warning/30 bg-warning/5 p-3">
              <div class="flex flex-wrap items-start gap-3">
                <UIcon
                  name="i-tabler-shield-exclamation"
                  class="mt-0.5 size-4 shrink-0 text-warning"
                />
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <p class="text-xs font-medium text-default">
                      {{ t('settingsAI.consent.title') }}
                    </p>
                    <SettingsRestartBadge />
                  </div>
                  <p class="mt-1 text-xs leading-relaxed text-dimmed">
                    {{ t('settingsAI.consent.hint') }}
                  </p>
                </div>
                <UButton
                  v-if="profile.workflowConsent"
                  size="xs"
                  variant="soft"
                  color="warning"
                  icon="i-tabler-shield-off"
                  @click="revokeWorkflowUse(profile)"
                >
                  {{ t('settingsAI.consent.revoke') }}
                </UButton>
                <UButton
                  v-else
                  size="xs"
                  variant="soft"
                  color="primary"
                  icon="i-tabler-shield-check"
                  :disabled="!profile.persisted"
                  @click="grantWorkflowUse(profile)"
                >
                  {{ t('settingsAI.consent.grant') }}
                </UButton>
              </div>
            </div>

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
                    finish: results[profile.slot]!.finish,
                  })
                }}
              </span>
              <span v-else class="break-all">{{ results[profile.slot]!.error }}</span>
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
import {
  backend,
  type AIModelProfile,
  type AIProfileCapabilities,
  type AIProfileTestResult,
  type AIProviderKind,
} from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import SettingsRestartBadge from '@/components/settings/SettingsRestartBadge.vue'
import SettingsSection from '@/components/settings/SettingsSection.vue'

interface AIModelProfileDraft extends AIModelProfile {
  apiKey: string
  persisted: boolean
}

type CapabilityKey = keyof AIProfileCapabilities

const { t } = useI18n()
const { confirm } = useConfirm()
const store = useSettingsStore()
const profiles = computed<AIModelProfile[]>(() => store.data?.ai.profiles ?? [])
const draft = ref<AIModelProfileDraft[]>([])
const expandedSlot = ref('')
const testing = reactive<Record<string, boolean>>({})
const results = reactive<Record<string, AIProfileTestResult>>({})
const revealed = reactive<Record<string, boolean>>({})
const secretStatus = reactive<Record<string, boolean>>({})

const providerItems = computed(() => [
  { label: t('settingsAI.provider.openai_responses'), value: 'openai-responses' },
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
    draft.value = profiles.value.map((profile) => ({
      ...profile,
      capabilities: { ...profile.capabilities },
      apiKey: '',
      persisted: true,
    }))
    void refreshSecretStatus()
  },
  { immediate: true },
)

const providerName = (provider: AIProviderKind) =>
  t(
    provider === 'openai-responses'
      ? 'settingsAI.provider.openai_responses'
      : 'settingsAI.provider.anthropic_messages',
  )
const providerIcon = (provider: AIProviderKind) =>
  provider === 'openai-responses' ? 'i-tabler-brand-openai' : 'i-tabler-letter-a'
const toggleExpanded = (slot: string) =>
  (expandedSlot.value = expandedSlot.value === slot ? '' : slot)

function uniqueSlot(): string {
  const taken = new Set([
    ...draft.value.map((profile) => profile.slot),
    ...(store.data?.network.httpOrigins ?? []).map((origin) => origin.slot),
    ...(store.data?.applications.profiles ?? []).map((profile) => profile.slot),
    ...(store.data?.automation.win32Targets ?? []).map((target) => target.slot),
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
    model: '',
    maxOutputTokens: 4096,
    capabilities: {
      structuredOutput: false,
      toolCalling: false,
      parallelTools: false,
      background: false,
      zeroRetention: false,
    },
    evaluation: 'unverified',
    apiKey: '',
    persisted: false,
  })
  expandedSlot.value = slot
}

async function commit(): Promise<boolean> {
  if (
    draft.value.some(
      (profile) => profile.persisted && (!profile.label.trim() || !profile.model.trim()),
    )
  ) {
    return false
  }
  const savable = draft.value.filter(
    (profile) => profile.slot && profile.label.trim() && profile.model.trim(),
  )
  const ok = await store.patchAIProfiles(savable.map(profileMetadata))
  if (ok) {
    for (const profile of savable) profile.persisted = true
  }
  return ok
}

async function onProvider(index: number, provider: AIProviderKind): Promise<void> {
  draft.value[index].provider = provider
  await commit()
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
    model: profile.model.trim(),
    maxOutputTokens: profile.maxOutputTokens,
    capabilities: { ...profile.capabilities },
    evaluation: profile.evaluation,
    ...(profile.evaluationSuite ? { evaluationSuite: profile.evaluationSuite } : {}),
    ...(profile.workflowConsent ? { workflowConsent: profile.workflowConsent } : {}),
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
  const slots = profiles.value.map((profile) => profile.slot)
  const status = (await backend.ai.secretStatus(slots)) ?? {}
  for (const key of Object.keys(secretStatus)) delete secretStatus[key]
  Object.assign(secretStatus, status)
}

async function saveAPIKey(profile: AIModelProfileDraft): Promise<void> {
  if (!profile.apiKey || !(await commit())) return
  const ok = await backend.ai.setAPIKey(profile.slot, profile.apiKey)
  if (!ok) return
  profile.apiKey = ''
  revealed[profile.slot] = false
  await refreshSecretStatus()
}

async function deleteAPIKey(slot: string): Promise<void> {
  const ok = await confirm({
    title: t('settingsAI.confirm.delete_key_title'),
    description: t('settingsAI.confirm.delete_key_hint'),
    confirmText: t('settingsAI.profiles.apikey_remove'),
    cancelText: t('common.cancel'),
    color: 'error',
  })
  if (ok === true && (await backend.ai.deleteAPIKey(slot))) await refreshSecretStatus()
}

async function testProfile(profile: AIModelProfileDraft): Promise<void> {
  if (!(await commit())) return
  testing[profile.slot] = true
  delete results[profile.slot]
  const result = await backend.ai.testProfile(profileMetadata(profile), profile.apiKey)
  testing[profile.slot] = false
  if (result) results[profile.slot] = result
}

async function grantWorkflowUse(profile: AIModelProfileDraft): Promise<void> {
  if (!(await commit())) return
  await backend.ai.grantWorkflowUse(profile.slot)
}

async function revokeWorkflowUse(profile: AIModelProfileDraft): Promise<void> {
  await backend.ai.revokeWorkflowUse(profile.slot)
}
</script>
