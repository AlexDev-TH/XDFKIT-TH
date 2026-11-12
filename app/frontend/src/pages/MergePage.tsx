import { ChevronDown, ChevronRight, CircleCheck, FileCode2, FolderOpen, GitMerge, LoaderCircle } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { FileSlot, formatName } from '../components/FileSlot'
import { Empty, Notice, PageHead, Segmented, Select, Switch } from '../components/ui'
import { engineMessage, errorMessage, hasKey, type Message } from '../i18n'
import { backend } from '../lib/backend'
import { count, fileName } from '../lib/format'
import type { FileInfo, MergeAction, MergeResult, OutFormat } from '../lib/types'
import { useApp, useDropZone } from '../state'

type Mode = 'keep' | 'update' | 'replace'
type DescPolicy = '' | 'keep' | 'fill' | 'replace'
type CommentPolicy = '' | 'keep' | 'fill' | 'append' | 'replace'
const modes: Mode[] = ['keep', 'update', 'replace']
const mergeFormats: OutFormat[] = ['json', 'yaml', 'kp', 'xdf']

function Report({ actions }: { actions: MergeAction[] }) {
  const { t } = useApp()
  const [open, setOpen] = useState<string | null>(null)
  const groups = useMemo(() => {
    const m = new Map<string, MergeAction[]>()
    for (const a of actions) m.set(a.action, [...(m.get(a.action) ?? []), a])
    return [...m.entries()]
  }, [actions])
  if (groups.length === 0) return <Empty title={t('merge.noActions')} />
  return (
    <div className="report">
      {groups.map(([name, items]) => {
        const key = `action.${name}`
        const expanded = open === name
        return (
          <div className="report-group" key={name}>
            <button type="button" className="report-head" aria-expanded={expanded} onClick={() => setOpen(expanded ? null : name)}>
              {expanded ? <ChevronDown size={15} aria-hidden="true" /> : <ChevronRight size={15} aria-hidden="true" />}
              <span>{hasKey(key) ? t(key) : name}</span>
              <span className="faint mono">{name}</span>
              <span className="report-count">{items.length}</span>
            </button>
            {expanded && (
              <div className="report-items">
                {items.map((a, i) => <div key={i}>{a.key || '-'}{a.field ? `  ·  ${a.field}` : ''}</div>)}
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}

export function MergePage() {
  const { t, def, image, inspect, notify, fail } = useApp()
  const [incoming, setIncoming] = useState<FileInfo | null>(null)
  const [mode, setMode] = useState<Mode>('keep')
  const [smart, setSmart] = useState(false)
  const [description, setDescription] = useState<DescPolicy>('')
  const [comment, setComment] = useState<CommentPolicy>('')
  const [to, setTo] = useState<OutFormat>('json')
  const [running, setRunning] = useState(false)
  const [result, setResult] = useState<MergeResult | null>(null)
  const [error, setError] = useState<Message | null>(null)

  const take = (files: FileInfo[]) => {
    const f = files.find(x => !x.error && x.kind === 'definition')
    if (f) setIncoming(f)
    else if (files[0]) notify('error', t('drop.notDefinition', { name: files[0].name }))
  }
  useDropZone('incoming', take)

  // A report describes one set of inputs and options.
  useEffect(() => {
    setResult(null)
    setError(null)
  }, [def?.path, incoming?.path, mode, smart, description, comment, to])

  if (!def) {
    return (
      <>
        <PageHead title={t('merge.title')} sub={t('merge.subtitle')} />
        <Empty icon={<GitMerge size={30} />} title={t('merge.needBase')} />
      </>
    )
  }

  const pick = async () => {
    try {
      take(await inspect(await backend.pickOpen(t('merge.incoming'), 'definition', false, def.dir)))
    } catch (err) {
      fail(err)
    }
  }

  const run = async (dryRun: boolean) => {
    let output = ''
    try {
      if (!dryRun) {
        const suggested = await backend.suggestOutput(def.path, to, '', 'merged')
        output = await backend.pickSave(t('common.saveAs'), suggested, to)
        if (!output) return
      }
      setRunning(true)
      setError(null)
      const r = await backend.merge({
        base: def.path, incoming: incoming?.path ?? '', output, to,
        smart, update: mode === 'update', replace: mode === 'replace',
        description, comment,
        image: image?.path ?? '', staticAxes: false,
        // The save dialog has already asked before replacing an existing file.
        overwrite: true, dryRun,
      })
      setResult(r)
    } catch (err) {
      setResult(null)
      setError(errorMessage(t, err))
    } finally {
      setRunning(false)
    }
  }

  const ready = !!incoming || smart

  return (
    <>
      <PageHead title={t('merge.title')} sub={t('merge.subtitle')}>
        {running && <LoaderCircle size={18} className="spin" aria-hidden="true" />}
        <button type="button" className="btn" disabled={!ready || running} onClick={() => run(true)}>{t('merge.preview')}</button>
        <button type="button" className="btn primary" disabled={!ready || running} onClick={() => run(false)}>{t('merge.save')}</button>
      </PageHead>
      <div className="merge-grid">
        <div className="stack">
          <section>
            <div className="section-label">{t('merge.base')} + {t('merge.incoming')}</div>
            <div className="stack" style={{ gap: 8 }}>
              <div className="slot boxed">
                <div className="slot-icon" aria-hidden="true"><FileCode2 size={18} /></div>
                <div className="slot-body">
                  <div className="slot-label">{t('merge.base')}</div>
                  <div className="slot-name"><span className="ellipsis" title={def.path}>{def.name}</span></div>
                  <div className="slot-meta">{formatName[def.format]} · {t('ws.objects', { n: count(def.objects) })}</div>
                </div>
              </div>
              <FileSlot boxed drop="incoming" label={t('merge.incoming')} hint={t('merge.incoming.hint')} icon={<GitMerge size={18} />}
                file={incoming} onPick={pick} onClear={() => setIncoming(null)} />
            </div>
          </section>

          <section>
            <div className="section-label">{t('merge.mode')}</div>
            <div className="options" role="radiogroup" aria-label={t('merge.mode')}>
              {modes.map(m => (
                <button key={m} type="button" className="option" role="radio" aria-checked={mode === m} onClick={() => setMode(m)}>
                  <span className="option-mark" aria-hidden="true" />
                  <span className="grow">
                    <span className="switch-title" style={{ display: 'block' }}>{t(`merge.mode.${m}`)}</span>
                    <span className="hint">{t(`merge.mode.${m}.hint`)}</span>
                  </span>
                </button>
              ))}
            </div>
          </section>

          <section>
            <div className="section-label">{t('convert.options')}</div>
            <div className="panel panel-pad">
              <Switch checked={smart} onChange={setSmart} title={t('merge.smart')} hint={t('merge.smart.hint')} />
              <div className="field">
                <div className="field-text"><div className="switch-title">{t('merge.description')}</div></div>
                <Select label={t('merge.description')} value={description} onChange={setDescription}
                  options={(['', 'keep', 'fill', 'replace'] as const).map(p => ({ value: p, label: t(`merge.policy.${p}`) }))} />
              </div>
              <div className="field">
                <div className="field-text"><div className="switch-title">{t('merge.comment')}</div></div>
                <Select label={t('merge.comment')} value={comment} onChange={setComment}
                  options={(['', 'keep', 'fill', 'append', 'replace'] as const).map(p => ({ value: p, label: t(`merge.policy.${p}`) }))} />
              </div>
              <div className="field">
                <div className="field-text"><div className="switch-title">{t('merge.format')}</div></div>
                <Segmented label={t('merge.format')} value={to} onChange={setTo}
                  options={mergeFormats.map(f => ({ value: f, label: formatName[f] }))} />
              </div>
            </div>
          </section>

        </div>

        <div className="stack">
          <section>
            <div className="section-label">{t('merge.report')}</div>
            {error && <Notice tone="error" message={error} />}
            {!result && !error && <Empty title={t('merge.preview')}><div>{t('merge.subtitle')}</div></Empty>}
            {result && (
              <div className="stack" style={{ gap: 10 }}>
                {result.output ? (
                  <div className="result">
                    <div className="result-head">
                      <CircleCheck size={16} aria-hidden="true" />
                      <span className="ellipsis">{fileName(result.output)}</span>
                      <span className="grow" />
                      <button type="button" className="btn small" onClick={() => backend.reveal(result.output).catch(fail)}>
                        <FolderOpen size={15} aria-hidden="true" />{t('common.reveal')}
                      </button>
                    </div>
                    <div className="result-body">
                      <div className="path">{result.output}</div>
                      {result.metaOutput && <div className="muted">+ {fileName(result.metaOutput)}</div>}
                    </div>
                  </div>
                ) : (
                  <Notice>{t('merge.previewNote')}</Notice>
                )}
                {(result.warnings ?? []).map((w, i) => <Notice key={i} tone="warn" message={engineMessage(t, w)} />)}
                <Report actions={result.actions} />
              </div>
            )}
          </section>
        </div>
      </div>
    </>
  )
}
