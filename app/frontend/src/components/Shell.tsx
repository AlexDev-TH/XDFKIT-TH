import { ArrowRightLeft, CircleX, Cpu, FileCode2, Fingerprint, GitMerge, Info, Monitor, Moon, Sun, Table2, Wrench, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import banner from '../assets/banner.jpg'
import logo from '../assets/logo.png'
import { backend } from '../lib/backend'
import { count } from '../lib/format'
import type { Versions } from '../lib/types'
import { useApp, type Page } from '../state'
import { FileSlot } from './FileSlot'
import { Segmented } from './ui'

/** The logo, painted in the theme's ink through its own shape. */
export function Logo() {
  return <span className="logo" role="img" aria-label="XDFKIT-TH" style={{ ['--logo' as string]: `url(${logo})` }} />
}

const pages: { id: Page; icon: typeof Table2; key: 'nav.convert' | 'nav.explore' | 'nav.lint' | 'nav.merge' | 'nav.verify' }[] = [
  { id: 'convert', icon: ArrowRightLeft, key: 'nav.convert' },
  { id: 'explore', icon: Table2, key: 'nav.explore' },
  { id: 'lint', icon: Wrench, key: 'nav.lint' },
  { id: 'merge', icon: GitMerge, key: 'nav.merge' },
  { id: 'verify', icon: Fingerprint, key: 'nav.verify' },
]

export function Sidebar({ versions, onAbout }: { versions: Versions | null; onAbout: () => void }) {
  const { t, page, setPage, lang, setLang, theme, setTheme, def } = useApp()
  return (
    <aside className="sidebar">
      <div className="brand"><Logo /></div>
      <nav className="nav">
        {pages.map(p => (
          <button key={p.id} type="button" className="nav-item" aria-current={page === p.id ? 'page' : undefined} onClick={() => setPage(p.id)}>
            <p.icon size={17} aria-hidden="true" />
            <span className="ellipsis">{t(p.key)}</span>
            {p.id === 'explore' && def && <span className="nav-badge">{count(def.objects)}</span>}
          </button>
        ))}
      </nav>
      <div className="sidebar-foot">
        <div className="setting">
          <span className="setting-label">{t('settings.language')}</span>
          <Segmented full label={t('settings.language')} value={lang} onChange={setLang}
            options={[{ value: 'th', label: 'ไทย' }, { value: 'en', label: 'English' }]} />
        </div>
        <div className="setting">
          <span className="setting-label">{t('settings.theme')}</span>
          <Segmented full label={t('settings.theme')} value={theme} onChange={setTheme}
            options={[
              { value: 'dark', label: <Moon size={15} aria-label={t('theme.dark')} />, title: t('theme.dark') },
              { value: 'light', label: <Sun size={15} aria-label={t('theme.light')} />, title: t('theme.light') },
              { value: 'system', label: <Monitor size={15} aria-label={t('theme.system')} />, title: t('theme.system') },
            ]} />
        </div>
        <button type="button" className="version" onClick={onAbout}>
          XDFKIT-TH {versions?.app ?? ''} · {t('nav.about')}
        </button>
      </div>
    </aside>
  )
}

/** The two files every page works on: the definition and, optionally, its image. */
export function WorkspaceBar() {
  const { t, def, image, open, clearDef, clearImage, fail } = useApp()
  const pick = (kind: 'definition' | 'image') => async () => {
    try {
      const dir = (kind === 'definition' ? def?.dir : image?.dir) ?? def?.dir ?? image?.dir ?? ''
      const title = kind === 'definition' ? t('ws.definition') : t('ws.image')
      await open(await backend.pickOpen(title, kind, kind === 'definition', dir))
    } catch (err) {
      fail(err)
    }
  }
  return (
    <div className="workspace">
      <FileSlot label={t('ws.definition')} hint={t('ws.definition.hint')} icon={<FileCode2 size={18} />}
        file={def} onPick={pick('definition')} onClear={clearDef} drop="definition" />
      <FileSlot label={t('ws.image')} hint={t('ws.image.hint')} icon={<Cpu size={18} />}
        file={image} onPick={pick('image')} onClear={clearImage} drop="image" />
    </div>
  )
}

export function Toasts() {
  const { toasts, dismiss, t } = useApp()
  return (
    <div className="toasts" aria-live="polite">
      {toasts.map(x => (
        <div key={x.id} className={`toast ${x.tone}`}>
          {x.tone === 'error' ? <CircleX size={16} aria-hidden="true" /> : <Info size={16} aria-hidden="true" />}
          <div className="grow">
            {x.text}
            {x.detail && <span className="toast-detail">{x.detail}</span>}
          </div>
          <button type="button" onClick={() => dismiss(x.id)} aria-label={t('common.close')}><X size={14} /></button>
        </div>
      ))}
    </div>
  )
}

export function About({ versions, onClose }: { versions: Versions | null; onClose: () => void }) {
  const { t } = useApp()
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onClose()
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])
  return (
    <div className="scrim" onMouseDown={e => e.target === e.currentTarget && onClose()}>
      <div className="dialog" role="dialog" aria-modal="true" aria-label={t('nav.about')}>
        <img className="dialog-banner" src={banner} alt="XDFKIT-TH" />
        <div className="dialog-body">
          <p className="selectable">{t('about.body')}</p>
          <dl className="facts" style={{ marginTop: 14 }}>
            <dt>{t('about.author')}</dt>
            <dd>AlexDev-TH</dd>
            <dt>{t('about.app')}</dt>
            <dd className="mono">XDFKIT-TH {versions?.app}</dd>
            <dt>{t('about.engine')}</dt>
            <dd className="mono">{versions?.engine}</dd>
            <dt>{t('about.source')}</dt>
            <dd className="mono">github.com/nyetlabs/xdfkit</dd>
            <dt>{t('about.license')}</dt>
            <dd className="mono">LGPL-3.0-or-later</dd>
          </dl>
          <div className="dialog-foot">
            <button type="button" className="btn" onClick={onClose} autoFocus>{t('common.close')}</button>
          </div>
        </div>
      </div>
    </div>
  )
}

/** The app's and the engine's versions, read once. */
export function useVersions(): Versions | null {
  const [v, setV] = useState<Versions | null>(null)
  useEffect(() => {
    backend.version().then(setV, () => setV(null))
  }, [])
  return v
}
