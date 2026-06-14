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
      <div class="flex items-center gap-0.5 shrink-0">
        <!-- 「?」节点说明 — 概述 + 示例收进单击弹层 (仅当该 kind 有 description / example) -->
        <UPopover v-if="description || example">
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-help-circle"
            :title="t('inspector.help_tooltip')"
          />
          <template #content>
            <div class="p-3 max-w-xs space-y-3">
              <p v-if="description" class="text-[12px] text-toned leading-relaxed">{{ description }}</p>
              <div v-if="example" class="rounded-md bg-elevated/40 border border-default/40 px-2.5 py-2">
                <div class="flex items-center gap-1.5 mb-1">
                  <UIcon name="i-tabler-bulb" class="size-3.5 text-amber-400 shrink-0" />
                  <span class="text-[11px] font-medium text-toned">{{ t('inspector.example_title') }}</span>
                </div>
                <p class="text-[12px] text-toned leading-relaxed whitespace-pre-line">{{ example }}</p>
              </div>
            </div>
          </template>
        </UPopover>
        <!-- 复制菜单: 节点 ID / JSON / 脚本调用信息。下拉项点完即关、无处内联 →
             走短 toast (ui.md「反馈方式」决策树第 2 条)。 -->
        <UDropdownMenu :items="copyMenuItems">
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-copy"
            :title="t('inspector.copy_menu_tooltip')"
          />
        </UDropdownMenu>
        <UButton
          size="xs"
          variant="ghost"
          color="error"
          icon="i-tabler-trash"
          :title="t('inspector.delete_node_tooltip')"
          @click="$emit('delete')"
        />
      </div>
    </header>

    <SectionHeader :title="t('editor.inspector.group_basics')" icon="i-tabler-adjustments" class="-mx-4 mt-2 mb-4" />

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

    <!-- 打印日志 (LogEnabled) — 勾选后该节点执行时吐通用 dump 日志 -->
    <section class="mb-4">
      <UFormField :label="t('inspector.log_enabled_label')" :hint="t('inspector.log_enabled_hint')">
        <USwitch
          :model-value="node.logEnabled ?? false"
          size="sm"
          @update:model-value="(v: boolean) => $emit('log-enabled-update', v)"
        />
      </UFormField>
    </section>

    <SectionHeader :title="t('editor.inspector.group_inputs')" icon="i-tabler-login-2" class="-mx-4 mt-5 mb-4" />

    <!-- 并发警告 -->
    <section
      v-if="concurrencyWarning"
      class="mb-5 rounded-md bg-warning/10 border border-warning/30 px-3 py-2.5"
    >
      <div class="flex items-start gap-2">
        <UIcon name="i-tabler-alert-triangle" class="size-3.5 text-warning shrink-0 mt-0.5" />
        <div class="text-[12px] text-warning">
          <div class="font-medium leading-tight">{{ t('inspector.concurrency_warn_title') }}</div>
          <div class="text-warning/80 mt-1 leading-relaxed">{{ concurrencyWarning }}</div>
        </div>
      </div>
    </section>

    <!-- Expr 链提示 + 一键合并按钮 -->
    <section
      v-if="exprChainHint"
      class="mb-5 rounded-md bg-warning/10 border border-warning/30 px-3 py-2.5"
    >
      <div class="flex items-start gap-2">
        <UIcon name="i-tabler-info-circle" class="size-3.5 text-warning shrink-0 mt-0.5" />
        <div class="text-[12px] text-warning flex-1">
          <div class="font-medium leading-tight">{{ t('inspector.expr_chain_title') }}</div>
          <div class="text-warning/80 mt-1 leading-relaxed font-mono text-[11px]">
            value → {{ exprChainHint.targetID }}.{{ exprChainHint.targetPin }}
          </div>
          <div class="text-warning/80 mt-1 mb-2 leading-relaxed">
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
            :create-item="'always'"
            :items="allSubgraphTagsList"
            size="sm"
            :placeholder="t('node.Subgraph.inspector.subgraph_tags_placeholder')"
            :disabled="!boundSubgraph"
            @update:model-value="(v: string[]) => onPatchSubgraph({ tags: v })"
            @create="(v: string) => onPatchSubgraph({ tags: [...((boundSubgraph as any)?.tags ?? []), v] })"
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
        <!-- 「发布到库」已删 — 子图全局化后天生在池里, 子图库直接可见。 -->
        <p class="text-[10px] text-dimmed leading-snug">
          {{ t('node.Subgraph.inspector.footer_meta_hint') }}<br />
          {{ t('node.Subgraph.inspector.footer_delete_hint') }}
        </p>
      </div>
    </section>

    <!-- MouseCalibration 节点 — 多 profile 后不再跟"全局单值"比对报 FOREIGN (各游戏 counts 本就不同)。 -->
    <section v-else-if="node.kind === 'MouseCalibration'" class="space-y-3">
      <div class="rounded-md bg-elevated/30 border border-default/40 p-3 space-y-2">
        <div class="flex items-baseline gap-2">
          <span class="text-xs text-toned">{{ t('node.MouseCalibration.inspector.counts_label') }}</span>
        </div>
        <div class="flex items-center gap-2">
          <span
            class="text-2xl font-mono tabular-nums"
            :class="mcCounts > 0 ? 'text-success' : 'text-error'"
          >{{ mcCounts }}</span>
          <span class="text-[11px] text-dimmed">{{ mcCounts > 0 ? t('node.MouseCalibration.inspector.calibrated') : t('node.MouseCalibration.inspector.not_calibrated') }}</span>
        </div>
        <p class="text-[11px] text-dimmed leading-relaxed">
          {{ t('node.MouseCalibration.inspector.counts_hint') }}<br />
          <span class="text-error/80">{{ t('node.MouseCalibration.inspector.counts_warn') }}</span>
        </p>
        <UButton
          size="sm"
          color="primary"
          variant="solid"
          icon="i-tabler-target"
          block
          @click="onOpenCalibrator"
        >{{ t('node.MouseCalibration.inspector.start_calibrate') }}</UButton>

        <!-- 未校准 + 设置里有档 → 「从设置加载」: 单档直接填, 多档下拉选 -->
        <template v-if="mcCounts === 0 && mouseProfiles.length > 0">
          <UButton
            v-if="mouseProfiles.length === 1"
            size="sm"
            variant="soft"
            color="primary"
            icon="i-tabler-download"
            block
            @click="loadProfileIntoNode(mouseProfiles[0].label)"
          >{{ t('node.MouseCalibration.inspector.load_from_settings_one', { label: mouseProfiles[0].label || '?', n: mouseProfiles[0].counts360 }) }}</UButton>
          <USelect
            v-else
            :items="profileSelectItems"
            :placeholder="t('node.MouseCalibration.inspector.load_from_settings_pick')"
            icon="i-tabler-download"
            size="sm"
            class="w-full"
            @update:model-value="(v: string) => loadProfileIntoNode(v)"
          />
        </template>

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
              :model-value="mcCounts"
              size="sm"
              class="w-full mt-2"
              @update:model-value="(v: number) => setMcCounts(v)"
            />
          </template>
        </UCollapsible>
      </div>
    </section>

    <!-- WindowTarget: 运行时切换目标游戏窗口 -->
    <section v-else-if="node.kind === 'WindowTarget'" class="mb-5 space-y-4">
      <!-- 子图内提示 -->
      <p v-if="editorStore.editorPath.length > 0" class="text-xs text-warning">
        {{ t('node.WindowTarget.subgraph_hint') }}
      </p>

      <!-- 捕获按钮 (F9 全局热键流程) -->
      <div>
        <UButton
          :icon="capturing ? 'i-tabler-loader-2' : 'i-tabler-target'"
          :loading="capturing"
          size="sm"
          block
          @click="toggleWindowCapture"
        >
          {{ capturing ? t('node.WindowTarget.inspector.capture_waiting', { hk: captureHk }) : t('node.WindowTarget.inspector.capture_start', { hk: captureHk }) }}
        </UButton>
        <p class="text-xs text-dimmed mt-1">
          {{ t('node.WindowTarget.inspector.capture_hint_a', { hk: captureHk }) }}
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
      <div v-else class="rounded-md bg-warning/10 border border-warning/30 px-3 py-2 text-[11px] text-warning">
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

    <!-- 动态输入声明 (spec.dynamicInputs: Expr/Script) — 编辑 config.Inputs[],
         画布 data-in 引脚 + 下方 literal 区 + code 补全据此联动。独立 v-if (不进上面
         bespoke v-else-if 链): 这些节点还要渲染通用 literal 区。 -->
    <section v-if="specHasDynamicInputs" class="mb-5">
      <h4 class="text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed mb-3">
        {{ t('inspector.dyn_inputs_title') }}
      </h4>
      <DynamicInputsEditor :node="node" @update="emit('update', $event)" />
    </section>

    <!-- 数据输入 — 每个未连线 data-in pin 一个 widget-aware 编辑器, 写回 config.literal[pin]。
         连线的 pin 不显 (值走 data 边)。有专属 section 的 kind (BESPOKE_EDITOR_KINDS) 这里返空。 -->
    <section v-if="normalLiterals.length > 0" class="mb-5">
      <h4 class="text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed mb-3">
        {{ t('inspector.literal_section') }}
      </h4>
      <div class="space-y-4">
        <div v-for="lit in normalLiterals" :key="lit.name" class="space-y-1.5">
          <label class="block text-xs text-toned">
            {{ fieldFor(lit.name) ? t(fieldFor(lit.name)!.label) : lit.name }}
            <span class="text-[10px] text-dimmed font-mono ml-1">({{ lit.type }})</span>
          </label>
          <VarNameInput
            v-if="fieldFor(lit.name)?.semantic === 'varname'"
            :model-value="String(getLiteral(lit.name) ?? '')"
            :declared-vars="declaredVars ?? []"
            :scope="nodeScope"
            @update:model-value="(v: string) => setLiteral(lit.name, v)"
            @declare-var="(a) => emit('declare-var', a)"
          />
          <StructuredInput
            v-else-if="fieldFor(lit.name)?.schema"
            :schema="fieldFor(lit.name)!.schema!"
            :model-value="getLiteral(lit.name)"
            :field-path="lit.name"
            :kind="node!.kind"
            @update:model-value="(v: any) => setLiteral(lit.name, v)"
            @pick-color="onColorPick"
          />
          <!-- 模板字段 (WaitTemplate/ClickTemplate/CheckTemplate) → 多选缩略图拾取器 + 现截一张 -->
          <TemplatePickerField
            v-else-if="fieldFor(lit.name)?.widgetKind === 'template-picker'"
            :model-value="asTemplateList(getLiteral(lit.name))"
            :pin="lit.name"
            @update:model-value="(v: string[]) => setLiteral(lit.name, v)"
          />
          <PinInput
            v-else
            :type="(lit.type as any)"
            :widget-kind="fieldFor(lit.name)?.widgetKind"
            :options="fieldFor(lit.name)?.options"
            :placeholder="fieldFor(lit.name)?.placeholder"
            :min="fieldFor(lit.name)?.min"
            :max="fieldFor(lit.name)?.max"
            :step="fieldFor(lit.name)?.step"
            :input-names="dynamicInputNames"
            :declared-vars="declaredVars"
            :model-value="getLiteral(lit.name)"
            @update:model-value="(v: any) => setLiteral(lit.name, v)"
            @declare-var="(a) => emit('declare-var', a)"
          />
          <p
            v-if="fieldFor(lit.name)?.hint && te(fieldFor(lit.name)!.hint!)"
            class="text-[11px] text-dimmed leading-snug"
          >{{ t(fieldFor(lit.name)!.hint!) }}</p>
        </div>
      </div>
    </section>

    <!-- 输出捕获 — semantic==='capture' 的输入聚成默认折叠组, 折叠头带「总数 / 已填」徽章。
         填变量名 → 节点运行时把对应输出值写进该变量。 -->
    <section v-if="captureLiterals.length > 0" class="mb-5">
      <button
        type="button"
        class="w-full flex items-center gap-2 mb-3 group"
        @click="captureOpen = !captureOpen"
      >
        <UIcon
          :name="captureOpen ? 'i-tabler-chevron-down' : 'i-tabler-chevron-right'"
          class="size-3.5 text-dimmed shrink-0"
        />
        <h4 class="text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed">
          {{ t('inspector.capture_section') }}
        </h4>
        <span
          class="text-[10px] font-mono px-1.5 py-0.5 rounded"
          :class="captureFilledCount > 0 ? 'bg-primary/15 text-primary' : 'bg-elevated text-dimmed'"
        >{{ captureFilledCount }}/{{ captureLiterals.length }}</span>
      </button>
      <div v-if="captureOpen" class="space-y-4">
        <div v-for="lit in captureLiterals" :key="lit.name" class="space-y-1.5">
          <label class="block text-xs text-toned">
            {{ fieldFor(lit.name) ? t(fieldFor(lit.name)!.label) : lit.name }}
            <span class="text-[10px] text-dimmed font-mono ml-1">({{ fieldFor(lit.name)?.captureType ?? lit.type }})</span>
          </label>
          <VarNameInput
            :model-value="String(getLiteral(lit.name) ?? '')"
            :declared-vars="declaredVars ?? []"
            :capture-type="fieldFor(lit.name)?.captureType"
            scope="auto"
            @update:model-value="(v: string) => setLiteral(lit.name, v)"
            @declare-var="(a) => emit('declare-var', a)"
          />
          <p
            v-if="fieldFor(lit.name)?.hint && te(fieldFor(lit.name)!.hint!)"
            class="text-[11px] text-dimmed leading-snug"
          >{{ t(fieldFor(lit.name)!.hint!) }}</p>
        </div>
      </div>
    </section>

    <p
      v-else-if="normalLiterals.length === 0 && !hasBespokeSection && !specHasDynamicInputs"
      class="text-[12px] text-dimmed"
    >{{ t('inspector.no_config') }}</p>

    <!-- 输出 — 出口 pin 速览 (只读): exec 出口 + data 出口。无出口 → 占位。 -->
    <SectionHeader :title="t('editor.inspector.group_outputs')" icon="i-tabler-logout-2" class="-mx-4 mt-5 mb-3" />
    <div v-if="outPins.exec.length || outPins.data.length" class="space-y-1.5">
      <div
        v-for="pn in outPins.exec"
        :key="'x-' + pn"
        class="flex items-center gap-2 text-[11px]"
      >
        <UIcon name="i-tabler-arrow-right" class="size-3.5 text-dimmed shrink-0" />
        <span class="text-toned">{{ pn }}</span>
        <span class="ml-auto text-[10px] text-dimmed font-mono">exec</span>
      </div>
      <div
        v-for="dp in outPins.data"
        :key="'d-' + dp.name"
        class="flex items-center gap-2 text-[11px]"
      >
        <UIcon name="i-tabler-variable" class="size-3.5 text-dimmed shrink-0" />
        <span class="text-toned">{{ dp.name }}</span>
        <span v-if="dp.type" class="ml-auto text-[10px] text-dimmed font-mono">{{ dp.type }}</span>
      </div>
    </div>
    <p v-else class="text-[11px] text-dimmed">{{ t('editor.inspector.outputs_none') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, toRef } from 'vue'
import { Events } from '@wailsio/runtime'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import type { GraphNode } from '@/lib/backend'
import { backend } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import SwitchInspector from './inspector/SwitchInspector.vue'
import DynamicInputsEditor from './inspector/DynamicInputsEditor.vue'
import { getSpec } from './nodeRegistry/registry'
import ClipTimeline from './ClipTimeline.vue'
import TemplatePickerField from './TemplatePickerField.vue'
import SectionHeader from '@/components/common/SectionHeader.vue'
import { useI18n } from 'vue-i18n'
import { KIND_LABEL_ZH, KIND_DESCRIPTION, KIND_EXAMPLE, KIND_VISUAL, PIN_SPECS, edgeKind, pinsFor } from './pinSpec'

const { t, te } = useI18n()

import PinInput from './inline/PinInput.vue'
import StructuredInput from './inline/StructuredInput.vue'
import VarNameInput from './inline/VarNameInput.vue'
import { unconnectedDataInPins } from '@/composables/containerEditor/pinLiterals'
import { type VarType } from '@/lib/variableRef'
import { NODE_FIELD_SCHEMAS, type Field } from './nodeFieldSchemas'
import { useSettingsStore } from '@/stores/settings'
import { useHotkeysStore } from '@/stores/hotkeys'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { useEditorBusStore } from '@/stores/editorBus'
import { useClipsStore } from '@/stores/clips'
import { useToast } from '@nuxt/ui/composables'
import { useConfirm } from '@/composables/useConfirm'
import { useScreenPick } from '@/composables/containerEditor/useScreenPick'
import { fillColorLiteral } from '@/composables/containerEditor/colorRange'
import { useConcurrencyWarning } from '@/composables/containerEditor/useConcurrencyWarning'

const props = defineProps<{
  node: GraphNode | null
  declaredVars?: { name: string; type: VarType }[]
  nodes?: GraphNode[]
  edges?: { from: string; to: string }[]
}>()
const emit = defineEmits<{
  update: [config: Record<string, any>]
  'label-update': [v: string]
  'log-enabled-update': [v: boolean]
  delete: []
  'request-record': [opts: { mode: 'precise' | 'simple'; replaceNodeID: string }]
  'declare-var': [args: { name: string; type: VarType; default: unknown }]
}>()

const settingsStore = useSettingsStore()
// 节点未校准时「从设置加载」用: 全部校准档. 多档 → 下拉选, 单档 → 直接填.
const mouseProfiles = computed(() => settingsStore.mouseProfiles)
function loadProfileIntoNode(label: string) {
  const p = mouseProfiles.value.find((x) => x.label === label)
  if (p) setMcCounts(p.counts360)
}
const profileSelectItems = computed(() =>
  mouseProfiles.value.map((p) => ({ label: `${p.label || '?'} · ${p.counts360}`, value: p.label })),
)

// 模板字段值容错: undefined / 单 string (迁移前残留) / string[] → string[]。
function asTemplateList(v: any): string[] {
  if (Array.isArray(v)) return v.filter((x): x is string => typeof x === 'string')
  if (typeof v === 'string' && v) return [v]
  return []
}

// Inline pin literal — Inspector 版.
// 对每个 PIN_SPECS[kind].dataIn 里没连入边的 pin, 暴露一个绑 config.literal[pinName] 的编辑器.
// 用全部返回值 (含 point) — Inspector 是宽面板, point 也在此编辑。判定逻辑跟画布共用纯函数。
const dataInLiterals = computed(() => {
  if (!props.node) return []
  const dataIn = PIN_SPECS[props.node.kind]?.dataIn ?? {}
  return unconnectedDataInPins(
    props.node.kind,
    dataIn,
    props.node.config,
    props.edges ?? [],
    props.node.id,
  )
})

// spec.dynamicInputs 标志 (Expr/Script) — 驱动「输入口」声明编辑 section。
const specHasDynamicInputs = computed(
  () => !!props.node && !!getSpec(props.node.kind)?.dynamicInputs,
)

// 动态输入名 (config.Inputs[] 声明, 镜像后端 ParseDynamicInputDecls) — 喂 code widget 补全。
const dynamicInputNames = computed<string[]>(() => {
  const raw = props.node?.config?.Inputs
  if (!Array.isArray(raw)) return []
  return raw
    .map((d: any) => String(d?.Name ?? ''))
    .filter((n) => n !== '')
})

// semantic==='capture' 的输入聚成「输出捕获」折叠组, 其余正常平铺。
function isCaptureLit(name: string): boolean {
  return fieldFor(name)?.semantic === 'capture'
}
const normalLiterals = computed(() => dataInLiterals.value.filter((l) => !isCaptureLit(l.name)))
const captureLiterals = computed(() => dataInLiterals.value.filter((l) => isCaptureLit(l.name)))
// 已填的捕获项数 (literal 非空字符串) — 折叠头徽章用。
const captureFilledCount = computed(
  () => captureLiterals.value.filter((l) => String(getLiteral(l.name) ?? '').trim() !== '').length,
)
// 折叠状态: 默认折叠。
const captureOpen = ref(false)

// 节点 scope — 传给 VarNameInput，影响补全行为。
// Scope pin 字面量 = config.literal.Scope (跟后端 + 真实存盘 shape 对齐)。
const nodeScope = computed(
  () =>
    ((props.node?.config?.literal as Record<string, unknown> | undefined)?.Scope as
      | 'auto'
      | 'global'
      | 'local'
      | undefined) ?? 'auto',
)

// 读 pin 字面量: config.literal[pin] 优先, 顶层 config[pin] fallback —
// 镜像后端 PinValue / newInputs 优先级。让尚未跑迁移脚本的旧数据 (值在顶层 config) 也能正确显示。
function getLiteral(pin: string): any {
  const lit = props.node?.config?.literal as Record<string, unknown> | undefined
  if (lit && pin in lit) return lit[pin]
  return props.node?.config?.[pin]
}
// 写回唯一走 config.literal[pin] (input-editing-unification guardrail: 不写顶层同名 key)。
function setLiteral(pin: string, v: any) {
  if (!props.node) return
  const cfg = { ...props.node.config }
  cfg.literal = { ...cfg.literal, [pin]: v }
  emit('update', cfg)
}
// 批量写多个 config.literal pin (一次 emit) — 屏幕拾取 point 同时写 XRatio+YRatio 用。
function setLiteralBatch(patch: Record<string, any>) {
  if (!props.node) return
  const cfg = { ...props.node.config }
  cfg.literal = { ...cfg.literal, ...patch }
  emit('update', cfg)
}
// 查 pin 对应的 widget 元数据 (类型/选项), 喂给 PinInput 渲染正确控件。
// 动态 input (Expr config.Inputs[]) 在 fields 里查不到 → undefined, PinInput 走 pinType fallback。
function fieldFor(pin: string): Field | undefined {
  return fields.value.find((f) => f.key === pin)
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
      // edge kind 由 (srcNode.kind, srcPin) 推导 (无 edge.kind 字段).
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
// MouseCalibration Counts360 读写 — 正源 config.literal.Counts360。
// 读 fallback: literal.Counts360 → 顶层 Counts360 → 顶层 counts360 (旧小写遗留, 迁移脚本会清)。
const mcCounts = computed<number>(() => {
  const cfg = props.node?.config as any
  if (!cfg) return 0
  const raw = cfg.literal?.Counts360 ?? cfg.Counts360 ?? cfg.counts360 ?? 0
  return Number(raw) || 0
})
function setMcCounts(v: number) {
  setLiteral('Counts360', v)
}

async function onOpenCalibrator() {
  if (!props.node) return
  // 开独立置顶校准 HUD 窗 (用户自己切到目标游戏按 F8), 等结果写进本节点 config.literal.Counts360。
  const id = 'calib-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now()
  const ok = await backend.tools.openCalibratorHUD(id)
  if (!ok) return // 开窗失败 (invoke 已 toast)
  const r = await awaitWailsEvent<{ id: string; counts?: number; cancelled?: boolean }>(
    'calibration:result',
    (p) => p?.id === id,
  )
  if (!r.cancelled && typeof r.counts === 'number' && r.counts > 0) setMcCounts(r.counts)
}

const toastForSync = useToast()
const { confirm: confirmDialog } = useConfirm()

// 复制节点 ID / 完整 JSON / 脚本调用信息 到剪贴板。
// 留住下拉 (e.preventDefault) → 被点项原地闪「已复制 ✓」~1500ms, 成功不弹 toast, 仅错误弹
// (ui.md「反馈方式」: 能原地就原地)。
const copiedItem = ref<'id' | 'json' | 'script' | null>(null)
let copiedTimer = 0
async function copyAndFlash(text: string, which: 'id' | 'json' | 'script', e?: Event) {
  e?.preventDefault() // 留住菜单, 好原地显「已复制」
  try {
    await navigator.clipboard.writeText(text)
    copiedItem.value = which
    window.clearTimeout(copiedTimer)
    copiedTimer = window.setTimeout(() => {
      copiedItem.value = null
    }, 1500)
  } catch (err: any) {
    toastForSync.add({ title: t('toast.copy_failed'), description: errorMessage(err), color: 'error' })
  }
}

// 节点脚本调用形态, 带当前已填实参: Kind({ Pin: value, ... })。
// object key = canonical PascalCase pin 名 (脚本调节点的 key 必须逐字匹配后端 Spec pin 名);
// 只列已设 literal 的 data-in pin (连线取值的 pin 无 literal, 跳过)。
function buildScriptCall(): string {
  if (!props.node) return ''
  const kind = props.node.kind
  const dataIn = PIN_SPECS[kind]?.dataIn ?? {}
  const parts: string[] = []
  for (const pin of Object.keys(dataIn)) {
    const v = getLiteral(pin)
    if (v === undefined || v === null || v === '') continue
    parts.push(`${pin}: ${JSON.stringify(v)}`)
  }
  return parts.length ? `${kind}({ ${parts.join(', ')} })` : `${kind}()`
}

// 被点项 label/icon 切「已复制 ✓」(copiedItem 命中), 其余正常。
const copyMenuItems = computed(() => [
  [
    {
      label: copiedItem.value === 'id' ? t('common.copied') : t('inspector.copy_id'),
      icon: copiedItem.value === 'id' ? 'i-tabler-check' : 'i-tabler-id',
      onSelect: (e: Event) => {
        if (props.node) void copyAndFlash(props.node.id, 'id', e)
      },
    },
    {
      label: copiedItem.value === 'json' ? t('common.copied') : t('inspector.copy_json'),
      icon: copiedItem.value === 'json' ? 'i-tabler-check' : 'i-tabler-braces',
      onSelect: (e: Event) => {
        if (props.node) void copyAndFlash(JSON.stringify(props.node, null, 2), 'json', e)
      },
    },
    {
      label: copiedItem.value === 'script' ? t('common.copied') : t('inspector.copy_script'),
      icon: copiedItem.value === 'script' ? 'i-tabler-check' : 'i-tabler-code',
      onSelect: (e: Event) => {
        void copyAndFlash(buildScriptCall(), 'script', e)
      },
    },
  ],
])

// Subgraph 调用节点：1:1 模型，只显示绑定的子图（不需 USelect 选择）
const editorStore = useContainerEditorStore()

const boundSubgraph = computed(() => {
  const sgID = props.node?.config?.SubgraphID
  if (!sgID) return null
  // only visible (non-anonymous) subgraphs are valid Subgraph-call targets.
  // CollapsedNode-backers (isAnonymous) shouldn't show editable label/tags here.
  return editorStore.visibleSubgraphs.find((s) => s.id === sgID) ?? null
})

function onEnterSubgraph() {
  const sgID = props.node?.config?.SubgraphID
  if (!sgID) return
  editorStore.pushPath(String(sgID))
}

// (「发布到库」已删 — 子图全局化后天生在池里。)

// 所有子图聚合 tags（给 UInputMenu autocomplete）— 排除 isAnonymous (用 visibleSubgraphs).
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
  // label/desc/tags 不在 sg.graph 里, activeGraph 深 watch 看不见 — 显式记改动归属本容器。
  editorStore.touchSubgraph(editorStore.activeContainerID, (boundSubgraph.value as any).id)
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
// 使用示例 — 仅当该 kind 配了 example 翻译 (te) 才返非空, 驱动 header「?」弹层显示.
const example = computed(() => {
  if (!props.node) return ''
  const key = KIND_EXAMPLE[props.node.kind]
  return key && te(key) ? t(key) : ''
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

// 有专属 Inspector section 的 kind — 这些不显通用「数据输入」section (BESPOKE_EDITOR_KINDS 同源),
// 也不显 "no_config" 占位 (它们有自己的 UI)。
const BESPOKE_SECTION_KINDS = new Set(['Subgraph', 'MouseCalibration', 'WindowTarget', 'PlayClip', 'Switch'])
const hasBespokeSection = computed(() => !!props.node && BESPOKE_SECTION_KINDS.has(props.node.kind))

// 「输出」组速览: 当前 config 下的出口 pin(exec + data), 只读展示。
// pinsFor 已含动态出口(Switch.cases / Parallel.n 等), dataOut 类型从 PIN_SPECS 反查。
const outPins = computed(() => {
  if (!props.node) return { exec: [] as string[], data: [] as { name: string; type: string }[] }
  const p = pinsFor(props.node.kind, props.node.config)
  const dataTypes = PIN_SPECS[props.node.kind]?.dataOut ?? {}
  return {
    exec: p.execOut,
    data: p.dataOut.map((name) => ({ name, type: String(dataTypes[name] ?? '') })),
  }
})

// 屏幕拾取 → 回填 config.literal (PascalCase Spec.Input 名 + 正确类型):
//   - point: XRatio/YRatio (Number pin, ClickAt/Scroll) — 存 number 不存字符串。
//   - rect:  Region ({x,y,w,h} object, DetectColor Rect pin) — runtime buildDataWireFor coerce 成 node.Rect。
//   - color: Range / 对应 pin (DetectColor/DetectColorHSV) — tuple 直传数组, object 映射 hsv 字段。

// 按目标 pin 决定 schemaType + colorSpace (开吸管前和回填时都用).
function colorMetaFor(fieldPath: string): { schemaType: 'tuple' | 'object'; colorSpace: 'hsv' | 'rgb' } {
  const sc = fieldFor(fieldPath)?.schema
  const schemaType = sc?.type === 'tuple' ? 'tuple' : 'object'
  // object schema 恒 hsv; tuple 读 config.literal.Mode (缺/空 → hsv)
  const colorSpace =
    schemaType === 'object' ? 'hsv' : String(getLiteral('Mode') ?? '') === 'rgb' ? 'rgb' : 'hsv'
  return { schemaType, colorSpace }
}

const { picking, canPickPoint, canPickRect, onPickPoint, onPickRect, onPickColor, onOpenHUD } = useScreenPick({
  node: toRef(props, 'node'),
  applyPoint: (x, y) => setLiteralBatch({ XRatio: round4(x), YRatio: round4(y) }),
  // fieldPath 由 onPickRect(fieldPath) 透传 — DetectColor 固定走 'Region'
  // Region 是 Geometry 类型 ({ pct, overrides }) — 回填必须写 .pct 外壳, 否则
  // GeometryWidget 读不到 .pct 会整体回退成全 0 (不显示框选结果). overrides 原样保留。
  applyRect: (_fieldPath, r) => {
    const cur = (getLiteral('Region') ?? {}) as { overrides?: unknown[] }
    setLiteral('Region', {
      pct: { x: round3(r[0]), y: round3(r[1]), w: round3(r[2]), h: round3(r[3]) },
      overrides: cur.overrides ?? [],
    })
  },
  applyColor: (fieldPath, range, hueWrap) => {
    const { schemaType } = colorMetaFor(fieldPath)
    setLiteral(fieldPath, fillColorLiteral(range, schemaType))
    if (hueWrap) {
      toastForSync.add({
        title: t('inspector.color_pick_huewrap_title'),
        description: t('inspector.color_pick_huewrap_desc'),
        color: 'warning',
        icon: 'i-tabler-color-swatch',
      })
    }
  },
})

// 吸管按钮点击 — 按 pin 决定 colorSpace 后开颜色拾取器.
function onColorPick(fieldPath: string) {
  const { colorSpace } = colorMetaFor(fieldPath)
  void onPickColor(fieldPath, colorSpace)
}
function round4(n: number): number {
  return Math.round(n * 1e4) / 1e4
}
function round3(n: number): number {
  return Math.round(n * 1e3) / 1e3
}

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
  emit('update', { ...props.node.config, keepRanges: next })
}

function updateRange(idx: number, field: 'fromMs' | 'toMs', valMs: number) {
  if (!props.node) return
  const next = currentRanges()
  if (idx < 0 || idx >= next.length) return
  if (field === 'fromMs') next[idx].fromUs = Math.max(0, Math.floor(valMs * 1000))
  else next[idx].toUs = Math.max(0, Math.floor(valMs * 1000))
  emit('update', { ...props.node.config, keepRanges: next })
}

function removeRange(idx: number) {
  if (!props.node) return
  const next = currentRanges()
  next.splice(idx, 1)
  emit('update', { ...props.node.config, keepRanges: next })
}

function onTimelineAdd(r: { fromMs: number; toMs: number }) {
  if (!props.node) return
  const cur = currentRanges()
  cur.push({ fromUs: r.fromMs * 1000, toUs: r.toMs * 1000 })
  cur.sort((a, b) => a.fromUs - b.fromUs)
  emit('update', { ...props.node.config, keepRanges: cur })
}

function onTimelineUpdate(idx: number, r: { fromMs: number; toMs: number }) {
  if (!props.node) return
  const cur = currentRanges()
  if (idx < 0 || idx >= cur.length) return
  cur[idx] = { fromUs: r.fromMs * 1000, toUs: r.toMs * 1000 }
  cur.sort((a, b) => a.fromUs - b.fromUs)
  emit('update', { ...props.node.config, keepRanges: cur })
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

// ─── WindowTarget section ──────────────────────────────────────────────────
// 双向绑定 — config 顶层 PascalCase 字段, 对齐 internal/nodes/system/window_target.go Spec.Inputs.
// 直接 mutate props.node.config 让父图 deep watch 标 dirty (跟 PlayClip keepRanges 一样).
// 绑 config.literal (pin 字面量正源). seed: 优先沿用已有 literal, 否则从顶层 config 旧值迁入,
// 再否则用默认。v-model 写 literal.* → 跟 runtime PinString 同源。顶层旧 key 由迁移脚本清理。
const wtConfig = computed(() => {
  if (props.node?.kind !== 'WindowTarget') return null as any
  if (!props.node.config) (props.node as any).config = {}
  const cfg = props.node.config as any
  if (!cfg.literal) cfg.literal = {}
  const lit = cfg.literal as Record<string, any>
  const seed = (k: string, def: string) => {
    if (lit[k] === undefined) lit[k] = cfg[k] !== undefined ? cfg[k] : def
  }
  seed('Title', '')
  seed('Class', '')
  seed('ProcessName', '')
  seed('TitleMatch', 'exact')
  return lit
})

const titleMatchOptions = computed(() => [
  { value: 'exact', label: t('node.WindowTarget.inspector.title_match_exact') },
  { value: 'regex', label: t('node.WindowTarget.inspector.title_match_regex') },
])

const capturing = ref(false)
const captureID = ref('')
// 窗口捕获键: 读热键中心 tools.window-capture 实时绑定 (用户可在「快捷键」页 rebind), 回退 F9。
const hotkeysStore = useHotkeysStore()
const captureHk = computed(() => hotkeysStore.keyFor('tools.window-capture', 'F9'))

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
    const id = (await backend.tools.startWindowTargetCapture()) as string
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
    // 把捕获时的 resolution 存到 node config — 给 ROI 节点 metadata 用
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
