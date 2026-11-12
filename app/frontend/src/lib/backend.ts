// The one place the page talks to Go. Every call is a bound method of App
// (app.go); the generated stubs are cast to the hand-written types.
import * as Go from '../../wailsjs/go/main/App'
import type {
  ConvertJob, ConvertResult, Definition, FileInfo, FixJob, FixResult, LintResult, MapView,
  MergeJob, MergeResult, OutFormat, PickKind, VerifyItem, Versions,
} from './types'

const call = <T>(p: Promise<unknown>) => p as Promise<T>

export const backend = {
  inspectMany: (paths: string[]) => call<FileInfo[]>(Go.InspectMany(paths)),
  openDefinition: (path: string) => call<Definition>(Go.OpenDefinition(path)),
  mapDetail: (path: string, index: number, image: string) => call<MapView>(Go.MapDetail(path, index, image)),
  convert: (job: ConvertJob) => call<ConvertResult>(Go.Convert(job as never)),
  lint: (input: string, image: string, family: string) => call<LintResult>(Go.Lint(input, image, family)),
  fix: (job: FixJob) => call<FixResult>(Go.Fix(job as never)),
  merge: (job: MergeJob) => call<MergeResult>(Go.Merge(job as never)),
  verify: (paths: string[]) => call<VerifyItem[]>(Go.Verify(paths)),
  pickOpen: (title: string, kind: PickKind, multiple = false, dir = '') =>
    call<string[]>(Go.PickOpen({ title, kind, multiple, dir } as never)),
  pickSave: (title: string, path: string, format: OutFormat) => Go.PickSave(title, path, format),
  pickDir: (title: string, dir: string) => Go.PickDir(title, dir),
  suggestOutput: (input: string, format: OutFormat, dir = '', tag = '') => Go.SuggestOutput(input, format, dir, tag),
  exists: (path: string) => Go.Exists(path),
  reveal: (path: string) => Go.Reveal(path),
  startupFiles: () => Go.StartupFiles(),
  version: () => call<Versions>(Go.Version()),
}

/** A failure from Go: its own coded errors arrive as "[[code]] detail". */
export interface Failure {
  code: string
  detail: string
}

export function failure(err: unknown): Failure {
  const text = typeof err === 'string' ? err : err instanceof Error ? err.message : String(err)
  const m = /^\[\[([\w-]+)\]\]\s*(.*)$/s.exec(text)
  return m ? { code: m[1], detail: m[2] } : { code: '', detail: text }
}
