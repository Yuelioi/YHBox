<!-- 节点模板字段内联触发器 (NodeInspector 的 widgetKind==='template-picker' 渲染它).
     触发按钮 (已选缩略图+数量) + 已选 chips (缺失告警) + 内嵌 TemplateExplorerModal 的 pick 模式.
     v-model:string[] ↔ 节点 config.literal.Templates (多选 GUID 列表). 取代旧 TemplatePicker. -->
<template>
  <div class="space-y-1.5">
    <UButton
      variant="outline"
      color="neutral"
      size="sm"
      class="w-full justify-start font-normal"
      :title="selectedNames.join(', ') || t('template.picker.not_selected')"
      @click="open = true"
    >
      <div class="flex items-center gap-2 min-w-0 flex-1">
        <img v-if="firstThumb" :src="firstThumb" class="size-6 rounded object-contain bg-elevated shrink-0" alt="" />
        <UIcon
          v-else
          :name="selected.length ? 'i-tabler-photo' : 'i-tabler-photo-plus'"
          class="size-4 shrink-0"
          :class="selected.length ? 'text-dimmed' : 'text-warning'"
        />
        <div class="min-w-0 flex-1 text-left">
          <div class="text-xs text-highlighted truncate">{{ summaryLabel }}</div>
        </div>
        <UIcon name="i-tabler-chevron-down" class="size-3.5 text-dimmed shrink-0" />
      </div>
    </UButton>

    <TemplateExplorerModal v-model:open="open" pick-mode :model-value="selected" @update:model-value="onUpdate" />

    <div v-if="selected.length" class="flex flex-wrap gap-1">
      <span
        v-for="guid in selected"
        :key="guid"
        class="inline-flex items-center gap-1 rounded bg-elevated/60 border border-default/40 pl-1.5 pr-1 py-0.5 text-[10px]"
        :class="tplStore.map[guid] ? 'text-toned' : 'text-error/90 border-error/30'"
      >
        <UIcon v-if="!tplStore.map[guid]" name="i-tabler-alert-triangle" class="size-3" />
        <span class="truncate max-w-[140px]">{{ tplStore.map[guid]?.name || guid }}</span>
        <UIcon
          name="i-tabler-x"
          class="size-3 cursor-pointer hover:text-error"
          @click="onUpdate(selected.filter((g) => g !== guid))"
        />
      </span>
    </div>
    <p v-else class="text-[10px] text-dimmed">{{ t('template.picker.no_template_selected') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTemplatesStore } from '@/stores/templates'
import TemplateExplorerModal from '@/components/containers/TemplateExplorerModal.vue'

const { t } = useI18n()
const props = defineProps<{ modelValue: string[] }>()
const emit = defineEmits<{ 'update:modelValue': [v: string[]] }>()
const tplStore = useTemplatesStore()
const open = ref(false)

// modelValue 容错: undefined / 单 string (迁移前残留) / string[] → string[].
const selected = computed<string[]>(() => {
  const v = props.modelValue as unknown
  if (Array.isArray(v)) return v.filter((x): x is string => typeof x === 'string')
  if (typeof v === 'string' && v) return [v]
  return []
})
function onUpdate(v: string[]) {
  emit('update:modelValue', v)
}

onMounted(() => {
  if (Object.keys(tplStore.map).length === 0) void tplStore.reload()
})

const selectedNames = computed(() => selected.value.map((g) => tplStore.map[g]?.name || g))
const summaryLabel = computed(() => {
  if (selected.value.length === 0) return t('template.picker.select_placeholder')
  if (selected.value.length === 1) return tplStore.map[selected.value[0]]?.name || selected.value[0]
  return t('template.picker.selected_count', { n: selected.value.length })
})

// 首个已选模板缩略图 (触发按钮上显示).
const firstThumb = ref<string | undefined>(undefined)
watch(
  selected,
  async (guids) => {
    firstThumb.value = undefined
    const s = guids[0] ? tplStore.map[guids[0]] : undefined
    if (s?.firstBlobSha) {
      const r = await tplStore.readBlobDataURL(s.firstBlobSha)
      if (typeof r === 'string') firstThumb.value = r
    }
  },
  { immediate: true },
)
</script>
