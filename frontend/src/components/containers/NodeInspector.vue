<template>
  <div v-if="!node" class="text-sm text-dimmed">{{ t('inspector.no_selection') }}</div>

  <div v-else>
    <!-- Header: 大图标 + 中文名 + ID -->
    <header class="flex items-start gap-3 pb-4 mb-4 border-b border-default">
      <div
        class="size-10 rounded-lg flex items-center justify-center shrink-0"
        :class="[visual.bg, visual.border, 'border']"
      >
        <UIcon :name="visual.icon" class="size-5 text-default" />
      </div>
      <div class="min-w-0 flex-1">
        <h3 class="text-sm font-medium text-highlighted leading-tight">{{ label }}</h3>
        <p class="text-[11px] text-dimmed font-mono truncate mt-0.5">
          {{ node.kind }} · {{ node.id }}
        </p>
      </div>
      <UButton
        size="xs"
        variant="ghost"
        color="error"
        icon="i-tabler-trash"
        :title="t('inspector.delete_node_tooltip')"
        @click="$emit('delete')"
      />
    </header>

    <!-- 标签 (Label) — 用户可编辑的节点显示名 -->
    <section class="mb-4">
      <UFormField :label="t('inspector.label_field_label')" :hint="t('inspector.label_field_hint')">
        <UInput
          :model-value="node.label ?? ''"
          :placeholder="t('inspector.label_field_placeholder')"
          size="sm"
          class="w-full"
          @update:model-value="(v: string) => $emit('label-update', v)"
        />
      </UFormField>
    </section>

    <!-- 用法说明 -->
    <section
      v-if="description"
      class="mb-5 rounded-md bg-elevated/30 border border-default/40 px-3 py-2.5"
    >
      <div class="flex items-start gap-2">
        <UIcon name="i-tabler-info-circle" class="size-3.5 text-primary shrink-0 mt-0.5" />
        <p class="text-[12px] text-toned leading-relaxed">{{ description }}</p>
      </div>
    </section>

    <!-- 并发警告 -->
    <section
      v-if="concurrencyWarning"
      class="mb-5 rounded-md bg-amber-500/10 border border-amber-500/30 px-3 py-2.5"
    >
      <div class="flex items-start gap-2">
        <UIcon name="i-tabler-alert-triangle" class="size-3.5 text-amber-300 shrink-0 mt-0.5" />
        <div class="text-[12px] text-amber-300">
          <div class="font-medium leading-tight">{{ t('inspector.concurrency_warn_title') }}</div>
          <div class="text-amber-300/80 mt-1 leading-relaxed">{{ concurrencyWarning }}</div>
        </div>
      </div>
    </section>

    <!-- Expr 链提示 + 一键合并按钮 -->
    <section
      v-if="exprChainHint"
      class="mb-5 rounded-md bg-amber-500/10 border border-amber-500/30 px-3 py-2.5"
    >
      <div class="flex items-start gap-2">
        <UIcon name="i-tabler-info-circle" class="size-3.5 text-amber-300 shrink-0 mt-0.5" />
        <div class="text-[12px] text-amber-300 flex-1">
          <div class="font-medium leading-tight">{{ t('inspector.expr_chain_title') }}</div>
          <div class="text-amber-300/80 mt-1 leading-relaxed font-mono text-[11px]">
            value → {{ exprChainHint.targetID }}.{{ exprChainHint.targetPin }}
          </div>
          <div class="text-amber-300/80 mt-1 mb-2 leading-relaxed">
            {{ t('inspector.expr_chain_desc') }}
          </div>
          <UButton size="xs" color="warning" variant="soft" icon="i-tabler-arrow-merge" @click="onFuseExpr">
            {{ t('inspector.expr_chain_fuse') }}
          </UButton>
        </div>
      </div>
    </section>

    <!-- 屏幕选择工具：根据 kind 显示对应快捷 -->
    <section
      v-if="canPickPoint || canPickRect"
      class="mb-4 rounded-md bg-primary/5 border border-primary/30 p-3 space-y-2"
    >
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-crosshair" class="size-3.5 text-primary" />
        <span class="text-[11px] text-toned">{{ t('inspector.screen_pick_label') }}</span>
      </div>
      <div class="flex items-center gap-2 flex-wrap">
        <UButton
          v-if="canPickPoint"
          size="xs"
          variant="soft"
          color="primary"
          icon="i-tabler-pointer"
          :loading="picking"
          @click="onPickPoint"
        >
          {{ t('inspector.screen_pick_point') }}
        </UButton>
        <UButton
          v-if="canPickRect"
          size="xs"
          variant="soft"
          color="primary"
          icon="i-tabler-frame"
          :loading="picking"
          @click="onPickRect"
        >
          {{ t('inspector.screen_pick_rect') }}
        </UButton>
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-pointer"
          @click="onOpenHUD"
        >
          {{ t('inspector.screen_pick_hud') }}
        </UButton>
      </div>
      <p class="text-[10px] text-dimmed leading-snug">
        {{ t('inspector.screen_pick_hint', { action: canPickRect ? t('inspector.screen_pick_action_drag') : t('inspector.screen_pick_action_click') }) }}
      </p>
    </section>

    <!-- Subgraph 节点：1:1 模型 — 节点 ↔ 子图 强绑定 + 外部统一编辑 -->
    <section v-if="node.kind === 'Subgraph'" class="space-y-3">
      <div class="rounded-md bg-elevated/30 border border-default/40 p-3 space-y-3">
        <!-- 头部：图标 + 节点数 + 进入按钮 -->
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-package" class="size-4 text-fuchsia-300" />
          <span class="text-xs text-toned">{{ t('node.Subgraph.inspector.binding_label') }}</span>
          <UBadge size="xs" variant="soft" color="neutral" class="ml-auto">
            {{ t('containers.node_count', { n: boundSubgraph?.graph?.nodes?.length ?? 0 }) }}
          </UBadge>
        </div>

        <!-- 子图 label 编辑 -->
        <div class="space-y-1">
          <label class="text-[11px] text-toned">{{ t('node.Subgraph.inspector.subgraph_label_field') }}</label>
          <UInput
            :model-value="boundSubgraph?.label ?? ''"
            size="sm"
            :placeholder="t('node.Subgraph.inspector.subgraph_label_placeholder')"
            :disabled="!boundSubgraph"
            @update:model-value="(v: string) => onPatchSubgraph({ label: v })"
          />
        </div>

        <!-- 子图描述编辑 -->
        <div class="space-y-1">
          <label class="text-[11px] text-toned">{{ t('node.Subgraph.inspector.subgraph_description_field') }}</label>
          <UTextarea
            :model-value="(boundSubgraph as any)?.description ?? ''"
            size="sm"
            :rows="2"
            :placeholder="t('node.Subgraph.inspector.subgraph_description_placeholder')"
            :disabled="!boundSubgraph"
            @update:model-value="(v: string) => onPatchSubgraph({ description: v })"
          />
        </div>

        <!-- 子图标签 tags -->
        <div class="space-y-1">
          <label class="text-[11px] text-toned">{{ t('node.Subgraph.inspector.subgraph_tags_field') }}</label>
          <UInputMenu
            :model-value="(boundSubgraph as any)?.tags ?? []"
            multiple
            creatable
            :items="allSubgraphTagsList"
            size="sm"
            :placeholder="t('node.Subgraph.inspector.subgraph_tags_placeholder')"
            :disabled="!boundSubgraph"
            @update:model-value="(v: string[]) => onPatchSubgraph({ tags: v })"
          />
        </div>

        <UButton
          size="sm"
          variant="soft"
          color="primary"
          icon="i-tabler-arrow-right"
          block
          :disabled="!boundSubgraph"
          @click="onEnterSubgraph"
        >
          {{ t('node.Subgraph.inspector.enter_subgraph') }}
        </UButton>
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-cloud-upload"
          block
          :disabled="!boundSubgraph || publishing"
          :loading="publishing"
          @click="onPublishToLibrary"
        >
          {{ publishing ? t('node.Subgraph.inspector.publishing') : t('node.Subgraph.inspector.publish_to_library') }}
        </UButton>
        <p class="text-[10px] text-dimmed leading-snug">
          {{ t('node.Subgraph.inspector.footer_meta_hint') }}<br />
          {{ t('node.Subgraph.inspector.footer_delete_hint') }}
        </p>
      </div>
    </section>

    <!-- MouseCalibration 节点 — 强制 sync 形态防止误用 -->
    <section v-else-if="node.kind === 'MouseCalibration'" class="space-y-3">
      <div
        v-if="isCalibrationForeign"
        class="rounded-md bg-amber-500/10 border border-amber-500/30 px-3 py-2.5 text-[12px] text-amber-300"
      >
        <UIcon name="i-tabler-alert-triangle" class="size-3.5 inline mr-1 align-middle" />
        {{ t('node.MouseCalibration.inspector.foreign_warn', { nodeVal: node.config?.counts360, globalVal: globalCounts360 }) }}<br />
        {{ t('node.MouseCalibration.inspector.foreign_hint') }}
        <div class="mt-2 flex gap-1.5 flex-wrap">
          <UButton
            size="xs"
            color="warning"
            variant="solid"
            icon="i-tabler-refresh"
            @click="$emit('update', { ...node.config, counts360: globalCounts360 })"
          >{{ t('node.MouseCalibration.inspector.override_with_local', { n: globalCounts360 }) }}</UButton>
          <UButton
            size="xs"
            variant="ghost"
            color="warning"
            icon="i-tabler-bolt"
            @click="onSyncAllFromForeign"
          >{{ t('node.MouseCalibration.inspector.sync_all') }}</UButton>
        </div>
      </div>

      <div class="rounded-md bg-elevated/30 border border-default/40 p-3 space-y-2">
        <div class="flex items-baseline gap-2">
          <span class="text-xs text-toned">{{ t('node.MouseCalibration.inspector.counts_label') }}</span>
        </div>
        <div class="flex items-center gap-2">
          <span
            class="text-2xl font-mono tabular-nums"
            :class="(node.config?.counts360 ?? 0) > 0 ? 'text-emerald-300' : 'text-rose-300'"
          >{{ node.config?.counts360 ?? 0 }}</span>
          <span class="text-[11px] text-dimmed">{{ (node.config?.counts360 ?? 0) > 0 ? t('node.MouseCalibration.inspector.calibrated') : t('node.MouseCalibration.inspector.not_calibrated') }}</span>
        </div>
        <p class="text-[11px] text-dimmed leading-relaxed">
          {{ t('node.MouseCalibration.inspector.counts_hint') }}<br />
          <span class="text-rose-300/80">{{ t('node.MouseCalibration.inspector.counts_warn') }}</span>
        </p>
        <UButton
          size="sm"
          color="primary"
          variant="solid"
          icon="i-tabler-target"
          block
          @click="onOpenCalibrator"
        >{{ t('node.MouseCalibration.inspector.start_calibrate') }}</UButton>

        <UCollapsible class="mt-3">
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-chevron-right"
            class="w-full justify-start"
          >{{ t('node.MouseCalibration.inspector.advanced_manual') }}</UButton>

          <template #content>
            <UInputNumber
              :model-value="node.config?.counts360 ?? 0"
              size="sm"
              class="w-full mt-2"
              @update:model-value="(v: number) => $emit('update', { ...(node?.config ?? {}), counts360: v })"
            />
          </template>
        </UCollapsible>
      </div>
    </section>

    <!-- WindowTarget: v3 Phase B 声明目标游戏窗口 + input/capture backend -->
    <section v-else-if="node.kind === 'WindowTarget'" class="mb-5 space-y-4">
      <!-- 捕获按钮 (F9 全局热键流程) -->
      <div>
        <UButton
          :icon="capturing ? 'i-tabler-loader-2' : 'i-tabler-target'"
          :loading="capturing"
          size="sm"
          block
          @click="toggleWindowCapture"
        >
          {{ capturing ? t('node.WindowTarget.inspector.capture_waiting') : t('node.WindowTarget.inspector.capture_start') }}
        </UButton>
        <p class="text-xs text-dimmed mt-1">
          {{ t('node.WindowTarget.inspector.capture_hint_a') }}
          {{ t('node.WindowTarget.inspector.capture_hint_b') }}
        </p>
      </div>

      <!-- match section -->
      <div class="border border-default rounded-lg p-3 space-y-2">
        <h4 class="text-sm font-semibold">{{ t('node.WindowTarget.inspector.match_section') }}</h4>
        <UFormField :label="t('node.WindowTarget.inspector.title_label')">
          <UInput v-model="wtConfig.Title" :placeholder="t('node.WindowTarget.inspector.title_placeholder')" />
        </UFormField>
        <UFormField :label="t('node.WindowTarget.inspector.class_label')">
          <UInput v-model="wtConfig.Class" placeholder="UnrealWindow" />
        </UFormField>
        <UFormField :label="t('node.WindowTarget.inspector.process_label')">
          <UInput v-model="wtConfig.ProcessName" placeholder="game.exe" />
        </UFormField>
        <UFormField :label="t('node.WindowTarget.inspector.title_match_label')">
          <USelect
            v-model="wtConfig.TitleMatch"
            class="w-full"
            :items="titleMatchOptions"
          />
        </UFormField>
      </div>

      <!-- runtime section -->
      <div class="border border-default rounded-lg p-3 space-y-2">
        <h4 class="text-sm font-semibold">{{ t('node.WindowTarget.inspector.runtime_section') }}</h4>
        <UFormField :label="t('node.WindowTarget.inspector.input_backend_label')">
          <USelect
            v-model="wtConfig.InputBackend"
            class="w-full"
            :items="inputBackendOptions"
          />
        </UFormField>
        <UFormField :label="t('node.WindowTarget.inspector.capture_backend_label')">
          <USelect
            v-model="wtConfig.CaptureBackend"
            class="w-full"
            :items="captureBackendOptions"
          />
        </UFormField>
      </div>
    </section>

    <!-- PlayClip: clip 绑死显示 (一节点一 clip, 不允许下拉换) + 重录覆盖 + 裁剪段编辑 -->
    <section v-else-if="node.kind === 'PlayClip'" class="mb-5 space-y-3">
      <!-- 绑定的 clip 概要 (只读) -->
      <div
        v-if="selectedClip"
        class="rounded-md bg-elevated/30 border border-default/40 px-3 py-2.5 text-[11px] space-y-1.5"
      >
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-vinyl" class="size-3.5 text-emerald-400 shrink-0" />
          <span class="text-default font-medium truncate">{{ selectedClip.label || selectedClip.id }}</span>
        </div>
        <div class="flex items-center gap-3 text-[10px] text-dimmed">
          <span class="flex items-center gap-1"><UIcon name="i-tabler-clock" class="size-3" />{{ formatDuration(selectedClip.durationUs) }}</span>
          <span class="flex items-center gap-1"><UIcon name="i-tabler-calendar" class="size-3" />{{ formatDate(selectedClip.createdAt) }}</span>
        </div>
        <div
          v-if="selectedClip.tags && selectedClip.tags.length"
          class="flex items-center gap-1 flex-wrap"
        >
          <UBadge
            v-for="t in selectedClip.tags"
            :key="t"
            size="xs"
            color="neutral"
            variant="subtle"
          >{{ t }}</UBadge>
        </div>
        <div class="text-[10px] text-dimmed font-mono break-all">{{ selectedClip.id }}</div>
      </div>
      <div v-else class="rounded-md bg-amber-500/10 border border-amber-500/30 px-3 py-2 text-[11px] text-amber-300">
        <UIcon name="i-tabler-alert-triangle" class="size-3 inline mr-1" />
        {{ t('node.PlayClip.inspector.clip_missing', { id: node.config?.ClipID || t('node.PlayClip.inspector.clip_unset_placeholder') }) }}
      </div>

      <!-- 重新录制覆盖 (一节点一 clip, 不允许下拉切换; 想换 clip 就重录) -->
      <div class="flex items-center gap-2">
        <UButton
          size="xs"
          color="primary"
          variant="soft"
          icon="i-tabler-circle-dot"
          class="flex-1"
          @click="$emit('request-record', { mode: 'precise', replaceNodeID: node.id })"
        >{{ t('node.PlayClip.inspector.record_precise') }}</UButton>
        <UButton
          size="xs"
          color="neutral"
          variant="soft"
          icon="i-tabler-zap"
          class="flex-1"
          @click="$emit('request-record', { mode: 'simple', replaceNodeID: node.id })"
        >{{ t('node.PlayClip.inspector.record_simple') }}</UButton>
      </div>
      <p class="text-[10px] text-dimmed leading-snug -mt-1">
        {{ t('node.PlayClip.inspector.bind_hint') }}
      </p>

      <!-- keepRanges 编辑器 -->
      <div>
        <!-- 可视化 timeline (拖拽添加/调长度/删) -->
        <ClipTimeline
          v-if="selectedClip"
          class="mb-3"
          :duration-ms="Math.floor(selectedClip.durationUs / 1000)"
          :ranges="keepRanges"
          @add="onTimelineAdd"
          @update="onTimelineUpdate"
          @remove="removeRange"
        />

        <div class="flex items-center justify-between mb-1">
          <span class="text-[11px] text-toned">{{ t('node.PlayClip.inspector.keep_ranges_label') }}</span>
          <UButton size="xs" variant="ghost" icon="i-tabler-plus" @click="addRange">{{ t('common.add') }}</UButton>
        </div>
        <p class="text-[10px] text-dimmed mb-2 leading-snug">
          {{ t('node.PlayClip.inspector.keep_ranges_hint') }}
        </p>
        <div v-if="keepRanges.length === 0" class="text-[10px] text-dimmed italic">
          {{ t('node.PlayClip.inspector.full_playback') }}
        </div>
        <div v-else class="space-y-1.5">
          <div
            v-for="(r, idx) in keepRanges"
            :key="idx"
            class="flex items-center gap-1.5"
          >
            <UInput
              :model-value="r.fromMs"
              type="number"
              size="xs"
              class="w-24"
              placeholder="from ms"
              @update:model-value="updateRange(idx, 'fromMs', Number($event))"
            />
            <span class="text-[10px] text-dimmed">→</span>
            <UInput
              :model-value="r.toMs"
              type="number"
              size="xs"
              class="w-24"
              placeholder="to ms"
              @update:model-value="updateRange(idx, 'toMs', Number($event))"
            />
            <UButton
              size="xs"
              variant="ghost"
              color="error"
              icon="i-tabler-x"
              @click="removeRange(idx)"
            />
          </div>
        </div>
      </div>
    </section>

    <!-- Switch: 多路分支 case 编辑器 -->
    <section v-else-if="node.kind === 'Switch'" class="mb-5">
      <SwitchInspector
        :node="node"
        :edges="edges ?? []"
        @update="emit('update', $event)"
      />
    </section>

    <!-- Data-in pin literal 编辑 (未连入边时 → 走 config.literal inline 值) -->
    <section v-if="dataInLiterals.length > 0" class="mb-5">
      <h4 class="text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed mb-3">
        {{ t('inspector.literal_section') }}
      </h4>
      <div class="space-y-3">
        <div v-for="lit in dataInLiterals" :key="lit.name" class="space-y-1.5">
          <label class="block text-xs text-toned">
            {{ lit.name }}
            <span class="text-[10px] text-dimmed font-mono ml-1">({{ lit.type }})</span>
          </label>
          <PinLiteral
            :type="(lit.type as any)"
            :model-value="getLiteral(lit.name)"
            @update:model-value="(v: any) => setLiteral(lit.name, v)"
          />
        </div>
      </div>
    </section>

    <!-- Config fields (non-pin config — enum/path/template/etc) -->
    <section v-if="fields.length > 0">
      <h4 class="text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed mb-3">{{ t('inspector.config_section') }}</h4>
      <div class="space-y-4">
        <div v-for="field in fields" :key="field.key" class="space-y-1.5">
          <label class="block text-xs text-toned">{{ t(field.label) }}</label>
          <!-- v4: 'expr' field type removed; v3 expr inputs migrated to data-in pin literals
               (shown in "数据输入 (literal)" section above). -->
          <USelect
            v-if="field.type === 'select'"
            :model-value="getCfg(field.key)"
            :items="field.options ?? []"
            size="md"
            class="w-full"
            :ui="{ content: 'min-w-[280px]' }"
            @update:model-value="setCfg(field.key, String($event))"
          />
          <UInput
            v-else-if="field.type === 'text'"
            :model-value="getCfg(field.key)"
            size="md"
            :placeholder="field.placeholder"
            @update:model-value="setCfg(field.key, String($event))"
          />
          <USelect
            v-else-if="field.type === 'var-name-select'"
            :model-value="getCfg(field.key)"
            :items="varOptions"
            size="md"
            class="w-full"
            :ui="{ content: 'min-w-[280px]' }"
            :placeholder="t('inspector.select_var_placeholder')"
            @update:model-value="setCfg(field.key, String($event))"
          />
          <TemplatePicker
            v-else-if="field.type === 'template-picker'"
            :model-value="getCfg(field.key)"
            @update:model-value="setCfg(field.key, $event)"
          />
          <KeyCapture
            v-else-if="field.type === 'key-capture'"
            :model-value="getCfg(field.key)"
            @update:model-value="setCfg(field.key, $event)"
          />
          <p v-if="field.hint" class="text-[11px] text-dimmed leading-snug">{{ t(field.hint) }}</p>
        </div>
      </div>
    </section>

    <p v-else class="text-[12px] text-dimmed">{{ t('inspector.no_config') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, toRef } from 'vue'
import { Events } from '@wailsio/runtime'
import type { GraphNode } from '@/lib/backend'
import { backend } from '@/lib/backend'
// v4: ExpressionInput no longer imported — v3 'expr' field type removed in nodeFieldSchemas
// (config strings like $vars.X are gone; data-in pin literals handle their replacement).
import SwitchInspector from './inspector/SwitchInspector.vue'
import TemplatePicker from './TemplatePicker.vue'
import KeyCapture from './KeyCapture.vue'
import ClipTimeline from './ClipTimeline.vue'
import { useI18n } from 'vue-i18n'
import { KIND_LABEL_ZH, KIND_DESCRIPTION, KIND_VISUAL, PIN_SPECS, edgeKind } from './pinSpec'

const { t } = useI18n()
import PinLiteral from './inline/PinLiteral.vue'
import { NODE_FIELD_SCHEMAS, type Field } from './nodeFieldSchemas'
import { useSettingsStore } from '@/stores/settings'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { useEditorBusStore } from '@/stores/editorBus'
import { useClipsStore } from '@/stores/clips'
import { useToast } from '@nuxt/ui/composables'
import { useConfirm } from '@/composables/useConfirm'
import { useScreenPick } from '@/composables/containerEditor/useScreenPick'
import { useConcurrencyWarning } from '@/composables/containerEditor/useConcurrencyWarning'

const props = defineProps<{
  node: GraphNode | null
  varNames?: string[]
  nodes?: GraphNode[]
  edges?: { from: string; to: string }[]
}>()
const emit = defineEmits<{
  update: [config: Record<string, any>]
  'label-update': [v: string]
  delete: []
  'request-record': [opts: { mode: 'precise' | 'simple'; replaceNodeID: string }]
}>()

const settingsStore = useSettingsStore()
const globalCounts360 = computed(() => settingsStore.data?.ui?.mouseCounts360 ?? 0)

// Inline pin literal — Inspector 版.
// 对每个 PIN_SPECS[kind].dataIn 里没连入边的 pin, 暴露一个绑 config.literal[pinName] 的编辑器.
interface LiteralEntry { name: string; type: string }
const dataInLiterals = computed<LiteralEntry[]>(() => {
  if (!props.node) return []
  const spec = PIN_SPECS[props.node.kind]
  if (!spec) return []
  const incomingPins = new Set<string>()
  for (const e of props.edges ?? []) {
    // v4 C2: derive edge kind via (srcNode.kind, srcPin) — edge.kind field is gone.
    const [src, srcPin] = (e.from ?? '').split('.')
    const srcNode = (props.nodes ?? []).find((n: any) => n.id === src)
    if (!srcNode || edgeKind(srcNode.kind, srcPin) !== 'data') continue
    const [tgt, pin] = (e.to ?? '').split('.')
    if (tgt === props.node.id) incomingPins.add(pin)
  }
  // Expr has dynamic inputs (config.inputs[]); include those.
  const out: LiteralEntry[] = []
  if (props.node.kind === 'Expr') {
    for (const inp of (props.node.config?.inputs ?? []) as Array<{ name: string; type: string }>) {
      if (!incomingPins.has(inp.name)) out.push({ name: inp.name, type: inp.type ?? 'any' })
    }
  }
  for (const [name, type] of Object.entries(spec.dataIn ?? {})) {
    if (!incomingPins.has(name)) out.push({ name, type: String(type) })
  }
  return out
})

function getLiteral(pin: string): any {
  return props.node?.config?.literal?.[pin]
}
function setLiteral(pin: string, v: any) {
  if (!props.node) return
  const cfg = { ...(props.node.config ?? {}) }
  cfg.literal = { ...(cfg.literal ?? {}), [pin]: v }
  emit('update', cfg)
}

// 一键 fusion — Inspector 通过 editorBus store 请求, ContainerEditorView watch + 处理.
function onFuseExpr() {
  if (!exprChainHint.value || !props.node) return
  useEditorBusStore().requestExprFusion({
    sourceID: props.node.id,
    targetID: exprChainHint.value.targetID,
    targetPin: exprChainHint.value.targetPin,
  })
}

// Expr 链检测 — 如果当前 Expr 节点的 value out 唯一连到另一 Expr 的 input,
// Inspector 显示合并建议 + 按钮.
interface ChainHint { targetID: string; targetPin: string }
const exprChainHint = computed<ChainHint | null>(() => {
  if (!props.node || props.node.kind !== 'Expr') return null
  if (!props.nodes || !props.edges) return null
  const myID = props.node.id
  const outgoing = (props.edges ?? []).filter(
    (e: any) => {
      const [src, srcPin] = (e.from ?? '').split('.')
      // v4 C2: derive edge kind via (srcNode.kind, srcPin) — edge.kind field is gone.
      if (src !== myID || srcPin !== 'value') return false
      const srcNode = (props.nodes ?? []).find((n: any) => n.id === src)
      return srcNode ? edgeKind(srcNode.kind, srcPin) === 'data' : false
    },
  )
  if (outgoing.length !== 1) return null
  const [tgtID, tgtPin] = (outgoing[0].to ?? '').split('.')
  const tgtNode = (props.nodes ?? []).find((n: any) => n.id === tgtID)
  if (tgtNode?.kind !== 'Expr') return null
  return { targetID: tgtID, targetPin: tgtPin }
})
const isCalibrationForeign = computed(() => {
  if (!props.node || props.node.kind !== 'MouseCalibration') return false
  const nodeVal = props.node.config?.counts360 ?? 0
  return nodeVal > 0 && globalCounts360.value > 0 && nodeVal !== globalCounts360.value
})

function onOpenCalibrator() {
  if (!props.node) return
  const cfg = props.node.config ?? {}
  window.dispatchEvent(new CustomEvent('open-calibrator-modal', {
    detail: {
      onSave: (counts: number) => emit('update', { ...cfg, counts360: counts }),
    },
  }))
}

// Plan B Task E.8: FOREIGN warning 旁加"同步所有容器"按钮
const toastForSync = useToast()
const { confirm: confirmDialog } = useConfirm()
async function onSyncAllFromForeign() {
  const cur = globalCounts360.value
  if (cur <= 0) return
  const yes = await confirmDialog({
    title: t('node.MouseCalibration.inspector.sync_confirm_title'),
    description: t('node.MouseCalibration.inspector.sync_confirm_desc', { cur }),
    color: 'primary',
    confirmText: t('node.MouseCalibration.inspector.sync_confirm_ok'),
  })
  if (yes !== true) return
  const r = (await backend.containers.syncLocalMouseCalibration(cur)) as any
  toastForSync.add({ title: t('node.MouseCalibration.inspector.sync_toast_ok', { n: r?.updated?.length ?? 0 }), color: 'success' })
}

// Subgraph 调用节点：1:1 模型，只显示绑定的子图（不需 USelect 选择）
const editorStore = useContainerEditorStore()

// v4: 'var-name-select' 字段类型的候选 = NodeInspector 的 varNames prop (从父容器拿).
// 父容器 (ContainerEditorView) 把 draft.vars.map(v=>v.name) 传进来作 ExpressionInput 自动完成,
// 这里复用同源. 后续可扩 prop 为 varDecls 带 type, 在 label 显示 "name (type)".
const varOptions = computed<{ label: string; value: string }[]>(() => {
  return (props.varNames ?? []).map((name) => ({ label: name, value: name }))
})

const boundSubgraph = computed(() => {
  const sgID = props.node?.config?.SubgraphID
  if (!sgID) return null
  // v4: only visible (non-anonymous) subgraphs are valid Subgraph-call targets.
  // CollapsedNode-backers (isAnonymous) shouldn't show editable label/tags here.
  return editorStore.visibleSubgraphs.find((s) => s.id === sgID) ?? null
})

function onEnterSubgraph() {
  const sgID = props.node?.config?.SubgraphID
  if (!sgID) return
  editorStore.pushPath(String(sgID))
}

const publishing = ref(false)
async function onPublishToLibrary() {
  const sgID = props.node?.config?.SubgraphID
  const cid = editorStore.activeContainerID
  if (!sgID || !cid || !boundSubgraph.value) return
  const yes = await confirmDialog({
    title: t('node.Subgraph.inspector.publish_confirm_title'),
    description: t('node.Subgraph.inspector.publish_confirm_desc', { name: boundSubgraph.value.label || sgID }),
    color: 'primary',
    confirmText: t('node.Subgraph.inspector.publish_confirm_ok'),
  })
  if (yes !== true) return
  publishing.value = true
  try {
    await backend.library.exportSubgraph(cid, String(sgID), true)
    toastForSync.add({
      title: t('node.Subgraph.inspector.publish_toast_ok'),
      description: `${String(sgID)}`,
      color: 'success',
      icon: 'i-tabler-cloud-upload',
    })
  } catch (e) {
    toastForSync.add({ title: t('node.Subgraph.inspector.publish_toast_fail'), description: String(e), color: 'error' })
  } finally {
    publishing.value = false
  }
}

// 所有子图聚合 tags（给 UInputMenu autocomplete）— v4: 排除 isAnonymous (用 visibleSubgraphs).
const allSubgraphTagsList = computed(() => {
  const set = new Set<string>()
  for (const sg of editorStore.visibleSubgraphs ?? []) {
    for (const t of (sg as any).tags ?? []) set.add(t)
  }
  return [...set]
})

// 主图 Inspector 编辑绑定子图的 label/description/tags 时直接 mutate store 里的 sg 对象.
// store ref 上的 deep watch (useContainerDraft) 会自动 fire 标 dirty —
// 不需要 window.dispatchEvent 显式通知 view (之前的桥接已删, 同棵 Vue 树没必要走 window 总线).
function onPatchSubgraph(patch: Record<string, any>) {
  if (!boundSubgraph.value) return
  Object.assign(boundSubgraph.value as any, patch)
  // dirty 由 useContainerDraft 的 watch(editorStore.subgraphsForCurrentContainer, deep) 自动监控.
}

// KIND_LABEL_ZH[k] 值是 i18n key, t() 渲染. fallback 走 kind 字面 (节点未注册).
const label = computed(() => {
  if (!props.node) return ''
  const key = KIND_LABEL_ZH[props.node.kind]
  return key ? t(key) : props.node.kind
})
// KIND_DESCRIPTION[k] 值是 i18n key, t() 渲染.
const description = computed(() => {
  if (!props.node) return ''
  const key = KIND_DESCRIPTION[props.node.kind]
  return key ? t(key) : ''
})
const visual = computed(() =>
  props.node
    ? (KIND_VISUAL[props.node.kind] ?? {
        icon: 'i-tabler-circle',
        bg: 'bg-muted',
        border: 'border-default',
      })
    : { icon: '', bg: '', border: '' },
)


const fields = computed<Field[]>(() => (props.node ? (NODE_FIELD_SCHEMAS[props.node.kind] ?? []) : []))

function getCfg(key: string): string {
  if (!props.node?.config) return ''
  const v = props.node.config[key]
  return v == null ? '' : String(v)
}

function setCfg(key: string, val: string) {
  if (!props.node) return
  const next = { ...props.node.config }
  if (val === '') delete next[key]
  else next[key] = val
  emit('update', next)
}

function setCfgBatch(patch: Record<string, string>) {
  if (!props.node) return
  const next = { ...props.node.config }
  for (const k in patch) {
    if (patch[k] === '') delete next[k]
    else next[k] = patch[k]
  }
  emit('update', next)
}

// 屏幕拾取 (打开 ScreenPicker 子窗口 → 回填 xRatio/yRatio 或 region)
const { picking, canPickPoint, canPickRect, onPickPoint, onPickRect, onOpenHUD } = useScreenPick({
  node: toRef(props, 'node'),
  applyPoint: (x, y) => setCfgBatch({ xRatio: x.toFixed(4), yRatio: y.toFixed(4) }),
  applyRect: (r) =>
    setCfg('region', `${r[0].toFixed(3)},${r[1].toFixed(3)},${r[2].toFixed(3)},${r[3].toFixed(3)}`),
})

// Parallel / Race 并发分支写同名变量警告
const { concurrencyWarning } = useConcurrencyWarning({
  node: toRef(props, 'node'),
  nodes: toRef(props, 'nodes'),
  edges: toRef(props, 'edges'),
})

// ─── PlayClip section ─────────────────────────────────────────────────────────
const clipsStore = useClipsStore()
onMounted(() => {
  void clipsStore.refresh()
  clipsStore.listen()
})

const selectedClip = computed(() => {
  if (props.node?.kind !== 'PlayClip') return null
  const id = props.node.config?.ClipID
  if (!id) return null
  return clipsStore.clips.find((c) => c.id === id) ?? null
})

// keepRanges 显示形态用 ms 便于人读, 存储形态用 us
const keepRanges = computed<{ fromMs: number; toMs: number }[]>(() => {
  if (props.node?.kind !== 'PlayClip') return []
  const raw = (props.node.config?.keepRanges ?? []) as { fromUs?: number; toUs?: number }[]
  return raw.map((r) => ({
    fromMs: Math.floor((r.fromUs ?? 0) / 1000),
    toMs: Math.floor((r.toUs ?? 0) / 1000),
  }))
})

function currentRanges(): { fromUs: number; toUs: number }[] {
  const raw = (props.node?.config?.keepRanges ?? []) as { fromUs?: number; toUs?: number }[]
  return raw.map((r) => ({ fromUs: r.fromUs ?? 0, toUs: r.toUs ?? 0 }))
}

function addRange() {
  if (!props.node) return
  const next = currentRanges()
  next.push({ fromUs: 0, toUs: 0 })
  emit('update', { ...(props.node.config ?? {}), keepRanges: next })
}

function updateRange(idx: number, field: 'fromMs' | 'toMs', valMs: number) {
  if (!props.node) return
  const next = currentRanges()
  if (idx < 0 || idx >= next.length) return
  if (field === 'fromMs') next[idx].fromUs = Math.max(0, Math.floor(valMs * 1000))
  else next[idx].toUs = Math.max(0, Math.floor(valMs * 1000))
  emit('update', { ...(props.node.config ?? {}), keepRanges: next })
}

function removeRange(idx: number) {
  if (!props.node) return
  const next = currentRanges()
  next.splice(idx, 1)
  emit('update', { ...(props.node.config ?? {}), keepRanges: next })
}

function onTimelineAdd(r: { fromMs: number; toMs: number }) {
  if (!props.node) return
  const cur = currentRanges()
  cur.push({ fromUs: r.fromMs * 1000, toUs: r.toMs * 1000 })
  cur.sort((a, b) => a.fromUs - b.fromUs)
  emit('update', { ...(props.node.config ?? {}), keepRanges: cur })
}

function onTimelineUpdate(idx: number, r: { fromMs: number; toMs: number }) {
  if (!props.node) return
  const cur = currentRanges()
  if (idx < 0 || idx >= cur.length) return
  cur[idx] = { fromUs: r.fromMs * 1000, toUs: r.toMs * 1000 }
  cur.sort((a, b) => a.fromUs - b.fromUs)
  emit('update', { ...(props.node.config ?? {}), keepRanges: cur })
}

function formatDuration(us: number): string {
  const ms = us / 1000
  if (ms < 1000) return ms.toFixed(0) + ' ms'
  return (ms / 1000).toFixed(1) + ' s'
}

function formatDate(iso: string): string {
  try {
    const d = new Date(iso)
    return d.toLocaleDateString() + ' ' + d.toTimeString().slice(0, 5)
  } catch {
    return iso
  }
}

// ─── WindowTarget section (v3 Phase B) ─────────────────────────────────────
// 双向绑定 — config 顶层 PascalCase 字段, 对齐 internal/nodes/system/window_target.go Spec.Inputs.
// 直接 mutate props.node.config 让父图 deep watch 标 dirty (跟 PlayClip keepRanges 一样).
const wtConfig = computed(() => {
  if (props.node?.kind !== 'WindowTarget') return null as any
  if (!props.node.config) (props.node as any).config = {}
  const cfg = props.node.config as any
  if (cfg.Title === undefined) cfg.Title = ''
  if (cfg.Class === undefined) cfg.Class = ''
  if (cfg.ProcessName === undefined) cfg.ProcessName = ''
  if (cfg.TitleMatch === undefined) cfg.TitleMatch = 'exact'
  if (cfg.InputBackend === undefined) cfg.InputBackend = 'postmessage'
  if (cfg.CaptureBackend === undefined) cfg.CaptureBackend = 'auto'
  return cfg
})

const titleMatchOptions = computed(() => [
  { value: 'exact', label: t('node.WindowTarget.inspector.title_match_exact') },
  { value: 'regex', label: t('node.WindowTarget.inspector.title_match_regex') },
])
const inputBackendOptions = computed(() => [
  { value: 'postmessage', label: t('node.WindowTarget.inspector.input_backend_postmessage') },
])
const captureBackendOptions = computed(() => [
  { value: 'auto', label: t('node.WindowTarget.inspector.capture_backend_auto') },
  { value: 'gdi', label: t('node.WindowTarget.inspector.capture_backend_gdi') },
  { value: 'wgc', label: t('node.WindowTarget.inspector.capture_backend_wgc') },
])

const capturing = ref(false)
const captureID = ref('')

// 点按钮: 开 capture session → backend 注册 F9 全局热键; 或 cancel 已开的 session.
// 跟旧同步流程不同 — 这里立刻返回 captureID, 真正捕获在 'windowtarget:captured' event.
async function toggleWindowCapture() {
  if (capturing.value) {
    if (captureID.value) {
      try {
        await backend.tools.cancelWindowTargetCapture(captureID.value)
      } catch {
        // cancel idempotent — 忽略所有错
      }
    }
    capturing.value = false
    captureID.value = ''
    return
  }
  try {
    const id = (await backend.tools.startWindowTargetCapture(0x78)) as string // VK_F9
    captureID.value = id
    capturing.value = true
  } catch (e: any) {
    console.error('startWindowTargetCapture failed', e)
    capturing.value = false
    captureID.value = ''
  }
}

// 监听 backend emit — 收到后填表 + 清 session 状态
let unsubWindowCapture: (() => void) | null = null
onMounted(() => {
  unsubWindowCapture = Events.On('windowtarget:captured', (ev: any) => {
    const raw = ev?.data ?? ev
    const data = Array.isArray(raw) ? raw[0] : raw
    if (!data) return
    capturing.value = false
    captureID.value = ''
    if (data.error) {
      console.warn('windowtarget:captured error', data.error)
      return
    }
    if (wtConfig.value) {
      wtConfig.value.Title = data.title ?? ''
      wtConfig.value.Class = data.class ?? ''
      wtConfig.value.ProcessName = data.processName ?? ''
    }
    // 把捕获时的 resolution 存到 node config — 给 Phase C ROI 节点 metadata 用
    if (props.node && data.clientW && data.clientH) {
      ;(props.node.config as any)._capturedAtResolution = [data.clientW, data.clientH]
    }
  })
})
onUnmounted(() => {
  if (unsubWindowCapture) unsubWindowCapture()
  // 组件 unmount = 用户切别的节点 / 关 inspector — 当前 session 不该悬挂
  // (否则 hotkey 一直占着, F9 触发后 event 没 listener 静默丢失)
  if (captureID.value) {
    backend.tools.cancelWindowTargetCapture(captureID.value).catch(() => {})
  }
})
</script>
