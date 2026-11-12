package model

import "testing"

// TestCellOffset: offsets worked by hand from TunerPro's rule (layout.go).
// TestSkips: WinOLS's skip bytes and the strides state the same cells. The
// offsets are WinOLS's own rule, (width+skip)*(n*line+i) + lineSkip*line.
func TestSkips(t *testing.T) {
	for _, c := range []struct {
		name                         string
		bits, skip, lineSkip, per, n int
		major, minor                 int
	}{
		{"packed", 16, 0, 0, 4, 3, 0, 0},
		// Byte pairs: a cell, then a byte that is not one.
		{"pairs", 8, 1, 0, 4, 3, 16, 16},
		// Each line ends in one more 16-bit value.
		{"line skip", 16, 0, 2, 5, 4, 0, 32},
		{"both", 8, 1, 3, 6, 2, 16, 40},
		{"one line", 16, 2, 0, 7, 1, 32, 0},
		// One cell to a line: what follows the cell follows the line.
		{"one cell lines", 16, 3, 0, 1, 5, 40, 40},
	} {
		d := Data{Bits: c.bits}
		if !d.SetSkips(c.bits/8, c.skip, c.lineSkip, c.n) {
			t.Errorf("%s: SetSkips refused", c.name)
			continue
		}
		if d.MajorStride != c.major || d.MinorStride != c.minor {
			t.Errorf("%s: strides %d and %d, want %d and %d", c.name, d.MajorStride, d.MinorStride, c.major, c.minor)
		}
		w := c.bits / 8
		for line := range c.n {
			for i := range c.per {
				want := (w+c.skip)*(c.per*line+i) + c.lineSkip*line
				if got := d.CellOffset(line, i, c.n, c.per); got != want {
					t.Errorf("%s: cell %d of line %d at %d, want %d", c.name, i, line, got, want)
				}
			}
		}
		skip, lineSkip, ok := d.Skips(c.n, c.per)
		wantLine := c.lineSkip
		if c.n == 1 {
			wantLine = 0
		}
		if !ok || skip != c.skip || lineSkip != wantLine {
			t.Errorf("%s: skips back %d and %d (%v), want %d and %d", c.name, skip, lineSkip, ok, c.skip, wantLine)
		}
	}
	// Layouts WinOLS has no form for.
	for name, d := range map[string]Data{
		"reversed row":         {Bits: 16, MajorStride: -16},
		"reversed rows":        {Bits: 16, MinorStride: -16},
		"overlapping cells":    {Bits: 16, MajorStride: 8},
		"part of a byte":       {Bits: 4},
		"column major":         {Bits: 8, ColumnMajor: true},
		"row gap under cell's": {Bits: 16, MajorStride: 32},
	} {
		if _, _, ok := d.Skips(3, 4); ok {
			t.Errorf("%s: stated as skip bytes", name)
		}
	}
	// With one line there is no row gap to state.
	if skip, lineSkip, ok := (Data{Bits: 16, MajorStride: 32}).Skips(1, 4); !ok || skip != 2 || lineSkip != 0 {
		t.Errorf("one line of spaced cells: %d and %d (%v)", skip, lineSkip, ok)
	}
}

// TestInverseForms: a 2d-inverse object and the column-major table it is shown
// as hold every cell at the same place.
func TestInverseForms(t *testing.T) {
	inv := &Object{Shape: "2d", Inverse: true, Rows: 3, Cols: 2, Data: Data{Bits: 16, MajorStride: 32, MinorStride: 48}}
	shown := inv.AsColumnMajor()
	if shown == inv || shown.Inverse || !shown.Data.ColumnMajor || shown.Rows != 2 || shown.Cols != 3 {
		t.Fatalf("shown form %+v", shown)
	}
	for line := range inv.Rows {
		for i := range inv.Cols {
			stored := inv.Data.CellOffset(line, i, inv.Rows, inv.Cols)
			// The stored line is the column; the cell's place in it is the row.
			if got := shown.Data.CellOffset(i, line, shown.Rows, shown.Cols); got != stored {
				t.Errorf("cell %d of line %d: shown at %d, stored at %d", i, line, got, stored)
			}
		}
	}
	back := shown.AsInverse()
	if !back.Inverse || back.Data.ColumnMajor || back.Rows != inv.Rows || back.Cols != inv.Cols || back.Data != inv.Data {
		t.Errorf("back to the stored form: %+v", back)
	}
	plain := &Object{Shape: "2d", Rows: 3, Cols: 2, Data: Data{Bits: 8}}
	if plain.AsColumnMajor() != plain || plain.AsInverse() != plain {
		t.Error("a plain table was changed")
	}
	// One row of column-major cells is a row of cells spaced by the minor stride.
	d := Data{Bits: 16, MinorStride: 80, ColumnMajor: true}
	r := d.rowForm(1)
	if r.ColumnMajor || r.MajorStride != 80 || r.MinorStride != 0 || r.CellOffset(0, 1, 1, 2) != d.CellOffset(0, 1, 1, 2) {
		t.Errorf("row form %+v", r)
	}
}

func TestCellOffset(t *testing.T) {
	for _, c := range []struct {
		name                 string
		d                    Data
		row, col, rows, cols int
		want                 int
	}{
		{"packed 16 bit", Data{Bits: 16}, 2, 3, 3, 4, 22},
		{"packed 8 bit", Data{Bits: 8}, 1, 2, 4, 16, 18},
		// One row of records ten bytes long: a cell in each.
		{"major stride", Data{Bits: 16, MajorStride: 80}, 0, 5, 1, 8, 50},
		// Byte pairs, the cell in the first byte of each.
		{"pairs", Data{Bits: 8, MajorStride: 16}, 0, 3, 1, 8, 6},
		// One column, the next row four bytes on.
		{"minor stride", Data{Bits: 16, MinorStride: 32}, 3, 0, 4, 1, 12},
		// Cells four bytes apart in a row; from the last cell of a row to the
		// first of the next is 29 bytes: a row every 24+29 = 53 bytes.
		{"both strides", Data{Bits: 16, MajorStride: 32, MinorStride: 232}, 2, 6, 3, 7, 130},
		{"both strides, first of row", Data{Bits: 16, MajorStride: 32, MinorStride: 232}, 1, 0, 3, 7, 53},
		// A minor stride equal to the cell width is no gap.
		{"minor is the width", Data{Bits: 16, MinorStride: 16}, 2, 1, 3, 4, 18},
		// Under half the width it is padding after the row: 4 bits here.
		{"minor as padding", Data{Bits: 16, MinorStride: 4}, 1, 0, 2, 2, 4},
		// A negative major stride stores a row backwards.
		{"reversed row, first", Data{Bits: 16, MajorStride: -16}, 0, 0, 1, 4, 6},
		{"reversed row, last", Data{Bits: 16, MajorStride: -16}, 0, 3, 1, 4, 0},
		// A negative minor stride stores the rows backwards. TunerPro takes
		// its size as padding too: row 0 is the third row stored, each row
		// four cells and one byte of padding, so cell 1 is at 2*5 + 1.
		{"reversed rows", Data{Bits: 8, MinorStride: -8}, 0, 1, 3, 4, 11},
		// Column by column: column 1 starts after the three cells of column 0.
		{"column major", Data{Bits: 16, ColumnMajor: true}, 2, 1, 3, 4, 10},
		{"column major, strided", Data{Bits: 8, MajorStride: 16, ColumnMajor: true}, 2, 0, 4, 1, 4},
	} {
		if got := c.d.CellOffset(c.row, c.col, c.rows, c.cols); got != c.want {
			t.Errorf("%s: cell (%d, %d) of %dx%d at byte %d, want %d", c.name, c.row, c.col, c.rows, c.cols, got, c.want)
		}
	}
}

func TestExtent(t *testing.T) {
	for _, c := range []struct {
		d          Data
		rows, cols int
		want       int
	}{
		{Data{Bits: 16}, 3, 4, 24},
		{Data{Bits: 8, MajorStride: 16}, 1, 8, 15},                     // last cell at byte 14
		{Data{Bits: 16, MajorStride: 32, MinorStride: 232}, 3, 7, 132}, // last cell at byte 130
		{Data{Bits: 16, MajorStride: -16}, 1, 4, 8},
	} {
		if got := c.d.Extent(c.rows, c.cols); got != c.want {
			t.Errorf("%+v %dx%d: extent %d, want %d", c.d, c.rows, c.cols, got, c.want)
		}
	}
	if (Data{Bits: 16}).Spaced() || (Data{Bits: 16, MajorStride: 16, MinorStride: 16}).Spaced() {
		t.Error("packed cells reported as spaced")
	}
	if !(Data{Bits: 8, MajorStride: 16}).Spaced() || !(Data{Bits: 8, ColumnMajor: true}).Spaced() {
		t.Error("spaced cells reported as packed")
	}
}
