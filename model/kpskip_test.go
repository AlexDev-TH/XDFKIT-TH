package model

import (
	"os"
	"path/filepath"
	"testing"

	"go.nyet.org/xdfkit/internal/testenv"
	"go.nyet.org/xdfkit/kp"
)

// skipModel has a curve of byte pairs with its axis in the other byte, a table
// whose lines each end in a value that is not a cell, a column-major table and
// a table stored backwards.
func skipModel() *Model {
	one := Conversion{Factor: 1}
	axis := Addr(0x100)
	m := &Model{Schema: SchemaID, Objects: []*Object{
		{
			ID: "PAIRS", Shape: "1d", Address: 0x101, Rows: 1, Cols: 8,
			Data:  Data{Bits: 8, MajorStride: 16},
			Value: Value{Conversion: one},
			X: &Axis{Source: "image", Stored: "absolute", Address: &axis,
				Data: &Data{Bits: 8, MajorStride: 16}, Value: Value{Conversion: one}},
		},
		{
			ID: "LINES", Shape: "2d", Address: 0x200, Rows: 4, Cols: 5,
			Data:  Data{Bits: 16, Endian: "little", MinorStride: 32},
			Value: Value{Conversion: one},
		},
		{
			ID: "COLUMNS", Shape: "2d", Address: 0x300, Rows: 2, Cols: 3,
			Data:  Data{Bits: 8, ColumnMajor: true},
			Value: Value{Conversion: one},
		},
		{
			ID: "BACK", Shape: "1d", Address: 0x400, Rows: 1, Cols: 4,
			Data:  Data{Bits: 16, Endian: "big", MajorStride: -16},
			Value: Value{Conversion: one},
		},
	}}
	m.AssignKeys()
	return m
}

// TestKPSkipBytes: a KP states gaps between cells by skip bytes. A pack of
// header 292 has them for axis points only. A WinOLS 5 pack would hold those
// of map cells too, but it cannot be written with other maps (kp.HeaderSealed),
// so it is turned down as a template.
func TestKPSkipBytes(t *testing.T) {
	m := skipModel()
	if noForm, tooOld := m.KPLayoutLost(KPDefaultHeader); noForm != 1 || tooOld != 2 {
		t.Errorf("header %d: %d without a form and %d needing a later pack, want 1 and 2", KPDefaultHeader, noForm, tooOld)
	}
	if noForm, tooOld := m.KPLayoutLost(503); noForm != 1 || tooOld != 0 {
		t.Errorf("header 503: %d without a form and %d needing a later pack, want 1 and 0", noForm, tooOld)
	}

	// The pack KP writes by itself: the axis keeps its skip, the cells are packed.
	f, err := m.KP(nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := f.Encode()
	if err != nil {
		t.Fatal(err)
	}
	g, err := kp.Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	got := FromKP(g)
	if g.HeaderID != KPDefaultHeader {
		t.Fatalf("header %d", g.HeaderID)
	}
	if km := g.Project.Maps[0]; km.X.SkipBytes != 1 {
		t.Errorf("paired axis: skip %d in the pack", km.X.SkipBytes)
	}
	if x := got.Objects[0].X; x == nil || x.Data == nil || x.Data.MajorStride != 16 {
		t.Errorf("paired axis read back as %+v", x)
	}
	if d := got.Objects[0].Data; d.Spaced() {
		t.Errorf("paired cells read back as %+v", d)
	}
	if km := g.Project.Maps[2]; km.Organization != kp.OrgTwoDInv || km.Cols != 2 || km.Rows != 3 || km.StoredCols != 3 || km.StoredRows != 2 {
		t.Errorf("columns: %v, %d cells in %d lines, shown %d by %d", km.Organization, km.Cols, km.Rows, km.StoredCols, km.StoredRows)
	}
	// Read back, the 2d-inverse map has every cell where the column-major table had it.
	o, back := m.Objects[2], got.Objects[2].AsColumnMajor()
	if back.Rows != o.Rows || back.Cols != o.Cols {
		t.Fatalf("columns read back %d by %d, want %d by %d", back.Rows, back.Cols, o.Rows, o.Cols)
	}
	for r := range o.Rows {
		for c := range o.Cols {
			if a, b := back.Data.CellOffset(r, c, back.Rows, back.Cols), o.Data.CellOffset(r, c, o.Rows, o.Cols); a != b {
				t.Errorf("columns: cell %d,%d at %d, want %d", r, c, a, b)
			}
		}
	}

	// The skip bytes a later pack holds are read into the same strides.
	later := &kp.Map{Organization: kp.OrgTwoD, Type: 3, Width: 2, Cols: 5, Rows: 4, SkipBytes: 2, LineSkipBytes: 2, X: &kp.Axis{}, Y: &kp.Axis{}}
	if d := objectFromKP(later, false).Data; d.MajorStride != 32 || d.MinorStride != 48 {
		t.Errorf("skip 2 and line skip 2 read as strides %d and %d, want 32 and 48", d.MajorStride, d.MinorStride)
	}

	// A WinOLS 5 pack is not a template, and is encoded only as it was read.
	b, err := os.ReadFile(filepath.Join(testenv.ModuleRoot(), "testdata", "kp", "503-fewmaps.kp"))
	if err != nil {
		t.Fatal(err)
	}
	template, err := kp.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.KP(template); err == nil {
		t.Error("a header 503 pack was taken as the template")
	}
	if _, err := template.Encode(); err != nil {
		t.Errorf("header 503 pack as read: %v", err)
	}
	template.Project.Maps = template.Project.Maps[:1]
	if _, err := template.Encode(); err == nil {
		t.Error("a header 503 pack was encoded with fewer maps")
	}
}
