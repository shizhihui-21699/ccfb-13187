// Shared types for the analysis result (mirrors gores JSON).
export interface KV { key: string; value: string }
export interface Module { path: string; version: string; sum?: string; replace?: string; main?: boolean }
export interface BuildInfo {
  goVersion: string
  path: string
  main?: Module
  deps: Module[]
  settings: KV[]
}
export interface Func {
  name: string
  entry: number
  end: number
  file: string
  line: number
  package: string
  isMethod: boolean
}
export interface GoString {
  addr: number
  text: string
  xrefs?: number[]
}
export interface PkgStat { path: string; count: number }
export interface Stats {
  funcCount: number
  stringCount: number
  stringXRefCount: number
  depCount: number
}
export interface Result {
  fileName: string
  format: string
  arch: string
  pclnVersion: string
  build: BuildInfo
  functions: Func[]
  strings: GoString[]
  packages: PkgStat[]
  stats: Stats
}
