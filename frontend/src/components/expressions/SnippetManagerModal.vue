<!-- 用户代码片段管理 (EditorModal「片段」下拉的「新建/管理」入口) —
     列表 (名称+首行预览, 行尾编辑/删除) ↔ 表单 (名称+内容) 两态切换。
     打开时 initialBody 非空 → 直接进新建表单 (选区入库流程)。 -->
<template>
  <BaseModal
    :open="open"
    :title="t('inspector.snippet_manager_title')"
    icon="i-tabler-template"
    size="2xl"
    @update:open="(v: boolean) => emit('update:open', v)"
  >
    <!-- 表单态 (新建/编辑) -->
    <div v-if="editing" class="space-y-2">
      <div class="flex gap-2">
        <UInput
          v-model="prefix"
          size="sm"
          class="w-40 shrink-0 font-mono"
          :placeholder="t('inspector.snippet_manager_prefix')"
          autofocus
        />
        <UInput
          v-model="name"
          size="sm"
          class="flex-1 min-w-0"
          :placeholder="t('inspector.snippet_manager_name')"
        />
      </div>
      <UInput
        v-model="description"
        size="sm"
        class="w-full"
        :placeholder="t('inspector.snippet_manager_description')"
      />
      <UTextarea
        v-model="body"
        :rows="10"
        class="w-full font-mono"
        :placeholder="t('inspector.snippet_manager_body')"
      />
      <p v-if="prefix && !prefixValid" class="text-[11px] text-error">
        {{ t('inspector.snippet_manager_prefix_invalid') }}
      </p>
    </div>

    <!-- 列表态 -->
    <div v-else class="space-y-0.5">
      <div v-if="list.length === 0" class="flex flex-col items-center gap-1.5 py-8 text-muted">
        <UIcon name="i-tabler-template-off" class="size-5 opacity-60" />
        <p class="text-[12px]">{{ t('inspector.snippet_manager_empty') }}</p>
      </div>
      <div
        v-for="s in list"
        :key="s.id"
        class="flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-elevated/60 group/row"
      >
        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-1.5 min-w-0">
            <span class="text-[11px] font-mono text-primary bg-primary/10 rounded px-1 shrink-0">{{ s.prefix }}</span>
            <span class="text-[12px] text-highlighted truncate">{{ s.name }}</span>
          </div>
          <div class="text-[11px] font-mono text-muted truncate">{{ s.description || firstLine(s.body) }}</div>
        </div>
        <UButton
          icon="i-tabler-pencil"
          variant="ghost"
          color="neutral"
          size="xs"
          class="shrink-0 opacity-0 group-hover/row:opacity-70 hover:!opacity-100"
          :title="t('common.edit')"
          @click="startEdit(s)"
        />
        <UButton
          icon="i-tabler-trash"
          variant="ghost"
          color="error"
          size="xs"
          class="shrink-0 opacity-0 group-hover/row:opacity-70 hover:!opacity-100"
          :title="t('common.delete')"
          @click="store.remove(s.id)"
        />
      </div>
    </div>

    <template #footer>
      <template v-if="editing">
        <UButton variant="ghost" color="neutral" @click="editing = null">
          {{ t('common.cancel') }}
        </UButton>
        <UButton color="primary" :disabled="!canSave" @click="save">
          {{ t('common.save') }}
        </UButton>
      </template>
      <template v-else>
        <UButton variant="soft" color="primary" icon="i-tabler-plus" class="mr-auto" @click="startNew('')">
          {{ t('inspector.snippet_manager_new') }}
        </UButton>
        <UButton variant="ghost" color="neutral" @click="emit('update:open', false)">
          {{ t('common.close') }}
        </UButton>
      </template>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseModal from '@/components/common/BaseModal.vue'
import {
  useCodeSnippetsStore,
  type CodeSnippet,
  type CodeSnippetLang,
} from '@/stores/codeSnippets'

const { t } = useI18n()

const props = defineProps<{
  open: boolean
  lang: CodeSnippetLang
  /** 给了 (含空串) = 打开即进新建表单并预填; undefined = 打开进列表 (管理)。 */
  initialBody?: string
}>()

const emit = defineEmits<{ 'update:open': [v: boolean] }>()

const store = useCodeSnippetsStore()
const list = computed(() => store.byLang(props.lang))

// editing: null = 列表态; 'new' = 新建; 其余 = 编辑中片段的 id。
const editing = ref<string | null>(null)
const prefix = ref('')
const name = ref('')
const description = ref('')
const body = ref('')

// 触发词 = 标识符形态才会被补全匹配到 (CM 词法按 identifier 截词)。
const prefixValid = computed(() => /^[A-Za-z_$][A-Za-z0-9_$]*$/.test(prefix.value.trim()))
const canSave = computed(() => prefixValid.value && !!name.value.trim() && !!body.value.trim())

watch(() => props.open, (open) => {
  if (!open) return
  void store.ensureLoaded()
  if (props.initialBody !== undefined) startNew(props.initialBody)
  else editing.value = null
})

function startNew(initial: string) {
  editing.value = 'new'
  prefix.value = ''
  name.value = ''
  description.value = ''
  body.value = initial
}

function startEdit(s: CodeSnippet) {
  editing.value = s.id
  prefix.value = s.prefix
  name.value = s.name
  description.value = s.description ?? ''
  body.value = s.body
}

function save() {
  const draft = {
    prefix: prefix.value.trim(),
    name: name.value.trim(),
    description: description.value.trim() || undefined,
    body: body.value,
  }
  if (editing.value === 'new') store.add(props.lang, draft)
  else if (editing.value) store.update(editing.value, draft)
  editing.value = null
}

function firstLine(s: string): string {
  const i = s.indexOf('\n')
  return i === -1 ? s : `${s.slice(0, i)} …`
}
</script>
