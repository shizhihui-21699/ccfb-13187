<template>
  <div>
    <h3>编译信息</h3>
    <table class="kv">
      <tbody>
        <tr><td class="dim">Go 版本</td><td class="mono">{{ b?.goVersion || '—' }}</td></tr>
        <tr><td class="dim">主模块路径</td><td class="mono">{{ b?.path || '—' }}</td></tr>
        <tr v-if="b?.main"><td class="dim">主模块版本</td><td class="mono">{{ b.main.version || '(devel)' }}</td></tr>
      </tbody>
    </table>

    <h3>第三方依赖({{ deps.length }})</h3>
    <table>
      <thead><tr><th>模块</th><th>版本</th><th>替换</th></tr></thead>
      <tbody>
        <tr v-for="m in deps" :key="m.path">
          <td class="mono">{{ m.path }}</td>
          <td class="mono accent">{{ m.version }}</td>
          <td class="mono dim">{{ m.replace || '' }}</td>
        </tr>
        <tr v-if="!deps.length"><td colspan="3" class="dim">无(仅标准库)</td></tr>
      </tbody>
    </table>

    <h3>构建参数</h3>
    <table>
      <tbody>
        <tr v-for="s in settings" :key="s.key">
          <td class="mono dim" style="width:220px">{{ s.key }}</td>
          <td class="mono">{{ s.value }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import type { Result } from '~/types'

const props = defineProps<{ result: Result }>()
const b = computed(() => props.result.build)
const deps = computed(() => props.result.build?.deps ?? [])
const settings = computed(() => (props.result.build?.settings ?? []).filter(s => s.key !== 'vcs'))
</script>

<style scoped>
h3 { margin: 18px 0 8px; font-size: 14px; color: var(--dim); }
.kv td { border: 0; padding: 3px 10px 3px 0; }
.accent { color: var(--accent2); }
</style>
