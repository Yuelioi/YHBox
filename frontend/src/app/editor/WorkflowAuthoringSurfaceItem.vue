<template>
  <div v-if="item.kind === 'config'" class="space-y-2">
    <GeneratedFieldEditor
      :field="item.field"
      :model-value="effectiveConfigValue"
      :state-variables="variables"
      :select-items="targetOptions"
      :select-placeholder="t('workflow.inspector.select_target')"
      @update:model-value="
        emit('command', {
          kind: 'set-config',
          nodeId: node.id,
          fieldId: item.field.id,
          value: $event,
        })
      "
    />
    <div v-if="targetCapability" class="flex items-center gap-2 text-[11px]">
      <UBadge
        :color="hasOverride ? 'warning' : inheritedTarget ? 'primary' : 'error'"
        variant="soft"
        size="sm"
      >
        {{
          t(
            hasOverride
              ? 'workflow.inspector.target_overridden'
              : inheritedTarget
                ? 'workflow.inspector.target_inherited'
                : 'workflow.inspector.target_missing',
          )
        }}
      </UBadge>
      <span v-if="inheritedTarget && !hasOverride" class="truncate text-muted">
        {{ inheritedTarget }}
      </span>
      <UButton
        v-if="hasOverride && inheritedTarget"
        class="ml-auto"
        color="neutral"
        variant="ghost"
        size="xs"
        :label="t('workflow.inspector.restore_inherited')"
        @click="emit('command', { kind: 'clear-config', nodeId: node.id, fieldId: item.field.id })"
      />
    </div>
    <div
      v-if="targetCapability && targetOptions?.length === 0"
      class="flex items-center gap-2 rounded-lg border border-warning/30 bg-warning/10 px-3 py-2"
    >
      <p class="min-w-0 flex-1 text-[11px] leading-5 text-warning">
        {{ t('workflow.inspector.no_installed_target') }}
      </p>
      <UButton
        :to="{ path: '/settings', query: { section: targetSettingsSection } }"
        :label="t('workflow.inspector.configure_target')"
        icon="i-tabler-settings"
        color="warning"
        variant="soft"
        size="xs"
      />
    </div>
  </div>

  <WorkflowInputBindingEditor
    v-else-if="item.kind === 'input'"
    :node="node"
    :port="item.port"
    :target-slot="targetSlot"
    :connected="connectedInputIds?.has(item.port.id)"
    @command="emit('command', $event)"
    @capture-template="emit('capture-template')"
  />

  <div
    v-else
    class="flex items-center gap-2 rounded-lg border border-default bg-muted/20 px-3 py-2"
  >
    <span
      class="size-2 rounded-full"
      :style="{ backgroundColor: item.port.type.color || '#a1a1aa' }"
      aria-hidden="true"
    />
    <div class="min-w-0 flex-1">
      <p class="truncate text-xs font-medium text-toned">{{ portTitle }}</p>
      <p class="truncate text-[10px] text-dimmed">{{ typeTitle }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { TargetDefault } from '../../../../contracts/workflow/3.1/workflow-source'
import type { CapabilityProjection } from '../../../../contracts/node/3.1/authoring-projection'
import type { EditorCommand, Node, NodeProjection } from './EditorSession'
import type { AuthoringSurfaceItem } from './authoringSurface'
import GeneratedFieldEditor from './GeneratedFieldEditor.vue'
import WorkflowInputBindingEditor from './WorkflowInputBindingEditor.vue'
import { useSettingsStore } from '@/stores/settings'

const props = defineProps<{
  item: AuthoringSurfaceItem
  node: Node
  projection: NodeProjection
  variables: string[]
  targetDefaults: TargetDefault[]
  targetSlot?: string
  connectedInputIds?: ReadonlySet<string>
}>()
const emit = defineEmits<{
  command: [command: EditorCommand]
  'capture-template': []
}>()
const { t, te } = useI18n()
const settingsStore = useSettingsStore()
const configFieldID = computed(() => (props.item.kind === 'config' ? props.item.field.id : ''))
const targetCapability = computed<CapabilityProjection | undefined>(() =>
  props.projection.capabilities.find(
    (candidate) => candidate.targetSlotConfigKey === configFieldID.value,
  ),
)
const inheritedTarget = computed(() => {
  const target = targetCapability.value?.targetSlot
  return props.targetDefaults.find((candidate) => candidate.target === target)?.slot ?? ''
})
const hasOverride = computed(() =>
  configFieldID.value
    ? Object.prototype.hasOwnProperty.call(props.node.config, configFieldID.value)
    : false,
)
const effectiveConfigValue = computed(() => {
  if (!configFieldID.value) return undefined
  return hasOverride.value
    ? props.node.config[configFieldID.value]
    : inheritedTarget.value || undefined
})
const targetOptions = computed<Array<{ label: string; value: string }> | undefined>(() => {
  const capability = targetCapability.value
  if (!capability) return undefined
  const settings = settingsStore.data
  if (!settings) return []
  if (
    capability.targetKinds.some((kind) =>
      settings.automation.targets.some((target) => target.targetKind === kind),
    )
  ) {
    return settings.automation.targets
      .filter((target) => capability.targetKinds.includes(target.targetKind))
      .map((target) => ({ label: `${target.label} · ${target.slot}`, value: target.slot }))
  }
  if (capability.targetKinds.includes('installed-application'))
    return settings.applications.profiles.map((application) => ({
      label: `${application.label} · ${application.slot}`,
      value: application.slot,
    }))
  if (capability.targetKinds.includes('ai-model'))
    return settings.ai.profiles.map((profile) => ({
      label: `${profile.label} · ${profile.slot}`,
      value: profile.slot,
    }))
  if (capability.targetKinds.includes('http-origin'))
    return settings.network.httpOrigins.map((origin) => ({
      label: `${origin.label} · ${origin.slot}`,
      value: origin.slot,
    }))
  return []
})
const targetSettingsSection = computed<'automation' | 'applications' | 'ai' | 'network'>(() => {
  const kinds = targetCapability.value?.targetKinds ?? []
  if (kinds.includes('installed-application')) return 'applications'
  if (kinds.includes('ai-model')) return 'ai'
  if (kinds.includes('http-origin')) return 'network'
  return 'automation'
})
const portTitle = computed(() => {
  if (props.item.kind === 'config') return props.item.field.id
  const key = props.item.port.titleKey
  return key && te(key) ? t(key) : props.item.port.id
})
const typeTitle = computed(() => {
  if (props.item.kind === 'config') return props.item.field.control
  const key = props.item.port.type.titleKey
  return key && te(key) ? t(key) : props.item.port.type.label
})
</script>
