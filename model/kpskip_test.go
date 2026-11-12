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
// header 292 has them for axis points only; a WinOLS 5 pack as template holds
// those of map cells too, and they read back as the strides they came from.
func TestKPSkipBytes(t *testing.T) {
	m := skipModel()
	if noForm, tooOld := m.KPLayoutLost(KPDefaultHeader); noForm != 1 || tooOld != 2 {
		t.Errorf("header %d: %d without a form and %d needing a later pack, want 1 and 2", KPDefaultHeader, noForm, tooOld)
	}
	if noForm, tooOld := m.KPLayoutLost(503); noForm != 1 || tooOld != 0 {
		t.Errorf("header 503: %d without a form and %d needing a later pack, want 1 and 0", noForm, tooOld)
	}

	back := func(template *kp.File) (*kp.File, *Model) {
		t.Helper()
		f, err := m.KP(template)
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
		return g, FromKP(g)
	}

	// The pack KP writes by itself: the axis keeps its skip, the cells are packed.
	f, got := back(nil)
	if f.HeaderID != KPDefaultHeader {
		t.Fatalf("header %d", f.HeaderID)
	}
	if x := got.Objects[0].X; x == nil || x.Data == nil || x.Data.MajorStride != 16 {
		t.Errorf("header %d: paired axis read back as %+v", f.HeaderID, x)
	}
	if d := got.Objects[0].Data; d.Spaced() {
		t.Errorf("header %d: paired cells read back as %+v", f.HeaderID, d)
	}
	if o := got.Objects[2]; !o.Inverse || o.Rows != 3 || o.Cols != 2 {
		t.Errorf("header %d: column-major table read back as %+v", f.HeaderID, o)
	}

	b, err := os.ReadFile(filepath.Join(testenv.ModuleRoot(), "testdata", "kp", "503-fewmaps.kp"))
	if err != nil {
		t.Fatal(err)
	}
	template, err := kp.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	f, got = back(template)
	if f.HeaderID != 503 || len(f.Project.Maps) != len(m.Objects) {
		t.Fatalf("header %d, %d maps", f.HeaderID, len(f.Project.Maps))
	}
	if km := f.Project.Maps[0]; km.SkipBytes != 1 || km.LineSkipBytes != 0 || km.X.SkipBytes != 1 || km.End != 0x101+16 {
		t.Errorf("pairs: skip %d, line skip %d, axis skip %d, end %#x", km.SkipBytes, km.LineSkipBytes, km.X.SkipBytes, km.End)
	}
	if km := f.Project.Maps[1]; km.SkipBytes != 0 || km.LineSkipBytes != 2 || km.End != 0x200+4*5*2+4*2 {
		t.Errorf("lines: skip %d, line skip %d, end %#x", km.SkipBytes, km.LineSkipBytes, km.End)
	}
	if km := f.Project.Maps[2]; km.Organization != kp.OrgTwoDInv || km.Cols != 2 || km.Rows != 3 || km.StoredCols != 3 || km.StoredRows != 2 {
		t.Errorf("columns: %v, %d cells in %d lines, shown %d by %d", km.Organization, km.Cols, km.Rows, km.StoredCols, km.StoredRows)
	}
	// The table stored backwards has no form: it is written packed.
	if km := f.Project.Maps[3]; km.SkipBytes != 0 || km.LineSkipBytes != 0 {
		t.Errorf("backwards table: skip %d, line skip %d", km.SkipBytes, km.LineSkipBytes)
	}
	for i, want := range []Data{m.Objects[0].Data, m.Objects[1].Data} {
		if d := got.Objects[i].Data; d.MajorStride != want.MajorStride || d.MinorStride != want.MinorStride {
			t.Errorf("object %d: strides %d and %d read back, want %d and %d", i, d.MajorStride, d.MinorStride, want.MajorStride, want.MinorStride)
		}
	}
	// Every cell of the first three objects is where the model has it.
	for i, o := range m.Objects[:3] {
		g := got.Objects[i].AsColumnMajor()
		if g.Rows != o.Rows || g.Cols != o.Cols {
			t.Errorf("%s: %d by %d read back, want %d by %d", o.ID, g.Rows, g.Cols, o.Rows, o.Cols)
			continue
		}
		for r := range o.Rows {
			for c := range o.Cols {
				if a, b := g.Data.CellOffset(r, c, g.Rows, g.Cols), o.Data.CellOffset(r, c, o.Rows, o.Cols); a != b {
					t.Errorf("%s: cell %d,%d at %d, want %d", o.ID, r, c, a, b)
				}
			}
		}
	}
}
