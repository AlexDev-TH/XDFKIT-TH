import type { Num } from './types'

export function hex(n: number, pad = 0): string {
  return '0x' + n.toString(16).toUpperCase().padStart(pad, '0')
}

export function bytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(n < 10240 ? 1 : 0)} KB`
  return `${(n / 1024 / 1024).toFixed(n < 10 * 1024 * 1024 ? 2 : 1)} MB`
}

/** A cell or axis value at the definition's precision; a value that isn't finite shows as a dash. */
export function num(v: Num, precision: number): string {
  if (v === null || !Number.isFinite(v)) return '-'
  return v.toFixed(Math.max(0, Math.min(precision, 8)))
}

/** A factor or offset: as many digits as it takes, without float noise. */
export function plain(v: Num): string {
  if (v === null || !Number.isFinite(v)) return '-'
  const s = Number(v.toPrecision(10)).toString()
  return s.includes('e') ? v.toExponential(4) : s
}

export function fileName(path: string): string {
  return path.split(/[\\/]/).pop() ?? path
}

export function dirName(path: string): string {
  const i = Math.max(path.lastIndexOf('\\'), path.lastIndexOf('/'))
  return i > 0 ? path.slice(0, i) : ''
}

/** The middle of a long path gives way first: the drive and the file name stay readable. */
export function shortPath(path: string, max = 64): string {
  if (path.length <= max) return path
  const name = fileName(path)
  const head = path.slice(0, Math.max(8, max - name.length - 4))
  return `${head}...\\${name}`
}

export function count(n: number): string {
  return n.toLocaleString('en-US')
}
