import { useEffect, useState } from 'react'
import { Notice, Plate, Segmented } from '../components/ui'
import type { Key } from '../i18n'
import { num, plain } from '../lib/format'
import type { AxisView, MapView, Num } from '../lib/types'
import { useApp } from '../state'
import { Curve } from './Curve'
import { Heatmap } from './Heatmap'
import { Surface } from './Surface'

type Tab = 'values' | 'chart' | 'info' | 'json'

function storage(t: (k: Key) => string, bits: number, signed: boolean, float: boolean, endian: string): string {
  const parts = [`${bits} bit`]
  if (endian) parts.push(t(endian === 'big' ? 'storage.big' : 'storage.little'))
  parts.push(float ? t('storage.float') : t(signed ? 'storage.signed' : 'storage.unsigned'))
  return parts.join(' · ')
}

/** The conversion as the model defines it: factor × raw + offset, or factor / raw + offset. */
function conversion(factor: Num, offset: Num, reciprocal?: boolean): string {
  const f = plain(factor)
  const o = offset ?? 0
  const tail = o === 0 ? '' : o > 0 ? ` + ${plain(o)}` : ` − ${plain(-o)}`
  return reciprocal ? `${f} / raw${tail}` : `raw × ${f}${tail}`
}

function AxisFacts({ title, axis }: { title: string; axis: AxisView }) {
  const { t } = useApp()
  const source = t(`axis.source.${axis.source}` as Key)
  const stored = axis.source === 'image' && axis.stored ? ` · ${t(`axis.stored.${axis.stored}` as Key)}` : ''
  return (
    <div className="facts-group">
      <div className="section-label">{title} · {t('detail.points', { n: axis.count })}</div>
      <dl className="facts">
        <dt>{t('detail.storage')}</dt>
        <dd>{source}{stored}</dd>
        {axis.address && (
          <>
            <dt>{t('detail.address')}</dt>
            <dd className="mono">{axis.address}{axis.bits ? ` · ${storage(t, axis.bits, !!axis.signed, !!axis.float, axis.endian ?? '')}` : ''}</dd>
          </>
        )}
        {axis.present && (
          <>
            <dt>{t('detail.conversion')}</dt>
            <dd className="mono">{conversion(axis.factor, axis.offset, axis.reciprocal)}</dd>
          </>
        )}
        {(axis.units || axis.description) && (
          <>
            <dt>{t('detail.units')}</dt>
            <dd>{[axis.units, axis.description].filter(Boolean).join(' · ')}</dd>
          </>
        )}
      </dl>
      {axis.values.length > 0 ? (
        <div className="axis-values">{axis.values.map((v, i) => <span key={i}>{num(v, axis.precision)}</span>)}</div>
      ) : axis.note ? (
        <div className="hint" style={{ marginTop: 6 }}>{t(`note.axis.${axis.note}` as Key)}</div>
      ) : null}
    </div>
  )
}

function Info({ view }: { view: MapView }) {
  const { t } = useApp()
  const hasCells = view.cells.length > 0
  return (
    <>
      <div className="facts-group">
        <dl className="facts">
          <dt>{t('detail.address')}</dt>
          <dd className="mono">{view.address} {t('common.to')} {view.endAddress}</dd>
          <dt>{t('detail.cells')}</dt>
          <dd className="mono">{view.cols} × {view.rows}</dd>
          <dt>{t('detail.storage')}</dt>
          <dd>{storage(t, view.bits, view.signed, view.float, view.endian)}</dd>
          <dt>{t('detail.conversion')}</dt>
          <dd className="mono">{conversion(view.factor, view.offset, view.reciprocal)}</dd>
          {(view.units || view.valueDescription) && (
            <>
              <dt>{t('detail.units')}</dt>
              <dd>{[view.units, view.valueDescription].filter(Boolean).join(' · ')}</dd>
            </>
          )}
          {hasCells && (
            <>
              <dt>{t('detail.range')}</dt>
              <dd className="mono">{num(view.cellMin, view.precision)} {t('common.to')} {num(view.cellMax, view.precision)}</dd>
            </>
          )}
          {view.inverse && (
            <>
              <dt>{t('detail.layout')}</dt>
              <dd>{t('detail.layoutInverse')}</dd>
            </>
          )}
          {view.categories.length > 0 && (
            <>
              <dt>{t('detail.category')}</dt>
              <dd>{view.categories.join(' · ')}</dd>
            </>
          )}
          {view.comment && (
            <>
              <dt>{t('detail.comment')}</dt>
              <dd style={{ whiteSpace: 'pre-wrap' }}>{view.comment}</dd>
            </>
          )}
        </dl>
      </div>
      {view.shape !== 'value' && <AxisFacts title={t('detail.axisX')} axis={view.x} />}
      {view.shape === '2d' && <AxisFacts title={t('detail.axisY')} axis={view.y} />}
    </>
  )
}

/** One object in full: its values, its shape drawn, its definition. */
export function Detail({ view }: { view: MapView }) {
  const { t } = useApp()
  const [tab, setTab] = useState<Tab>('values')
  const [raw, setRaw] = useState(false)
  const hasCells = view.cells.length > 0
  const charted = hasCells && view.shape !== 'value' && view.cols > 1 && (view.shape === '1d' || view.rows > 1)
  // A tab the next object doesn't have falls back to its values.
  useEffect(() => {
    if (tab === 'chart' && !charted) setTab('values')
  }, [tab, charted])

  const tabs: { value: Tab; label: string }[] = [{ value: 'values', label: t('detail.values') }]
  if (charted) tabs.push({ value: 'chart', label: t(view.shape === '2d' ? 'detail.surface' : 'detail.curve') })
  tabs.push({ value: 'info', label: t('detail.info') }, { value: 'json', label: t('detail.json') })

  return (
    <>
      <div className="detail-head">
        <div className="row" style={{ alignItems: 'flex-start' }}>
          <div className="grow">
            <div className="detail-id">{view.id || '-'}</div>
            {view.description && <div className="detail-desc">{view.description}</div>}
          </div>
          <Plate>{t(`shape.${view.shape}`)}</Plate>
        </div>
      </div>
      <div className="detail-tabs">
        <Segmented label={view.id} value={tab} onChange={setTab} options={tabs} />
        <span className="grow" />
        {tab === 'values' && hasCells && view.shape !== 'value' && (
          <Segmented label={t('detail.values')} value={raw ? 'raw' : 'phys'} onChange={v => setRaw(v === 'raw')}
            options={[{ value: 'phys', label: t('detail.phys') }, { value: 'raw', label: t('detail.raw') }]} />
        )}
      </div>
      <div className="detail-body">
        {tab === 'values' && (
          hasCells ? (
            view.shape === 'value' ? (
              <>
                <div className="hero-number">
                  <strong>{num(view.cells[0][0], view.precision)}</strong>
                  {view.units && <span>{view.units}</span>}
                </div>
                <dl className="facts">
                  <dt>{t('detail.raw')}</dt>
                  <dd className="mono">{num(view.raw[0][0], view.float ? 4 : 0)}</dd>
                  <dt>{t('detail.conversion')}</dt>
                  <dd className="mono">{conversion(view.factor, view.offset, view.reciprocal)}</dd>
                </dl>
              </>
            ) : (
              <Heatmap view={view} raw={raw} />
            )
          ) : (
            <>
              <Notice tone={view.note === 'needs-image' ? 'info' : 'warn'}>{t(`note.${view.note ?? 'unsupported'}` as Key)}</Notice>
              <div style={{ marginTop: 16 }}><Info view={view} /></div>
            </>
          )
        )}
        {tab === 'chart' && charted && (view.shape === '2d' ? <Surface view={view} /> : <Curve view={view} />)}
        {tab === 'info' && <Info view={view} />}
        {tab === 'json' && <pre className="code">{view.json}</pre>}
      </div>
    </>
  )
}
