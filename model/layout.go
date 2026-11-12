package model

// Where cells sit when they are not simply packed row after row.
//
// TunerPro places a cell by two strides and an order flag
// (mmedmajorstridebits, mmedminorstridebits and type flag 0x04 in an XDF).
// The rule below is TunerPro's own, read from TunerPro RT v6: its cell offset
// routine at 0x4EAD40, which the cell reader adds to the cell address
// (0x4EA8D0) and the axis code calls with row 0 and the point's index as the
// column (0x57BE10).
//
//   - The major stride is the distance from one cell to the next along a row;
//     0 means the cell width. Negative: the cells of a row are in reverse.
//   - The minor stride is the distance from the last cell of a row to the
//     first cell of the next; 0 or the cell width means the next row follows
//     on. A stride under half the cell width is extra padding after the row
//     instead. Negative: the rows are in reverse.
//   - Column major swaps rows and columns in all of the above.

// Spaced reports cells that are not packed row after row.
func (d Data) Spaced() bool {
	return d.MajorStride != 0 && d.MajorStride != d.Bits || d.MinorStride != 0 && d.MinorStride != d.Bits || d.ColumnMajor
}

// CellOffset is the distance in bytes from the address of the cells to cell
// (row, col) of a table of rows by cols cells.
func (d Data) CellOffset(row, col, rows, cols int) int {
	bits, step, minor := d.Bits, d.MajorStride, d.MinorStride
	inner, outer, innerCount, outerCount := col, row, cols, rows
	if d.ColumnMajor {
		inner, outer, innerCount, outerCount = row, col, rows, cols
	}
	switch {
	case step == 0:
		step = bits
	case step < 0:
		step, inner = -step, innerCount-inner-1
	}
	if minor < 0 {
		outer = outerCount - outer - 1
	}
	gap := minor - bits
	if abs(minor) < abs(gap) {
		gap = minor
	}
	return (outer*(bits+step*(innerCount-1)) + step*inner + outer*abs(gap)) / 8
}

// Extent is the number of bytes from the address of the cells to the end of
// the cell that lies furthest from it.
func (d Data) Extent(rows, cols int) int {
	rows, cols = max(rows, 1), max(cols, 1)
	if !d.Spaced() {
		return rows * cols * (d.Bits / 8)
	}
	far := 0
	// The furthest cell is at a corner, whichever way the strides run.
	for _, r := range []int{0, rows - 1} {
		for _, c := range []int{0, cols - 1} {
			far = max(far, d.CellOffset(r, c, rows, cols))
		}
	}
	return far + (d.Bits+7)/8
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
