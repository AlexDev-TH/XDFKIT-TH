// The shapes the Go side sends (app.go, explore.go). Written by hand rather
// than taken from wailsjs/go/models.ts: that file types the engine's enums as
// numbers, while they travel as their names.

export type Format = 'kp' | 'xdf' | 'a2l' | 'damos' | 'json' | 'yaml' | 'csv' | 'bin'
export type OutFormat = 'xdf' | 'kp' | 'json' | 'yaml' | 'csv'
export type Shape = 'value' | '1d' | '2d'

export interface FileInfo {
  path: string
  name: string
  dir: string
  size: number
  kind: 'definition' | 'image' | 'meta' | 'unknown'
  format: Format | ''
  error?: string
  objects: number
  values: number
  curves: number
  maps: number
  categories: number
  project?: string
  projectVersion?: string
  origin?: string
  kpHeader?: number
  metaPath?: string
  stamp?: string
  thai?: number
  warnings?: string[]
  sha256?: string
  family?: string
  familySource?: string
}

export interface Category {
  id: number
  name: string
  count: number
}

export interface MapRow {
  index: number
  id: string
  description: string
  comment?: string
  categories: string[]
  shape: Shape
  rows: number
  cols: number
  address: number
  bits: number
  signed: boolean
  float: boolean
  endian: string
  units: string
  alternates: number
}

export interface Definition {
  info: FileInfo
  categories: Category[]
  rows: MapRow[]
}

export type Num = number | null

export interface AxisView {
  present: boolean
  source: string
  stored?: string
  address?: string
  bits?: number
  signed?: boolean
  float?: boolean
  endian?: string
  description?: string
  units?: string
  precision: number
  factor: Num
  offset: Num
  reciprocal?: boolean
  mirror?: boolean
  count: number
  values: Num[]
  note?: string
}

export interface MapView {
  index: number
  id: string
  key?: string
  description: string
  comment?: string
  categories: string[]
  shape: Shape
  inverse?: boolean
  rows: number
  cols: number
  address: string
  endAddress: string
  bits: number
  signed: boolean
  float: boolean
  endian: string
  valueDescription?: string
  units: string
  precision: number
  factor: Num
  offset: Num
  reciprocal?: boolean
  base: number
  x: AxisView
  y: AxisView
  cells: Num[][]
  raw: Num[][]
  note?: string
  cellMin: Num
  cellMax: Num
  json: string
}

export interface ConvertJob {
  input: string
  to: OutFormat
  output: string
  outDir: string
  image: string
  template: string
  tuner: string
  origin: string
  title: string
  staticAxes: boolean
  writeMeta: boolean
  overwrite: boolean
}

export interface ConvertResult {
  input: string
  output: string
  metaOutput?: string
  from: string
  to: string
  bytes: number
  warnings?: string[]
  error?: string
}

export interface AxisRef {
  index: number
  key?: string
  id: string
  name: string
  axis?: string
}

export interface FindingFix {
  dataSource?: string
  signed?: boolean
  end?: number
  points?: number
  addr?: number
  start?: number
  type?: string
}

export interface Finding {
  id: string
  rule: string
  confidence: 'high' | 'medium' | 'low' | 'n/a'
  message: string
  addr: number
  points: number
  dataSource?: string
  type: string
  signed: boolean
  values?: number[]
  subtracted?: number[]
  opposite?: number[]
  cause?: string
  fix?: FindingFix
  maps: AxisRef[]
}

export interface LintResult {
  family: string
  familySource: string
  findings: Finding[]
  warnings?: string[]
}

export interface FixJob {
  input: string
  image: string
  family: string
  output: string
  only: string[]
  minConfidence: string
  overwrite: boolean
  dryRun: boolean
}

export interface FixResult {
  output: string
  applied: string[]
  remaining: Finding[]
  warnings?: string[]
  bytes: number
}

export interface MergeJob {
  base: string
  incoming: string
  output: string
  to: OutFormat
  smart: boolean
  update: boolean
  replace: boolean
  description: string
  comment: string
  image: string
  staticAxes: boolean
  overwrite: boolean
  dryRun: boolean
}

export interface MergeAction {
  action: string
  key?: string
  field?: string
}

export interface MergeResult {
  output: string
  metaOutput?: string
  from: string
  to: string
  actions: MergeAction[]
  warnings?: string[]
  bytes: number
}

export interface VerifyItem {
  path: string
  name: string
  status?: string
  digests?: { canon: string; digest: string; status: string }[]
  error?: string
}

export interface Versions {
  app: string
  engine: string
}

export type PickKind = 'definition' | 'image' | 'kp' | 'meta' | 'json' | 'any'
