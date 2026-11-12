import { CircleCheck, CircleX, FileCode2, FolderOpen, LoaderCircle, Plus, X } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import banner from '../assets/banner.jpg'
import { FileMeta, FileSlot, formatName } from '../components/FileSlot'
import { Notice, PageHead, Plate, Select, Switch } from '../components/ui'
import { engineMessage, errorMessage } from '../i18n'
import { backend } from '../lib/backend'
import { bytes, dirName, fileName } from '../lib/format'
import type { ConvertResult, FileInfo, OutFormat } from '../lib/types'
import { useApp, useDropZone } from '../state'

const outFormats: OutFormat[] = ['xdf', 'kp', 'json', 'yaml', 'csv']
const origins = ['', 'damos', 'a2l', 'manual', 'auto'] as const
type Origin = (typeof origins)[number]

/** Formats whose definitions can be given an origin when converted (app.go, Convert). */
const takesOrigin = (from: string, to: OutFormat) =>
  (from === 'xdf' || from === 'a2l' || from === 'damos' || from === 'kp') && !(from === 'kp' && to === 'kp')

export function ConvertPage() {
  const { t, def, image, extras, open, removeExtra, clearDef, fail, inspect, notify, refresh } = useApp()
  const [to, setTo] = useState<OutFormat>('xdf')
  const [staticAxes, setStaticAxes] = useState(false)
  const [writeMeta, setWriteMeta] = useState(true)
  const [overwrite, setOverwrite] = useState(false)
  const [origin, setOrigin] = useState<Origin>('')
  const [template, setTemplate] = useState<FileInfo | null>(null)
  const [outDir, setOutDir] = useState('')
  const [savePath, setSavePath] = useState('')
  const [suggested, setSuggested] = useState('')
  const [running, setRunning] = useState(false)
  const [results, setResults] = useState<ConvertResult[]>([])

  const inputs = useMemo(() => (def ? [def, ...extras] : []), [def, extras])
  const single = inputs.length === 1 ? inputs[0] : null

  // A name chosen for one file doesn't carry over to another file or format.
  useEffect(() => setSavePath(''), [single?.path, to])
  useEffect(() => {
    let live = true
    if (!single) {
      setSuggested('')
      return
    }
    backend.suggestOutput(single.path, to, outDir).then(p => live && setSuggested(p), () => live && setSuggested(''))
    return () => {
      live = false
    }
  }, [single, to, outDir])

  const takeTemplate = (files: FileInfo[]) => {
    const f = files.find(x => !x.error && x.kind === 'definition' && x.format === 'kp')
    if (f) setTemplate(f)
    else if (files[0]) notify('error', t('drop.notDefinition', { name: files[0].name }))
  }
  useDropZone('template', takeTemplate)

  if (!def) return <Hero />

  const pickFolder = async () => {
    try {
      const dir = await backend.pickDir(t('convert.chooseFolder'), outDir || def.dir)
      if (dir) {
        setOutDir(dir)
        setSavePath('')
      }
    } catch (err) {
      fail(err)
    }
  }
  const pickSave = async () => {
    if (!single) return
    try {
      const p = await backend.pickSave(t('common.saveAs'), savePath || suggested, to)
      if (p) setSavePath(p)
    } catch (err) {
      fail(err)
    }
  }
  const addFiles = async () => {
    try {
      await open(await backend.pickOpen(t('convert.addFiles'), 'definition', true, def.dir), 'extra')
    } catch (err) {
      fail(err)
    }
  }
  const pickTemplate = async () => {
    try {
      takeTemplate(await inspect(await backend.pickOpen(t('opt.template'), 'kp', false, def.dir)))
    } catch (err) {
      fail(err)
    }
  }

  const run = async () => {
    setRunning(true)
    setResults([])
    const done: ConvertResult[] = []
    for (const f of inputs) {
      try {
        done.push(await backend.convert({
          input: f.path, to,
          output: single ? savePath : '',
          outDir,
          image: image?.path ?? '',
          template: template?.path ?? '',
          tuner: '',
          origin,
          title: '',
          staticAxes: staticAxes && !!image,
          writeMeta,
          overwrite,
        }))
      } catch (err) {
        done.push({ input: f.path, output: '', from: f.format, to, bytes: 0, error: String(err) })
      }
      setResults([...done])
    }
    setRunning(false)
    // A metadata file written next to the open XDF changes how it reads.
    void refresh()
  }

  const allKP = inputs.every(f => f.format === 'kp')
  const showOrigin = inputs.some(f => takesOrigin(f.format, to))
  const showTemplate = to === 'kp' && !allKP

  return (
    <>
      <PageHead title={t('convert.title')} sub={t('convert.subtitle')}>
        {running && <LoaderCircle size={18} className="spin" aria-hidden="true" />}
        <button type="button" className="btn primary" disabled={running} onClick={run}>
          {running ? t('convert.running') : inputs.length > 1 ? t('convert.runMany', { n: inputs.length }) : t('convert.run')}
        </button>
      </PageHead>
      <div className="convert-grid">
        <div className="stack">
          <section>
            <div className="section-label">{t('convert.to')}</div>
            <div className="tiles">
              {outFormats.map(f => (
                <button key={f} type="button" className="tile" aria-pressed={to === f} onClick={() => setTo(f)}>
                  <span className="tile-name">{formatName[f]}</span>
                  <span className="tile-desc">{t(`fmt.${f}.desc`)}</span>
                </button>
              ))}
            </div>
          </section>

          <section>
            <div className="section-label">{t('convert.options')}</div>
            <div className="panel panel-pad">
              {(to === 'xdf' || to === 'csv') && (
                <div className="field">
                  <div className="field-text">
                    <div className="switch-title">{image ? image.name : t('ws.image')}</div>
                    <div className="hint">
                      {image ? t(to === 'xdf' ? 'opt.image.xdf' : 'opt.image.csv') : t('opt.image.none')}
                    </div>
                  </div>
                  <Plate outline>BIN</Plate>
                </div>
              )}
              {to === 'xdf' && (
                <>
                  <Switch checked={staticAxes && !!image} onChange={setStaticAxes} disabled={!image}
                    title={t('opt.staticAxes')}
                    hint={image ? t('opt.staticAxes.hint') : `${t('opt.staticAxes.hint')} (${t('opt.needImage')})`} />
                  <Switch checked={writeMeta} onChange={setWriteMeta} title={t('opt.writeMeta')} hint={t('opt.writeMeta.hint')} />
                </>
              )}
              {showTemplate && (
                <div className="field" style={{ display: 'block' }}>
                  <div className="switch-title">{t('opt.template')} <span className="faint">· {t('common.optional')}</span></div>
                  <div className="hint" style={{ marginBottom: 8 }}>{t('opt.template.hint')}</div>
                  <FileSlot boxed drop="template" label={t('opt.template')} hint="WinOLS KP" icon={<FileCode2 size={18} />}
                    file={template} onPick={pickTemplate} onClear={() => setTemplate(null)} />
                </div>
              )}
              {to === 'kp' && allKP && (
                <div className="field"><div className="hint">{t('convert.sameFormat', { format: 'KP' })}</div></div>
              )}
              {showOrigin && (
                <div className="field">
                  <div className="field-text">
                    <div className="switch-title">{t('opt.origin')}</div>
                    <div className="hint">{t('opt.origin.hint')}</div>
                  </div>
                  <Select label={t('opt.origin')} value={origin} onChange={setOrigin}
                    options={origins.map(o => ({ value: o, label: t(`origin.${o}`) }))} />
                </div>
              )}
              {(to === 'json' || to === 'yaml') && !showOrigin && (
                <div className="field"><div className="hint">{t('convert.noOptions')}</div></div>
              )}
            </div>
          </section>

          <section>
            <div className="section-label">{t('common.saveTo')}</div>
            <div className="panel panel-pad">
              <div className="field">
                <div className="field-text">
                  {single ? (
                    <div className="path">{savePath || suggested || '...'}</div>
                  ) : outDir ? (
                    <div className="path">{outDir}</div>
                  ) : (
                    <div>{t('convert.sameFolder')}</div>
                  )}
                </div>
                {(outDir || savePath) && (
                  <button type="button" className="btn small quiet icon" title={t('convert.sameFolder')} aria-label={t('convert.sameFolder')}
                    onClick={() => { setOutDir(''); setSavePath('') }}>
                    <X size={15} />
                  </button>
                )}
                <button type="button" className="btn small" onClick={pickFolder}>
                  <FolderOpen size={15} aria-hidden="true" />{t('convert.chooseFolder')}
                </button>
                {single && <button type="button" className="btn small" onClick={pickSave}>{t('common.saveAs')}</button>}
              </div>
              <Switch checked={overwrite} onChange={setOverwrite} title={t('common.overwrite')} />
            </div>
          </section>

        </div>

        <div className="stack">
          <section>
            <div className="row" style={{ marginBottom: 10 }}>
              <div className="section-label grow" style={{ margin: 0 }}>{t('convert.files')} · {inputs.length}</div>
              <button type="button" className="btn small" onClick={addFiles}><Plus size={15} aria-hidden="true" />{t('convert.addFiles')}</button>
            </div>
            <div className="panel filelist">
              {inputs.map((f, i) => (
                <div className="fileitem" key={f.path}>
                  <Plate>{formatName[f.format] ?? f.format}</Plate>
                  <div className="grow">
                    <div className="ellipsis" title={f.path} style={{ fontWeight: 500 }}>{f.name}</div>
                    <div className="slot-meta"><FileMeta file={f} /></div>
                  </div>
                  <button type="button" className="btn small quiet icon" aria-label={t('common.remove')} title={t('common.remove')}
                    onClick={() => (i === 0 ? clearDef() : removeExtra(f.path))}>
                    <X size={15} />
                  </button>
                </div>
              ))}
            </div>
          </section>

          {results.length > 0 && (
            <section>
              <div className="section-label">{t('convert.results')}</div>
              <div className="results">
                {results.map(r => <Result key={r.input} r={r} />)}
              </div>
            </section>
          )}
        </div>
      </div>
    </>
  )
}

function Result({ r }: { r: ConvertResult }) {
  const { t, fail } = useApp()
  if (r.error) {
    const m = errorMessage(t, r.error)
    return (
      <div className="result failed">
        <div className="result-head"><CircleX size={16} aria-hidden="true" /><span className="ellipsis">{fileName(r.input)}</span><span className="faint">{t('common.failed')}</span></div>
        <div className="result-body">
          <div>{m.text}</div>
          {m.original && <div className="path">{m.original}</div>}
        </div>
      </div>
    )
  }
  return (
    <div className="result">
      <div className="result-head">
        <CircleCheck size={16} aria-hidden="true" />
        <span className="ellipsis" title={r.output}>{fileName(r.output)}</span>
        <span className="faint mono" style={{ fontWeight: 400 }}>{bytes(r.bytes)}</span>
        <span className="grow" />
        <button type="button" className="btn small" onClick={() => backend.reveal(r.output).catch(fail)}>
          <FolderOpen size={15} aria-hidden="true" />{t('common.reveal')}
        </button>
      </div>
      <div className="result-body">
        <div className="path">{dirName(r.output)}</div>
        {r.metaOutput && <div className="muted">+ {fileName(r.metaOutput)}</div>}
        {(r.warnings ?? []).map((w, i) => {
          const m = engineMessage(t, w)
          return <div className="result-warn" key={i}><span>{m.text}{m.original && <span className="notice-original">{m.original}</span>}</span></div>
        })}
      </div>
    </div>
  )
}

/** The page before any file: what the app does and where to start. */
function Hero() {
  const { t, open, fail } = useApp()
  const pick = async () => {
    try {
      await open(await backend.pickOpen(t('hero.pick'), 'definition', true))
    } catch (err) {
      fail(err)
    }
  }
  return (
    <>
      <img className="hero-banner" src={banner} alt="XDFKIT-TH" />
      <div className="hero">
        <div className="grow">
          <h1 className="hero-title">{t('hero.title')}</h1>
          <p className="hero-body">{t('hero.body')}</p>
        </div>
        <button type="button" className="btn primary" onClick={pick}>{t('hero.pick')}</button>
      </div>
      <div className="formats">
        <div className="panel format-list">
          <div className="section-label">{t('hero.reads')}</div>
          <div className="format-row">
            {['KP', 'XDF', 'A2L', 'DAMOS', 'JSON', 'YAML'].map(f => <Plate key={f} large>{f}</Plate>)}
          </div>
        </div>
        <div className="panel format-list">
          <div className="section-label">{t('hero.writes')}</div>
          <div className="format-row">
            {['XDF', 'KP', 'JSON', 'YAML', 'CSV'].map(f => <Plate key={f} large>{f}</Plate>)}
          </div>
        </div>
      </div>
      <div style={{ marginTop: 14 }}><Notice>{t('drop.meta')}</Notice></div>
    </>
  )
}
