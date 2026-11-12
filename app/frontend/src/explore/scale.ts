import type { Num } from '../lib/types'

/** Where v sits between lo and hi, 0 to 1; a flat range sits in the middle. */
export function unit(v: number, lo: number, hi: number): number {
  if (!(hi > lo)) return 0.5
  return Math.min(1, Math.max(0, (v - lo) / (hi - lo)))
}

// Magnitude is one neutral ramp in even lightness steps (OKLCH, no chroma).
// The low end sits next to the surface and the high end stands away from it,
// so the ramp turns round between the themes rather than being flipped.
const ramp = {
  dark: { lo: 0.2, hi: 0.96 },
  light: { lo: 0.97, hi: 0.2 },
}

export function lightness(u: number, dark: boolean): number {
  const r = dark ? ramp.dark : ramp.light
  return r.lo + (r.hi - r.lo) * u
}

/** A cell's fill and the ink that reads on it. */
export function heat(u: number, dark: boolean): { fill: string; ink: string } {
  const l = lightness(u, dark)
  return { fill: `oklch(${l.toFixed(3)} 0 0)`, ink: l > 0.62 ? '#0a0a0a' : '#f5f5f5' }
}

export function rampCss(dark: boolean): string {
  return `linear-gradient(to right, oklch(${lightness(0, dark)} 0 0), oklch(${lightness(1, dark)} 0 0))`
}

/** A quieter span of the same ramp, for fills that carry lines drawn in the text ink. */
export function shade(u: number, dark: boolean): string {
  const l = dark ? 0.14 + 0.4 * u : 0.99 - 0.36 * u
  return `oklch(${l.toFixed(3)} 0 0)`
}

export function finite(v: Num): v is number {
  return v !== null && Number.isFinite(v)
}

export function extent(values: Num[]): [number, number] | null {
  let lo = Infinity
  let hi = -Infinity
  for (const v of values) {
    if (finite(v)) {
      lo = Math.min(lo, v)
      hi = Math.max(hi, v)
    }
  }
  return lo <= hi ? [lo, hi] : null
}

/** True when every point is a number and they run one way without repeating. */
export function monotonic(values: Num[]): boolean {
  if (values.length < 2 || !values.every(finite)) return false
  const v = values as number[]
  const up = v[1] > v[0]
  return v.every((x, i) => i === 0 || (up ? x > v[i - 1] : x < v[i - 1]))
}
