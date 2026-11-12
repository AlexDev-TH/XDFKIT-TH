package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.nyet.org/xdfkit/api"
	"go.nyet.org/xdfkit/canon"
	"go.nyet.org/xdfkit/model"
)

func cellRows(v MapView) [][]float64 {
	out := make([][]float64, len(v.Cells))
	for r, row := range v.Cells {
		for _, c := range row {
			out[r] = append(out[r], float64(c))
		}
	}
	return out
}

func axisValues(a AxisView) []float64 {
	out := make([]float64, len(a.Values))
	for i, v := range a.Values {
		out[i] = float64(v)
	}
	return out
}

// spacedModel is a curve stored as byte pairs (the axis point, then the
// value), as Keihin ECUs keep them, and a map stored column by column.
func spacedModel() (*model.Model, []byte) {
	img := make([]byte, 0x40)
	copy(img[0x10:], []byte{1, 10, 2, 20, 3, 30, 4, 40}) // axis, value, axis, value
	copy(img[0x30:], []byte{1, 2, 3, 4, 5, 6})           // column 0, column 1, column 2
	one := model.Conversion{Factor: 1}
	axisAddr := model.Addr(0x10)
	m := &model.Model{Schema: model.SchemaID, Objects: []*model.Object{
		{
			ID: "Pairs", Shape: "1d", Address: 0x11, Rows: 1, Cols: 4,
			Data:  model.Data{Bits: 8, MajorStride: 16},
			Value: model.Value{Conversion: one},
			X: &model.Axis{Source: "image", Stored: "absolute", Address: &axisAddr,
				Data: &model.Data{Bits: 8, MajorStride: 16}, Value: model.Value{Conversion: one}},
		},
		{
			ID: "Columns", Shape: "2d", Address: 0x30, Rows: 2, Cols: 3,
			Data:  model.Data{Bits: 8, ColumnMajor: true},
			Value: model.Value{Conversion: one},
		},
	}}
	return m, img
}

// TestSpacedCells: the map view reads cells and axis points where TunerPro
// places them.
func TestSpacedCells(t *testing.T) {
	m, img := spacedModel()
	pairs := viewObject(m, 0, img)
	if got := cellRows(pairs); len(got) != 1 || !slices.Equal(got[0], []float64{10, 20, 30, 40}) {
		t.Errorf("paired values %v, want 10 20 30 40", got)
	}
	if got := axisValues(pairs.X); !slices.Equal(got, []float64{1, 2, 3, 4}) {
		t.Errorf("paired axis %v (%s), want 1 2 3 4", got, pairs.X.Note)
	}
	if pairs.EndAddress != "0x18" {
		t.Errorf("paired values end at %s, want 0x18", pairs.EndAddress)
	}
	cols := viewObject(m, 1, img)
	if got := cellRows(cols); len(got) != 2 || !slices.Equal(got[0], []float64{1, 3, 5}) || !slices.Equal(got[1], []float64{2, 4, 6}) {
		t.Errorf("column-major cells %v, want [1 3 5] [2 4 6]", got)
	}
	// Cells that reach past the image are not read at all.
	if v := viewObject(m, 0, img[:0x16]); v.Note != "out-of-range" {
		t.Errorf("cells past the image: note %q", v.Note)
	}
}

// TestSpacedThroughXDF: the layout is written to an XDF as TunerPro reads it,
// comes back from it, reaches the CSV value ranges, and a KP is told it
// cannot hold it.
func TestSpacedThroughXDF(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	m, img := spacedModel()
	doc, err := canon.MarshalStamped(m, "test")
	if err != nil {
		t.Fatal(err)
	}
	src, bin := filepath.Join(dir, "spaced.json"), filepath.Join(dir, "spaced.bin")
	if err := os.WriteFile(src, doc, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, img, 0o644); err != nil {
		t.Fatal(err)
	}
	x := a.Convert(ConvertJob{Input: src, To: api.XDF, Image: bin})
	if x.Error != "" {
		t.Fatal(x.Error)
	}
	text := string(read(t, x.Output))
	for _, want := range []string{
		`mmedaddress="0x11" mmedelementsizebits="8" mmedrowcount="1" mmedcolcount="4" mmedmajorstridebits="16"`,
		`mmedtypeflags="0x04" mmedaddress="0x30"`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the XDF lacks %s", want)
		}
	}
	// Read back without its metadata file: the XDF alone says where the cells are.
	if err := os.Remove(metaName(x.Output)); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	d, err := a.loadDef(x.Output)
	if err != nil {
		t.Fatal(err)
	}
	if got := d.model.Objects[0].Data; got.MajorStride != 16 || got.ColumnMajor {
		t.Errorf("paired values read back as %+v", got)
	}
	if got := d.model.Objects[0].X; got == nil || got.Data == nil || got.Data.MajorStride != 16 {
		t.Errorf("paired axis read back as %+v", got)
	}
	if got := d.model.Objects[1].Data; !got.ColumnMajor {
		t.Errorf("column-major cells read back as %+v", got)
	}
	v, err := a.MapDetail(x.Output, 0, bin)
	if err != nil {
		t.Fatal(err)
	}
	if got := cellRows(v); len(got) != 1 || !slices.Equal(got[0], []float64{10, 20, 30, 40}) {
		t.Errorf("values through the XDF %v, want 10 20 30 40", got)
	}
	// The CSV map list reads the same cells: 10 to 40, not the axis bytes between them.
	c := a.Convert(ConvertJob{Input: src, To: api.CSV, Image: bin})
	if c.Error != "" {
		t.Fatal(c.Error)
	}
	if csv := string(read(t, c.Output)); !strings.Contains(csv, `"10.0","40.0","0xa","0x28"`) {
		t.Errorf("CSV value range of the paired values:\n%s", csv)
	}
	// A KP holds packed cells only: the conversion says so.
	k := a.Convert(ConvertJob{Input: src, To: api.KP})
	if k.Error != "" {
		t.Fatal(k.Error)
	}
	if !slices.Contains(k.Warnings, "[[kp-layout]] 2") {
		t.Errorf("KP warnings %v, want the two spaced tables reported", k.Warnings)
	}
}
