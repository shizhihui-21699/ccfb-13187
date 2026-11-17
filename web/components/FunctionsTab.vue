<template>
  <div>
    <div class="toolbar">
      <input v-model="q" class="search" placeholder="搜索函数名 / 文件路径,如 main. 或 http" />
      <label class="dim"><input type="checkbox" v-model="userOnly" /> 只看非标准库</label>
      <span class="dim">{{ filtered.length }} / {{ result.functions.length }}</span>
    </div>
    <table>
      <thead>
        <tr>
          <th>地址</th><th>函数名</th><th>源文件</th><th>行</th><th>大小</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="f in page" :key="f.entry">
          <td class="mono dim">{{ hex(f.entry) }}</td>
          <td class="mono">{{ f.name }}</td>
          <td class="mono dim" :title="f.file">{{ shorten(f.file) }}</td>
          <td class="mono dim">{{ f.line }}</td>
          <td class="mono dim">{{ f.end - f.entry }}</td>
        </tr>
      </tbody>
    </table>
    <div class="pager">
      <button class="btn ghost" :disabled="pageIdx === 0" @click="pageIdx--">上一页</button>
      <span class="dim">{{ pageIdx + 1 }} / {{ pages }}</span>
      <button class="btn ghost" :disabled="pageIdx >= pages - 1" @click="pageIdx++">下一页</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Result } from '~/types'

const props = defineProps<{ result: Result }>()

const q = ref('')
const userOnly = ref(false)
const pageIdx = ref(0)
const pageSize = 200

const STDLIB_PREFIX = ['runtime', 'internal/', 'vendor/', 'sync', 'syscall', 'fmt', 'io', 'os', 'strings', 'bytes', 'strconv', 'errors', 'sort', 'math', 'unicode', 'time', 'context', 'reflect', 'encoding/', 'crypto/', 'net', 'hash', 'regexp', 'bufio', 'path', 'compress/', 'encoding']

const filtered = computed(() => {
  const needle = q.value.trim().toLowerCase()
  let list = props.result.functions ?? []
  if (userOnly.value) {
    list = list.filter(f => !STDLIB_PREFIX.some(p => f.package === p || f.package.startsWith(p)))
  }
  if (needle) {
    list = list.filter(f =>
      f.name.toLowerCase().includes(needle) || f.file.toLowerCase().includes(needle))
  }
  return list
})

const pages = computed(() => Math.max(1, Math.ceil(filtered.value.length / pageSize)))
const page = computed(() => filtered.value.slice(pageIdx.value * pageSize, (pageIdx.value + 1) * pageSize))

watch([q, userOnly], () => { pageIdx.value = 0 })
</script>

<style scoped>
.pager { display: flex; gap: 12px; align-items: center; margin-top: 12px; }
</style>
