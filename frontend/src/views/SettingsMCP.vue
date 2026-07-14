<template>
  <div class="settings-page">
    <SettingsSection
      :title="t('settingsMCP.accessTitle')"
      :description="t('settingsMCP.accessHint')"
      icon="i-tabler-shield-lock"
    >
      <div class="grid gap-3 sm:grid-cols-2">
        <div class="mcp-capability mcp-capability--active">
          <UIcon name="i-tabler-eye" class="size-5 text-success" />
          <div>
            <p class="text-sm font-medium text-default">{{ t('settingsMCP.readOnlyTitle') }}</p>
            <p class="mt-1 text-xs text-dimmed">{{ t('settingsMCP.readOnlyHint') }}</p>
          </div>
          <UBadge size="xs" color="success" variant="subtle">{{
            t('settingsMCP.alwaysOn')
          }}</UBadge>
        </div>
        <div class="mcp-capability" :class="armed ? 'mcp-capability--danger' : ''">
          <UIcon
            name="i-tabler-bolt"
            class="size-5"
            :class="armed ? 'text-error' : 'text-dimmed'"
          />
          <div>
            <p class="text-sm font-medium text-default">{{ t('settingsMCP.executeTitle') }}</p>
            <p class="mt-1 text-xs text-dimmed">{{ t('settingsMCP.executeHint') }}</p>
          </div>
          <UBadge size="xs" :color="armed ? 'error' : 'neutral'" variant="subtle">{{
            t(armed ? 'settingsMCP.enabled' : 'settingsMCP.disabled')
          }}</UBadge>
        </div>
      </div>

      <div class="flex items-start gap-3 rounded-lg border border-error/35 bg-error/5 p-4">
        <UIcon name="i-tabler-alert-triangle" class="mt-0.5 size-5 shrink-0 text-error" />
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-default">{{ t('settingsMCP.armLabel') }}</p>
          <p class="mt-1 text-xs leading-relaxed text-dimmed">{{ t('settingsMCP.armWarning') }}</p>
        </div>
        <USwitch
          :model-value="armed"
          :aria-label="t('settingsMCP.armLabel')"
          @update:model-value="onToggleArmed"
        />
      </div>
    </SettingsSection>

    <SettingsSection
      :title="t('settingsMCP.connectionTitle')"
      :description="t('settingsMCP.urlHint')"
      icon="i-tabler-link"
    >
      <SettingsRow :label="t('settingsMCP.urlLabel')" :hint="t('settingsMCP.localOnlyHint')">
        <div class="flex min-w-0 items-center gap-2">
          <UInput
            :model-value="MCP_URL"
            readonly
            size="sm"
            class="min-w-0 flex-1 font-mono"
            :aria-label="t('settingsMCP.urlLabel')"
          />
          <UButton
            size="sm"
            variant="soft"
            color="primary"
            icon="i-tabler-copy"
            :aria-label="t('settingsMCP.copy')"
            @click="copyUrl"
            >{{ t('settingsMCP.copy') }}</UButton
          >
        </div>
      </SettingsRow>
      <p class="sr-only" aria-live="polite">{{ copyMessage }}</p>
    </SettingsSection>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import SettingsRow from '@/components/settings/SettingsRow.vue'
import SettingsSection from '@/components/settings/SettingsSection.vue'

const { t } = useI18n()
const { confirm } = useConfirm()
const toast = useToast()
const store = useSettingsStore()
const MCP_URL = 'http://127.0.0.1:8765/mcp'
const armed = computed(() => store.data?.mcp?.armed ?? false)
const copyMessage = ref('')

async function onToggleArmed(value: boolean) {
  if (value) {
    const ok = await confirm({
      title: t('settingsMCP.confirmTitle'),
      description: t('settingsMCP.confirmHint'),
      confirmText: t('settingsMCP.confirmAction'),
      cancelText: t('common.cancel'),
      color: 'error',
    })
    if (ok !== true) return
  }
  await store.patch({ mcp: { armed: value } })
}
async function copyUrl() {
  try {
    await navigator.clipboard.writeText(MCP_URL)
    copyMessage.value = t('settingsMCP.copied')
    toast.add({ title: copyMessage.value, color: 'success', icon: 'i-tabler-copy-check' })
  } catch {
    copyMessage.value = t('settingsMCP.copyFailed')
    toast.add({ title: copyMessage.value, color: 'error', icon: 'i-tabler-copy-off' })
  }
}
</script>
