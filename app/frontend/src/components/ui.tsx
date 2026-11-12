import { Info, TriangleAlert, CircleX } from 'lucide-react'
import type { ReactNode } from 'react'
import type { Message } from '../i18n'

/** The slanted plate of the logo: names a format, a rule, a status. */
export function Plate({ children, outline, large }: { children: ReactNode; outline?: boolean; large?: boolean }) {
  const cls = ['plate', outline && 'outline', large && 'large'].filter(Boolean).join(' ')
  return <span className={cls}>{outline ? <span>{children}</span> : children}</span>
}

export function Segmented<V extends string>({ value, options, onChange, label, full }: {
  value: V
  options: { value: V; label: ReactNode; title?: string }[]
  onChange: (v: V) => void
  label: string
  full?: boolean
}) {
  return (
    <div className={full ? 'segmented full' : 'segmented'} role="group" aria-label={label}>
      {options.map(o => (
        <button key={o.value} type="button" aria-pressed={o.value === value} title={o.title} onClick={() => onChange(o.value)}>
          {o.label}
        </button>
      ))}
    </div>
  )
}

export function Switch({ checked, onChange, title, hint, disabled }: {
  checked: boolean
  onChange: (v: boolean) => void
  title: ReactNode
  hint?: ReactNode
  disabled?: boolean
}) {
  return (
    <label className={disabled ? 'switch disabled' : 'switch'}>
      <input type="checkbox" checked={checked} disabled={disabled} onChange={e => onChange(e.target.checked)} />
      <span className="switch-track" aria-hidden="true" />
      <span className="grow">
        <span className="switch-title">{title}</span>
        {hint && <span className="hint" style={{ display: 'block' }}>{hint}</span>}
      </span>
    </label>
  )
}

export function Select<V extends string>({ value, options, onChange, label }: {
  value: V
  options: { value: V; label: string }[]
  onChange: (v: V) => void
  label: string
}) {
  return (
    <select className="select" aria-label={label} value={value} onChange={e => onChange(e.target.value as V)}>
      {options.map(o => (
        <option key={o.value} value={o.value}>{o.label}</option>
      ))}
    </select>
  )
}

export function Notice({ tone = 'info', children, message }: {
  tone?: 'info' | 'warn' | 'error'
  children?: ReactNode
  message?: Message
}) {
  const Icon = tone === 'error' ? CircleX : tone === 'warn' ? TriangleAlert : Info
  return (
    <div className={`notice ${tone}`} role={tone === 'error' ? 'alert' : undefined}>
      <Icon size={16} aria-hidden="true" />
      <div className="grow">
        {children}
        {message && (
          <>
            {message.text}
            {message.original && <span className="notice-original">{message.original}</span>}
          </>
        )}
      </div>
    </div>
  )
}

export function PageHead({ title, sub, children }: { title: string; sub?: string; children?: ReactNode }) {
  return (
    <header className="page-head">
      <div className="grow">
        <h1 className="page-title">{title}</h1>
        {sub && <p className="page-sub">{sub}</p>}
      </div>
      {children}
    </header>
  )
}

export function Empty({ icon, title, children }: { icon?: ReactNode; title: string; children?: ReactNode }) {
  return (
    <div className="empty">
      {icon}
      <div className="empty-title">{title}</div>
      {children}
    </div>
  )
}
