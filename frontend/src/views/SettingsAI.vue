<template>
  <div class="settings-page">
    <div class="flex items-start gap-3 rounded-xl border border-warning/35 bg-warning/5 p-4">
      <UIcon name="i-tabler-shield-lock" class="mt-0.5 size-5 shrink-0 text-warning" />
      <div>
        <p class="text-sm font-medium text-default">{{ t('settingsAI.security.title') }}</p>
        <p class="mt-1 text-xs leading-relaxed text-dimmed">{{ t('settingsAI.security.hint') }}</p>
      </div>
    </div>

    <SettingsSection
      :title="t('settingsAI.connections.title')"
      :description="t('settingsAI.connections.hint')"
      icon="i-tabler-plug-connected"
      :badge="String(draft.length)"
    >
      <template #actions>
        <UButton
          size="sm"
          color="primary"
          variant="soft"
          icon="i-tabler-plus"
          @click="addConnection"
        >
          {{ t('settingsAI.connections.add') }}
        </UButton>
      </template>

      <div v-if="draft.length" class="space-y-3">
        <article
          v-for="(connection, index) in draft"
          :key="connection.id"
          class="ai-connection"
          :class="connection.id === defaultId ? 'ai-connection--default' : ''"
        >
          <button
            class="flex min-w-0 flex-1 items-center gap-3 text-left"
            type="button"
            :aria-expanded="expandedId === connection.id"
            @click="toggleExpanded(connection.id)"
          >
            <span
              class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-default bg-elevated"
            >
              <UIcon
                :name="
                  connection.protocol === 'anthropic'
                    ? 'i-tabler-letter-a'
                    : 'i-tabler-brand-openai'
                "
                class="size-4 text-toned"
              />
            </span>
            <span class="min-w-0 flex-1">
              <span class="flex flex-wrap items-center gap-2">
                <span class="truncate text-sm font-medium text-default">{{
                  connection.label || t('settingsAI.connections.unnamed')
                }}</span>
                <UBadge
                  v-if="connection.id === defaultId"
                  size="xs"
                  color="primary"
                  variant="subtle"
                  >{{ t('settingsAI.connections.default_badge') }}</UBadge
                >
              </span>
              <span class="mt-1 block truncate text-xs text-dimmed"
                >{{ protocolName(connection.protocol) }} ·
                {{ connection.baseURL || t('settingsAI.connections.official_endpoint') }}</span
              >
            </span>
            <UIcon
              name="i-tabler-chevron-down"
              class="size-4 shrink-0 text-dimmed transition-transform"
              :class="expandedId === connection.id ? 'rotate-180' : ''"
            />
          </button>

          <div v-if="expandedId === connection.id" class="ai-connection__details">
            <div class="grid gap-4 sm:grid-cols-2">
              <UFormField :label="t('settingsAI.connections.name_label')" required>
                <UInput
                  v-model="connection.label"
                  size="sm"
                  :placeholder="t('settingsAI.connections.label_placeholder')"
                  @change="commit"
                />
              </UFormField>
              <UFormField :label="t('settingsAI.connections.protocol_label')">
                <USelect
                  :model-value="connection.protocol"
                  :items="protocolItems"
                  size="sm"
                  @update:model-value="(value: 'openai' | 'anthropic') => onProtocol(index, value)"
                />
              </UFormField>
            </div>
            <UFormField
              :label="t('settingsAI.connections.endpoint_label')"
              :hint="t('settingsAI.connections.endpoint_hint')"
            >
              <UInput
                v-model="connection.baseURL"
                size="sm"
                :placeholder="t('settingsAI.connections.baseurl_placeholder')"
                @change="commit"
              />
            </UFormField>
            <UFormField
              :label="t('settingsAI.connections.apikey_label')"
              :hint="t('settingsAI.connections.apikey_hint')"
            >
              <UInput
                v-model="connection.apiKey"
                :type="revealed[connection.id] ? 'text' : 'password'"
                size="sm"
                :placeholder="t('settingsAI.connections.apikey_placeholder')"
                @change="commit"
              >
                <template #trailing>
                  <UButton
                    :icon="revealed[connection.id] ? 'i-tabler-eye-off' : 'i-tabler-eye'"
                    variant="link"
                    color="neutral"
                    size="xs"
                    :aria-label="t('settingsAI.connections.reveal')"
                    @click="revealed[connection.id] = !revealed[connection.id]"
                  />
                </template>
              </UInput>
            </UFormField>
            <div class="flex flex-wrap items-end gap-3 border-t border-default/60 pt-4">
              <UFormField
                :label="t('settingsAI.connections.testmodel_label')"
                class="min-w-52 flex-1"
              >
                <UInput
                  v-model="testModels[connection.id]"
                  size="sm"
                  :placeholder="t('settingsAI.connections.testmodel_placeholder')"
                />
              </UFormField>
              <UButton
                size="sm"
                variant="soft"
                color="primary"
                icon="i-tabler-plug"
                :loading="testing[connection.id]"
                @click="test(index)"
                >{{ t('settingsAI.connections.test') }}</UButton
              >
              <UButton
                v-if="connection.id !== defaultId"
                size="sm"
                variant="ghost"
                color="neutral"
                icon="i-tabler-star"
                @click="setDefault(connection.id)"
                >{{ t('settingsAI.connections.set_default') }}</UButton
              >
              <UButton
                size="sm"
                variant="ghost"
                color="error"
                icon="i-tabler-trash"
                @click="removeConnection(connection)"
                >{{ t('settingsAI.connections.delete') }}</UButton
              >
            </div>
            <div
              v-if="results[connection.id]"
              class="flex items-start gap-2 rounded-lg border p-3 text-xs"
              :class="
                results[connection.id]!.ok
                  ? 'border-success/30 bg-success/5 text-success'
                  : 'border-error/30 bg-error/5 text-error'
              "
              role="status"
            >
              <UIcon
                :name="results[connection.id]!.ok ? 'i-tabler-circle-check' : 'i-tabler-circle-x'"
                class="mt-0.5 size-4 shrink-0"
              />
              <span v-if="results[connection.id]!.ok">{{
                results[connection.id]!.models?.length
                  ? t('settingsAI.connections.ok_models') +
                    ' ' +
                    results[connection.id]!.models.join(', ')
                  : t('settingsAI.connections.ok_no_models')
              }}</span>
              <span v-else class="break-all">{{ results[connection.id]!.error }}</span>
            </div>
          </div>
        </article>
      </div>

      <div v-else class="settings-empty-state">
        <UIcon name="i-tabler-plug-off" class="size-6 text-dimmed" />
        <p class="text-sm font-medium text-default">{{ t('settingsAI.connections.empty') }}</p>
        <UButton
          size="sm"
          color="primary"
          variant="soft"
          icon="i-tabler-plus"
          @click="addConnection"
          >{{ t('settingsAI.connections.add') }}</UButton
        >
      </div>
    </SettingsSection>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type AIConnection, type AITestResult } from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import SettingsSection from '@/components/settings/SettingsSection.vue'

const { t } = useI18n()
const { confirm } = useConfirm()
const store = useSettingsStore()
const connections = computed<AIConnection[]>(() => store.data?.ai.connections ?? [])
const defaultId = computed(() => store.data?.ai.default ?? '')
const draft = ref<AIConnection[]>([])
const expandedId = ref('')
const testing = reactive<Record<string, boolean>>({})
const results = reactive<Record<string, AITestResult>>({})
const revealed = reactive<Record<string, boolean>>({})
const testModels = reactive<Record<string, string>>({})
const protocolItems = computed(() => [
  { label: t('settingsAI.protocol.openai'), value: 'openai' },
  { label: t('settingsAI.protocol.anthropic'), value: 'anthropic' },
])

watch(
  connections,
  () => {
    draft.value = connections.value.map((connection) => ({ ...connection }))
  },
  { immediate: true },
)
const protocolName = (protocol: AIConnection['protocol']) => t(`settingsAI.protocol.${protocol}`)
const toggleExpanded = (id: string) => (expandedId.value = expandedId.value === id ? '' : id)
function uniqueLabel(base: string) {
  const taken = new Set(connections.value.map((connection) => connection.label))
  if (!taken.has(base)) return base
  for (let index = 2; ; index++) {
    const candidate = `${base} ${index}`
    if (!taken.has(candidate)) return candidate
  }
}
async function commit() {
  for (const connection of draft.value)
    if (connection.baseURL && !/^https?:\/\//i.test(connection.baseURL))
      connection.baseURL = `http://${connection.baseURL}`
  await store.patchAIConnections(draft.value.map((connection) => ({ ...connection })))
}
async function onProtocol(index: number, value: AIConnection['protocol']) {
  draft.value[index].protocol = value
  await commit()
}
async function setDefault(id: string) {
  await store.patchAIConnections(
    draft.value.map((connection) => ({ ...connection })),
    id,
  )
}
async function addConnection() {
  const connection: AIConnection = {
    id: crypto.randomUUID(),
    label: uniqueLabel(t('settingsAI.connections.new_label')),
    protocol: 'openai',
    baseURL: '',
    apiKey: '',
  }
  const ok = await store.patchAIConnections([...connections.value, connection])
  if (ok) expandedId.value = connection.id
}
async function removeConnection(connection: AIConnection) {
  const refs = (await backend.ai.nodesUsingConnection(connection.id)) ?? []
  const ok = await confirm({
    title: t('settingsAI.confirm.delete_title', { name: connection.label }),
    description: refs.length
      ? t('settingsAI.confirm.delete_referenced', {
          n: refs.length,
          containers: [...new Set(refs.map((ref) => ref.containerName))].join('、'),
        })
      : t('settingsAI.confirm.delete_unused'),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    color: 'error',
  })
  if (ok !== true) return
  const list = connections.value.filter((item) => item.id !== connection.id)
  await store.patchAIConnections(list, connection.id === defaultId.value ? '' : undefined)
}
async function test(index: number) {
  const connection = draft.value[index]
  if (!connection) return
  await commit()
  testing[connection.id] = true
  delete results[connection.id]
  const result = await backend.ai.testConnection({ ...connection }, testModels[connection.id] ?? '')
  testing[connection.id] = false
  if (result) results[connection.id] = result
}
</script>
