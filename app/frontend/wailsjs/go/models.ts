export namespace api {
	
	export class DigestCheck {
	    canon: string;
	    digest: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new DigestCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.canon = source["canon"];
	        this.digest = source["digest"];
	        this.status = source["status"];
	    }
	}

}

export namespace lint {
	
	export class AxisRef {
	    index: number;
	    key?: string;
	    id: string;
	    name: string;
	    axis?: string;
	
	    static createFrom(source: any = {}) {
	        return new AxisRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.key = source["key"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.axis = source["axis"];
	    }
	}
	export class Fix {
	    dataSource?: number;
	    signed?: boolean;
	    end?: number;
	    points?: number;
	    addr?: number;
	    start?: number;
	    type?: number;
	
	    static createFrom(source: any = {}) {
	        return new Fix(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dataSource = source["dataSource"];
	        this.signed = source["signed"];
	        this.end = source["end"];
	        this.points = source["points"];
	        this.addr = source["addr"];
	        this.start = source["start"];
	        this.type = source["type"];
	    }
	}
	export class RecordAxis {
	    addr: number;
	    bits: number;
	
	    static createFrom(source: any = {}) {
	        return new RecordAxis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.addr = source["addr"];
	        this.bits = source["bits"];
	    }
	}
	export class Record {
	    start?: number;
	    counts?: number[];
	    axes: RecordAxis[];
	    values?: number;
	
	    static createFrom(source: any = {}) {
	        return new Record(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = source["start"];
	        this.counts = source["counts"];
	        this.axes = this.convertValues(source["axes"], RecordAxis);
	        this.values = source["values"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Finding {
	    id: string;
	    rule: string;
	    confidence: string;
	    message: string;
	    addr: number;
	    points: number;
	    dataSource?: number;
	    type: number;
	    signed: boolean;
	    values?: number[];
	    subtracted?: number[];
	    opposite?: number[];
	    later?: number[];
	    earlier?: number[];
	    narrow?: number[];
	    countByte?: number;
	    record?: Record;
	    cause?: string;
	    end?: number;
	    fix?: Fix;
	    maps: AxisRef[];
	
	    static createFrom(source: any = {}) {
	        return new Finding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.rule = source["rule"];
	        this.confidence = source["confidence"];
	        this.message = source["message"];
	        this.addr = source["addr"];
	        this.points = source["points"];
	        this.dataSource = source["dataSource"];
	        this.type = source["type"];
	        this.signed = source["signed"];
	        this.values = source["values"];
	        this.subtracted = source["subtracted"];
	        this.opposite = source["opposite"];
	        this.later = source["later"];
	        this.earlier = source["earlier"];
	        this.narrow = source["narrow"];
	        this.countByte = source["countByte"];
	        this.record = this.convertValues(source["record"], Record);
	        this.cause = source["cause"];
	        this.end = source["end"];
	        this.fix = this.convertValues(source["fix"], Fix);
	        this.maps = this.convertValues(source["maps"], AxisRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	

}

export namespace main {
	
	export class AxisView {
	    present: boolean;
	    source: string;
	    stored?: string;
	    address?: string;
	    bits?: number;
	    signed?: boolean;
	    float?: boolean;
	    endian?: string;
	    description?: string;
	    units?: string;
	    precision: number;
	    factor: number;
	    offset: number;
	    reciprocal?: boolean;
	    mirror?: boolean;
	    count: number;
	    values: number[];
	    note?: string;
	
	    static createFrom(source: any = {}) {
	        return new AxisView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.present = source["present"];
	        this.source = source["source"];
	        this.stored = source["stored"];
	        this.address = source["address"];
	        this.bits = source["bits"];
	        this.signed = source["signed"];
	        this.float = source["float"];
	        this.endian = source["endian"];
	        this.description = source["description"];
	        this.units = source["units"];
	        this.precision = source["precision"];
	        this.factor = source["factor"];
	        this.offset = source["offset"];
	        this.reciprocal = source["reciprocal"];
	        this.mirror = source["mirror"];
	        this.count = source["count"];
	        this.values = source["values"];
	        this.note = source["note"];
	    }
	}
	export class Category {
	    id: number;
	    name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.count = source["count"];
	    }
	}
	export class ConvertJob {
	    input: string;
	    to: string;
	    output: string;
	    outDir: string;
	    image: string;
	    template: string;
	    tuner: string;
	    origin: string;
	    title: string;
	    staticAxes: boolean;
	    writeMeta: boolean;
	    overwrite: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConvertJob(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input = source["input"];
	        this.to = source["to"];
	        this.output = source["output"];
	        this.outDir = source["outDir"];
	        this.image = source["image"];
	        this.template = source["template"];
	        this.tuner = source["tuner"];
	        this.origin = source["origin"];
	        this.title = source["title"];
	        this.staticAxes = source["staticAxes"];
	        this.writeMeta = source["writeMeta"];
	        this.overwrite = source["overwrite"];
	    }
	}
	export class ConvertResult {
	    input: string;
	    output: string;
	    metaOutput?: string;
	    from: string;
	    to: string;
	    bytes: number;
	    warnings?: string[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ConvertResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input = source["input"];
	        this.output = source["output"];
	        this.metaOutput = source["metaOutput"];
	        this.from = source["from"];
	        this.to = source["to"];
	        this.bytes = source["bytes"];
	        this.warnings = source["warnings"];
	        this.error = source["error"];
	    }
	}
	export class MapRow {
	    index: number;
	    id: string;
	    description: string;
	    comment?: string;
	    categories: string[];
	    shape: string;
	    rows: number;
	    cols: number;
	    address: number;
	    bits: number;
	    signed: boolean;
	    float: boolean;
	    endian: string;
	    units: string;
	    alternates: number;
	
	    static createFrom(source: any = {}) {
	        return new MapRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.id = source["id"];
	        this.description = source["description"];
	        this.comment = source["comment"];
	        this.categories = source["categories"];
	        this.shape = source["shape"];
	        this.rows = source["rows"];
	        this.cols = source["cols"];
	        this.address = source["address"];
	        this.bits = source["bits"];
	        this.signed = source["signed"];
	        this.float = source["float"];
	        this.endian = source["endian"];
	        this.units = source["units"];
	        this.alternates = source["alternates"];
	    }
	}
	export class FileInfo {
	    path: string;
	    name: string;
	    dir: string;
	    size: number;
	    kind: string;
	    format: string;
	    error?: string;
	    objects: number;
	    values: number;
	    curves: number;
	    maps: number;
	    categories: number;
	    project?: string;
	    projectVersion?: string;
	    origin?: string;
	    kpHeader?: number;
	    metaPath?: string;
	    stamp?: string;
	    thai?: number;
	    warnings?: string[];
	    sha256?: string;
	    family?: string;
	    familySource?: string;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.dir = source["dir"];
	        this.size = source["size"];
	        this.kind = source["kind"];
	        this.format = source["format"];
	        this.error = source["error"];
	        this.objects = source["objects"];
	        this.values = source["values"];
	        this.curves = source["curves"];
	        this.maps = source["maps"];
	        this.categories = source["categories"];
	        this.project = source["project"];
	        this.projectVersion = source["projectVersion"];
	        this.origin = source["origin"];
	        this.kpHeader = source["kpHeader"];
	        this.metaPath = source["metaPath"];
	        this.stamp = source["stamp"];
	        this.thai = source["thai"];
	        this.warnings = source["warnings"];
	        this.sha256 = source["sha256"];
	        this.family = source["family"];
	        this.familySource = source["familySource"];
	    }
	}
	export class Definition {
	    info: FileInfo;
	    categories: Category[];
	    rows: MapRow[];
	
	    static createFrom(source: any = {}) {
	        return new Definition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.info = this.convertValues(source["info"], FileInfo);
	        this.categories = this.convertValues(source["categories"], Category);
	        this.rows = this.convertValues(source["rows"], MapRow);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class FixJob {
	    input: string;
	    image: string;
	    family: string;
	    output: string;
	    only: string[];
	    minConfidence: string;
	    overwrite: boolean;
	    dryRun: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FixJob(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input = source["input"];
	        this.image = source["image"];
	        this.family = source["family"];
	        this.output = source["output"];
	        this.only = source["only"];
	        this.minConfidence = source["minConfidence"];
	        this.overwrite = source["overwrite"];
	        this.dryRun = source["dryRun"];
	    }
	}
	export class FixResult {
	    output: string;
	    applied: string[];
	    remaining: lint.Finding[];
	    warnings?: string[];
	    bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new FixResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.output = source["output"];
	        this.applied = source["applied"];
	        this.remaining = this.convertValues(source["remaining"], lint.Finding);
	        this.warnings = source["warnings"];
	        this.bytes = source["bytes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LintResult {
	    family: string;
	    familySource: string;
	    findings: lint.Finding[];
	    warnings?: string[];
	
	    static createFrom(source: any = {}) {
	        return new LintResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.family = source["family"];
	        this.familySource = source["familySource"];
	        this.findings = this.convertValues(source["findings"], lint.Finding);
	        this.warnings = source["warnings"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class MapView {
	    index: number;
	    id: string;
	    key?: string;
	    description: string;
	    comment?: string;
	    categories: string[];
	    shape: string;
	    inverse?: boolean;
	    rows: number;
	    cols: number;
	    address: string;
	    endAddress: string;
	    bits: number;
	    signed: boolean;
	    float: boolean;
	    endian: string;
	    valueDescription?: string;
	    units: string;
	    precision: number;
	    factor: number;
	    offset: number;
	    reciprocal?: boolean;
	    base: number;
	    x: AxisView;
	    y: AxisView;
	    cells: number[][];
	    raw: number[][];
	    note?: string;
	    cellMin: number;
	    cellMax: number;
	    json: string;
	
	    static createFrom(source: any = {}) {
	        return new MapView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.id = source["id"];
	        this.key = source["key"];
	        this.description = source["description"];
	        this.comment = source["comment"];
	        this.categories = source["categories"];
	        this.shape = source["shape"];
	        this.inverse = source["inverse"];
	        this.rows = source["rows"];
	        this.cols = source["cols"];
	        this.address = source["address"];
	        this.endAddress = source["endAddress"];
	        this.bits = source["bits"];
	        this.signed = source["signed"];
	        this.float = source["float"];
	        this.endian = source["endian"];
	        this.valueDescription = source["valueDescription"];
	        this.units = source["units"];
	        this.precision = source["precision"];
	        this.factor = source["factor"];
	        this.offset = source["offset"];
	        this.reciprocal = source["reciprocal"];
	        this.base = source["base"];
	        this.x = this.convertValues(source["x"], AxisView);
	        this.y = this.convertValues(source["y"], AxisView);
	        this.cells = source["cells"];
	        this.raw = source["raw"];
	        this.note = source["note"];
	        this.cellMin = source["cellMin"];
	        this.cellMax = source["cellMax"];
	        this.json = source["json"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MergeJob {
	    base: string;
	    incoming: string;
	    output: string;
	    to: string;
	    smart: boolean;
	    update: boolean;
	    replace: boolean;
	    description: string;
	    comment: string;
	    image: string;
	    staticAxes: boolean;
	    overwrite: boolean;
	    dryRun: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MergeJob(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.base = source["base"];
	        this.incoming = source["incoming"];
	        this.output = source["output"];
	        this.to = source["to"];
	        this.smart = source["smart"];
	        this.update = source["update"];
	        this.replace = source["replace"];
	        this.description = source["description"];
	        this.comment = source["comment"];
	        this.image = source["image"];
	        this.staticAxes = source["staticAxes"];
	        this.overwrite = source["overwrite"];
	        this.dryRun = source["dryRun"];
	    }
	}
	export class MergeResult {
	    output: string;
	    metaOutput?: string;
	    from: string;
	    to: string;
	    actions: model.Action[];
	    warnings?: string[];
	    bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new MergeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.output = source["output"];
	        this.metaOutput = source["metaOutput"];
	        this.from = source["from"];
	        this.to = source["to"];
	        this.actions = this.convertValues(source["actions"], model.Action);
	        this.warnings = source["warnings"];
	        this.bytes = source["bytes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PickRequest {
	    title: string;
	    kind: string;
	    multiple: boolean;
	    dir: string;
	
	    static createFrom(source: any = {}) {
	        return new PickRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.kind = source["kind"];
	        this.multiple = source["multiple"];
	        this.dir = source["dir"];
	    }
	}
	export class VerifyItem {
	    path: string;
	    name: string;
	    status?: string;
	    digests?: api.DigestCheck[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new VerifyItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.digests = this.convertValues(source["digests"], api.DigestCheck);
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Versions {
	    app: string;
	    engine: string;
	
	    static createFrom(source: any = {}) {
	        return new Versions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.app = source["app"];
	        this.engine = source["engine"];
	    }
	}

}

export namespace model {
	
	export class Action {
	    action: string;
	    key?: string;
	    field?: string;
	
	    static createFrom(source: any = {}) {
	        return new Action(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.key = source["key"];
	        this.field = source["field"];
	    }
	}

}

