<!-- yt 脚本控制台 — 写 JS 对当前容器(含子图)批量改节点。CodeMirror(复用 scriptEditorExtensions
     + yt.* 补全) + 运行按钮(Ctrl+Enter) + 输出区。执行/落地由 useYtConsole 的 run 注入 (prop)。 -->
<template>
  <BaseModal
    :open="open"
    :title="t('editor.jsConsole.title')"
    icon="i-tabler-terminal-2"
    size="3xl"
    tall
    @update:open="(v: boolean) => emit('update:open', v)"
  >
    <div class="flex h-full min-h-0 flex-col gap-3">
      <p class="shrink-0 text-xs text-muted">{{ t('editor.jsConsole.hint') }}</p>
      <div ref="host" class="shrink-0 overflow-hidden rounded-md border border-default" />
      <div class="min-h-0 flex-1 overflow-y-auto">
        <div v-if="report" class="space-y-2 font-mono text-xs">
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
import { nextTick, onBeforeUnmount, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { EditorView, keymap } from "@codemirror/view";
import { scriptEditorExtensions } from "@/lib/scriptCompletions";
import { YT_COMPLETIONS } from "@/lib/ytConsole/completions";
import type { RunResult } from "@/lib/ytConsole/executor";
import BaseModal from "@/components/common/BaseModal.vue";

const props = defineProps<{ open: boolean; run: (code: string) => RunResult }>();
const emit = defineEmits<{ "update:open": [v: boolean] }>();
const { t } = useI18n();

const host = ref<HTMLElement | null>(null);
const report = ref<RunResult | null>(null);
let view: EditorView | null = null;

function onRun(): void {
  if (!view) return;
  report.value = props.run(view.state.doc.toString());
}

function mountEditor(): void {
  if (!host.value || view) return;
  view = new EditorView({
    parent: host.value,
    doc: "",
    extensions: [
      ...scriptEditorExtensions({
        completions: () => YT_COMPLETIONS,
        placeholder: t("editor.jsConsole.placeholder"),
        minHeight: "12em",
      }),
      keymap.of([{ key: "Mod-Enter", run: () => (onRun(), true) }]),
    ],
  });
}

// 懒挂载: 打开才建编辑器, 关闭销毁 (省内存 + 每次开是干净的)。
watch(
  () => props.open,
  async (o) => {
    if (o) {
      await nextTick();
      mountEditor();
    } else {
      view?.destroy();
      view = null;
    }
  },
);
onBeforeUnmount(() => {
  view?.destroy();
  view = null;
});
</script>
