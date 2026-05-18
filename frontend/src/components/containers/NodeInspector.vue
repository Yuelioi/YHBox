<template>
  <div v-if="!node" class="text-sm text-dimmed">未选中节点</div>

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
        title="删除节点"
        @click="$emit('delete')"
      />
    </header>

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
          <div class="font-medium leading-tight">并发分支写入同一变量</div>
          <div class="text-amber-300/80 mt-1 leading-relaxed">{{ concurrencyWarning }}</div>
        </div>
      </div>
    </section>

    <!-- v4 §5.5 Expr 链提示 (Phase C.10) — detect Expr→Expr 单 fan-out, 建议合并 -->
    <section
      v-if="exprChainHint"
      class="mb-5 rounded-md bg-amber-500/10 border border-amber-500/30 px-3 py-2.5"
    >
      <div class="flex items-start gap-2">
        <UIcon name="i-tabler-info-circle" class="size-3.5 text-amber-300 shrink-0 mt-0.5" />
        <div class="text-[12px] text-amber-300">
          <div class="font-medium leading-tight">检测到 Expr 链</div>
          <div class="text-amber-300/80 mt-1 leading-relaxed font-mono text-[11px]">
            value → {{ exprChainHint.targetID }}.{{ exprChainHint.targetPin }}
          </div>
          <div class="text-amber-300/80 mt-1 leading-relaxed">
            建议合并为单一 Expr 节点减少节点数。 Phase D 将提供 "合并到 B" 按钮; 当前请手动合并 (把当前节点的 expr 替换成被引用的 input pin 后嵌入下游 Expr).
          </div>
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
        <span class="text-[11px] text-toned">屏幕拾取</span>
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
          截屏选点
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
          截屏框选 ROI
        </UButton>
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-pointer"
          @click="onOpenHUD"
        >
          鼠标 HUD
        </UButton>
      </div>
      <p class="text-[10px] text-dimmed leading-snug">
        打开独立窗口截当前游戏画面，{{ canPickRect ? '拖矩形' : '点一下' }}后自动回填字段
      </p>
    </section>

    <!-- Subgraph 节点：1:1 模型 — 节点 ↔ 子图 强绑定 + 外部统一编辑 -->
    <section v-if="node.kind === 'Subgraph'" class="space-y-3">
      <div class="rounded-md bg-elevated/30 border border-default/40 p-3 space-y-3">
        <!-- 头部：图标 + 节点数 + 进入按钮 -->
        <div class="flex items-center gap-2">
          <UIcon name="i-tabler-package" class="size-4 text-fuchsia-300" />
          <span class="text-xs text-toned">绑定子图</span>
          <UBadge size="xs" variant="soft" color="neutral" class="ml-auto">
            {{ (boundSubgraph?.graph?.nodes?.length ?? 0) }} 节点
          </UBadge>
        </div>

        <!-- 子图 label 编辑 -->
        <div class="space-y-1">
          <label class="text-[11px] text-toned">标签</label>
          <UInput
            :model-value="boundSubgraph?.label ?? ''"
            size="sm"
            placeholder="子图名称"
            :disabled="!boundSubgraph"
            @update:model-value="(v: string) => onPatchSubgraph({ label: v })"
          />
        </div>

        <!-- 子图描述编辑 -->
        <div class="space-y-1">
          <label class="text-[11px] text-toned">描述</label>
          <UTextarea
            :model-value="(boundSubgraph as any)?.description ?? ''"
            size="sm"
            :rows="2"
            placeholder="可选 · 给自己或队友看的说明"
            :disabled="!boundSubgraph"
            @update:model-value="(v: string) => onPatchSubgraph({ description: v })"
          />
        </div>

        <!-- 子图标签 tags -->
        <div class="space-y-1">
          <label class="text-[11px] text-toned">tags</label>
          <UInputMenu
            :model-value="(boundSubgraph as any)?.tags ?? []"
            multiple
            creatable
            :items="allSubgraphTagsList"
            size="sm"
            placeholder="添加标签..."
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
          进入子图编辑节点内容
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
          {{ publishing ? '发布中...' : '发布此子图到库' }}
        </UButton>
        <p class="text-[10px] text-dimmed leading-snug">
          子图元信息（标签/描述/tags）可在此处编辑，无需进入子图。<br />
          删除此节点会同时删除对应子图（如无其他引用）。
        </p>
      </div>
    </section>

    <!-- MouseCalibration：v2 spec §1.6 防误用形态 -->
    <section v-else-if="node.kind === 'MouseCalibration'" class="space-y-3">
      <div
        v-if="isCalibrationForeign"
        class="rounded-md bg-amber-500/10 border border-amber-500/30 px-3 py-2.5 text-[12px] text-amber-300"
      >
        <UIcon name="i-tabler-alert-triangle" class="size-3.5 inline mr-1 align-middle" />
        这个容器似乎来自别的机器（节点 {{ node.config?.counts360 }} / 全局 {{ globalCounts360 }}）<br />
        请用本机重新校准；或点击下方按钮一键覆盖
        <div class="mt-2 flex gap-1.5 flex-wrap">
          <UButton
            size="xs"
            color="warning"
            variant="solid"
            icon="i-tabler-refresh"
            @click="$emit('update', { ...node.config, counts360: globalCounts360 })"
          >用本机值（{{ globalCounts360 }}）覆盖此节点</UButton>
          <UButton
            size="xs"
            variant="ghost"
            color="warning"
            icon="i-tabler-bolt"
            @click="onSyncAllFromForeign"
          >⚡ 同步所有容器</UButton>
        </div>
      </div>

      <div class="rounded-md bg-elevated/30 border border-default/40 p-3 space-y-2">
        <div class="flex items-baseline gap-2">
          <span class="text-xs text-toned">本机 360° HID counts</span>
        </div>
        <div class="flex items-center gap-2">
          <span
            class="text-2xl font-mono tabular-nums"
            :class="(node.config?.counts360 ?? 0) > 0 ? 'text-emerald-300' : 'text-rose-300'"
          >{{ node.config?.counts360 ?? 0 }}</span>
          <span class="text-[11px] text-dimmed">{{ (node.config?.counts360 ?? 0) > 0 ? '✅ 已校准' : '❌ 未校准' }}</span>
        </div>
        <p class="text-[11px] text-dimmed leading-relaxed">
          转 360° 你的鼠标硬件累积上报多少 |dx|；跟硬件 DPI、OS 灵敏度、游戏内灵敏度都有关。<br />
          <span class="text-rose-300/80">⚠ 这个值必须是你本机+游戏实测的，不是从别人容器导入的值。</span>
        </p>
        <UButton
          size="sm"
          color="primary"
          variant="solid"
          icon="i-tabler-target"
          block
          @click="onOpenCalibrator"
        >▶ 开始校准</UButton>

        <UCollapsible class="mt-3">
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-chevron-right"
            class="w-full justify-start"
          >高级（手动输入）</UButton>

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
          {{ capturing ? '等待 F9 按键 (再点取消)' : '捕获目标窗口 (按 F9)' }}
        </UButton>
        <p class="text-xs text-dimmed mt-1">
          点开后切到游戏窗口, 按 F9 即可捕获 title/class/processName.
          若 F9 被游戏反作弊吞掉, 联系开发者换其他键 (当前 1.0 写死 F9).
        </p>
      </div>

      <!-- match section -->
      <div class="border border-default rounded-lg p-3 space-y-2">
        <h4 class="text-sm font-semibold">窗口匹配 (match)</h4>
        <UFormField label="标题 (title)">
          <UInput v-model="wtMatch.title" placeholder="异环" />
        </UFormField>
        <UFormField label="类名 (class)">
          <UInput v-model="wtMatch.class" placeholder="UnrealWindow" />
        </UFormField>
        <UFormField label="进程名 (processName)">
          <UInput v-model="wtMatch.processName" placeholder="game.exe" />
        </UFormField>
        <UFormField label="title 匹配方式">
          <USelect
            v-model="wtMatch.titleMatch"
            class="w-full"
            :items="[
              { value: 'exact', label: '精确匹配 (区分大小写)' },
              { value: 'regex', label: '正则 RE2 (partial match)' },
            ]"
          />
        </UFormField>
      </div>

      <!-- runtime section -->
      <div class="border border-default rounded-lg p-3 space-y-2">
        <h4 class="text-sm font-semibold">运行后端 (runtime)</h4>
        <UFormField label="输入后端 (inputBackend)">
          <USelect
            v-model="wtRuntime.inputBackend"
            class="w-full"
            :items="[{ value: 'postmessage', label: 'PostMessage (后台输入, 1.0 默认)' }]"
          />
        </UFormField>
        <UFormField label="截图后端 (captureBackend)">
          <USelect
            v-model="wtRuntime.captureBackend"
            class="w-full"
            :items="[
              { value: 'auto', label: 'auto (按 OS 选, Win10+ 用 WGC)' },
              { value: 'gdi', label: 'GDI (所有 Windows)' },
              { value: 'wgc', label: 'WGC (要 Win10 1903+)' },
            ]"
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
        clip {{ node.config?.clipID || '(未设)' }} 不在 clips 库. 重新录制覆盖.
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
        >重新录制 (精准)</UButton>
        <UButton
          size="xs"
          color="neutral"
          variant="soft"
          icon="i-tabler-zap"
          class="flex-1"
          @click="$emit('request-record', { mode: 'simple', replaceNodeID: node.id })"
        >重录 (简易)</UButton>
      </div>
      <p class="text-[10px] text-dimmed leading-snug -mt-1">
        一个 PlayClip 节点绑死一个 clip — 想换内容请重新录制覆盖, 不要切换 clip 引用 (避免删 clip 后这里指向不存在的 ID).
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
          <span class="text-[11px] text-toned">裁剪段 (keepRanges)</span>
          <UButton size="xs" variant="ghost" icon="i-tabler-plus" @click="addRange">添加</UButton>
        </div>
        <p class="text-[10px] text-dimmed mb-2 leading-snug">
          不指定 = 整段播放. 加多段后只播这些段, 跨段的停顿会自动压缩.
        </p>
        <div v-if="keepRanges.length === 0" class="text-[10px] text-dimmed italic">
          无, 整段播放
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

    <!-- v4 §7.1: data-in pin literal editors (no incoming edge → set inline value) -->
    <section v-if="dataInLiterals.length > 0" class="mb-5">
      <h4 class="text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed mb-3">
        数据输入 (literal)
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
      <h4 class="text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed mb-3">配置</h4>
      <div class="space-y-4">
        <div v-for="field in fields" :key="field.key" class="space-y-1.5">
          <label class="block text-xs text-toned">{{ field.label }}</label>
          <ExpressionInput
            v-if="field.type === 'expr'"
            :model-value="getCfg(field.key)"
            :placeholder="field.placeholder"
            :expected-type="field.exprType"
            :var-names="varNames"
            @update:model-value="setCfg(field.key, $event)"
          />
          <USelect
            v-else-if="field.type === 'select'"
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
            placeholder="选择变量"
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
          <p v-if="field.hint" class="text-[11px] text-dimmed leading-snug">{{ field.hint }}</p>
        </div>
      </div>
    </section>

    <p v-else class="text-[12px] text-dimmed">此节点无可配置项。</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, toRef } from 'vue'
import { Events } from '@wailsio/runtime'
import type { GraphNode } from '@/lib/backend'
import { backend } from '@/lib/backend'
import ExpressionInput from '@/components/expressions/ExpressionInput.vue'
import SwitchInspector from './inspector/SwitchInspector.vue'
import TemplatePicker from './TemplatePicker.vue'
import KeyCapture from './KeyCapture.vue'
import ClipTimeline from './ClipTimeline.vue'
import { KIND_LABEL_ZH, KIND_DESCRIPTION, KIND_VISUAL, PIN_SPECS } from './pinSpec'
import PinLiteral from './inline/PinLiteral.vue'
import { NODE_FIELD_SCHEMAS, type Field } from './nodeFieldSchemas'
import { useSettingsStore } from '@/stores/settings'
import { useContainerEditorStore } from '@/stores/containerEditor'
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
  delete: []
  'request-record': [opts: { mode: 'precise' | 'simple'; replaceNodeID: string }]
}>()

const settingsStore = useSettingsStore()
const globalCounts360 = computed(() => settingsStore.data?.ui?.mouseCounts360 ?? 0)

// v4 §7.1 inline pin literal (Inspector edition — in-canvas UE-style render deferred).
// For each data-in pin in PIN_SPECS[kind].dataIn that lacks an incoming data edge,
// expose a literal editor bound to config.literal[pinName].
interface LiteralEntry { name: string; type: string }
const dataInLiterals = computed<LiteralEntry[]>(() => {
  if (!props.node) return []
  const spec = PIN_SPECS[props.node.kind]
  if (!spec) return []
  const incomingPins = new Set<string>()
  for (const e of props.edges ?? []) {
    if ((e as any).kind !== 'data') continue
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

// v4 §5.5 第二层: Expr 链检测 — 如果当前 Expr 节点的 value out 唯一连到另一 Expr 的 input,
// Inspector 显示提示建议合并 (Phase D 加 fusion 按钮; 当前只提示).
interface ChainHint { targetID: string; targetPin: string }
const exprChainHint = computed<ChainHint | null>(() => {
  if (!props.node || props.node.kind !== 'Expr') return null
  if (!props.nodes || !props.edges) return null
  const myID = props.node.id
  const outgoing = (props.edges ?? []).filter(
    (e: any) => {
      const [src, srcPin] = (e.from ?? '').split('.')
      return src === myID && srcPin === 'value' && e.kind === 'data'
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
    title: '同步到所有容器？',
    description: `把本机 counts360 = ${cur} 同步到所有本地容器（不只是这个节点）`,
    color: 'primary',
    confirmText: '同步',
  })
  if (yes !== true) return
  const r = (await backend.containers.syncLocalMouseCalibration(cur)) as any
  toastForSync.add({ title: `已同步 ${r?.updated?.length ?? 0} 个容器`, color: 'success' })
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
  const sgID = props.node?.config?.subgraphId
  if (!sgID) return null
  return editorStore.subgraphsForCurrentContainer.find((s) => s.id === sgID) ?? null
})

function onEnterSubgraph() {
  const sgID = props.node?.config?.subgraphId
  if (!sgID) return
  editorStore.pushPath(String(sgID))
}

// 发布当前绑定子图到库 (容器→库, 反向 copy-on-use)
const publishing = ref(false)
async function onPublishToLibrary() {
  const sgID = props.node?.config?.subgraphId
  const cid = editorStore.activeContainerID
  if (!sgID || !cid || !boundSubgraph.value) return
  const yes = await confirmDialog({
    title: '发布子图到库？',
    description: `将「${boundSubgraph.value.label || sgID}」深拷贝一份到全局库；容器里原子图不变。后续可在 库 入口 拖入其他容器。`,
    color: 'primary',
    confirmText: '发布',
  })
  if (yes !== true) return
  publishing.value = true
  try {
    const r = (await backend.library.publishFromContainer(cid, String(sgID))) as any
    toastForSync.add({
      title: '已发布到库',
      description: `新库 ID: ${r?.id ?? '?'}`,
      color: 'success',
      icon: 'i-tabler-cloud-upload',
    })
  } catch (e) {
    toastForSync.add({ title: '发布失败', description: String(e), color: 'error' })
  } finally {
    publishing.value = false
  }
}

// 所有子图聚合 tags（给 UInputMenu autocomplete）
const allSubgraphTagsList = computed(() => {
  const set = new Set<string>()
  for (const sg of editorStore.subgraphsForCurrentContainer ?? []) {
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

const label = computed(() =>
  props.node ? (KIND_LABEL_ZH[props.node.kind] ?? props.node.kind) : '',
)
const description = computed(() => (props.node ? (KIND_DESCRIPTION[props.node.kind] ?? '') : ''))
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
  const id = props.node.config?.clipID
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
// 双向绑定 — config 嵌套 {match, runtime}. 直接 mutate props.node.config 让
// 父图 deep watch 标 dirty (跟 PlayClip keepRanges 一样的写法).
const wtMatch = computed(() => {
  if (props.node?.kind !== 'WindowTarget') return null
  if (!props.node.config) (props.node as any).config = {}
  if (!(props.node.config as any).match) {
    ;(props.node.config as any).match = {
      title: '',
      class: '',
      processName: '',
      titleMatch: 'exact',
    }
  }
  return (props.node.config as any).match
})

const wtRuntime = computed(() => {
  if (props.node?.kind !== 'WindowTarget') return null
  if (!props.node.config) (props.node as any).config = {}
  if (!(props.node.config as any).runtime) {
    ;(props.node.config as any).runtime = {
      inputBackend: 'postmessage',
      captureBackend: 'auto',
    }
  }
  return (props.node.config as any).runtime
})

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
    if (wtMatch.value) {
      wtMatch.value.title = data.title ?? ''
      wtMatch.value.class = data.class ?? ''
      wtMatch.value.processName = data.processName ?? ''
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
