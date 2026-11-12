import { X } from 'lucide-react'
import type { ReactNode } from 'react'
import { bytes, count } from '../lib/format'
import type { FileInfo } from '../lib/types'
import { useApp } from '../state'
import { Plate } from './ui'

export const formatName: Record<string, string> = {
  kp: 'KP', xdf: 'XDF', a2l: 'A2L', damos: 'DAMOS', json: 'JSON', yaml: 'YAML', csv: 'CSV', bin: 'BIN',
}

/** What a loaded file is, in a line: a definition's contents or an image's identity. */
export function FileMeta({ file }: { file: FileInfo }) {
  const { t } = useApp()
  if (file.kind === 'image') {
    return (
      <>
        <span className="mono">{bytes(file.size)}</span>
        <span>{file.family ? t('ws.family', { family: file.family.toUpperCase() }) : t('ws.familyUnknown')}</span>
        {file.familySource && <span className="mono">{file.familySource.replace(/^ident /, '')}</span>}
      </>
    )
  }
  return (
    <>
      <span>{t('ws.objects', { n: count(file.objects) })}</span>
      {file.maps > 0 && <span>{t('stat.maps')} {count(file.maps)}</span>}
      {file.curves > 0 && <span>{t('stat.curves')} {count(file.curves)}</span>}
      {file.values > 0 && <span>{t('stat.values')} {count(file.values)}</span>}
      {file.metaPath && <span>{t('ws.withMeta')}</span>}
      {file.thai ? <span>{t('ws.thai', { n: count(file.thai) })}</span> : null}
      {file.stamp && <span>{t(`status.${file.stamp}` as 'status.clean')}</span>}
    </>
  )
}

/** A place a file goes: empty and asking, or holding one. It is also a drop
 *  zone when given a name the page registered. */
export function FileSlot({ label, hint, icon, file, onPick, onClear, drop, boxed }: {
  label: string
  hint: string
  icon: ReactNode
  file: FileInfo | null
  onPick: () => void
  onClear?: () => void
  drop?: string
  boxed?: boolean
}) {
  const { t } = useApp()
  const cls = ['slot', boxed && 'boxed', !file && 'vacant'].filter(Boolean).join(' ')
  return (
    <div className={cls} data-drop={drop}>
      <div className="slot-icon" aria-hidden="true">{icon}</div>
      <div className="slot-body">
        <div className="slot-label">{label}</div>
        {file ? (
          <>
            <div className="slot-name" title={file.path}>
              <span className="ellipsis">{file.name}</span>
              <Plate>{formatName[file.format] ?? file.format}</Plate>
            </div>
            <div className="slot-meta"><FileMeta file={file} /></div>
          </>
        ) : (
          <div className="slot-meta">{hint}</div>
        )}
      </div>
      <div className="slot-actions">
        <button type="button" className="btn small" onClick={onPick}>
          {file ? t('common.change') : t('common.choose')}
        </button>
        {file && onClear && (
          <button type="button" className="btn small quiet icon" onClick={onClear} aria-label={t('common.remove')} title={t('common.remove')}>
            <X size={15} />
          </button>
        )}
      </div>
    </div>
  )
}
