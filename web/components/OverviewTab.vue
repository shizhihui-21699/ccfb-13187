<template>
  <div>
    <div class="cards">
      <div class="card"><div class="k">编译器</div><div class="v">{{ r.build?.goVersion || '—' }}</div></div>
      <div class="card"><div class="k">格式 / 架构</div><div class="v">{{ r.format }} / {{ r.arch }}</div></div>
      <div class="card"><div class="k">pclntab</div><div class="v">{{ r.pclnVersion }}</div></div>
      <div class="card"><div class="k">函数</div><div class="v">{{ r.stats.funcCount }}</div></div>
      <div class="card"><div class="k">字符串(有引用)</div><div class="v">{{ r.stats.stringCount }} ({{ r.stats.stringXRefCount }})</div></div>
      <div class="card"><div class="k">第三方依赖</div><div class="v">{{ r.stats.depCount }}</div></div>
    </div>

    <h3>主模块</h3>
    <p class="mono">{{ r.build?.path || '—' }} <span class="dim">{{ r.build?.main?.version }}</span></p>

    <h3>包分布(按函数数,前 20)</h3>
    <div class="pkglist">
      <div v-for="p in topPkgs" :key="p.path" class="pkgrow">
        <span class="mono pkgname" :title="p.path">{{ p.path }}</span>
        <span class="bar" :style="{ width: (p.count / maxPkg * 100) + '%' }"></span>
        <span class="dim">{{ p.count }}</span>
      </div>
    </div>

    <template v-if="interesting.length">
      <h3>可疑字符串(URL / 路径 / 命令)</h3>
      <ul class="ioc">
        <li v-for="(s, i) in interesting" :key="i" class="mono">{{ s }}</li>
      </ul>
    </template>
  </div>
</template>

<script setup lang="ts">
import type { Result } from '~/types'

const props = defineProps<{ result: Result }>()
const r = props.result

const topPkgs = computed(() => (r.packages ?? []).slice(0, 20))
const maxPkg = computed(() => topPkgs.value[0]?.count ?? 1)

const interesting = computed(() => {
  const out: string[] = []
  for (const s of r.strings ?? []) {
    const t = s.text
    if (/https?:\/\//.test(t) || t.includes('C:\\') || t.startsWith('/etc/') ||
        t.startsWith('/tmp/') || t.includes('powershell') || t.includes('cmd.exe') ||
        t.includes('/bin/sh') || t.includes('schtasks')) {
      out.push(t.length > 160 ? t.slice(0, 160) + '…' : t)
    }
    if (out.length >= 60) break
  }
  return out
})
</script>

<style scoped>
.cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 10px; margin-bottom: 18px; }
.card { background: var(--bg2); border: 1px solid var(--border); border-radius: 8px; padding: 10px 14px; }
.card .k { color: var(--dim); font-size: 12px; }
.card .v { font-size: 18px; font-family: ui-monospace, monospace; color: var(--accent2); }
h3 { margin: 18px 0 8px; font-size: 14px; color: var(--dim); }
.pkglist { display: grid; gap: 3px; }
.pkgrow { display: grid; grid-template-columns: minmax(200px, 380px) 1fr 50px; align-items: center; gap: 8px; }
.pkgname { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.bar { height: 10px; background: var(--accent); border-radius: 3px; opacity: .7; min-width: 2px; }
.ioc { padding-left: 18px; }
.ioc li { color: var(--warn); margin: 2px 0; word-break: break-all; }
</style>
