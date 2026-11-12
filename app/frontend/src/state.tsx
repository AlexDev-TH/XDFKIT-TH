import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode, type RefObject } from 'react'
import { WindowSetDarkTheme, WindowSetLightTheme } from '../wailsjs/runtime/runtime'
import { engineMessage, errorMessage, translator, type Lang, type Message, type T } from './i18n'
import { backend } from './lib/backend'
import type { FileInfo } from './lib/types'

export type ThemePref = 'dark' | 'light' | 'system'
export type Page = 'convert' | 'explore' | 'lint' | 'merge' | 'verify'

export interface Toast {
  id: number
  tone: 'info' | 'error'
  text: string
  detail?: string
}

/** Where a drop landed: a zone a page registered, or anywhere else in the window. */
export type DropHandler = (files: FileInfo[]) => void

interface AppState {
  lang: Lang
  setLang: (l: Lang) => void
  theme: ThemePref
  setTheme: (t: ThemePref) => void
  dark: boolean
  t: T
  page: Page
  setPage: (p: Page) => void

  def: FileInfo | null
  image: FileInfo | null
  /** More definitions queued for the same conversion. */
  extras: FileInfo[]
  busy: boolean
  /** Inspect paths and file them: an image, the definition, further definitions. */
  open: (paths: string[], as?: 'extra') => Promise<void>
  /** Inspect paths without filing them (for a page's own slots). */
  inspect: (paths: string[]) => Promise<FileInfo[]>
  clearDef: () => void
  clearImage: () => void
  removeExtra: (path: string) => void
  /** Re-read the definition after a file next to it changed. */
  refresh: () => Promise<void>

  toasts: Toast[]
  notify: (tone: Toast['tone'], text: string, detail?: string) => void
  dismiss: (id: number) => void
  /** Show a failure from Go in the page's language. */
  fail: (err: unknown) => void

  /** Pages register their own drop zones by name (data-drop="name"). */
  zones: RefObject<Map<string, DropHandler>>
}

const Ctx = createContext<AppState | null>(null)

export function useApp(): AppState {
  const v = useContext(Ctx)
  if (!v) throw new Error('useApp outside AppProvider')
  return v
}

function stored<V extends string>(key: string, allowed: readonly V[], fallback: V): V {
  try {
    const v = localStorage.getItem(key)
    return allowed.includes(v as V) ? (v as V) : fallback
  } catch {
    return fallback
  }
}

function store(key: string, value: string) {
  try {
    localStorage.setItem(key, value)
  } catch {
    // Storage can be unavailable; the choice then lasts for this session only.
  }
}

const systemDark = () => window.matchMedia('(prefers-color-scheme: dark)').matches

// Windows paths name the same file in either case.
const samePath = (a: string, b: string) => a.toLowerCase() === b.toLowerCase()

export function AppProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(() => stored('xdfkit.lang', ['th', 'en'] as const, 'th'))
  const [theme, setThemeState] = useState<ThemePref>(() => stored('xdfkit.theme', ['dark', 'light', 'system'] as const, 'dark'))
  const [sysDark, setSysDark] = useState(systemDark)
  const [page, setPage] = useState<Page>('convert')
  const [def, setDef] = useState<FileInfo | null>(null)
  const [image, setImage] = useState<FileInfo | null>(null)
  const [extras, setExtras] = useState<FileInfo[]>([])
  const [busy, setBusy] = useState(false)
  const [toasts, setToasts] = useState<Toast[]>([])
  const nextToast = useRef(1)
  const zones = useRef(new Map<string, DropHandler>())

  const dark = theme === 'system' ? sysDark : theme === 'dark'
  const t = useMemo(() => translator(lang), [lang])

  useEffect(() => {
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const on = () => setSysDark(mq.matches)
    mq.addEventListener('change', on)
    return () => mq.removeEventListener('change', on)
  }, [])

  useEffect(() => {
    document.documentElement.dataset.theme = dark ? 'dark' : 'light'
    try {
      // The title bar follows the page (main.go sets its colours for both).
      if (dark) WindowSetDarkTheme()
      else WindowSetLightTheme()
    } catch {
      // Outside the app window (a plain browser) there is no title bar to set.
    }
  }, [dark])

  useEffect(() => {
    document.documentElement.lang = lang
  }, [lang])

  const setLang = useCallback((l: Lang) => {
    setLangState(l)
    store('xdfkit.lang', l)
  }, [])
  const setTheme = useCallback((v: ThemePref) => {
    setThemeState(v)
    store('xdfkit.theme', v)
  }, [])

  const dismiss = useCallback((id: number) => setToasts(ts => ts.filter(x => x.id !== id)), [])
  const notify = useCallback((tone: Toast['tone'], text: string, detail?: string) => {
    const id = nextToast.current++
    setToasts(ts => [...ts.slice(-3), { id, tone, text, detail }])
    window.setTimeout(() => dismiss(id), tone === 'error' ? 9000 : 4500)
  }, [dismiss])
  const fail = useCallback((err: unknown) => {
    const m: Message = errorMessage(t, err)
    notify('error', m.text, m.original)
  }, [notify, t])

  const inspect = useCallback(async (paths: string[]) => {
    if (paths.length === 0) return []
    setBusy(true)
    try {
      return await backend.inspectMany(paths)
    } finally {
      setBusy(false)
    }
  }, [])

  const open = useCallback(async (paths: string[], as?: 'extra') => {
    let files: FileInfo[]
    try {
      files = await inspect(paths)
    } catch (err) {
      fail(err)
      return
    }
    const defs: FileInfo[] = []
    for (const f of files) {
      if (f.error) {
        const m = errorMessage(t, f.error)
        notify('error', t('drop.failed', { name: f.name }), m.original ? `${m.text}: ${m.original}` : m.text)
      } else if (f.kind === 'image') {
        setImage(f)
      } else if (f.kind === 'definition') {
        defs.push(f)
      } else if (f.kind === 'meta') {
        notify('info', t('drop.meta'))
      } else {
        notify('error', t('drop.unknown', { name: f.name }))
      }
    }
    if (defs.length === 0) return
    // One entry per file, whatever the spelling of its path.
    const merged = (list: FileInfo[], more: FileInfo[]) => {
      const out = [...list]
      for (const f of more) {
        if (!out.some(x => samePath(x.path, f.path))) out.push(f)
      }
      return out
    }
    if (as === 'extra' && def) {
      setExtras(ex => merged(ex, defs.filter(d => !samePath(d.path, def.path))))
    } else {
      setDef(defs[0])
      setExtras(merged([], defs.slice(1)).filter(d => !samePath(d.path, defs[0].path)))
    }
    for (const d of defs) {
      for (const w of d.warnings ?? []) notify('info', `${d.name}: ${engineMessage(t, w).text}`)
    }
  }, [def, fail, inspect, notify, t])

  const refresh = useCallback(async () => {
    if (!def) return
    try {
      const [f] = await backend.inspectMany([def.path])
      if (f && !f.error) setDef(f)
    } catch {
      // The file may be gone; the loaded summary stays until the user changes it.
    }
  }, [def])

  const value: AppState = {
    lang, setLang, theme, setTheme, dark, t, page, setPage,
    def, image, extras, busy, open, inspect,
    clearDef: () => {
      setDef(extras[0] ?? null)
      setExtras(extras.slice(1))
    },
    clearImage: () => setImage(null),
    removeExtra: path => setExtras(ex => ex.filter(x => x.path !== path)),
    refresh,
    toasts, notify, dismiss, fail, zones,
  }
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

/** Registers a named drop zone for as long as the calling component is mounted. */
export function useDropZone(name: string, handler: DropHandler) {
  const { zones } = useApp()
  const latest = useRef(handler)
  latest.current = handler
  useEffect(() => {
    const map = zones.current
    map.set(name, files => latest.current(files))
    return () => {
      map.delete(name)
    }
  }, [name, zones])
}
