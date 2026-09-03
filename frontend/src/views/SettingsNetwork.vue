<template>
  <div class="settings-page">
    <SettingsSection :title="t('settingsNetwork.origins.title')" icon="i-tabler-world-www">
      <template #badge>
        <UBadge size="xs" color="neutral" variant="subtle">{{ draft.length }}</UBadge>
      </template>
      <template #actions>
        <UButton size="sm" color="primary" variant="soft" icon="i-tabler-plus" @click="addOrigin">
          {{ t('settingsNetwork.origins.add') }}
        </UButton>
      </template>

      <div v-if="draft.length" class="settings-collection">
        <article v-for="origin in draft" :key="origin.slot" class="ai-profile">
          <button
            type="button"
            class="settings-entity-summary cursor-pointer text-left"
            :aria-expanded="expandedSlot === origin.slot"
            @click="toggleExpanded(origin.slot)"
          >
            <span
              class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-default bg-elevated"
            >
              <UIcon name="i-tabler-world-www" class="size-4 text-toned" />
            </span>
            <span class="min-w-0 flex-1">
              <span class="flex flex-wrap items-center gap-2">
                <span class="truncate text-sm font-medium text-default">
                  {{ origin.label || t('settingsNetwork.origins.unnamed') }}
                </span>
              </span>
              <span class="mt-1 block truncate text-xs text-dimmed">
                {{ origin.origin || t('settingsNetwork.origins.origin_missing') }} ·
                <code>{{ origin.slot }}</code>
              </span>
            </span>
            <UIcon
              name="i-tabler-chevron-down"
              class="size-4 shrink-0 text-dimmed transition-transform"
              :class="expandedSlot === origin.slot ? 'rotate-180' : ''"
            />
          </button>

          <div v-if="expandedSlot === origin.slot" class="ai-profile__details">
            <div class="grid gap-4 sm:grid-cols-2">
              <UFormField :label="t('settingsNetwork.origins.name_label')" required>
                <UInput
                  v-model="origin.label"
                  size="sm"
                  :placeholder="t('settingsNetwork.origins.name_placeholder')"
                  @change="commit"
                />
              </UFormField>
              <UFormField
                :label="t('settingsNetwork.origins.slot_label')"
                :hint="t('settingsNetwork.origins.slot_hint')"
              >
                <UInput :model-value="origin.slot" size="sm" disabled class="font-mono" />
              </UFormField>
            </div>

            <UFormField
              :label="t('settingsNetwork.origins.origin_label')"
              :hint="t('settingsNetwork.origins.origin_hint')"
              required
            >
              <UInput
                v-model="origin.origin"
                type="url"
                size="sm"
                class="font-mono"
                placeholder="https://api.example.com"
                @change="commit"
              />
            </UFormField>

            <div class="grid gap-4 sm:grid-cols-2">
              <UFormField
                :label="t('settingsNetwork.origins.byte_limit_label')"
                :hint="t('settingsNetwork.origins.byte_limit_hint')"
              >
                <UInputNumber
                  v-model="origin.responseByteLimit"
                  :min="1"
                  :step="1024"
                  size="sm"
                  class="w-full"
                  @change="commit"
                />
              </UFormField>
              <UFormField :label="t('settingsNetwork.origins.timeout_label')">
                <UInputNumber
                  v-model="origin.timeoutMilliseconds"
                  :min="1"
                  :step="100"
                  size="sm"
                  class="w-full"
                  @change="commit"
                />
              </UFormField>
            </div>

            <div class="flex items-center border-t border-default/60 pt-4">
              <UButton
                class="ml-auto"
                size="sm"
                variant="ghost"
                color="error"
                icon="i-tabler-trash"
                @click="removeOrigin(origin)"
              >
                {{ t('settingsNetwork.origins.delete') }}
              </UButton>
            </div>
          </div>
        </article>
      </div>

      <div v-else class="settings-empty-state">
        <UIcon name="i-tabler-world-www" class="size-6 text-dimmed" />
        <p class="text-sm font-medium text-default">{{ t('settingsNetwork.origins.empty') }}</p>
        <p class="max-w-md text-center text-xs leading-relaxed text-dimmed">
          {{ t('settingsNetwork.origins.empty_hint') }}
        </p>
        <UButton size="sm" color="primary" variant="soft" icon="i-tabler-plus" @click="addOrigin">
          {{ t('settingsNetwork.origins.add') }}
        </UButton>
      </div>
    </SettingsSection>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { type HTTPOriginProfile } from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import SettingsSection from '@/components/settings/SettingsSection.vue'

interface HTTPOriginDraft extends HTTPOriginProfile {
  persisted: boolean
}

const { t } = useI18n()
const { confirm } = useConfirm()
const store = useSettingsStore()
const origins = computed<HTTPOriginProfile[]>(() => store.data?.network.httpOrigins ?? [])
const draft = ref<HTTPOriginDraft[]>([])
const expandedSlot = ref('')

watch(
  origins,
  () => {
    draft.value = origins.value.map((origin) => ({ ...origin, persisted: true }))
  },
  { immediate: true },
)

function toggleExpanded(slot: string) {
  expandedSlot.value = expandedSlot.value === slot ? '' : slot
}
function uniqueSlot(): string {
  const taken = new Set([
    ...draft.value.map((origin) => origin.slot),
    ...(store.data?.ai.profiles ?? []).map((profile) => profile.slot),
    ...(store.data?.applications.profiles ?? []).map((profile) => profile.slot),
    ...(store.data?.automation.targets ?? []).map((target) => target.slot),
  ])
  if (!taken.has('http')) return 'http'
  for (let index = 2; ; index++) {
    const candidate = `http-${index}`
    if (!taken.has(candidate)) return candidate
  }
}
function uniqueLabel(base: string): string {
  const taken = new Set(draft.value.map((origin) => origin.label))
  if (!taken.has(base)) return base
  for (let index = 2; ; index++) {
    const candidate = `${base} ${index}`
    if (!taken.has(candidate)) return candidate
  }
}
function addOrigin() {
  const slot = uniqueSlot()
  draft.value.push({
    slot,
    label: uniqueLabel(t('settingsNetwork.origins.new_label')),
    origin: '',
    responseByteLimit: 262144,
    timeoutMilliseconds: 10000,
    persisted: false,
  })
  expandedSlot.value = slot
}
function metadata(origin: HTTPOriginDraft): HTTPOriginProfile {
  return {
    slot: origin.slot,
    label: origin.label.trim(),
    origin: origin.origin.trim(),
    responseByteLimit: origin.responseByteLimit,
    timeoutMilliseconds: origin.timeoutMilliseconds,
  }
}
async function commit(): Promise<boolean> {
  if (
    draft.value.some(
      (origin) => origin.persisted && (!origin.label.trim() || !origin.origin.trim()),
    )
  )
    return false
  const savable = draft.value.filter(
    (origin) => origin.slot && origin.label.trim() && origin.origin.trim(),
  )
  const ok = await store.patchHTTPOrigins(savable.map(metadata))
  if (ok) for (const origin of savable) origin.persisted = true
  return ok
}
async function removeOrigin(origin: HTTPOriginDraft) {
  if (!origin.persisted) {
    draft.value = draft.value.filter((candidate) => candidate.slot !== origin.slot)
    return
  }
  const accepted = await confirm({
    title: t('settingsNetwork.confirm.delete_title', { name: origin.label }),
    description: t('settingsNetwork.confirm.delete_hint'),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    color: 'error',
  })
  if (accepted === true)
    await store.patchHTTPOrigins(
      draft.value.filter((candidate) => candidate.slot !== origin.slot).map(metadata),
    )
}
</script>
