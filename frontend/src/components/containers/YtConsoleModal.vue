<!-- yt 脚本控制台 — 写 JS 对当前容器(含子图)批量改节点。复用共享 <CodeEditor> 主体
     (工具栏/参考面板/状态栏/补全/折叠, 跟 Script/Expr 一致), 外加 运行 + 输出区。
     执行/落地由 useYtConsole 的 run 注入 (prop)。 -->
<template>
  <BaseModal
    :open="open"
    :title="t('editor.jsConsole.title')"
    icon="i-tabler-terminal-2"
    size="6xl"
    tall
    @update:open="(v: boolean) => emit('update:open', v)"
  >
    <div class="flex h-full min-h-0 flex-col gap-3">
      <p class="shrink-0 text-xs text-muted">{{ t('editor.jsConsole.hint') }}</p>
      <div class="min-h-0 flex-1">
        <CodeEditor
          v-model:model-value="code"
          :extensions="ytExtensions"
          :reference="reference"
          commentable
          foldable
          lang-label="JavaScript"
          @submit="onRun"
        />
      </div>
      <div v-if="report" class="shrink-0 max-h-[28vh] overflow-y-auto border-t border-default pt-2 space-y-2 font-mono text-xs">
        <div v-if="report.error" class="whitespace-pre-wrap text-error">{{ report.error }}</div>
        <div v-else class="text-primary">
          {{ t('editor.jsConsole.applied', { nodes: report.nodeCount, pins: report.pinCount }) }}
        </div>
        <div v-if="report.rejected.length" class="text-warning">
          <div>{{ t('editor.jsConsole.rejected', { n: report.rejected.length }) }}</div>
          <ul class="space-y-0.5 pl-3">
            <li v-for="(rej, i) in report.rejected" :key="i">
              {{ rej.nodeId }}({{ rej.kind }}) · {{ rej.pin }} — {{ rej.reason }}
            </li>
          </ul>
        </div>
        <pre v-if="report.logs.length" class="whitespace-pre-wrap text-muted">{{ report.logs.join('\n') }}</pre>
      </div>
    </div>

    <template #footer>
      <span class="mr-auto text-xs text-muted">Ctrl+Enter</span>
      <UButton color="primary" icon="i-tabler-player-play-filled" @click="onRun">
        {{ t('editor.jsConsole.run') }}
      </UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { Extension } from "@codemirror/state";
import { scriptEditorExtensions } from "@/lib/scriptCompletions";
import { YT_COMPLETIONS, ytHoverDoc, YT_ENTRIES } from "@/lib/ytConsole/completions";
import type { RunResult } from "@/lib/ytConsole/executor";
import BaseModal from "@/components/common/BaseModal.vue";
import CodeEditor, { type RefItem } from "@/components/expressions/CodeEditor.vue";

const props = defineProps<{ open: boolean; run: (code: string) => RunResult }>();
const emit = defineEmits<{ "update:open": [v: boolean] }>();
const { t } = useI18n();

const code = ref("");
const report = ref<RunResult | null>(null);

// 参考面板项 (右侧"文档"栏) — 跟补全/hover 同一份 YT_ENTRIES, 单一来源。
const reference: RefItem[] = YT_ENTRIES.map((e) => ({
  label: e.label,
  detail: e.sig,
  desc: e.desc,
  docs: e.desc,
  insert: e.label,
  caretBack: 0,
  group: e.label.startsWith("yt.") ? "yt" : "NodeHandle",
}));

function ytExtensions(): Extension[] {
  return scriptEditorExtensions({
    completions: () => YT_COMPLETIONS,
    hoverDoc: ytHoverDoc,
    lintMessages: {
      syntaxError: (line: number) => t("inspector.editor_syntax_error_line", { line }),
      unknownVar: (name: string) => t("inspector.editor_unknown_var", { name }),
    },
    placeholder: t("editor.jsConsole.placeholder"),
    modal: true,
  });
}

function onRun(): void {
  report.value = props.run(code.value);
}

// 每次开窗清掉上次的报告 (代码保留, 方便接着调)。
watch(
  () => props.open,
  (o) => {
    if (o) report.value = null;
  },
);
</script>
