package model

import "testing"

// TestCellOffset: offsets worked by hand from TunerPro's rule (layout.go).
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
