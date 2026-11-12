import { useVirtualizer } from '@tanstack/react-virtual'
import { LoaderCircle, Search, Table2, X } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Detail } from '../explore/Detail'
import { Empty, Notice, PageHead, Segmented, Select } from '../components/ui'
import { errorMessage, type Message } from '../i18n'
import { backend } from '../lib/backend'
import { count, hex } from '../lib/format'
import type { Definition, MapRow, MapView } from '../lib/types'
import { useApp } from '../state'

type ShapeFilter = 'all' | 'value' | '1d' | '2d'
const ROW = 36

function typeOf(r: MapRow): string {
  if (r.float) return 'f32'
  return `${r.signed ? 's' : 'u'}${r.bits}`
}

export function ExplorePage({ active }: { active: boolean }) {
  const { t, def, image } = useApp()
  const [data, setData] = useState<Definition | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<Message | null>(null)
  const [query, setQuery] = useState('')
  const [shape, setShape] = useState<ShapeFilter>('all')
  const [category, setCategory] = useState('')
  const [selected, setSelected] = useState<number | null>(null)
  const [view, setView] = useState<MapView | null>(null)
  const [viewError, setViewError] = useState<Message | null>(null)
  const scroller = useRef<HTMLDivElement>(null)

  // The list is read the first time the page is shown for a definition, and
  // again whenever that file changes (its size or what it holds).
  const defKey = def ? `${def.path}|${def.size}|${def.objects}|${def.metaPath ?? ''}` : ''
  const loadedKey = useRef('')
  useEffect(() => {
    if (!def) {
      loadedKey.current = ''
      setData(null)
      setSelected(null)
      setView(null)
      return
    }
    if (!active || loadedKey.current === defKey) return
    loadedKey.current = defKey
    let live = true
    setLoading(true)
    setError(null)
    setSelected(null)
    setView(null)
    backend.openDefinition(def.path).then(
      d => live && (setData(d), setLoading(false)),
      err => live && (setData(null), setError(errorMessage(t, err)), setLoading(false)),
    )
    return () => {
      live = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [defKey, active])

  const rows = useMemo(() => {
    if (!data) return []
    const q = query.trim().toLowerCase()
    return data.rows.filter(r => {
      if (shape !== 'all' && r.shape !== shape) return false
      if (category && !r.categories.includes(category)) return false
      if (!q) return true
      return r.id.toLowerCase().includes(q) || r.description.toLowerCase().includes(q)
        || hex(r.address).toLowerCase().includes(q) || r.units.toLowerCase().includes(q)
    })
  }, [data, query, shape, category])

  const virtual = useVirtualizer({ count: rows.length, getScrollElement: () => scroller.current, estimateSize: () => ROW, overscan: 14 })

  const imagePath = image?.path ?? ''
  useEffect(() => {
    if (selected === null || !def) {
      setView(null)
      return
    }
    let live = true
    setViewError(null)
    backend.mapDetail(def.path, selected, imagePath).then(
      v => live && setView(v),
      err => live && (setView(null), setViewError(errorMessage(t, err))),
    )
    return () => {
      live = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selected, def?.path, imagePath, data])

  if (!def) {
    return (
      <>
        <PageHead title={t('explore.title')} sub={t('explore.subtitle')} />
        <Empty icon={<Table2 size={30} />} title={t('explore.needDefinition')} />
      </>
    )
  }

  const move = (delta: number) => {
    if (rows.length === 0) return
    const at = rows.findIndex(r => r.index === selected)
    const next = Math.min(rows.length - 1, Math.max(0, at < 0 ? 0 : at + delta))
    setSelected(rows[next].index)
    virtual.scrollToIndex(next)
  }
  const onKey = (e: React.KeyboardEvent) => {
    const step: Record<string, number> = { ArrowDown: 1, ArrowUp: -1, PageDown: 12, PageUp: -12 }
    if (e.key in step) {
      e.preventDefault()
      move(step[e.key])
    }
  }
  const filtered = query !== '' || shape !== 'all' || category !== ''
  const clear = () => {
    setQuery('')
    setShape('all')
    setCategory('')
  }
  const categories = data?.categories.filter(c => c.count > 0) ?? []

  return (
    <>
      <PageHead title={t('explore.title')} sub={t('explore.subtitle')} />
      {error && <Notice tone="error" message={error} />}
      {loading && <div className="row muted"><LoaderCircle size={16} className="spin" />{t('common.working')}</div>}
      {data && (
        <div className="explore">
          <div className="explore-list">
            <div className="filters">
              <label className="input">
                <Search size={15} aria-hidden="true" />
                <input value={query} onChange={e => setQuery(e.target.value)} placeholder={t('explore.search')} aria-label={t('explore.search')} spellCheck={false} />
                {query && <button type="button" className="btn small quiet icon" onClick={() => setQuery('')} aria-label={t('explore.clear')}><X size={14} /></button>}
              </label>
              <Segmented label={t('col.type')} value={shape} onChange={setShape}
                options={[
                  { value: 'all', label: t('explore.all') },
                  { value: '2d', label: t('shape.2d') },
                  { value: '1d', label: t('shape.1d') },
                  { value: 'value', label: t('shape.value') },
                ]} />
              {categories.length > 0 && (
                <Select label={t('detail.category')} value={category} onChange={setCategory}
                  options={[{ value: '', label: t('explore.allCategories') }, ...categories.map(c => ({ value: c.name, label: `${c.name} (${c.count})` }))]} />
              )}
            </div>
            <div className="table">
              <div className="thead">
                <span>{t('col.name')}</span>
                <span>{t('col.description')}</span>
                <span>{t('col.address')}</span>
                <span>{t('col.size')}</span>
                <span>{t('col.type')}</span>
                <span>{t('col.units')}</span>
              </div>
              <div className="tbody" ref={scroller} onKeyDown={onKey} role="listbox" aria-label={t('explore.title')} tabIndex={-1}>
                {rows.length === 0 ? (
                  <div className="detail-empty" style={{ height: '100%' }}>
                    <div>
                      <div>{t('explore.none')}</div>
                      {filtered && <button type="button" className="btn small" style={{ marginTop: 10 }} onClick={clear}>{t('explore.clear')}</button>}
                    </div>
                  </div>
                ) : (
                  <div style={{ height: virtual.getTotalSize(), position: 'relative' }}>
                    {virtual.getVirtualItems().map(item => {
                      const r = rows[item.index]
                      return (
                        <button key={r.index} type="button" className="trow" role="option" aria-selected={r.index === selected}
                          style={{ top: item.start }} onClick={() => setSelected(r.index)}>
                          <span className="mono" style={{ fontWeight: 500 }}>{r.id || '-'}</span>
                          <span className="muted">{r.description}</span>
                          <span className="mono">{hex(r.address)}</span>
                          <span className="mono">{r.shape === 'value' ? '1' : `${r.cols}×${r.rows}`}</span>
                          <span className="mono muted">{typeOf(r)}</span>
                          <span className="muted">{r.units}</span>
                        </button>
                      )
                    })}
                  </div>
                )}
              </div>
              <div className="table-foot">
                <span>{t('explore.shown', { n: count(rows.length), total: count(data.rows.length) })}</span>
                <span>
                  {t('stat.maps')} {count(data.info.maps)} · {t('stat.curves')} {count(data.info.curves)} · {t('stat.values')} {count(data.info.values)}
                </span>
              </div>
            </div>
          </div>
          <div className="detail">
            {viewError ? (
              <div className="detail-body"><Notice tone="error" message={viewError} /></div>
            ) : view ? (
              <Detail view={view} />
            ) : (
              <div className="detail-empty">{t('explore.pick')}</div>
            )}
          </div>
        </div>
      )}
    </>
  )
}
