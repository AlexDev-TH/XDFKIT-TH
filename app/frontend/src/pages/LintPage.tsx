import { CircleCheck, FolderOpen, LoaderCircle, Wrench } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Empty, Notice, PageHead, Plate, Select } from '../components/ui'
import { engineMessage, errorMessage, hasKey, type Key, type Message, type T } from '../i18n'
import { backend } from '../lib/backend'
import { fileName, hex } from '../lib/format'
import type { Finding, FixResult, LintResult } from '../lib/types'
import { useApp } from '../state'

type Family = '' | 'me7' | 'm3'

/** What a fix changes, in words. */
function fixText(t: T, f: Finding): string[] {
  const x = f.fix
  if (!x) return []
  const out: string[] = []
  if (x.dataSource !== undefined) out.push(t('fix.dataSource', { v: x.dataSource }))
  if (x.signed !== undefined) out.push(t('fix.signed', { v: t(x.signed ? 'storage.signed' : 'storage.unsigned') }))
  if (x.type !== undefined) out.push(t('fix.type', { v: x.type }))
  if (x.points !== undefined) out.push(t('fix.points', { v: x.points }))
  if (x.addr !== undefined) out.push(t('fix.addr', { v: hex(x.addr) }))
  if (x.start !== undefined) out.push(t('fix.start', { v: hex(x.start) }))
  if (x.end !== undefined) out.push(t('fix.end', { v: hex(x.end) }))
  return out
}

function FindingCard({ f, checked, applied, onToggle }: { f: Finding; checked: boolean; applied: boolean; onToggle: () => void }) {
  const { t } = useApp()
  const ruleKey = `rule.${f.rule}`
  const fixes = fixText(t, f)
  const shown = f.maps.slice(0, 10)
  const conf = f.confidence
  return (
    <div className={checked ? 'finding selected' : 'finding'}>
      <div className="finding-check">
        <input type="checkbox" className="check" checked={checked} disabled={!f.fix || applied} onChange={onToggle}
          aria-label={`${f.rule} ${hex(f.addr)}`} />
      </div>
      <div>
        <div className="finding-head">
          <Plate>{f.rule}</Plate>
          <span className="mono">{hex(f.addr)}</span>
          <span className={conf === 'high' ? 'chip solid' : conf === 'n/a' ? 'chip dashed' : 'chip'}>{t(`conf.${conf}` as Key)}</span>
          {applied && <span className="chip solid"><CircleCheck size={13} aria-hidden="true" />{t('common.done')}</span>}
        </div>
        <div className="finding-title" style={{ marginTop: 6 }}>{hasKey(ruleKey) ? t(ruleKey) : f.rule}</div>
        <div className="finding-msg">{f.message}</div>
        {fixes.length > 0 ? (
          <div className="finding-fix"><span className="faint">{t('lint.proposed')}</span><span>{fixes.join(' · ')}</span></div>
        ) : (
          <div className="finding-fix"><span className="faint">{t('lint.reportOnly')}</span></div>
        )}
        <div className="finding-maps">
          {shown.map((m, i) => (
            <span key={i} className="chip mono" title={m.name}>{m.id || m.name}{m.axis ? ` /${m.axis}` : ''}</span>
          ))}
          {f.maps.length > shown.length && <span className="chip">+{f.maps.length - shown.length}</span>}
        </div>
      </div>
    </div>
  )
}

export function LintPage() {
  const { t, def, image, fail } = useApp()
  const [family, setFamily] = useState<Family>('')
  const [result, setResult] = useState<LintResult | null>(null)
  const [error, setError] = useState<Message | null>(null)
  const [running, setRunning] = useState(false)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [fixed, setFixed] = useState<FixResult | null>(null)
  const [fixing, setFixing] = useState(false)

  // Findings belong to one pair of files.
  useEffect(() => {
    setResult(null)
    setError(null)
    setFixed(null)
    setSelected(new Set())
  }, [def?.path, image?.path])

  const head = <PageHead title={t('lint.title')} sub={t('lint.subtitle')} />
  if (!def) {
    return <>{head}<Empty icon={<Wrench size={30} />} title={t('lint.needDefinition')} /></>
  }
  const lintable = def.format === 'kp' || def.format === 'json' || def.format === 'yaml'

  const run = async () => {
    if (!image) return
    setRunning(true)
    setError(null)
    setFixed(null)
    try {
      const r = await backend.lint(def.path, image.path, family)
      setResult(r)
      setSelected(new Set(r.findings.filter(f => f.fix && f.confidence === 'high').map(f => f.id)))
    } catch (err) {
      setResult(null)
      setError(errorMessage(t, err))
    } finally {
      setRunning(false)
    }
  }

  const fix = async () => {
    if (!image || !result || selected.size === 0) return
    try {
      const suggested = await backend.suggestOutput(def.path, 'kp', '', 'fixed')
      const output = await backend.pickSave(t('common.saveAs'), suggested, 'kp')
      if (!output) return
      setFixing(true)
      // The save dialog has already asked before replacing an existing file.
      const r = await backend.fix({
        input: def.path, image: image.path, family, output,
        only: [...selected], minConfidence: '', overwrite: true, dryRun: false,
      })
      setFixed(r)
    } catch (err) {
      fail(err)
    } finally {
      setFixing(false)
    }
  }

  const fixable = result?.findings.filter(f => f.fix) ?? []
  const applied = new Set(fixed?.applied ?? [])
  const toggle = (id: string) => setSelected(s => {
    const n = new Set(s)
    if (!n.delete(id)) n.add(id)
    return n
  })

  return (
    <>
      {head}
      {!lintable && <Notice tone="warn">{t('lint.wrongFormat', { format: def.format.toUpperCase() })}</Notice>}
      {lintable && !image && <Notice tone="warn">{t('lint.needImage')}</Notice>}
      {lintable && image && (
        <div className="toolbar">
          <span className="muted">{t('lint.family')}</span>
          <Select label={t('lint.family')} value={family} onChange={setFamily}
            options={[
              { value: '', label: image.family ? `${t('lint.family.auto')} (${image.family.toUpperCase()})` : t('lint.family.auto') },
              { value: 'me7', label: t('lint.family.me7') },
              { value: 'm3', label: t('lint.family.m3') },
            ]} />
          <span className="grow" />
          {running && <LoaderCircle size={18} className="spin" aria-hidden="true" />}
          <button type="button" className="btn primary" disabled={running} onClick={run}>{result ? t('lint.rerun') : t('lint.run')}</button>
        </div>
      )}
      {error && <Notice tone="error" message={error} />}
      {(result?.warnings ?? []).map((w, i) => <div key={i} style={{ marginBottom: 8 }}><Notice message={engineMessage(t, w)} /></div>)}

      {fixed && (
        <div style={{ marginBottom: 14 }}>
          <div className="result">
            <div className="result-head">
              <CircleCheck size={16} aria-hidden="true" />
              <span>{t('lint.fixed', { n: fixed.applied.length, m: fixed.remaining.length })}</span>
              <span className="grow" />
              <button type="button" className="btn small" onClick={() => backend.reveal(fixed.output).catch(fail)}>
                <FolderOpen size={15} aria-hidden="true" />{t('common.reveal')}
              </button>
            </div>
            <div className="result-body">
              <div>{t('common.written')}: <strong>{fileName(fixed.output)}</strong></div>
              <div className="path">{fixed.output}</div>
            </div>
          </div>
        </div>
      )}

      {result && result.findings.length === 0 && (
        <Empty icon={<CircleCheck size={30} />} title={t('lint.clean')}>
          <div>{t('lint.clean.body')}</div>
          <div className="faint">{t('lint.detected', { family: result.family.toUpperCase(), source: result.familySource })}</div>
        </Empty>
      )}

      {result && result.findings.length > 0 && (
        <>
          <div className="summary">
            <strong>{t('lint.found', { n: result.findings.length })}</strong>
            <span className="chip">{t('lint.fixable', { n: fixable.length })}</span>
            <span className="chip mono">{t('lint.detected', { family: result.family.toUpperCase(), source: result.familySource })}</span>
            <span className="grow" />
            <button type="button" className="btn small" onClick={() => setSelected(new Set(fixable.filter(f => f.confidence === 'high').map(f => f.id)))}>{t('lint.selectHigh')}</button>
            <button type="button" className="btn small" onClick={() => setSelected(new Set(fixable.map(f => f.id)))}>{t('lint.selectAll')}</button>
            <button type="button" className="btn small" onClick={() => setSelected(new Set())}>{t('lint.selectNone')}</button>
            <button type="button" className="btn primary" disabled={selected.size === 0 || fixing} onClick={fix}>
              {fixing ? t('lint.fixing') : t('lint.fix', { n: selected.size })}
            </button>
          </div>
          <div className="findings">
            {result.findings.map(f => (
              <FindingCard key={f.id} f={f} checked={selected.has(f.id)} applied={applied.has(f.id)} onToggle={() => toggle(f.id)} />
            ))}
          </div>
        </>
      )}
    </>
  )
}
