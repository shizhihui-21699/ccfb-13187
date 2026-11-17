export function hex(n: number, pad = 8): string {
  return '0x' + n.toString(16).padStart(pad, '0')
}

export function baseName(p: string): string {
  const i = Math.max(p.lastIndexOf('/'), p.lastIndexOf('\\'))
  return i >= 0 ? p.slice(i + 1) : p
}

export function shorten(p: string, keep = 3): string {
  const parts = p.split('/')
  if (parts.length <= keep + 1) return p
  return '…/' + parts.slice(-keep).join('/')
}
