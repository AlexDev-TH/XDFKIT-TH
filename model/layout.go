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

// WinOLS states a layout by two counts of bytes instead, the script's
// SkipBytes and LineSkipBytes. Its cell address routine, read from a WinOLS 5
// build, puts element i of stored line l, with n elements to a line, at
//
//	(width+skip)*(n*l+i) + lineSkip*l
//
// from the map's start, and axis point i at (width+skip)*i. So skip is the
// major stride less the cell, and lineSkip what the gap after a row adds to it.

// SetSkips sets the strides of cells that WinOLS spaces by skip bytes after
// each cell and lineSkip bytes after each of lines stored lines. width is the
// cell size in bytes as the record states it. It reports false, leaving the
// rows packed, when the lines would overlap.
func (d *Data) SetSkips(width, skip, lineSkip, lines int) bool {
	d.MajorStride, d.MinorStride = 0, 0
	if width <= 0 {
		width = d.Bits / 8
	}
	pitch := 8 * (width + skip)
	if pitch != d.Bits {
		d.MajorStride = pitch
	}
	if lines <= 1 {
		return pitch >= d.Bits
	}
	gap := pitch + 8*lineSkip - d.Bits
	if gap > 0 {
		d.MinorStride = gap + d.Bits
	}
	return gap >= 0 && pitch >= d.Bits
}

// Skips states d's strides the way WinOLS does, for a table of lines stored
// lines with per cells each. ok is false when WinOLS has no such layout: cells
// narrower than a byte or closer than their width, cells or rows in reverse,
// column-major order, or a row gap smaller than the gap between cells.
// Lines of one cell have no gap between cells: what follows each line is
// stated as the skip after each cell.
func (d Data) Skips(lines, per int) (skip, lineSkip int, ok bool) {
	step := d.MajorStride
	if step == 0 || per <= 1 {
		step = d.Bits
	}
	if d.Bits <= 0 || d.Bits%8 != 0 || step%8 != 0 || step < d.Bits || d.MinorStride < 0 || d.ColumnMajor {
		return 0, 0, false
	}
	skip = (step - d.Bits) / 8
	if lines <= 1 {
		return skip, 0, true
	}
	gap := d.MinorStride - d.Bits
	if abs(d.MinorStride) < abs(gap) {
		gap = d.MinorStride
	}
	if gap = abs(gap); gap%8 != 0 || gap/8 < skip {
		return 0, 0, false
	}
	if per <= 1 {
		return gap / 8, 0, true
	}
	return skip, gap/8 - skip, true
}

// rowForm returns d for a table of one row with its cells stated along the
// row. Column-major order makes each cell of such a table a column of its
// own, so only the minor stride spaces them.
func (d Data) rowForm(rows int) Data {
	if !d.ColumnMajor || rows != 1 {
		return d
	}
	gap := d.MinorStride - d.Bits
	if abs(d.MinorStride) < abs(gap) {
		gap = d.MinorStride
	}
	step := d.Bits + abs(gap)
	if d.MinorStride < 0 {
		step = -step
	}
	if step == d.Bits {
		step = 0
	}
	d.MajorStride, d.MinorStride, d.ColumnMajor = step, 0, false
	return d
}

// A 2d-inverse KP map and a column-major table hold the same cells. WinOLS
// shows a 2d-inverse map with its stored lines as the columns: its cell
// address routine swaps the column and the row before it places the cell.
// The model keeps such a map in stored order, with Inverse set.

// AsColumnMajor returns a 2d-inverse object as the column-major table its map
// window shows: rows and columns swapped, every cell where it was. Any other
// object is returned as it is.
func (o *Object) AsColumnMajor() *Object {
	if !o.Inverse || o.Shape != "2d" || o.Data.ColumnMajor {
		return o
	}
	c := *o
	c.Rows, c.Cols, c.Inverse = o.Cols, o.Rows, false
	c.Data.ColumnMajor = true
	return &c
}

// AsInverse returns a column-major table as the 2d-inverse object that holds
// the same cells in stored order. Any other object is returned as it is.
func (o *Object) AsInverse() *Object {
	if o.Inverse || o.Shape != "2d" || !o.Data.ColumnMajor {
		return o
	}
	c := *o
	c.Rows, c.Cols, c.Inverse = o.Cols, o.Rows, true
	c.Data.ColumnMajor = false
	return &c
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
