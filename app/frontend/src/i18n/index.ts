import { en } from './en'
import { th, type Key } from './th'
import { failure } from '../lib/backend'

export type Lang = 'th' | 'en'
export type { Key }

const dict: Record<Lang, Record<Key, string>> = { th, en }

export type Vars = Record<string, string | number>
export type T = (key: Key, vars?: Vars) => string

export function translator(lang: Lang): T {
  return (key, vars) => {
    const s = dict[lang][key] ?? dict.en[key] ?? key
    return vars ? s.replace(/\{(\w+)\}/g, (_, k: string) => String(vars[k] ?? '')) : s
  }
}

export function hasKey(key: string): key is Key {
  return key in th
}

// The engine words its own messages in English. Those a user meets most often
// are matched here and shown in the page's language; anything else passes
// through as the engine wrote it.
const engineRules: [RegExp, Key][] = [
  [/^template is not the file the model was read from$/, 'engine.template'],
  [/^KP writes the first category of each map$/, 'engine.firstCategory'],
  [/no Bosch identification string in the image/, 'engine.noIdent'],
  [/cells at (0x[0-9a-fA-F]+) run past the image end/, 'engine.pastEnd'],
  [/^XDF stamps can't be verified yet$/, 'engine.xdfStamp'],
  [/subset of the original definition$/, 'engine.subset'],
  [/^stamp (.+)$/, 'engine.stamp'],
  [/^no image$/, 'engine.noImage'],
  [/^\[\[thai-lost\]\] (\d+)$/, 'engine.thaiLost'],
  [/^(\d+) XDFPATCH elements? ignored$/, 'engine.xdfPatch'],
  [/^\d+ (XDF[A-Z]+) elements? ignored$/, 'engine.ignored'],
  [/^\[\[kp-layout\]\] (\d+)$/, 'engine.stride'],
  [/^\[\[kp-header\]\] (\d+)$/, 'engine.strideOld'],
  [/cells start at bit \d+/, 'engine.colMajor'],
  [/only 8, 16 and 32 can be kept/, 'engine.bits'],
  [/is missing or of another size; its labels are used/, 'engine.linkMissing'],
  [/are not evenly spaced; its labels are used/, 'engine.linkUneven'],
  [/the points are read in reverse/, 'engine.reverseAxis'],
  [/^BASEOFFSET /, 'engine.baseOffset'],
  [/equation .*; read as X$/s, 'engine.equation'],
]

export interface Message {
  /** In the page's language. */
  text: string
  /** The engine's own wording, when text is a translation of it. */
  original?: string
}

/** An engine warning or error, in the page's language where it is a known one. */
export function engineMessage(t: T, message: string): Message {
  for (const [re, key] of engineRules) {
    const m = re.exec(message)
    // A message the app made itself has no engine wording to show beside it.
    if (m) return { text: t(key, { v: m[1] ?? '' }), original: message.startsWith('[[') ? undefined : message }
  }
  return { text: message }
}

/** Any failure from Go: the app's coded errors are worded here, the engine's passed through. */
export function errorMessage(t: T, err: unknown): Message {
  const f = failure(err)
  const key = `err.${f.code}`
  if (f.code && hasKey(key)) return { text: t(key), original: f.detail || undefined }
  return engineMessage(t, f.detail)
}
