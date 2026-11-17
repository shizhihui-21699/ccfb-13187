import type { Result } from '~/types'

const result = ref<Result | null>(null)
const loading = ref(false)
const error = ref('')
const warning = ref('')

export function useAnalysis() {
  async function analyze(file: File) {
    loading.value = true
    error.value = ''
    warning.value = ''
    try {
      const fd = new FormData()
      fd.append('file', file)
      const res = await fetch('/api/analyze', { method: 'POST', body: fd })
      warning.value = res.headers.get('X-Gores-Warning') ?? ''
      if (!res.ok) throw new Error(`服务器错误 ${res.status}: ${await res.text()}`)
      result.value = await res.json()
    } catch (e: any) {
      error.value = e?.message ?? String(e)
    } finally {
      loading.value = false
    }
  }

  function reset() {
    result.value = null
    error.value = ''
    warning.value = ''
  }

  const funcNameByEntry = computed(() => {
    const m = new Map<number, string>()
    for (const f of result.value?.functions ?? []) m.set(f.entry, f.name)
    return m
  })

  return { result, loading, error, warning, analyze, reset, funcNameByEntry }
}
