package main

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.nyet.org/xdfkit/api"
	"go.nyet.org/xdfkit/canon"
	"go.nyet.org/xdfkit/model"
)

// testdata/tunerpro.xdf is in the shape TunerPro writes: axes linked to other
// tables, axes with typed labels, an equation saved with a line break, a
// cell with its own equation after the table's, and Thai help text as
// Windows-874 byte references.

func objectNamed(t *testing.T, m *model.Model, id string) *model.Object {
	t.Helper()
	for _, o := range m.Objects {
		if o.ID == id {
			return o
		}
	}
	t.Fatalf("no object %q", id)
	return nil
}

// engineRead is what the engine alone makes of a file.
func engineRead(t *testing.T, path string) *model.Model {
	t.Helper()
	data := read(t, path)
	req := api.ConvertRequest{To: api.JSON, Name: filepath.Base(path)}
	if api.Detect(data) == api.XDF {
		_, _, req.Meta = siblingMeta(path)
	}
	out, _, err := api.Convert(data, req)
	if err != nil {
		t.Fatal(err)
	}
	m := new(model.Model)
	if err := canon.Unmarshal(out, m); err != nil {
		t.Fatal(err)
	}
	return m
}

func checkTunerProTables(t *testing.T, m *model.Model, what string) {
	t.Helper()
	ign := objectNamed(t, m, "Ignition Map Gear 1")
	// The table's own equation, not the one of cell (1,1) that follows it.
	if c := ign.Value.Conversion; c.Factor != 0.75 || c.Offset != -48 || c.Reciprocal {
		t.Errorf("%s: ignition formula %+v, want 0.75*X - 48", what, c)
	}
	if ign.Rows != 3 || ign.Cols != 4 || ign.Address != 0x30 || ign.Data.Bits != 8 {
		t.Errorf("%s: ignition cells %dx%d at %#x, %d bit", what, ign.Cols, ign.Rows, ign.Address, ign.Data.Bits)
	}
	// Linked axes are the linked tables' cells.
	if x := ign.X; x == nil || x.Source != "image" || x.Address == nil || *x.Address != 0x10 || x.Data.Bits != 16 || x.Data.Endian != "little" || x.Value.Units != "rpm" {
		t.Errorf("%s: ignition x axis %+v", what, x)
	}
	if y := ign.Y; y == nil || y.Source != "image" || y.Address == nil || *y.Address != 0x20 || y.Data.Bits != 8 || y.Value.Conversion.Factor != 0.5 {
		t.Errorf("%s: ignition y axis %+v", what, y)
	}
	// Typed labels are fixed axis points.
	fuel := objectNamed(t, m, "Fuel Trim")
	if x := fuel.X; x == nil || x.Source != "editable" || !slices.Equal(x.Values, []float64{1000, 2500.5, 4000}) || x.Value.Units != "rpm" {
		t.Errorf("%s: fuel x axis %+v", what, x)
	}
	if y := fuel.Y; y == nil || y.Source != "editable" || !slices.Equal(y.Values, []float64{-10, 20}) {
		t.Errorf("%s: fuel y axis %+v", what, y)
	}
	// Labels that only count stay a plain count.
	if y := objectNamed(t, m, "Load Axis").Y; y != nil && y.Source != "ordinal" {
		t.Errorf("%s: load axis table y %+v, want ordinal", what, y)
	}
	if rev := objectNamed(t, m, "Rev Limit"); rev.Shape != "value" || rev.Address != 0x50 || rev.Data.Bits != 16 {
		t.Errorf("%s: rev limit %+v", what, rev)
	}
}

func TestTunerProXDF(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	const src = "testdata/tunerpro.xdf"
	const help = "แกน RPM และ Load"

	d, err := a.loadDef(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range d.warnings {
		t.Errorf("warning reading the XDF: %s", w)
	}
	checkTunerProTables(t, d.model, "XDF")
	if got := objectNamed(t, d.model, "Ignition Map Gear 1").Description; got != help {
		t.Errorf("Thai help text %q, want %q", got, help)
	}

	// To a KP: WinOLS lists a map by its name, so the title is the name and
	// the help text the comment. The engine alone shows what the KP holds.
	kpPath := filepath.Join(dir, "tunerpro.kp")
	if r := a.Convert(ConvertJob{Input: src, To: api.KP, Output: kpPath}); r.Error != "" {
		t.Fatalf("XDF to KP: %s (%v)", r.Error, r.Warnings)
	}
	raw := engineRead(t, kpPath)
	ign := objectNamed(t, raw, "Ignition Map Gear 1")
	wantComment, _, _ := encodeThai(help, api.KP)
	if ign.Description != "Ignition Map Gear 1" || ign.Comment != wantComment {
		t.Errorf("KP name %q comment %q, want the title and the help text's bytes %q", ign.Description, ign.Comment, wantComment)
	}
	if rev := objectNamed(t, raw, "Rev Limit"); rev.Description != "Rev Limit" {
		t.Errorf("a map without help text is named %q in the KP", rev.Description)
	}
	kd, err := a.loadDef(kpPath)
	if err != nil {
		t.Fatal(err)
	}
	checkTunerProTables(t, kd.model, "KP")

	// Back to an XDF: the whole title, and the help text alone as the description.
	back := filepath.Join(dir, "back.xdf")
	if r := a.Convert(ConvertJob{Input: kpPath, To: api.XDF, Output: back}); r.Error != "" {
		t.Fatalf("KP to XDF: %s", r.Error)
	}
	text := string(read(t, back))
	if !strings.Contains(text, "<title>Ignition Map Gear 1</title>") {
		t.Error("the XDF title is not the whole name")
	}
	if strings.Contains(text, "<description>Ignition Map Gear 1") || strings.Contains(text, "<description>Rev Limit") {
		t.Error("the XDF description repeats the title")
	}
	bd, err := a.loadDef(back)
	if err != nil {
		t.Fatal(err)
	}
	checkTunerProTables(t, bd.model, "XDF from KP")
	if got := objectNamed(t, bd.model, "Ignition Map Gear 1"); got.Description != help {
		t.Errorf("XDF from KP: help text %q, want %q", got.Description, help)
	}
}
