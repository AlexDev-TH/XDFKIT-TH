import { useState } from 'react'
import { num } from '../lib/format'
import type { MapView } from '../lib/types'
import { useApp } from '../state'
import { extent, finite, monotonic } from './scale'

const W = 560
const H = 280
const M = { top: 16, right: 18, bottom: 46, left: 64 }

/** A curve's cells against its axis: one line, read off by pointing at it. */
export function Curve({ view }: { view: MapView }) {
  const { t } = useApp()
  const [hover, setHover] = useState<number | null>(null)
  const ys = view.cells[0] ?? []
  const n = ys.length
  // The axis's own values place the points when they run one way; otherwise
  // the points are evenly spaced by index.
  const byAxis = view.x.values.length === n && monotonic(view.x.values)
  const xs = byAxis ? (view.x.values as number[]) : ys.map((_, i) => i)
  const xr = extent(xs) ?? [0, 1]
  const yr0 = extent(ys)
  if (!yr0 || n < 2) return null
  const pad = (yr0[1] - yr0[0]) * 0.08 || Math.abs(yr0[0]) * 0.05 || 1
  const yr = [yr0[0] - pad, yr0[1] + pad]
  const px = (x: number) => M.left + ((x - xr[0]) / (xr[1] - xr[0] || 1)) * (W - M.left - M.right)
  const py = (y: number) => H - M.bottom - ((y - yr[0]) / (yr[1] - yr[0])) * (H - M.top - M.bottom)
  const pts = ys.map((y, i) => (finite(y) ? { i, x: px(xs[i]), y: py(y) } : null)).filter(p => p !== null)
  const path = pts.map((p, k) => `${k === 0 ? 'M' : 'L'}${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' ')
  const ticksY = [0, 0.25, 0.5, 0.75, 1].map(u => yr0[0] + (yr0[1] - yr0[0]) * u)
  const ticksX = n <= 2 ? [0, n - 1] : [0, Math.floor((n - 1) / 2), n - 1]
  const xLabel = (i: number) => (view.x.values.length === n ? num(view.x.values[i], view.x.precision) : String(i))

  const onMove = (e: React.PointerEvent<SVGSVGElement>) => {
    const box = e.currentTarget.getBoundingClientRect()
    const x = ((e.clientX - box.left) / box.width) * W
    let best = pts[0]
    for (const p of pts) if (Math.abs(p.x - x) < Math.abs(best.x - x)) best = p
    setHover(best.i)
  }
  const at = hover === null ? null : pts.find(p => p.i === hover) ?? null

  return (
    <div className="chart-wrap">
      <svg className="chart" viewBox={`0 0 ${W} ${H}`} role="img"
        aria-label={`${view.id}: ${t('detail.points', { n })}`}
        onPointerMove={onMove} onPointerLeave={() => setHover(null)}>
        {ticksY.map((v, i) => (
          <g key={i}>
            <line x1={M.left} x2={W - M.right} y1={py(v)} y2={py(v)} stroke="var(--line)" strokeWidth="1" />
            <text x={M.left - 8} y={py(v) + 4} textAnchor="end">{num(v, view.precision)}</text>
          </g>
        ))}
        {ticksX.map(i => (
          <text key={i} x={px(xs[i])} y={H - M.bottom + 19} textAnchor={i === 0 ? 'start' : i === n - 1 ? 'end' : 'middle'}>{xLabel(i)}</text>
        ))}
        {view.x.units && <text x={W - M.right} y={H - 7} textAnchor="end" style={{ fill: 'var(--fg-3)' }}>{view.x.units}</text>}
        {at && <line x1={at.x} x2={at.x} y1={M.top} y2={H - M.bottom} stroke="var(--fg-3)" strokeWidth="1" strokeDasharray="3 3" />}
        <path d={path} fill="none" stroke="var(--fg)" strokeWidth="2" strokeLinejoin="round" strokeLinecap="round" />
        {n <= 48 && pts.map(p => (
          <circle key={p.i} cx={p.x} cy={p.y} r={p.i === hover ? 5 : 3.5} fill="var(--fg)" stroke="var(--bg)" strokeWidth="2" />
        ))}
        {at && n > 48 && <circle cx={at.x} cy={at.y} r="5" fill="var(--fg)" stroke="var(--bg)" strokeWidth="2" />}
      </svg>
      {at && (
        <div className="chart-tip" style={{
          left: `${(at.x / W) * 100}%`, top: `${(at.y / H) * 100}%`,
          transform: `translate(${at.x > W * 0.7 ? 'calc(-100% - 12px)' : '12px'}, -120%)`,
        }}>
          <div>X {xLabel(at.i)}{view.x.units ? ` ${view.x.units}` : ''}</div>
          <div><strong>{num(ys[at.i], view.precision)}</strong>{view.units ? ` ${view.units}` : ''}</div>
        </div>
      )}
    </div>
  )
}
