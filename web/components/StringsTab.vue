<template>
  <div>
    <div class="toolbar">
      <input v-model="q" class="search" placeholder="搜索字符串内容,如 http、.exe、/bin/" />
      <label class="dim"><input type="checkbox" v-model="xrefOnly" /> 只看到有函数引用的</label>
      <span class="dim">{{ filtered.length }} / {{ result.strings.length }}</span>
    </div>
    <table>
      <thead>
        <tr>
          <th>地址</th><th>字符串</th><th>引用它的函数</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="s in page" :key="s.addr">
          <td class="mono dim">{{ hex(s.addr) }}</td>
          <td class="mono str">{{ s.text }}</td>
          <td class="mono">
            <span v-for="(x, i) in s.xrefs ?? []" :key="i" class="xref">
              {{ funcNames.get(x) ?? hex(x) }}
            </span>
            <span v-if="!(s.xrefs?.length)" class="dim">—</span>
          </td>
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

const props = defineProps<{ result: Result; funcNames: Map<number, string> }>()

const q = ref('')
const xrefOnly = ref(true)
const pageIdx = ref(0)
const pageSize = 200

const filtered = computed(() => {
  const needle = q.value.trim().toLowerCase()
  let list = props.result.strings ?? []
  if (xrefOnly.value) list = list.filter(s => s.xrefs?.length)
  if (needle) list = list.filter(s => s.text.toLowerCase().includes(needle))
  return list
})

const pages = computed(() => Math.max(1, Math.ceil(filtered.value.length / pageSize)))
const page = computed(() => filtered.value.slice(pageIdx.value * pageSize, (pageIdx.value + 1) * pageSize))

watch([q, xrefOnly], () => { pageIdx.value = 0 })
</script>

<style scoped>
.str { word-break: break-all; max-width: 46vw; white-space: pre-wrap; }
.xref { color: var(--accent); margin-right: 10px; }
.pager { display: flex; gap: 12px; align-items: center; margin-top: 12px; }
</style>
