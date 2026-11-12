import { num } from '../lib/format'
import type { AxisView, MapView, Num } from '../lib/types'
import { useApp } from '../state'
import { extent, finite, heat, rampCss, unit } from './scale'

/** Axis points as headers: the axis's own values, or the cell index without them. */
function headers(axis: AxisView, n: number): string[] {
  if (axis.values.length === n) return axis.values.map(v => num(v, axis.precision))
  return Array.from({ length: n }, (_, i) => String(i))
}

/** The cells as a table, each filled by where its value sits in the map's range. */
export function Heatmap({ view, raw }: { view: MapView; raw: boolean }) {
  const { t, dark } = useApp()
  const cells: Num[][] = raw ? view.raw : view.cells
  const precision = raw ? (view.float ? 4 : 0) : view.precision
  const range = extent(cells.flat())
  const cols = headers(view.x, view.cols)
  const rows = headers(view.y, view.rows)
  const unitsX = view.x.units ? ` ${view.x.units}` : ''
  const unitsY = view.y.units ? ` ${view.y.units}` : ''
  return (
    <>
      <div className="heat-wrap">
        <table className="heat">
          <thead>
            <tr>
              <th scope="col">{view.shape === '2d' ? 'Y \\ X' : 'X'}</th>
              {cols.map((c, i) => <th key={i} scope="col">{c}</th>)}
            </tr>
          </thead>
          <tbody>
            {cells.map((row, r) => (
              <tr key={r}>
                <th scope="row">{view.shape === '2d' ? rows[r] : view.units || t('detail.value')}</th>
                {row.map((v, c) => {
                  const h = finite(v) && range ? heat(unit(v, range[0], range[1]), dark) : null
                  const where = view.shape === '2d' ? `X ${cols[c]}${unitsX} · Y ${rows[r]}${unitsY}` : `X ${cols[c]}${unitsX}`
                  return (
                    <td key={c} style={h ? { background: h.fill, color: h.ink } : undefined}
                      title={`${where}\n${t('detail.phys')} ${num(view.cells[r][c], view.precision)}${view.units ? ' ' + view.units : ''}\n${t('detail.raw')} ${num(view.raw[r][c], view.float ? 4 : 0)}`}>
                      {num(v, precision)}
                    </td>
                  )
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {range && range[1] > range[0] && (
        <div className="heat-legend">
          <span>{num(range[0], precision)}</span>
          <span className="heat-ramp" style={{ background: rampCss(dark) }} />
          <span>{num(range[1], precision)}</span>
          {!raw && view.units && <span className="faint">{view.units}</span>}
        </div>
      )}
    </>
  )
}
