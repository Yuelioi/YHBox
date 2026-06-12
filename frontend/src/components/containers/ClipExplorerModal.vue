<!-- clip 管理 modal (编辑器内入口: 左 rail 🎬). 包全局 ClipManager; 开时刷一次列表. -->
<template>
  <BaseModal
    v-model:open="modelOpen"
    :title="t('clip.manager.title')"
    icon="i-tabler-movie"
    size="5xl"
  >
    <ClipManager />
  </BaseModal>
</template>

<script setup lang="ts">
import { watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'
import { useClipsStore } from '@/stores/clips'
import BaseModal from '@/components/common/BaseModal.vue'
import ClipManager from '@/components/containers/ClipManager.vue'

const { t } = useI18n()
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ 'update:open': [v: boolean] }>()
const modelOpen = useDialogOpen(props, emit)

const clips = useClipsStore()
watch(modelOpen, (open) => {
  if (open) void clips.refresh()
})
</script>
