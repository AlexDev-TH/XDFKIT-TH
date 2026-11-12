package main

import (
	"cmp"
	"encoding/csv"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"go.nyet.org/xdfkit/api"
	"go.nyet.org/xdfkit/model"
	"go.nyet.org/xdfkit/xdf"
)

// The one pack in the engine's archive that comes with its flash image.
const (
	packKP  = "../testdata/archive/ecuxplot/8N0906018CB.kp"
	packBin = "../testdata/archive/ecuxplot/8N0906018CB.bin"
)

func read(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestCellsMatchEngine: the cells the map view reads from the image have the
// raw and converted range the engine's own CSV writer reports for every
// object of a real pack.
func TestCellsMatchEngine(t *testing.T) {
	a := NewApp()
	d, err := a.loadDef(packKP)
	if err != nil {
		t.Fatal(err)
	}
	img := read(t, packBin)
	out, _, err := api.Convert(read(t, packKP), api.ConvertRequest{To: api.CSV, Name: "8N0906018CB.kp", Image: img})
	if err != nil {
		t.Fatal(err)
	}
	r := csv.NewReader(strings.NewReader(string(out)))
	r.FieldsPerRecord = -1 // rows end with the comment, which the header doesn't name
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	col := map[string]int{}
	for i, h := range rows[0] {
		col[h] = i
	}
	// The CSV lists objects by address, in source order at the same address.
	objs := slices.SortedStableFunc(slices.Values(d.model.Objects), func(x, y *model.Object) int {
		return cmp.Compare(x.Address, y.Address)
	})
	if len(objs) != len(rows)-1 {
		t.Fatalf("%d objects, %d CSV rows", len(objs), len(rows)-1)
	}
	checked := 0
	for i, o := range objs {
		row := rows[i+1]
		cells, raw, note := readCells(o, img)
		if row[col["Value min"]] == "-" {
			if cells != nil {
				t.Errorf("%s: cells read, CSV has none", o.ID)
			}
			continue
		}
		if cells == nil {
			t.Errorf("%s: no cells (%s), CSV has %s", o.ID, note, row[col["Value min"]])
			continue
		}
		lo, hi := slices.Min(raw), slices.Max(raw)
		c := o.Value.Conversion
		wantLo, wantHi := math.Min(c.Apply(lo), c.Apply(hi)), math.Max(c.Apply(lo), c.Apply(hi))
		for name, want := range map[string]float64{"Value min": wantLo, "Value max": wantHi} {
			got, err := strconv.ParseFloat(row[col[name]], 64)
			if err != nil {
				t.Fatalf("%s: %s %q: %v", o.ID, name, row[col[name]], err)
			}
			if got != want && !(math.IsNaN(got) && math.IsNaN(want)) {
				t.Errorf("%s: %s = %v, engine CSV has %v", o.ID, name, want, got)
			}
		}
		if !o.Data.Float {
			mask := uint64(1)<<o.Data.Bits - 1
			for name, v := range map[string]float64{"Value min*1": lo, "Value max*1": hi} {
				if got := "0x" + strconv.FormatUint(uint64(int64(v))&mask, 16); got != row[col[name]] {
					t.Errorf("%s: %s = %s, engine CSV has %s", o.ID, name, got, row[col[name]])
				}
			}
		}
		checked++
	}
	if checked < 100 {
		t.Fatalf("only %d objects compared", checked)
	}
}

var labelRe = regexp.MustCompile(`<LABEL index="(\d+)" value="([^"]*)"`)

// TestAxisPointsMatchEngine: the points shown for summed and reversed image
// axes are the labels the engine's XDF writer makes from the same image.
func TestAxisPointsMatchEngine(t *testing.T) {
	img := make([]byte, 64)
	copy(img[0x10:], []byte{20, 30, 40, 50, 16})                 // 8-bit points
	copy(img[0x20:], []byte{0x10, 0x00, 0x20, 0x00, 0x00, 0x7F}) // 16-bit little-endian points
	for _, tc := range []struct {
		name   string
		stored string
		addr   model.Addr
		data   model.Data
		n      int
		conv   model.Conversion
	}{
		{"subtract 8", "subtract", 0x10, model.Data{Bits: 8}, 5, model.Conversion{Factor: 40}},
		{"add 8", "add", 0x10, model.Data{Bits: 8}, 5, model.Conversion{Factor: 0.5, Offset: 3}},
		{"backwards 8", "backwards", 0x10, model.Data{Bits: 8}, 5, model.Conversion{Factor: 1}},
		{"subtract 16", "subtract", 0x20, model.Data{Bits: 16, Endian: "little"}, 3, model.Conversion{Factor: 0.25}},
		{"add 16 signed", "add", 0x20, model.Data{Bits: 16, Endian: "little", Signed: true}, 3, model.Conversion{Factor: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			addr, data := tc.addr, tc.data
			axis := &model.Axis{Source: "image", Stored: tc.stored, Address: &addr, Data: &data,
				Value: model.Value{Precision: 4, Conversion: tc.conv}}
			m := &model.Model{Schema: model.SchemaID, Objects: []*model.Object{{
				ID: "KF", Shape: "1d", Address: 0x30, Rows: 1, Cols: tc.n,
				Data: model.Data{Bits: 8}, Value: model.Value{Conversion: model.Conversion{Factor: 1}}, X: axis,
			}}}
			out, err := xdf.WriteLabels(m, img, "t")
			if err != nil {
				t.Fatal(err)
			}
			// Only the x axis: the y axis of a curve is one ordinal label.
			_, xAxis, _ := strings.Cut(string(out), `<XDFAXIS id="x"`)
			xAxis, _, _ = strings.Cut(xAxis, "</XDFAXIS>")
			labels := labelRe.FindAllStringSubmatch(xAxis, -1)
			if len(labels) != tc.n {
				t.Fatalf("%d labels on the XDF x axis, want %d:\n%s", len(labels), tc.n, xAxis)
			}
			pts, note := axisPoints(axis, tc.n, img)
			if note != "" || len(pts) != tc.n {
				t.Fatalf("points %v, note %q", pts, note)
			}
			for i, l := range labels {
				want, err := strconv.ParseFloat(l[2], 64)
				if err != nil {
					t.Fatal(err)
				}
				// A label is rounded to the decimals the writer allows.
				_, frac, _ := strings.Cut(l[2], ".")
				if tol := 0.5*math.Pow(10, -float64(len(frac))) + 1e-9; math.Abs(pts[i]-want) > tol {
					t.Errorf("point %d = %v, engine label %s", i, pts[i], l[2])
				}
			}
		})
	}
}

func TestAxisPointsWithoutImage(t *testing.T) {
	// Fixed values are display units; an ordinal axis converts 0, 1, 2.
	if pts, note := axisPoints(&model.Axis{Source: "editable", Values: []float64{1000, 2000, 3000},
		Value: model.Value{Conversion: model.Conversion{Factor: 1}}}, 3, nil); note != "" || !slices.Equal(pts, []float64{1000, 2000, 3000}) {
		t.Errorf("editable: %v %q", pts, note)
	}
	if pts, note := axisPoints(&model.Axis{Source: "ordinal",
		Value: model.Value{Conversion: model.Conversion{Factor: 2, Offset: 1}}}, 3, nil); note != "" || !slices.Equal(pts, []float64{1, 3, 5}) {
		t.Errorf("ordinal: %v %q", pts, note)
	}
	if pts, note := axisPoints(nil, 2, nil); note != "" || !slices.Equal(pts, []float64{0, 1}) {
		t.Errorf("no axis: %v %q", pts, note)
	}
	addr, data := model.Addr(0x10), model.Data{Bits: 8}
	x := &model.Axis{Source: "image", Address: &addr, Data: &data, Value: model.Value{Conversion: model.Conversion{Factor: 1}}}
	if pts, note := axisPoints(x, 4, nil); pts != nil || note != "needs-image" {
		t.Errorf("image axis, no image: %v %q", pts, note)
	}
	if pts, note := axisPoints(x, 4, make([]byte, 0x12)); pts != nil || note != "out-of-range" {
		t.Errorf("image axis past the end: %v %q", pts, note)
	}
}

func TestClassify(t *testing.T) {
	for path, want := range map[string][2]string{
		packKP:                                   {"definition", "kp"},
		packBin:                                  {"image", "bin"},
		"../testdata/a2l/versionless.a2l":        {"definition", "a2l"},
		"../testdata/damos/basic.dam":            {"definition", "damos"},
		"../testdata/kp/597-fewmaps.kp":          {"definition", "kp"},
		"../testdata/archive/ecuxplot/README.md": {"definition", "yaml"}, // text: read as a model document, and refused there
	} {
		kind, format := classify(path, read(t, path))
		if kind != want[0] || format != want[1] {
			t.Errorf("%s: %s %s, want %s %s", path, kind, format, want[0], want[1])
		}
	}
	if kind, _ := classify("x.meta.json", []byte(`{"schema":"x"}`)); kind != "meta" {
		t.Errorf("x.meta.json: %s, want meta", kind)
	}
}

func TestInspect(t *testing.T) {
	a := NewApp()
	kpInfo := a.Inspect(packKP)
	if kpInfo.Error != "" || kpInfo.Kind != "definition" || kpInfo.Format != "kp" || kpInfo.Objects == 0 ||
		kpInfo.Values+kpInfo.Curves+kpInfo.Maps != kpInfo.Objects || kpInfo.KPHeader == 0 {
		t.Errorf("KP: %+v", kpInfo)
	}
	bin := a.Inspect(packBin)
	if bin.Error != "" || bin.Kind != "image" || bin.Size != 1<<20 || bin.Family != "me7" || len(bin.SHA256) != 64 {
		t.Errorf("image: %+v", bin)
	}
	if bad := a.Inspect("../testdata/archive/ecuxplot/README.md"); bad.Error == "" {
		t.Errorf("a text file inspected as a definition without error: %+v", bad)
	}
	if missing := a.Inspect("no-such-file.kp"); missing.Error == "" {
		t.Error("missing file inspected without error")
	}
}

func TestOpenDefinitionAndDetail(t *testing.T) {
	a := NewApp()
	def, err := a.OpenDefinition(packKP)
	if err != nil {
		t.Fatal(err)
	}
	if len(def.Rows) != def.Info.Objects || len(def.Rows) == 0 {
		t.Fatalf("%d rows, %d objects", len(def.Rows), def.Info.Objects)
	}
	seen2d := false
	for _, r := range def.Rows {
		if r.Shape != "2d" || seen2d {
			continue
		}
		seen2d = true
		without, err := a.MapDetail(packKP, r.Index, "")
		if err != nil {
			t.Fatal(err)
		}
		if without.Note != "needs-image" || len(without.Cells) != 0 {
			t.Errorf("%s without image: note %q, %d rows", r.ID, without.Note, len(without.Cells))
		}
		v, err := a.MapDetail(packKP, r.Index, packBin)
		if err != nil {
			t.Fatal(err)
		}
		if v.Note != "" || len(v.Cells) != r.Rows || len(v.Cells[0]) != r.Cols {
			t.Errorf("%s: note %q, cells %dx%d, want %dx%d", r.ID, v.Note, len(v.Cells), len(v.Cells[0]), r.Rows, r.Cols)
		}
		if len(v.X.Values) != r.Cols || len(v.Y.Values) != r.Rows {
			t.Errorf("%s: %d x points (%s), %d y points (%s)", r.ID, len(v.X.Values), v.X.Note, len(v.Y.Values), v.Y.Note)
		}
		if !strings.Contains(v.JSON, `"shape": "2d"`) {
			t.Errorf("%s: JSON lacks the shape:\n%s", r.ID, v.JSON)
		}
	}
	if !seen2d {
		t.Fatal("no 2d map in the pack")
	}
	if _, err := a.MapDetail(packKP, len(def.Rows), ""); err == nil {
		t.Error("an index past the end was accepted")
	}
}

func TestConvertWritesAndProtectsFiles(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	in := filepath.Join(dir, "pack.kp")
	if err := os.WriteFile(in, read(t, packKP), 0o644); err != nil {
		t.Fatal(err)
	}
	// The image applies to XDF and is left out for JSON: one set of options
	// serves every format.
	for _, to := range []string{api.XDF, api.JSON, api.YAML, api.CSV} {
		r := a.Convert(ConvertJob{Input: in, To: to, Image: packBin, StaticAxes: to == api.XDF, WriteMeta: true})
		if r.Error != "" {
			t.Fatalf("%s: %s", to, r.Error)
		}
		if r.Output != filepath.Join(dir, "pack"+extOf[to]) || r.Bytes == 0 || r.From != api.KP {
			t.Errorf("%s: %+v", to, r)
		}
		if st, err := os.Stat(r.Output); err != nil || st.Size() != int64(r.Bytes) {
			t.Errorf("%s: output %v", to, err)
		}
		if (r.MetaOutput != "") != (to == api.XDF) {
			t.Errorf("%s: metadata file %q", to, r.MetaOutput)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "pack.meta.json")); err != nil {
		t.Errorf("no metadata file next to the XDF: %v", err)
	}
	// A second run does not replace the file unless told to.
	if r := a.Convert(ConvertJob{Input: in, To: api.XDF, WriteMeta: true}); !strings.HasPrefix(r.Error, "[[exists]]") {
		t.Errorf("existing output: error %q", r.Error)
	}
	if r := a.Convert(ConvertJob{Input: in, To: api.XDF, WriteMeta: true, Overwrite: true}); r.Error != "" {
		t.Errorf("overwrite: %s", r.Error)
	}
	// The input is never the output.
	if r := a.Convert(ConvertJob{Input: in, To: api.KP, Output: in, Overwrite: true}); !strings.HasPrefix(r.Error, "[[same-file]]") {
		t.Errorf("output onto the input: error %q", r.Error)
	}
	if got := read(t, in); len(got) != len(read(t, packKP)) {
		t.Error("the input file changed")
	}
	// XDF with its metadata file converts back to the same model JSON.
	back := a.Convert(ConvertJob{Input: filepath.Join(dir, "pack.xdf"), To: api.JSON, Output: filepath.Join(dir, "back.json")})
	if back.Error != "" || back.From != api.XDF {
		t.Fatalf("XDF back to JSON: %+v", back)
	}
	// An image is not a definition.
	if r := a.Convert(ConvertJob{Input: packBin, To: api.XDF, OutDir: dir}); !strings.HasPrefix(r.Error, "[[not-definition]]") {
		t.Errorf("image as input: error %q", r.Error)
	}
}

func TestSuggestOutput(t *testing.T) {
	a := NewApp()
	in := filepath.Join("C:", "maps", "a.kp")
	for _, tc := range []struct{ format, dir, tag, want string }{
		{api.XDF, "", "", filepath.Join("C:", "maps", "a.xdf")},
		{api.KP, "", "", filepath.Join("C:", "maps", "a.out.kp")},
		{api.KP, "", "fixed", filepath.Join("C:", "maps", "a.fixed.kp")},
		{api.JSON, filepath.Join("D:", "out"), "", filepath.Join("D:", "out", "a.json")},
	} {
		if got := a.SuggestOutput(in, tc.format, tc.dir, tc.tag); got != tc.want {
			t.Errorf("%s %q %q: %s, want %s", tc.format, tc.dir, tc.tag, got, tc.want)
		}
	}
}

func TestLintAndFix(t *testing.T) {
	a := NewApp()
	res, err := a.Lint(packKP, packBin, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Family != "me7" || res.Findings == nil {
		t.Fatalf("lint: %+v", res)
	}
	if _, err := a.Lint(packKP, "", ""); err == nil || !strings.HasPrefix(err.Error(), "[[no-image]]") {
		t.Errorf("lint without an image: %v", err)
	}
	out := filepath.Join(t.TempDir(), "fixed.kp")
	dry, err := a.Fix(FixJob{Input: packKP, Image: packBin, Output: out, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("a dry run wrote the output")
	}
	fixed, err := a.Fix(FixJob{Input: packKP, Image: packBin, Output: out})
	if err != nil {
		t.Fatal(err)
	}
	if fixed.Output != out || len(fixed.Applied) != len(dry.Applied) {
		t.Errorf("fix: %d applied, dry run %d, output %q", len(fixed.Applied), len(dry.Applied), fixed.Output)
	}
	if len(fixed.Remaining)+len(fixed.Applied) > len(res.Findings) {
		t.Errorf("%d findings, %d applied, %d remain", len(res.Findings), len(fixed.Applied), len(fixed.Remaining))
	}
	// The fixed file is a KP the app opens again.
	if info := a.Inspect(out); info.Error != "" || info.Objects == 0 {
		t.Errorf("fixed file: %+v", info)
	}
}

func TestMergeDryRunAndWrite(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	out := filepath.Join(dir, "merged.json")
	dry, err := a.Merge(MergeJob{Base: packKP, Incoming: packKP, To: api.JSON, Update: true, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(dry.Actions) == 0 {
		t.Error("a merge of a pack into itself reported nothing")
	}
	res, err := a.Merge(MergeJob{Base: packKP, Incoming: packKP, To: api.JSON, Update: true, Output: out})
	if err != nil {
		t.Fatal(err)
	}
	if res.Output != out || res.Bytes == 0 {
		t.Errorf("merge: %+v", res)
	}
	if v := a.Verify([]string{out, packBin}); v[0].Status != "clean" || v[1].Error == "" {
		t.Errorf("verify: %+v", v)
	}
}
