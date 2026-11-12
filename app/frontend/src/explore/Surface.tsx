import { useMemo, useRef, useState } from 'react'
import { num } from '../lib/format'
import type { AxisView, MapView } from '../lib/types'
import { useApp } from '../state'
import { extent, finite, shade, unit } from './scale'

const W = 560
const H = 400

interface P { x: number; y: number; d: number }

function span(axis: AxisView, n: number, to: string): string {
  if (axis.values.length !== n || n === 0) return `0 ${to} ${n - 1}`
  const u = axis.units ? ` ${axis.units}` : ''
  return `${num(axis.values[0], axis.precision)} ${to} ${num(axis.values[n - 1], axis.precision)}${u}`
}

/** A map drawn as the surface it is: a wireframe, turned by dragging. The
 *  grid is evenly spaced by cell; the table holds the exact values. */
export function Surface({ view }: { view: MapView }) {
  const { t, dark } = useApp()
  const [yaw, setYaw] = useState(-0.62)
  const [pitch, setPitch] = useState(0.5)
  const drag = useRef<{ x: number; y: number; yaw: number; pitch: number } | null>(null)
  const range = extent(view.cells.flat())
  const { rows, cols } = view

  const scene = useMemo(() => {
    if (!range || rows < 2 || cols < 2) return null
    const cy = Math.cos(yaw), sy = Math.sin(yaw), cp = Math.cos(pitch), sp = Math.sin(pitch)
    const scale = 300
    // A grid point (column, row, height), each -0.5 to 0.5, to the screen: turn
    // about the vertical axis, then tip towards the viewer.
    const project = (gx: number, gy: number, gz: number): P => {
      const rx = gx * cy - gy * sy
      const ry = gx * sy + gy * cy
      return { x: W / 2 + rx * scale, y: H / 2 + 18 + (ry * sp - gz * 0.62 * cp) * scale, d: ry * cp + gz * 0.62 * sp }
    }
    const grid: P[][] = view.cells.map((row, r) => row.map((v, c) => {
      const u = finite(v) ? unit(v, range[0], range[1]) : 0
      return project(c / (cols - 1) - 0.5, r / (rows - 1) - 0.5, u - 0.5)
    }))
    const quads: { pts: string; d: number; u: number }[] = []
    for (let r = 0; r < rows - 1; r++) {
      for (let c = 0; c < cols - 1; c++) {
        const corner = [grid[r][c], grid[r][c + 1], grid[r + 1][c + 1], grid[r + 1][c]]
        const vals = [view.cells[r][c], view.cells[r][c + 1], view.cells[r + 1][c + 1], view.cells[r + 1][c]]
        const mean = vals.reduce<number>((s, v) => s + (finite(v) ? v : range[0]), 0) / 4
        quads.push({
          pts: corner.map(p => `${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' '),
          d: (corner[0].d + corner[1].d + corner[2].d + corner[3].d) / 4,
          u: unit(mean, range[0], range[1]),
        })
      }
    }
    // Far faces first, so nearer ones cover them.
    quads.sort((a, b) => a.d - b.d)
    const floor = [[-0.5, -0.5], [0.5, -0.5], [0.5, 0.5], [-0.5, 0.5]].map(([x, y]) => project(x, y, -0.5))
    // The surface's four corners, in the floor's order, to tie it to the floor.
    const corners = [grid[0][0], grid[0][cols - 1], grid[rows - 1][cols - 1], grid[rows - 1][0]]
    // Each axis is labelled on the floor edge nearer the viewer.
    const mid = (a: P, b: P): P => ({ x: (a.x + b.x) / 2, y: (a.y + b.y) / 2, d: (a.d + b.d) / 2 })
    const xEdges = [mid(floor[0], floor[1]), mid(floor[3], floor[2])]
    const yEdges = [mid(floor[0], floor[3]), mid(floor[1], floor[2])]
    const near = (e: P[]) => (e[0].y > e[1].y ? e[0] : e[1])
    return { quads, floor, corners, xAt: near(xEdges), yAt: near(yEdges) }
  }, [view, range, rows, cols, yaw, pitch])

  if (!scene || !range) return null

  const onDown = (e: React.PointerEvent<SVGSVGElement>) => {
    e.currentTarget.setPointerCapture(e.pointerId)
    drag.current = { x: e.clientX, y: e.clientY, yaw, pitch }
  }
  const onMove = (e: React.PointerEvent<SVGSVGElement>) => {
    const d = drag.current
    if (!d) return
    setYaw(d.yaw + (e.clientX - d.x) * 0.008)
    setPitch(Math.min(1.25, Math.max(0.12, d.pitch + (e.clientY - d.y) * 0.006)))
  }
  const onKey = (e: React.KeyboardEvent<SVGSVGElement>) => {
    const step = { ArrowLeft: [-0.1, 0], ArrowRight: [0.1, 0], ArrowUp: [0, -0.06], ArrowDown: [0, 0.06] }[e.key]
    if (!step) return
    e.preventDefault()
    setYaw(y => y + step[0])
    setPitch(p => Math.min(1.25, Math.max(0.12, p + step[1])))
  }
  const label = (p: P) => ({ x: Math.min(W - 120, Math.max(120, p.x)), y: Math.min(H - 8, p.y + 24), textAnchor: 'middle' as const })

  return (
    <div className="chart-wrap">
      <svg className="chart surface" viewBox={`0 0 ${W} ${H}`} role="img" tabIndex={0}
        aria-label={`${view.id}: ${cols} × ${rows}, ${num(range[0], view.precision)} ${t('common.to')} ${num(range[1], view.precision)} ${view.units}`}
        onPointerDown={onDown} onPointerMove={onMove} onPointerUp={() => (drag.current = null)}
        onPointerCancel={() => (drag.current = null)} onKeyDown={onKey}>
        <polygon points={scene.floor.map(p => `${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' ')}
          fill="none" stroke="var(--line-2)" strokeWidth="1" strokeDasharray="4 4" />
        {scene.floor.map((p, i) => (
          <line key={i} x1={p.x} y1={p.y} x2={scene.corners[i].x} y2={scene.corners[i].y}
            stroke="var(--line-2)" strokeWidth="1" strokeDasharray="2 4" />
        ))}
        {scene.quads.map((q, i) => (
          <polygon key={i} points={q.pts} fill={shade(q.u, dark)} stroke="var(--fg)" strokeWidth="1" strokeLinejoin="round" strokeOpacity="0.85" />
        ))}
        <text {...label(scene.xAt)}>X {span(view.x, cols, t('common.to'))}</text>
        <text {...label(scene.yAt)}>Y {span(view.y, rows, t('common.to'))}</text>
        <text x={14} y={24} textAnchor="start">
          {num(range[0], view.precision)} {t('common.to')} {num(range[1], view.precision)}{view.units ? ` ${view.units}` : ''}
        </text>
      </svg>
      <span className="chart-hint">{t('detail.dragHint')}</span>
    </div>
  )
}
