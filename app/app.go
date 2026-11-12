package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.nyet.org/xdfkit/api"
	"go.nyet.org/xdfkit/canon"
	"go.nyet.org/xdfkit/kp"
	"go.nyet.org/xdfkit/lint"
	"go.nyet.org/xdfkit/model"
)

// AppVersion is the desktop app's own version; the engine reports api.Version.
var AppVersion = "0.1.0"

// maxFile is the largest file the app reads: definitions and flash images
// are far smaller, so anything bigger was picked by mistake.
const maxFile = 256 << 20

// App is bound to the window: every exported method is callable from the
// page. Methods take and return file paths; the engine works on the bytes.
type App struct {
	ctx context.Context

	mu     sync.Mutex
	defs   map[string]*loadedDef
	images map[string]*loadedImage
}

// NewApp creates the bound application object.
func NewApp() *App {
	return &App{defs: map[string]*loadedDef{}, images: map[string]*loadedImage{}}
}

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// appError is a failure the page words itself: it shows as "[[code]] detail".
// Engine errors pass through as the engine wrote them.
type appError struct{ code, detail string }

func (e appError) Error() string { return "[[" + e.code + "]] " + e.detail }

// fileStamp tells a cached read from a changed file.
type fileStamp struct {
	mod  time.Time
	size int64
}

func stampOf(path string) (fileStamp, error) {
	st, err := os.Stat(path)
	if err != nil {
		return fileStamp{}, err
	}
	if st.IsDir() {
		return fileStamp{}, appError{"is-folder", path}
	}
	if st.Size() > maxFile {
		return fileStamp{}, appError{"too-large", path}
	}
	return fileStamp{st.ModTime(), st.Size()}, nil
}

type loadedDef struct {
	at       fileStamp
	metaAt   fileStamp
	metaPath string
	from     string
	model    *model.Model
	warnings []string
	thai     int // texts read as Thai (thai.go)
}

type loadedImage struct {
	at   fileStamp
	data []byte
}

// metaName is the metadata file next to an XDF: NAME.meta.json for NAME.xdf.
func metaName(xdf string) string {
	return strings.TrimSuffix(xdf, filepath.Ext(xdf)) + ".meta.json"
}

// siblingMeta returns the metadata file next to an XDF, if there is one.
func siblingMeta(path string) (string, fileStamp, []byte) {
	name := metaName(path)
	st, err := stampOf(name)
	if err != nil {
		return "", fileStamp{}, nil
	}
	b, err := os.ReadFile(name)
	if err != nil {
		return "", fileStamp{}, nil
	}
	return name, st, b
}

// loadDef reads a definition file into the model, through the engine's own
// converter, and keeps it until the file changes.
func (a *App) loadDef(path string) (*loadedDef, error) {
	at, err := stampOf(path)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	d := a.defs[path]
	a.mu.Unlock()
	if d != nil && d.at == at {
		// An XDF is read with its metadata file: that one changing, appearing
		// or going away is a change too.
		metaPath, metaAt := "", fileStamp{}
		if st, err := stampOf(metaName(path)); err == nil && d.from == api.XDF {
			metaPath, metaAt = metaName(path), st
		}
		if metaPath == d.metaPath && metaAt == d.metaAt {
			return d, nil
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	d = &loadedDef{at: at, from: api.Detect(data)}
	req := api.ConvertRequest{To: api.JSON, Name: filepath.Base(path)}
	if d.from == api.XDF {
		d.metaPath, d.metaAt, req.Meta = siblingMeta(path)
	}
	out, resp, err := api.Convert(data, req)
	d.warnings = resp.Warnings
	if err != nil {
		return nil, err
	}
	d.model = new(model.Model)
	if err := canon.Unmarshal(out, d.model); err != nil {
		return nil, err
	}
	d.thai = readThai(d.model, d.from)
	if d.from == api.XDF {
		unlistTitles(d.model)
	}
	d.warnings = readableWarnings(d.warnings, d.from)
	a.mu.Lock()
	if len(a.defs) >= 6 {
		a.defs = map[string]*loadedDef{}
	}
	a.defs[path] = d
	a.mu.Unlock()
	return d, nil
}

// loadImage reads a flash image, kept until the file changes.
func (a *App) loadImage(path string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	at, err := stampOf(path)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	im := a.images[path]
	a.mu.Unlock()
	if im != nil && im.at == at {
		return im.data, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	if len(a.images) >= 3 {
		a.images = map[string]*loadedImage{}
	}
	a.images[path] = &loadedImage{at, data}
	a.mu.Unlock()
	return data, nil
}

// FileInfo describes a picked or dropped file: what it is and, for a
// definition, what it holds.
type FileInfo struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Dir    string `json:"dir"`
	Size   int64  `json:"size"`
	Kind   string `json:"kind"`   // definition, image, meta or unknown
	Format string `json:"format"` // kp, xdf, a2l, damos, json, yaml or bin
	Error  string `json:"error,omitempty"`

	// Definitions.
	Objects        int      `json:"objects"`
	Values         int      `json:"values"`
	Curves         int      `json:"curves"`
	Maps           int      `json:"maps"`
	Categories     int      `json:"categories"`
	Project        string   `json:"project,omitempty"`
	ProjectVersion string   `json:"projectVersion,omitempty"`
	Origin         string   `json:"origin,omitempty"`
	KPHeader       int      `json:"kpHeader,omitempty"`
	MetaPath       string   `json:"metaPath,omitempty"`
	Stamp          string   `json:"stamp,omitempty"`
	Thai           int      `json:"thai,omitempty"` // texts read as Thai from Windows-874 bytes
	Warnings       []string `json:"warnings,omitempty"`

	// Images.
	SHA256       string `json:"sha256,omitempty"`
	Family       string `json:"family,omitempty"`
	FamilySource string `json:"familySource,omitempty"`
}

var definitionExts = map[string]bool{".xdf": true, ".kp": true, ".a2l": true, ".dam": true, ".json": true}

var imageExts = map[string]bool{".bin": true, ".ori": true, ".mod": true, ".rom": true, ".dat": true, ".org": true}

// classify names what a file is. The engine detects definitions by content;
// its fallback (YAML) is narrowed here so that a flash image is not read as one.
func classify(path string, data []byte) (kind, format string) {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".meta.json") {
		return "meta", "json"
	}
	switch f := api.Detect(data); f {
	case api.KP, api.XDF, api.A2L, api.DAMOS, api.JSON:
		return "definition", f
	}
	ext := filepath.Ext(lower)
	if ext == ".yaml" || ext == ".yml" {
		return "definition", api.YAML
	}
	if imageExts[ext] {
		return "image", "bin"
	}
	if looksBinary(data) {
		// A file named as a definition that isn't one of the readable formats
		// (some programs save their own binary files as .xdf) is not an image.
		if definitionExts[ext] {
			return "unknown", ""
		}
		return "image", "bin"
	}
	return "definition", api.YAML
}

// looksBinary reports bytes that are not text: a NUL, or invalid UTF-8, near the start.
func looksBinary(data []byte) bool {
	head := data
	if len(head) > 8192 {
		head = head[:8192]
	}
	if bytes.IndexByte(head, 0) >= 0 {
		return true
	}
	// A cut at the limit can split a rune; allow that tail.
	for len(head) > 0 {
		r, n := utf8.DecodeRune(head)
		if r == utf8.RuneError && n == 1 && len(head) > 4 {
			return true
		}
		head = head[n:]
	}
	return false
}

// Inspect reads a file and says what it is.
func (a *App) Inspect(path string) FileInfo {
	info := FileInfo{Path: path, Name: filepath.Base(path), Dir: filepath.Dir(path), Kind: "unknown"}
	at, err := stampOf(path)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	info.Size = at.size
	data, err := os.ReadFile(path)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	info.Kind, info.Format = classify(path, data)
	switch info.Kind {
	case "unknown":
		info.Error = appError{"unreadable-definition", path}.Error()
	case "image":
		sum := sha256.Sum256(data)
		info.SHA256 = hex.EncodeToString(sum[:])
		if fam, src, err := lint.Family(data); err == nil {
			info.Family, info.FamilySource = fam, src
		}
	case "definition":
		d, err := a.loadDef(path)
		if err != nil {
			info.Error = err.Error()
			return info
		}
		info.Warnings = d.warnings
		info.MetaPath = d.metaPath
		info.Thai = d.thai
		m := d.model
		info.Objects = len(m.Objects)
		for _, o := range m.Objects {
			switch o.Shape {
			case "value":
				info.Values++
			case "1d":
				info.Curves++
			default:
				info.Maps++
			}
		}
		info.Categories = len(m.Categories)
		info.Project, info.ProjectVersion = m.Project.Name, m.Project.Version
		if m.Provenance != nil {
			info.Origin = m.Provenance.Origin
		}
		switch info.Format {
		case api.KP:
			if f, err := kp.Parse(data); err == nil {
				info.KPHeader = int(f.HeaderID)
			}
		case api.JSON, api.YAML:
			if v, err := api.Verify(data); err == nil {
				info.Stamp = v.Status
			}
		}
	}
	return info
}

// InspectMany inspects each path in order.
func (a *App) InspectMany(paths []string) []FileInfo {
	out := make([]FileInfo, len(paths))
	for i, p := range paths {
		out[i] = a.Inspect(p)
	}
	return out
}

// Versions are the app's and the engine's versions.
type Versions struct {
	App    string `json:"app"`
	Engine string `json:"engine"`
}

// Version reports the app and engine versions.
func (a *App) Version() Versions { return Versions{AppVersion, api.Version} }

// StartupFiles are the files named on the command line ("Open with").
func (a *App) StartupFiles() []string {
	out := []string{}
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if abs, err := filepath.Abs(arg); err == nil {
			if st, err := os.Stat(abs); err == nil && !st.IsDir() {
				out = append(out, abs)
			}
		}
	}
	return out
}

// PickRequest is a file dialog: Kind selects the file types offered.
type PickRequest struct {
	Title    string `json:"title"`
	Kind     string `json:"kind"` // definition, image, kp, meta, json or any
	Multiple bool   `json:"multiple"`
	Dir      string `json:"dir"`
}

func filters(kind string) []runtime.FileFilter {
	all := runtime.FileFilter{DisplayName: "All files (*.*)", Pattern: "*.*"}
	switch kind {
	case "definition":
		return []runtime.FileFilter{
			{DisplayName: "KP · XDF · A2L · DAMOS · JSON · YAML", Pattern: "*.kp;*.xdf;*.a2l;*.dam;*.json;*.yaml;*.yml"},
			all,
		}
	case "image":
		return []runtime.FileFilter{{DisplayName: "Flash image (*.bin;*.ori;*.mod)", Pattern: "*.bin;*.ori;*.mod;*.rom"}, all}
	case "kp":
		return []runtime.FileFilter{{DisplayName: "WinOLS KP (*.kp)", Pattern: "*.kp"}, all}
	case "meta":
		return []runtime.FileFilter{{DisplayName: "XDF metadata (*.meta.json)", Pattern: "*.meta.json;*.json"}, all}
	case "json":
		return []runtime.FileFilter{{DisplayName: "JSON · YAML", Pattern: "*.json;*.yaml;*.yml"}, all}
	}
	return []runtime.FileFilter{all}
}

func existingDir(dir string) string {
	if st, err := os.Stat(dir); err == nil && st.IsDir() {
		return dir
	}
	return ""
}

// PickOpen shows the open dialog and returns the chosen paths (none when
// cancelled).
func (a *App) PickOpen(req PickRequest) ([]string, error) {
	opts := runtime.OpenDialogOptions{Title: req.Title, Filters: filters(req.Kind), DefaultDirectory: existingDir(req.Dir)}
	if req.Multiple {
		paths, err := runtime.OpenMultipleFilesDialog(a.ctx, opts)
		if paths == nil {
			paths = []string{}
		}
		return paths, err
	}
	p, err := runtime.OpenFileDialog(a.ctx, opts)
	if err != nil || p == "" {
		return []string{}, err
	}
	return []string{p}, nil
}

var extOf = map[string]string{api.XDF: ".xdf", api.KP: ".kp", api.JSON: ".json", api.YAML: ".yaml", api.CSV: ".csv"}

var saveNames = map[string]string{
	api.XDF: "TunerPro XDF (*.xdf)", api.KP: "WinOLS KP (*.kp)", api.JSON: "JSON (*.json)",
	api.YAML: "YAML (*.yaml)", api.CSV: "CSV (*.csv)",
}

// PickSave shows the save dialog for an output of the given format, starting
// at path. It returns "" when cancelled.
func (a *App) PickSave(title, path, format string) (string, error) {
	ext := extOf[format]
	opts := runtime.SaveDialogOptions{
		Title:            title,
		DefaultDirectory: existingDir(filepath.Dir(path)),
		DefaultFilename:  filepath.Base(path),
	}
	if ext != "" {
		opts.Filters = []runtime.FileFilter{{DisplayName: saveNames[format], Pattern: "*" + ext}}
	}
	p, err := runtime.SaveFileDialog(a.ctx, opts)
	if err != nil || p == "" {
		return "", err
	}
	if ext != "" && !strings.EqualFold(filepath.Ext(p), ext) && !(format == api.YAML && strings.EqualFold(filepath.Ext(p), ".yml")) {
		p += ext
	}
	return p, nil
}

// PickDir shows the folder dialog. It returns "" when cancelled.
func (a *App) PickDir(title, dir string) (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: title, DefaultDirectory: existingDir(dir)})
}

// SuggestOutput names the output for input converted to a format: the same
// name with the new extension, in dir (default: next to the input). tag, if
// given, goes before the extension ("name.fixed.kp"); it is also added when
// the name would otherwise be the input itself.
func (a *App) SuggestOutput(input, format, dir, tag string) string {
	if dir == "" {
		dir = filepath.Dir(input)
	}
	base := filepath.Base(input)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	stem = strings.TrimSuffix(stem, ".meta")
	ext := extOf[format]
	if tag != "" {
		stem += "." + tag
	}
	out := filepath.Join(dir, stem+ext)
	if samePath(out, input) {
		out = filepath.Join(dir, stem+".out"+ext)
	}
	return out
}

// Exists reports whether a file is already there.
func (a *App) Exists(path string) bool {
	_, err := os.Stat(path)
	return path != "" && err == nil
}

// Reveal shows a file in Explorer, selected.
func (a *App) Reveal(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	// Explorer exits 1 even when it opened the folder; only a failed start counts.
	return exec.Command("explorer.exe", "/select,", filepath.Clean(path)).Start()
}

func samePath(a, b string) bool {
	x, err1 := filepath.Abs(a)
	y, err2 := filepath.Abs(b)
	return err1 == nil && err2 == nil && strings.EqualFold(filepath.Clean(x), filepath.Clean(y))
}

// writeFiles writes each output, or none: without overwrite, any that exists
// stops the write before the first byte. An output is never an input.
func writeFiles(overwrite bool, inputs []string, outs ...output) error {
	for _, o := range outs {
		for _, in := range inputs {
			if in != "" && samePath(o.path, in) {
				return appError{"same-file", o.path}
			}
		}
		if _, err := os.Stat(o.path); err == nil && !overwrite {
			return appError{"exists", o.path}
		}
	}
	for _, o := range outs {
		if err := os.MkdirAll(filepath.Dir(o.path), 0o755); err != nil {
			return err
		}
		flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
		if overwrite {
			flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		}
		w, err := os.OpenFile(o.path, flags, 0o644)
		if errors.Is(err, os.ErrExist) {
			return appError{"exists", o.path}
		} else if err != nil {
			return err
		}
		if _, err := w.Write(o.data); err != nil {
			w.Close()
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
	}
	return nil
}

type output struct {
	path string
	data []byte
}

func readOptional(path string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	if _, err := stampOf(path); err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

// ConvertJob is one conversion. Options that don't apply to the chosen
// formats are left out, so the page can keep one set of options for a batch.
type ConvertJob struct {
	Input      string `json:"input"`
	To         string `json:"to"`
	Output     string `json:"output"` // default: SuggestOutput
	OutDir     string `json:"outDir"`
	Image      string `json:"image"`
	Template   string `json:"template"`
	Tuner      string `json:"tuner"`
	Origin     string `json:"origin"`
	Title      string `json:"title"`
	StaticAxes bool   `json:"staticAxes"`
	WriteMeta  bool   `json:"writeMeta"`
	Overwrite  bool   `json:"overwrite"`
}

// ConvertResult reports one conversion.
type ConvertResult struct {
	Input      string   `json:"input"`
	Output     string   `json:"output"`
	MetaOutput string   `json:"metaOutput,omitempty"`
	From       string   `json:"from"`
	To         string   `json:"to"`
	Bytes      int      `json:"bytes"`
	Warnings   []string `json:"warnings,omitempty"`
	Error      string   `json:"error,omitempty"`
}

// Convert converts one definition file and writes the result.
func (a *App) Convert(job ConvertJob) ConvertResult {
	res := ConvertResult{Input: job.Input, To: job.To, Output: job.Output}
	fail := func(err error) ConvertResult {
		res.Error = err.Error()
		return res
	}
	if _, err := stampOf(job.Input); err != nil {
		return fail(err)
	}
	data, err := os.ReadFile(job.Input)
	if err != nil {
		return fail(err)
	}
	if kind, _ := classify(job.Input, data); kind != "definition" {
		return fail(appError{"not-definition", job.Input})
	}
	from := api.Detect(data)
	res.From = from
	if res.Output == "" {
		res.Output = a.SuggestOutput(job.Input, job.To, job.OutDir, "")
	}
	req := api.ConvertRequest{To: job.To, Name: filepath.Base(job.Input)}
	if job.To == api.XDF {
		req.StaticAxes = job.StaticAxes
		req.Title = job.Title
		if req.Title == "" {
			base := filepath.Base(res.Output)
			req.Title = strings.TrimSuffix(base, filepath.Ext(base))
		}
	}
	if job.To == api.XDF || job.To == api.CSV {
		if req.Image, err = a.loadImage(job.Image); err != nil {
			return fail(err)
		}
	}
	if job.To == api.KP && from != api.KP {
		if req.Template, err = readOptional(job.Template); err != nil {
			return fail(err)
		}
	}
	if from == api.XDF {
		_, _, req.Meta = siblingMeta(job.Input)
	}
	convertsModel := from != api.KP || job.To != api.KP
	if job.Origin != "" && convertsModel && (from == api.KP || from == api.XDF || from == api.A2L || from == api.DAMOS) {
		req.Origin = job.Origin
	}
	if convertsModel {
		if req.Tuner, err = readOptional(job.Tuner); err != nil {
			return fail(err)
		}
	}
	// A model goes through the app between the engine's read and its write:
	// Thai text is made readable (thai.go) and names are put where the output
	// format's program shows them (names.go). A KP to a KP is re-encoded as it is.
	tunerPro := from == api.XDF && req.Meta == nil
	var out, meta []byte
	if convertsModel {
		out, meta, res.Warnings, err = convertViaModel(data, req, from, tunerPro)
	} else {
		var resp api.ConvertResponse
		out, resp, err = api.Convert(data, req)
		meta, res.Warnings = resp.Meta, resp.Warnings
	}
	res.Warnings = readableWarnings(res.Warnings, from)
	if err != nil {
		return fail(err)
	}
	outs := []output{{res.Output, out}}
	if meta != nil && job.WriteMeta {
		res.MetaOutput = metaName(res.Output)
		outs = append(outs, output{res.MetaOutput, meta})
	}
	if err := writeFiles(job.Overwrite, []string{job.Input, job.Image, job.Template}, outs...); err != nil {
		res.MetaOutput = ""
		return fail(err)
	}
	res.Bytes = len(out)
	return res
}

// LintResult lists what lint found in a KP file checked against its image.
type LintResult struct {
	Family       string         `json:"family"`
	FamilySource string         `json:"familySource"`
	Findings     []lint.Finding `json:"findings"`
	Warnings     []string       `json:"warnings,omitempty"`
}

// Lint checks a KP file (or its model JSON) against the flash image. family
// is me7 or m3, or empty to read it from the image.
func (a *App) Lint(input, image, family string) (LintResult, error) {
	data, img, err := a.pair(input, image)
	if err != nil {
		return LintResult{}, err
	}
	resp, err := api.Lint(data, api.LintRequest{Image: img, Family: family})
	if resp.Findings == nil {
		resp.Findings = []lint.Finding{}
	}
	readableFindings(resp.Findings, api.Detect(data))
	return LintResult{resp.Family, resp.FamilySource, resp.Findings, resp.Warnings}, err
}

func (a *App) pair(input, image string) (data, img []byte, err error) {
	if _, err = stampOf(input); err != nil {
		return nil, nil, err
	}
	if data, err = os.ReadFile(input); err != nil {
		return nil, nil, err
	}
	if image == "" {
		return nil, nil, appError{"no-image", ""}
	}
	img, err = a.loadImage(image)
	return data, img, err
}

// FixJob selects the fixes to apply: the findings named in Only, else every
// fix of at least MinConfidence (default high). DryRun reports without
// writing.
type FixJob struct {
	Input         string   `json:"input"`
	Image         string   `json:"image"`
	Family        string   `json:"family"`
	Output        string   `json:"output"`
	Only          []string `json:"only"`
	MinConfidence string   `json:"minConfidence"`
	Overwrite     bool     `json:"overwrite"`
	DryRun        bool     `json:"dryRun"`
}

// FixResult lists the applied findings and those that remain afterwards.
type FixResult struct {
	Output    string         `json:"output"`
	Applied   []string       `json:"applied"`
	Remaining []lint.Finding `json:"remaining"`
	Warnings  []string       `json:"warnings,omitempty"`
	Bytes     int            `json:"bytes"`
}

// Fix applies lint fixes and writes the corrected file. The engine parses its
// own output back and compares it before anything is written.
func (a *App) Fix(job FixJob) (FixResult, error) {
	data, img, err := a.pair(job.Input, job.Image)
	if err != nil {
		return FixResult{}, err
	}
	to := api.KP
	switch strings.ToLower(filepath.Ext(job.Output)) {
	case ".json":
		to = api.JSON
	case ".yaml", ".yml":
		to = api.YAML
	}
	out, resp, err := api.Fix(data, api.FixRequest{
		LintRequest:   api.LintRequest{Image: img, Family: job.Family},
		Only:          job.Only,
		MinConfidence: job.MinConfidence,
		To:            to,
	})
	res := FixResult{Applied: resp.Applied, Remaining: resp.Remaining, Warnings: resp.Warnings}
	if res.Applied == nil {
		res.Applied = []string{}
	}
	if res.Remaining == nil {
		res.Remaining = []lint.Finding{}
	}
	readableFindings(res.Remaining, api.Detect(data))
	if err != nil || job.DryRun {
		return res, err
	}
	if job.Output == "" {
		return res, appError{"no-output", ""}
	}
	if err := writeFiles(job.Overwrite, []string{job.Input, job.Image}, output{job.Output, out}); err != nil {
		return res, err
	}
	res.Output, res.Bytes = job.Output, len(out)
	return res, nil
}

// MergeJob folds Incoming into Base (docs/design.md, Merge). DryRun reports
// the actions without writing.
type MergeJob struct {
	Base        string `json:"base"`
	Incoming    string `json:"incoming"`
	Output      string `json:"output"`
	To          string `json:"to"`
	Smart       bool   `json:"smart"`
	Update      bool   `json:"update"`
	Replace     bool   `json:"replace"` // the engine's --overwrite: replace a matched map
	Description string `json:"description"`
	Comment     string `json:"comment"`
	Image       string `json:"image"`
	StaticAxes  bool   `json:"staticAxes"`
	Overwrite   bool   `json:"overwrite"` // replace an existing output file
	DryRun      bool   `json:"dryRun"`
}

// MergeResult reports a merge: one action per decision.
type MergeResult struct {
	Output     string         `json:"output"`
	MetaOutput string         `json:"metaOutput,omitempty"`
	From       string         `json:"from"`
	To         string         `json:"to"`
	Actions    []model.Action `json:"actions"`
	Warnings   []string       `json:"warnings,omitempty"`
	Bytes      int            `json:"bytes"`
}

// Merge folds a second definition into a base definition.
func (a *App) Merge(job MergeJob) (MergeResult, error) {
	var res MergeResult
	if _, err := stampOf(job.Base); err != nil {
		return res, err
	}
	base, err := os.ReadFile(job.Base)
	if err != nil {
		return res, err
	}
	req := api.MergeRequest{
		To: job.To, Name: filepath.Base(job.Base),
		Smart: job.Smart, Update: job.Update, Overwrite: job.Replace,
		Description: job.Description, Comment: job.Comment,
	}
	if api.Detect(base) == api.XDF {
		_, _, req.Meta = siblingMeta(job.Base)
	}
	if job.Incoming != "" {
		if req.Incoming, err = readOptional(job.Incoming); err != nil {
			return res, err
		}
		req.IncomingName = filepath.Base(job.Incoming)
		if api.Detect(req.Incoming) == api.XDF {
			_, _, req.IncomingMeta = siblingMeta(job.Incoming)
		}
	}
	if job.To == api.XDF {
		req.StaticAxes = job.StaticAxes
		if req.Image, err = a.loadImage(job.Image); err != nil {
			return res, err
		}
		if job.Output != "" {
			b := filepath.Base(job.Output)
			req.Title = strings.TrimSuffix(b, filepath.Ext(b))
		}
	}
	var out, meta []byte
	var resp api.MergeResponse
	if docs, ok := a.thaiDocs(job.Base, job.Incoming); ok {
		out, meta, resp, err = mergeThai(docs, req)
		resp.From, resp.To = api.Detect(base), job.To
	} else {
		out, resp, err = api.Merge(base, req)
		meta = resp.Meta
	}
	res.From, res.To, res.Actions, res.Warnings = resp.From, resp.To, resp.Actions, resp.Warnings
	if res.Actions == nil {
		res.Actions = []model.Action{}
	}
	if err != nil || job.DryRun {
		return res, err
	}
	if job.Output == "" {
		return res, appError{"no-output", ""}
	}
	outs := []output{{job.Output, out}}
	if meta != nil {
		res.MetaOutput = metaName(job.Output)
		outs = append(outs, output{res.MetaOutput, meta})
	}
	if err := writeFiles(job.Overwrite, []string{job.Base, job.Incoming, job.Image}, outs...); err != nil {
		res.MetaOutput = ""
		return res, err
	}
	res.Output, res.Bytes = job.Output, len(out)
	return res, nil
}

// VerifyItem is the stamp status of one JSON or YAML file.
type VerifyItem struct {
	Path    string            `json:"path"`
	Name    string            `json:"name"`
	Status  string            `json:"status,omitempty"`
	Digests []api.DigestCheck `json:"digests,omitempty"`
	Error   string            `json:"error,omitempty"`
}

// Verify checks the edit stamp of each JSON or YAML file.
func (a *App) Verify(paths []string) []VerifyItem {
	out := make([]VerifyItem, len(paths))
	for i, p := range paths {
		it := VerifyItem{Path: p, Name: filepath.Base(p)}
		data, err := readOptional(p)
		if err == nil {
			if kind, format := classify(p, data); kind == "image" {
				err = appError{"not-definition", p}
			} else if format != api.JSON && format != api.YAML {
				err = appError{"no-stamp-format", fmt.Sprint(format)}
			}
		}
		if err == nil {
			var v api.VerifyResponse
			if v, err = api.Verify(data); err == nil {
				it.Status, it.Digests = v.Status, v.Digests
			}
		}
		if err != nil {
			it.Error = err.Error()
		}
		out[i] = it
	}
	return out
}

// engineModel reads a definition through the engine into the model, with its
// Thai text made readable.
func engineModel(data []byte, req api.ConvertRequest, from string) (*model.Model, []string, error) {
	req.To = api.JSON
	out, resp, err := api.Convert(data, req)
	if err != nil {
		return nil, resp.Warnings, err
	}
	m := new(model.Model)
	if err := canon.Unmarshal(out, m); err != nil {
		return nil, resp.Warnings, err
	}
	readThai(m, from)
	if from == api.XDF {
		unlistTitles(m)
	}
	return m, resp.Warnings, nil
}

// modelDoc is a model as the stamped JSON document the engine reads.
func modelDoc(m *model.Model) ([]byte, error) { return canon.MarshalStamped(m, "xdfkit "+api.Version) }

// thaiLost is the warning for Thai text holding characters that Windows-874
// has no byte for; the page words it.
func thaiLost(n int) string { return fmt.Sprintf("[[thai-lost]] %d", n) }

// writeDoc makes the document the engine writes a format from: for KP and XDF
// the model's Thai text becomes the bytes those formats hold.
func writeDoc(m *model.Model, to string) (doc []byte, warnings []string, err error) {
	if to == api.KP || to == api.XDF {
		if _, lost := writeThai(m, to); lost > 0 {
			warnings = append(warnings, thaiLost(lost))
		}
	}
	doc, err = modelDoc(m)
	return doc, warnings, err
}

// convertViaModel is api.Convert in two steps: the engine reads the
// definition, the app makes its Thai text readable and puts its names where
// the output format keeps them, and the engine writes the result from that
// model. tunerPro says the input is an XDF as TunerPro made it.
func convertViaModel(data []byte, req api.ConvertRequest, from string, tunerPro bool) (out, meta []byte, warnings []string, err error) {
	m, warnings, err := engineModel(data, api.ConvertRequest{Name: req.Name, Meta: req.Meta, Origin: req.Origin, Tuner: req.Tuner}, from)
	if err != nil {
		return nil, nil, warnings, err
	}
	switch {
	case req.To == api.KP && tunerPro:
		tunerProNames(m)
	case req.To == api.XDF && boschNames(m):
		listedTitles(m)
	case req.To == api.XDF:
		plainTitles(m)
	}
	doc, w, err := writeDoc(m, req.To)
	warnings = append(warnings, w...)
	if err != nil {
		return nil, nil, warnings, err
	}
	out, resp, err := api.Convert(doc, api.ConvertRequest{
		To: req.To, Name: req.Name, Template: req.Template, Image: req.Image,
		Refs: req.Refs, Title: req.Title, StaticAxes: req.StaticAxes,
	})
	return out, resp.Meta, append(warnings, resp.Warnings...), err
}

// thaiDocs returns the base and incoming definitions of a merge as model
// documents with readable text, when either holds Thai. Otherwise the engine
// merges the files as they are.
func (a *App) thaiDocs(base, incoming string) (docs [2][]byte, ok bool) {
	thai := false
	for i, path := range []string{base, incoming} {
		if path == "" {
			continue
		}
		d, err := a.loadDef(path)
		if err != nil {
			return docs, false
		}
		thai = thai || d.thai > 0 || modelHasThai(d.model)
		if docs[i], err = modelDoc(d.model); err != nil {
			return docs, false
		}
	}
	return docs, thai
}

// mergeThai is api.Merge over model documents with readable Thai text.
func mergeThai(docs [2][]byte, req api.MergeRequest) (out, meta []byte, resp api.MergeResponse, err error) {
	to := req.To
	if to == "" {
		to = api.JSON
	}
	merge := req
	merge.To, merge.Incoming, merge.Meta, merge.IncomingMeta = api.JSON, docs[1], nil, nil
	merge.Image, merge.Title, merge.StaticAxes = nil, "", false
	out, resp, err = api.Merge(docs[0], merge)
	if err != nil || to == api.JSON {
		return out, nil, resp, err
	}
	m := new(model.Model)
	if err := canon.Unmarshal(out, m); err != nil {
		return nil, nil, resp, err
	}
	doc, w, err := writeDoc(m, to)
	resp.Warnings = append(resp.Warnings, w...)
	if err != nil {
		return nil, nil, resp, err
	}
	out, cr, err := api.Convert(doc, api.ConvertRequest{To: to, Name: req.Name, Image: req.Image, Title: req.Title, StaticAxes: req.StaticAxes})
	resp.Warnings = append(resp.Warnings, cr.Warnings...)
	return out, cr.Meta, resp, err
}
