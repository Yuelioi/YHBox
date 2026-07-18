<template>
  <aside class="flex h-full w-[340px] shrink-0 flex-col border-l border-default bg-default">
    <div class="flex items-center justify-between border-b border-default px-4 py-3">
      <div class="min-w-0">
        <h2 class="truncate text-sm font-semibold text-highlighted">
          {{ t('workflow.inspector.title') }}
        </h2>
        <p class="truncate font-mono text-[10px] text-dimmed">
          {{ node?.id || t('workflow.inspector.no_selection') }}
        </p>
      </div>
      <UButton
        v-if="node"
        icon="i-tabler-trash"
        color="error"
        variant="ghost"
        size="xs"
        :aria-label="t('workflow.inspector.remove_node')"
        @click="emit('command', { kind: 'remove-node', nodeId: node.id })"
      />
    </div>

    <div
      v-if="!node || !projection"
      class="flex flex-1 items-center justify-center px-8 text-center"
    >
      <div>
        <UIcon name="i-tabler-pointer" class="mx-auto mb-3 size-6 text-dimmed" />
        <p class="text-xs text-muted">{{ t('workflow.inspector.select_hint') }}</p>
      </div>
    </div>

    <div v-else class="flex-1 space-y-6 overflow-y-auto p-4">
      <p
        v-if="projectionDescription"
        class="rounded-lg border border-default bg-elevated/30 px-3 py-2 text-[11px] leading-5 text-muted"
      >
        {{ projectionDescription }}
      </p>

      <section class="space-y-3">
        <label class="block text-xs font-medium text-toned" for="workflow-node-label">
          {{ t('workflow.inspector.label') }}
        </label>
        <UInput
          id="workflow-node-label"
          :model-value="node.label || ''"
          :placeholder="t('workflow.inspector.label_placeholder')"
          class="w-full"
          @change="setLabel"
        />
      </section>

      <section v-if="projection.configFields.length" class="space-y-3">
        <h3 class="text-xs font-semibold text-highlighted">
          {{ t('workflow.inspector.configuration') }}
        </h3>
        <div v-for="field in projection.configFields" :key="field.id" class="space-y-2">
          <GeneratedFieldEditor
            :field="field"
            :model-value="effectiveConfigValue(field.id)"
            :state-variables="variables.map((variable) => variable.name)"
            :select-items="targetItems(field.id)"
            :select-placeholder="t('workflow.inspector.select_target')"
            @update:model-value="
              emit('command', {
                kind: 'set-config',
                nodeId: node.id,
                fieldId: field.id,
                value: $event,
              })
            "
          />
          <div v-if="isTargetField(field.id)" class="flex items-center gap-2 text-[11px]">
            <UBadge
              :color="
                hasNodeOverride(field.id)
                  ? 'warning'
                  : inheritedTarget(field.id)
                    ? 'primary'
                    : 'error'
              "
              variant="soft"
              size="sm"
            >
              {{
                t(
                  hasNodeOverride(field.id)
                    ? 'workflow.inspector.target_overridden'
                    : inheritedTarget(field.id)
                      ? 'workflow.inspector.target_inherited'
                      : 'workflow.inspector.target_missing',
                )
              }}
            </UBadge>
            <span
              v-if="inheritedTarget(field.id) && !hasNodeOverride(field.id)"
              class="truncate text-muted"
            >
              {{ inheritedTarget(field.id) }}
            </span>
            <UButton
              v-if="hasNodeOverride(field.id) && inheritedTarget(field.id)"
              class="ml-auto"
              color="neutral"
              variant="ghost"
              size="xs"
              :label="t('workflow.inspector.restore_inherited')"
              @click="emit('command', { kind: 'clear-config', nodeId: node.id, fieldId: field.id })"
            />
          </div>
          <div
            v-if="isTargetField(field.id) && targetItems(field.id)?.length === 0"
            class="flex items-center gap-2 rounded-lg border border-warning/30 bg-warning/10 px-3 py-2"
          >
            <p class="min-w-0 flex-1 text-[11px] leading-5 text-warning">
              {{ t('workflow.inspector.no_installed_target') }}
            </p>
            <UButton
              :to="{ path: '/settings', query: { section: targetSettingsSection(field.id) } }"
              :label="t('workflow.inspector.configure_target')"
              icon="i-tabler-settings"
              color="warning"
              variant="soft"
              size="xs"
            />
          </div>
        </div>
      </section>

      <section v-if="projection.dataInputs.length" class="space-y-3">
        <h3 class="text-xs font-semibold text-highlighted">
          {{ t('workflow.inspector.inputs') }}
        </h3>
        <WorkflowInputBindingEditor
          v-for="port in projection.dataInputs"
          :key="port.id"
          :node="node"
          :port="port"
          @command="emit('command', $event)"
        />
      </section>

      <UCollapsible v-if="projection.capabilities.length || projection.statusEvents.length">
        <UButton
          :label="t('workflow.inspector.advanced')"
          icon="i-tabler-adjustments-horizontal"
          trailing-icon="i-tabler-chevron-down"
          color="neutral"
          variant="ghost"
          class="w-full justify-start"
        />
        <template #content>
          <div class="space-y-5 pt-3">
            <section v-if="projection.capabilities.length" class="space-y-3">
              <h3 class="text-xs font-semibold text-highlighted">
                {{ t('workflow.inspector.capabilities') }}
              </h3>
              <div
                v-for="capability in projection.capabilities"
                :key="capability.requirementId"
                class="rounded-lg border border-default px-3 py-2.5"
              >
                <p class="text-xs font-medium text-toned">{{ capability.requirementId }}</p>
                <p class="mt-1 text-[11px] text-muted">
                  {{ capability.operations.join(', ') }}
                </p>
                <p class="mt-1 font-mono text-[10px] text-dimmed">
                  {{ capability.risk }} / {{ capability.consent }}
                </p>
              </div>
            </section>

            <section v-if="projection.statusEvents.length" class="space-y-2">
              <h3 class="text-xs font-semibold text-highlighted">
                {{ t('workflow.inspector.observed_status') }}
              </h3>
              <p class="text-[11px] leading-5 text-muted">
                {{ t('workflow.inspector.status_hint') }}
              </p>
              <code class="block whitespace-pre-wrap text-[10px] text-toned">{{
                projection.statusEvents.map((event) => event.code).join('\n')
              }}</code>
            </section>
          </div>
        </template>
      </UCollapsible>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { TargetDefault, Variable } from '../../../../contracts/workflow/3.1/workflow-source'
import type { TypeProjection } from '../../../../contracts/node/3.1/authoring-projection'
import type { EditorCommand, Node, NodeProjection } from '@/app/editor/EditorSession'
import GeneratedFieldEditor from '@/app/editor/GeneratedFieldEditor.vue'
import WorkflowInputBindingEditor from '@/app/editor/WorkflowInputBindingEditor.vue'
import { useSettingsStore } from '@/stores/settings'

const props = defineProps<{
  node: Node | null
  projection: NodeProjection | null
  variables: Variable[]
  targetDefaults: TargetDefault[]
  types: TypeProjection[]
}>()
const emit = defineEmits<{ command: [command: EditorCommand] }>()
const { t, te } = useI18n()
const settingsStore = useSettingsStore()
const projectionDescription = computed(() => {
  const key = props.projection?.descriptionKey
  return key && te(key) ? t(key) : ''
})
function targetItems(fieldId: string): Array<{ label: string; value: string }> | undefined {
  const capability = targetCapability(fieldId)
  if (!capability) return undefined
  const settings = settingsStore.data
  if (!settings) return []
  if (
    capability.targetKinds.some((kind) =>
      settings.automation.targets.some((target) => target.targetKind === kind),
    )
  )
    return settings.automation.targets
      .filter((target) => capability.targetKinds.includes(target.targetKind))
      .map((target) => ({
        label: `${target.label} · ${target.slot}`,
        value: target.slot,
      }))
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
}

function targetCapability(fieldId: string) {
  return props.projection?.capabilities.find(
    (candidate) => candidate.targetSlotConfigKey === fieldId,
  )
}

function inheritedTarget(fieldId: string): string {
  const target = targetCapability(fieldId)?.targetSlot
  return props.targetDefaults.find((candidate) => candidate.target === target)?.slot ?? ''
}

function hasNodeOverride(fieldId: string): boolean {
  return Boolean(props.node && Object.prototype.hasOwnProperty.call(props.node.config, fieldId))
}

function effectiveConfigValue(fieldId: string): unknown {
  if (hasNodeOverride(fieldId)) return props.node?.config[fieldId]
  return inheritedTarget(fieldId) || undefined
}

function isTargetField(fieldId: string): boolean {
  return Boolean(targetCapability(fieldId))
}

function targetSettingsSection(fieldId: string): 'automation' | 'applications' | 'ai' | 'network' {
  const kinds = targetCapability(fieldId)?.targetKinds ?? []
  if (kinds.includes('installed-application')) return 'applications'
  if (kinds.includes('ai-model')) return 'ai'
  if (kinds.includes('http-origin')) return 'network'
  return 'automation'
}

function setLabel(event: Event): void {
  if (!props.node) return
  emit('command', {
    kind: 'set-node-label',
    nodeId: props.node.id,
    label: (event.target as HTMLInputElement).value,
  })
}
</script>
