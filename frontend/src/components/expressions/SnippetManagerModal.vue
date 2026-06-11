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
      <UInput
        v-model="name"
        size="sm"
        class="w-full"
        :placeholder="t('inspector.snippet_manager_name')"
        autofocus
      />
      <UTextarea
        v-model="body"
        :rows="10"
        class="w-full font-mono"
        :placeholder="t('inspector.snippet_manager_body')"
      />
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
          <div class="text-[12px] text-highlighted truncate">{{ s.name }}</div>
          <div class="text-[11px] font-mono text-muted truncate">{{ firstLine(s.body) }}</div>
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
        <UButton color="primary" :disabled="!name.trim() || !body.trim()" @click="save">
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
const name = ref('')
const body = ref('')

watch(() => props.open, (open) => {
  if (!open) return
  if (props.initialBody !== undefined) startNew(props.initialBody)
  else editing.value = null
})

function startNew(initial: string) {
  editing.value = 'new'
  name.value = ''
  body.value = initial
}

function startEdit(s: CodeSnippet) {
  editing.value = s.id
  name.value = s.name
  body.value = s.body
}

function save() {
  if (editing.value === 'new') store.add(props.lang, name.value.trim(), body.value)
  else if (editing.value) store.update(editing.value, { name: name.value.trim(), body: body.value })
  editing.value = null
}

function firstLine(s: string): string {
  const i = s.indexOf('\n')
  return i === -1 ? s : `${s.slice(0, i)} …`
}
</script>
