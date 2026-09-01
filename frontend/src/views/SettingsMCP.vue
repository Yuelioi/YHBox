<template>
  <div class="settings-page">
    <div class="settings-notice">
      <UIcon name="i-tabler-shield-lock" class="settings-notice__icon" />
      <div class="min-w-0">
        <p class="settings-notice__title">{{ t('settingsMCP.security_title') }}</p>
        <p class="settings-notice__description">{{ t('settingsMCP.security_hint') }}</p>
      </div>
    </div>

    <SettingsSection
      :title="t('settingsMCP.server_title')"
      :description="t('settingsMCP.server_hint')"
      icon="i-tabler-plug-connected"
    >
      <SettingsRow :label="t('settingsMCP.enabled_label')" :hint="t('settingsMCP.enabled_hint')">
        <template #meta>
          <UBadge :color="enabled ? 'success' : 'neutral'" size="xs" variant="subtle">
            {{ enabled ? t('settingsMCP.running') : t('settingsMCP.stopped') }}
          </UBadge>
        </template>
        <USwitch
          :model-value="enabled"
          :aria-label="t('settingsMCP.enabled_label')"
          @update:model-value="setEnabled"
        />
      </SettingsRow>

      <div class="border-t border-default/60" />

      <SettingsRow :label="t('settingsMCP.port_label')" :hint="t('settingsMCP.port_hint')">
        <UInputNumber
          :model-value="port"
          :min="1024"
          :max="65535"
          :disabled="enabled"
          class="w-36"
          @update:model-value="setPort"
        />
      </SettingsRow>

      <div class="border-t border-default/60" />

      <SettingsRow :label="t('settingsMCP.endpoint_label')" :hint="t('settingsMCP.endpoint_hint')">
        <div class="flex min-w-0 items-center gap-2">
          <code class="min-w-0 truncate text-xs text-toned">{{ endpoint }}</code>
          <UButton
            size="xs"
            color="neutral"
            variant="soft"
            icon="i-tabler-copy"
            :aria-label="t('settingsMCP.copy')"
            @click="copyEndpoint"
          >
            {{ t('settingsMCP.copy') }}
          </UButton>
        </div>
      </SettingsRow>
    </SettingsSection>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useSettingsStore } from '@/stores/settings'
import { backend } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import SettingsRow from '@/components/settings/SettingsRow.vue'
import SettingsSection from '@/components/settings/SettingsSection.vue'

const { t } = useI18n()
const toast = useToast()
const store = useSettingsStore()
const enabled = computed(() => store.data?.mcp.enabled ?? false)
const port = computed(() => store.data?.mcp.port ?? 39271)
const endpoint = computed(() => `http://127.0.0.1:${port.value}/mcp`)

async function setEnabled(value: boolean): Promise<void> {
  if (!(await store.patch({ mcp: { enabled: value } }))) return
  try {
    if (value) {
      await backend.mcp.registerCodex(port.value)
      toast.add({ title: t('settingsMCP.codex_registered'), color: 'success' })
    } else {
      await backend.mcp.unregisterCodex(port.value)
    }
  } catch (error) {
    toast.add({
      title: t('settingsMCP.codex_registration_failed'),
      description: errorMessage(error),
      color: 'error',
    })
  }
}

function setPort(value: number | undefined): void {
  if (value === undefined || !Number.isInteger(value)) return
  void store.patch({ mcp: { port: value } })
}

async function copyEndpoint(): Promise<void> {
  await navigator.clipboard.writeText(endpoint.value)
  toast.add({ title: t('settingsMCP.copied'), color: 'success' })
}
</script>
