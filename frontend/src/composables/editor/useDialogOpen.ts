// Bidirectional sync: 父 v-model:open ↔ 内部 modelOpen ref.
// 8 modal 重复 3 行模板 (CommandPalette / ContainerSettings / FindReferences /
// LibraryExplorer / NodeExplorer / NodeSearch / PromoteToVar / DeleteVarConfirm)
// 抽这里. backlog C4.
//
// Usage:
//   const props = defineProps<{ open: boolean }>()
//   const emit = defineEmits<{ 'update:open': [v: boolean] }>()
//   const modelOpen = useDialogOpen(props, emit)

import { ref, watch, type Ref } from 'vue'

type EmitOpen = (event: 'update:open', v: boolean) => void

export function useDialogOpen(
  props: { open: boolean },
  emit: EmitOpen,
): Ref<boolean> {
  const modelOpen = ref(props.open)
  watch(() => props.open, (v) => { modelOpen.value = v })
  watch(modelOpen, (v) => emit('update:open', v))
  return modelOpen
}
