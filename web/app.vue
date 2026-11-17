<template>
  <div class="page">
    <header class="topbar">
      <div class="brand">
        <span class="logo">⛏</span>
        <h1>gores</h1>
        <span class="tag">Go 二进制符号还原</span>
      </div>
      <div v-if="result" class="fileinfo">
        <span class="fname">{{ baseName(result.fileName) }}</span>
        <span class="badge">{{ result.format }}</span>
        <span class="badge">{{ result.arch }}</span>
        <span class="badge accent">{{ result.build?.goVersion || 'unknown' }}</span>
        <button class="btn ghost" @click="reset">分析新文件</button>
      </div>
    </header>

    <!-- upload -->
    <main v-if="!result" class="upload-wrap">
      <div
        class="dropzone"
        :class="{ over: dragOver }"
        @dragover.prevent="dragOver = true"
        @dragleave="dragOver = false"
        @drop.prevent="onDrop"
        @click="pickFile"
      >
        <div class="dz-icon">📦</div>
        <p class="dz-title">拖入去符号的 Go 二进制,或点击选择</p>
        <p class="dz-sub">支持 ELF / PE / Mach-O · amd64 / 386 / arm64 · Go 1.16+</p>
        <input ref="fileInput" type="file" hidden @change="onPick" />
      </div>
      <p v-if="loading" class="status">解析中,大文件可能需要几秒…</p>
      <p v-if="error" class="status err">{{ error }}</p>
    </main>

    <!-- dashboard -->
    <main v-else class="dash">
      <p v-if="warning" class="status err">部分解析失败:{{ warning }}</p>
      <nav class="tabs">
        <button
          v-for="t in tabs"
          :key="t.key"
          class="tab"
          :class="{ active: tab === t.key }"
          @click="tab = t.key"
        >
          {{ t.label }}
          <span v-if="t.count !== undefined" class="count">{{ t.count }}</span>
        </button>
      </nav>
      <OverviewTab v-if="tab === 'overview'" :result="result" />
      <FunctionsTab v-else-if="tab === 'functions'" :result="result" />
      <StringsTab v-else-if="tab === 'strings'" :result="result" :func-names="funcNameByEntry" />
      <ModulesTab v-else-if="tab === 'modules'" :result="result" />
    </main>
  </div>
</template>

<script setup lang="ts">
const { result, loading, error, warning, analyze, reset, funcNameByEntry } = useAnalysis()

const tab = ref('overview')
const dragOver = ref(false)
const fileInput = ref<HTMLInputElement>()

const tabs = computed(() => [
  { key: 'overview', label: '概览' },
  { key: 'functions', label: '函数', count: result.value?.stats.funcCount },
  { key: 'strings', label: '字符串', count: result.value?.stats.stringCount },
  { key: 'modules', label: '模块与构建', count: result.value?.stats.depCount },
])

function pickFile() {
  fileInput.value?.click()
}
function onPick(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (f) analyze(f)
}
function onDrop(e: DragEvent) {
  dragOver.value = false
  const f = e.dataTransfer?.files?.[0]
  if (f) analyze(f)
}
</script>

<style>
:root {
  --bg: #0d1117;
  --bg2: #161b22;
  --bg3: #1c2330;
  --fg: #d7dee8;
  --dim: #8b98ab;
  --accent: #58c4dc;
  --accent2: #7ee787;
  --warn: #f0883e;
  --border: #2a3342;
}
* { box-sizing: border-box; }
body {
  margin: 0;
  background: var(--bg);
  color: var(--fg);
  font: 14px/1.5 -apple-system, "SF Pro", "Segoe UI", "PingFang SC", sans-serif;
}
.page { min-height: 100vh; }
.topbar {
  display: flex; align-items: center; justify-content: space-between;
  gap: 12px; flex-wrap: wrap;
  padding: 10px 20px;
  background: var(--bg2);
  border-bottom: 1px solid var(--border);
}
.brand { display: flex; align-items: baseline; gap: 8px; }
.logo { font-size: 20px; }
.brand h1 { font-size: 18px; margin: 0; color: var(--accent); font-family: ui-monospace, monospace; }
.tag { color: var(--dim); font-size: 12px; }
.fileinfo { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.fname { font-family: ui-monospace, monospace; }
.badge {
  background: var(--bg3); border: 1px solid var(--border);
  padding: 1px 8px; border-radius: 10px; font-size: 12px; color: var(--dim);
}
.badge.accent { color: var(--accent2); border-color: var(--accent2); }
.btn {
  background: var(--accent); color: #04222b; border: 0; border-radius: 6px;
  padding: 5px 14px; cursor: pointer; font-weight: 600;
}
.btn.ghost { background: transparent; color: var(--accent); border: 1px solid var(--accent); }

.upload-wrap { display: grid; place-items: center; padding: 60px 20px; }
.dropzone {
  width: min(620px, 92vw); padding: 60px 30px; text-align: center;
  border: 2px dashed var(--border); border-radius: 14px; cursor: pointer;
  background: var(--bg2); transition: border-color .15s;
}
.dropzone.over { border-color: var(--accent); }
.dz-icon { font-size: 42px; }
.dz-title { font-size: 16px; margin: 12px 0 4px; }
.dz-sub { color: var(--dim); font-size: 13px; margin: 0; }
.status { margin-top: 18px; color: var(--dim); }
.status.err { color: var(--warn); }

.dash { padding: 16px 20px 40px; }
.tabs { display: flex; gap: 4px; border-bottom: 1px solid var(--border); margin-bottom: 14px; flex-wrap: wrap; }
.tab {
  background: none; border: 0; color: var(--dim); padding: 8px 14px;
  cursor: pointer; font-size: 14px; border-bottom: 2px solid transparent;
}
.tab.active { color: var(--fg); border-bottom-color: var(--accent); }
.count {
  background: var(--bg3); border-radius: 8px; padding: 0 6px;
  font-size: 11px; margin-left: 4px;
}

table { border-collapse: collapse; width: 100%; }
th, td { text-align: left; padding: 5px 10px; border-bottom: 1px solid var(--border); }
th { color: var(--dim); font-weight: 600; font-size: 12px; position: sticky; top: 0; background: var(--bg); }
td.mono, .mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12.5px; }
tr:hover td { background: var(--bg2); }
.toolbar { display: flex; gap: 10px; margin-bottom: 10px; align-items: center; flex-wrap: wrap; }
.search {
  background: var(--bg2); border: 1px solid var(--border); color: var(--fg);
  border-radius: 6px; padding: 6px 10px; width: 320px; outline: none;
}
.search:focus { border-color: var(--accent); }
.dim { color: var(--dim); }
.link { color: var(--accent); cursor: pointer; }
.pill { font-size: 11px; border-radius: 8px; padding: 0 6px; background: var(--bg3); color: var(--dim); }
</style>
