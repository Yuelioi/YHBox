import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { backend, type WorkflowSnippet, type WorkflowSnippetSummary } from '@/lib/backend'

export const useSnippetsStore = defineStore('workflow-snippets', () => {
  const items = ref<WorkflowSnippetSummary[]>([])
  const warnings = ref<Array<{ file: string; error: string }>>([])
  const loading = ref(false)
  const loaded = ref(false)

  const categories = computed(() =>
    [...new Set(items.value.map((item) => item.category?.trim()).filter(Boolean) as string[])].sort(
      (left, right) => left.localeCompare(right),
    ),
  )
  const tags = computed(() =>
    [...new Set(items.value.flatMap((item) => item.tags))].sort((left, right) =>
      left.localeCompare(right),
    ),
  )

  async function load(force = false): Promise<void> {
    if (loading.value || (loaded.value && !force)) return
    loading.value = true
    try {
      const result = await backend.snippets.list()
      items.value = result.items ?? []
      warnings.value = result.warnings ?? []
      loaded.value = true
    } finally {
      loading.value = false
    }
  }

  async function get(id: string): Promise<WorkflowSnippet> {
    return backend.snippets.get(id)
  }

  async function save(value: WorkflowSnippet): Promise<WorkflowSnippet> {
    const saved = await backend.snippets.save(value)
    await load(true)
    return saved
  }

  async function remove(id: string): Promise<void> {
    await backend.snippets.delete_(id)
    await load(true)
  }

  async function markUsed(id: string): Promise<void> {
    await backend.snippets.markUsed(id)
    await load(true)
  }

  return { items, warnings, loading, loaded, categories, tags, load, get, save, remove, markUsed }
})
