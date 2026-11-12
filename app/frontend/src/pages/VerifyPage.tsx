import { Fingerprint } from 'lucide-react'
import { useState } from 'react'
import { PageHead, Plate } from '../components/ui'
import { errorMessage, hasKey } from '../i18n'
import { backend } from '../lib/backend'
import type { FileInfo, VerifyItem } from '../lib/types'
import { useApp, useDropZone } from '../state'

export function VerifyPage() {
  const { t, def, fail } = useApp()
  const [items, setItems] = useState<VerifyItem[]>([])

  const add = async (paths: string[]) => {
    if (paths.length === 0) return
    try {
      const fresh = await backend.verify(paths)
      // The newest check of a file replaces its earlier one.
      setItems(old => [...fresh, ...old.filter(o => !fresh.some(f => f.path === o.path))])
    } catch (err) {
      fail(err)
    }
  }
  useDropZone('verify', (files: FileInfo[]) => void add(files.map(f => f.path)))

  const pick = async () => {
    try {
      await add(await backend.pickOpen(t('verify.add'), 'json', true, def?.dir ?? ''))
    } catch (err) {
      fail(err)
    }
  }

  return (
    <>
      <PageHead title={t('verify.title')} sub={t('verify.subtitle')} />
      <div className="dropzone" data-drop="verify">
        <Fingerprint size={30} aria-hidden="true" />
        <div>{t('verify.empty')}</div>
        <div className="row">
          <button type="button" className="btn" onClick={pick}>{t('verify.add')}</button>
          {items.length > 0 && <button type="button" className="btn quiet" onClick={() => setItems([])}>{t('verify.clear')}</button>}
        </div>
      </div>
      {items.length > 0 && (
        <div className="results" style={{ marginTop: 16 }}>
          {items.map(it => {
            const statusKey = `status.${it.status}`
            const m = it.error ? errorMessage(t, it.error) : null
            return (
              <div className={it.error ? 'result failed' : 'result'} key={it.path}>
                <div className="result-head">
                  <span className="ellipsis" title={it.path}>{it.name}</span>
                  <span className="grow" />
                  {it.status && (
                    <>
                      <span>{hasKey(statusKey) ? t(statusKey) : it.status}</span>
                      <Plate outline={it.status !== 'clean'}>{it.status}</Plate>
                    </>
                  )}
                  {it.error && <span className="faint">{t('common.failed')}</span>}
                </div>
                <div className="result-body">
                  <div className="path">{it.path}</div>
                  {m && <div>{m.text}</div>}
                  {(it.digests ?? []).map((d, i) => (
                    <div key={i} className="mono muted" style={{ fontSize: 12 }}>
                      {d.canon} · {d.status} · {d.digest.slice(0, 23)}...
                    </div>
                  ))}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </>
  )
}
