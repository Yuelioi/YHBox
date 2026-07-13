<template>
  <div class="settings-page">
    <section class="settings-section">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-sparkles" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">{{ t('settingsAI.title') }}</h2>
      </div>
      <p class="text-xs text-dimmed leading-relaxed">{{ t('settingsAI.intro') }}</p>
    </section>

    <section class="settings-section">
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-plug-connected" class="size-4 text-dimmed" />
        <h2 class="text-sm font-medium text-highlighted">
          {{ t('settingsAI.connections.title') }}
        </h2>
      </div>
      <p class="text-xs text-dimmed">{{ t('settingsAI.connections.hint') }}</p>

      <div v-if="draft.length" class="space-y-3">
        <div
          v-for="(c, i) in draft"
          :key="c.id"
          class="rounded-md border px-3 py-3 space-y-2.5"
          :class="
            c.id === defaultId
              ? 'border-primary/50 bg-primary/5'
              : 'border-default/60 bg-elevated/30'
          "
        >
          <div class="flex items-center gap-2">
            <URadio
              :model-value="defaultId"
              :value="c.id"
              :title="t('settingsAI.connections.set_default')"
              :aria-label="t('settingsAI.connections.set_default')"
              @update:model-value="() => setDefault(c.id)"
            />
            <UInput
              :model-value="c.label"
              size="sm"
              class="flex-1 min-w-0"
              :placeholder="t('settingsAI.connections.label_placeholder')"
              :aria-label="t('settingsAI.connections.label_placeholder')"
              @update:model-value="(v: string) => (draft[i].label = v)"
              @change="commit()"
            />
            <UButton
              size="xs"
              variant="soft"
              color="primary"
              icon="i-tabler-plug"
              :loading="testing[c.id]"
              @click="test(i)"
            >
              {{ t('settingsAI.connections.test') }}
            </UButton>
            <UButton
              size="xs"
              variant="ghost"
              color="error"
              icon="i-tabler-trash"
              :title="t('settingsAI.connections.delete')"
              :aria-label="t('settingsAI.connections.delete')"
              @click="removeConnection(c.id)"
            />
          </div>

          <div class="flex items-center gap-2">
            <USelect
              :model-value="c.protocol"
              :items="protocolItems"
              size="sm"
              class="w-44 shrink-0"
              :aria-label="t('settingsAI.connections.protocol_label')"
              @update:model-value="(v: 'openai' | 'anthropic') => onProtocol(i, v)"
            />
            <UInput
              :model-value="c.baseURL"
              size="sm"
              class="flex-1 min-w-0"
              :placeholder="t('settingsAI.connections.baseurl_placeholder')"
              :aria-label="t('settingsAI.connections.baseurl_placeholder')"
              @update:model-value="(v: string) => (draft[i].baseURL = v)"
              @change="commit()"
            />
          </div>

          <div class="flex items-center gap-2">
            <UInput
              :model-value="c.apiKey"
              :type="revealed[c.id] ? 'text' : 'password'"
              size="sm"
              class="flex-1 min-w-0"
              :placeholder="t('settingsAI.connections.apikey_placeholder')"
              :aria-label="t('settingsAI.connections.apikey_placeholder')"
              @update:model-value="(v: string) => (draft[i].apiKey = v)"
              @change="commit()"
            >
              <template #trailing>
                <UButton
                  :icon="revealed[c.id] ? 'i-tabler-eye-off' : 'i-tabler-eye'"
                  variant="link"
                  color="neutral"
                  size="xs"
                  :title="t('settingsAI.connections.reveal')"
                  :aria-label="t('settingsAI.connections.reveal')"
                  @click="revealed[c.id] = !revealed[c.id]"
                />
              </template>
            </UInput>
            <UInput
              :model-value="testModels[c.id] ?? ''"
              size="sm"
              class="w-44 shrink-0"
              :placeholder="t('settingsAI.connections.testmodel_placeholder')"
              :aria-label="t('settingsAI.connections.testmodel_placeholder')"
              @update:model-value="(v: string) => (testModels[c.id] = v)"
            />
          </div>

          <p class="text-xs text-dimmed">{{ t('settingsAI.connections.protocol_hint') }}</p>

          <div v-if="results[c.id]" class="text-xs">
            <div v-if="results[c.id]!.ok" class="text-success flex items-start gap-1.5">
              <UIcon name="i-tabler-circle-check" class="size-3.5 mt-0.5 shrink-0" />
              <span v-if="results[c.id]!.models?.length">
                {{ t('settingsAI.connections.ok_models') }}
                <span class="text-dimmed break-all">{{ results[c.id]!.models.join(', ') }}</span>
              </span>
              <span v-else>{{ t('settingsAI.connections.ok_no_models') }}</span>
            </div>
            <div v-else class="text-error flex items-start gap-1.5">
              <UIcon name="i-tabler-circle-x" class="size-3.5 mt-0.5 shrink-0" />
              <span class="break-all">{{ results[c.id]!.error }}</span>
            </div>
          </div>
        </div>
      </div>
      <p
        v-else
        class="text-xs text-dimmed py-8 text-center border border-dashed border-default/60 rounded-xl"
      >
        {{ t('settingsAI.connections.empty') }}
      </p>

      <UButton size="sm" color="primary" variant="soft" icon="i-tabler-plus" @click="addConnection">
        {{ t('settingsAI.connections.add') }}
      </UButton>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend, type AIConnection, type AITestResult } from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'

const { t } = useI18n()
const store = useSettingsStore()

const connections = computed<AIConnection[]>(() => store.data?.ai.connections ?? [])
const defaultId = computed(() => store.data?.ai.default ?? '')

const protocolItems = computed(() => [
  { label: t('settingsAI.protocol.openai'), value: 'openai' },
  { label: t('settingsAI.protocol.anthropic'), value: 'anthropic' },
])

// 文本字段输入期只改草稿, 改完(change=失焦/回车)才 patch —— 避免逐键触发后端校验弹错。
const draft = ref<AIConnection[]>([])
watch(
  connections,
  () => {
    draft.value = connections.value.map((c) => ({ ...c }))
  },
  { immediate: true },
)

const testing = reactive<Record<string, boolean>>({})
const results = reactive<Record<string, AITestResult>>({})
const revealed = reactive<Record<string, boolean>>({})
const testModels = reactive<Record<string, string>>({})

function uniqueLabel(base: string): string {
  const taken = new Set(connections.value.map((c) => c.label))
  if (!taken.has(base)) return base
  for (let i = 2; ; i++) {
    const cand = base + ' ' + i
    if (!taken.has(cand)) return cand
  }
}

// commit 把当前草稿整体写回; 缺 scheme 的 baseURL 补 http:// 再存。
function commit() {
  for (const c of draft.value) {
    if (c.baseURL && !/^https?:\/\//i.test(c.baseURL)) {
      c.baseURL = 'http://' + c.baseURL
    }
  }
  store.patchAIConnections(draft.value.map((x) => ({ ...x })))
}

function onProtocol(i: number, v: 'openai' | 'anthropic') {
  draft.value[i].protocol = v
  commit()
}

function setDefault(id: string) {
  store.patchAIConnections(connections.value, id)
}

function addConnection() {
  const c: AIConnection = {
    id: crypto.randomUUID(),
    label: uniqueLabel(t('settingsAI.connections.new_label')),
    protocol: 'openai',
    baseURL: '',
    apiKey: '',
  }
  store.patchAIConnections([...connections.value, c])
}

function removeConnection(id: string) {
  const list = connections.value.filter((c) => c.id !== id)
  store.patchAIConnections(list, id === defaultId.value ? '' : undefined)
}

async function test(i: number) {
  const c = draft.value[i]
  commit()
  testing[c.id] = true
  delete results[c.id]
  const res = await backend.ai.testConnection({ ...c }, testModels[c.id] ?? '')
  testing[c.id] = false
  if (res) results[c.id] = res
}
</script>
