<!-- 容器编辑器帮助弹窗：多 tab（快速上手 / 错误码 / 快捷键 / 节点速查）。
     入口: 面包屑栏「帮助」按钮 / 右面板「打开帮助」。弹窗模式照 NodeExplorerModal。 -->
<template>
  <BaseModal
    v-model:open="modelOpen"
    :title="t('editor.help.title')"
    icon="i-tabler-help-circle"
    size="3xl"
  >
    <div class="text-xs text-muted">
      <!-- Tabs -->
      <div class="flex flex-wrap gap-1 mb-3">
        <button
          v-for="tb in tabs"
          :key="tb.key"
          type="button"
          class="px-3 py-1.5 text-[12px] rounded-md transition-colors"
          :class="
            activeTab === tb.key
              ? 'bg-primary/15 text-primary'
              : 'text-dimmed hover:text-default hover:bg-elevated/40'
          "
          @click="activeTab = tb.key"
        >
          {{ t(tb.label) }}
        </button>
      </div>

      <!-- 快速上手 -->
          <div v-if="activeTab === 'getting_started'" class="space-y-4">
            <p class="text-toned leading-relaxed">{{ t('editor.help.gs.intro') }}</p>
            <ol class="space-y-1.5 list-decimal pl-5 marker:text-dimmed text-toned">
              <li v-for="i in 4" :key="i">{{ t(`editor.help.gs.step${i}`) }}</li>
            </ol>
            <div class="flex items-start gap-2 rounded-lg bg-amber-500/10 border border-amber-500/30 p-3">
              <UIcon name="i-tabler-shield-lock" class="size-4 text-amber-300 shrink-0 mt-0.5" />
              <div class="text-amber-300">
                <div class="font-medium mb-0.5">{{ t('editor.help.gs.uac_title') }}</div>
                <div class="text-[11px] leading-relaxed text-amber-200/90">{{ t('editor.help.gs.uac') }}</div>
              </div>
            </div>
          </div>

          <!-- 错误码参考 -->
          <div v-else-if="activeTab === 'errorcodes'" class="space-y-2">
            <p class="text-dimmed leading-relaxed">{{ t('editor.help.errorcodes_hint') }}</p>
            <div
              v-for="code in errorCodes"
              :key="code"
              class="flex items-start gap-3 rounded-lg bg-elevated/30 border border-default/50 p-2.5"
            >
              <code
                class="shrink-0 font-mono text-[10px] text-rose-300 bg-rose-500/10 border border-rose-500/20 rounded px-1.5 py-0.5"
              >{{ code }}</code>
              <div class="min-w-0">
                <div class="text-default mb-0.5">{{ errorCodeLabel(code) }}</div>
                <div class="text-[11px] text-muted leading-relaxed">{{ errorCodeDesc(code) }}</div>
              </div>
            </div>
          </div>

          <!-- 快捷键参考 -->
          <div v-else-if="activeTab === 'shortcuts'" class="space-y-2">
            <p class="text-dimmed leading-relaxed">{{ t('editor.help.shortcuts_hint') }}</p>
            <ul class="space-y-1.5">
              <li
                v-for="k in editorKeys"
                :key="k.key"
                class="flex items-center justify-between gap-3"
              >
                <span class="text-toned">{{ t(k.label) }}</span>
                <kbd
                  class="px-1.5 py-0.5 bg-elevated rounded border border-default text-[10px] font-mono shrink-0"
                >{{ k.hotkeyStr }}</kbd>
              </li>
            </ul>
          </div>

          <!-- 节点速查 -->
          <div v-else class="space-y-2">
            <p class="text-dimmed leading-relaxed">{{ t('editor.help.nodes_hint') }}</p>
            <div v-for="g in nodeGroups" :key="g.label" class="flex gap-3">
              <span class="text-toned shrink-0 w-14 font-medium">{{ t(g.label) }}</span>
              <span class="text-muted leading-relaxed">{{ t(g.desc) }}</span>
            </div>
          </div>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNodeRegistryStore } from '@/stores/nodeRegistry'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'
import { EDITOR_KEYS } from '@/composables/containerEditor/useEditorHotkeys'
import BaseModal from '@/components/common/BaseModal.vue'

const { t, te } = useI18n()

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ 'update:open': [v: boolean] }>()
const modelOpen = useDialogOpen(props, emit)

const tabs = [
  { key: 'getting_started', label: 'editor.help.tab.getting_started' },
  { key: 'errorcodes', label: 'editor.help.tab.errorcodes' },
  { key: 'shortcuts', label: 'editor.help.tab.shortcuts' },
  { key: 'nodes', label: 'editor.help.tab.nodes' },
] as const
const activeTab = ref<(typeof tabs)[number]['key']>('getting_started')

// 快捷键来自编辑器 hotkey 单源（label 已 i18n, hotkeyStr 字面值）。
const editorKeys = EDITOR_KEYS

// 错误码：码全集走后端注册表 (store 单源)，标签复用 errorcode.<code>，描述走 editor.help.errorcode_desc.<code>。
const nodeRegistry = useNodeRegistryStore()
const errorCodes = computed(() => nodeRegistry.errorCodes)
function errorCodeLabel(code: string): string {
  const key = `errorcode.${code}`
  return te(key) ? t(key) : code
}
function errorCodeDesc(code: string): string {
  const key = `editor.help.errorcode_desc.${code}`
  return te(key) ? t(key) : ''
}

// 节点类别速查（具体节点走 Tab 打开 NodeExplorer）。
const nodeGroups = [
  { label: 'editor.help.node_group.control.label', desc: 'editor.help.node_group.control.desc' },
  { label: 'editor.help.node_group.variable.label', desc: 'editor.help.node_group.variable.desc' },
  { label: 'editor.help.node_group.image.label', desc: 'editor.help.node_group.image.desc' },
  { label: 'editor.help.node_group.input.label', desc: 'editor.help.node_group.input.desc' },
  { label: 'editor.help.node_group.event.label', desc: 'editor.help.node_group.event.desc' },
  { label: 'editor.help.node_group.subgraph.label', desc: 'editor.help.node_group.subgraph.desc' },
  { label: 'editor.help.node_group.system.label', desc: 'editor.help.node_group.system.desc' },
  { label: 'editor.help.node_group.config.label', desc: 'editor.help.node_group.config.desc' },
  { label: 'editor.help.node_group.debug.label', desc: 'editor.help.node_group.debug.desc' },
]
</script>
