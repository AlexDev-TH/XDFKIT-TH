package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"

	"go.nyet.org/xdfkit/model"
)

// maxCells is the most cells sent to the page for one object. Calibration
// maps are far smaller; a larger block is a memory region, not a table.
const maxCells = 16384

// Category is a category of the open definition, with how many objects it holds.
type Category struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// MapRow is one object in the map list.
type MapRow struct {
	Index       int      `json:"index"`
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Comment     string   `json:"comment,omitempty"`
	Categories  []string `json:"categories"`
	Shape       string   `json:"shape"` // value, 1d or 2d
	Rows        int      `json:"rows"`
	Cols        int      `json:"cols"`
	Address     uint32   `json:"address"`
	Bits        int      `json:"bits"`
	Signed      bool     `json:"signed"`
	Float       bool     `json:"float"`
	Endian      string   `json:"endian"`
	Units       string   `json:"units"`
	Alternates  int      `json:"alternates"`
}

// Definition is an opened definition file: its summary and every object.
type Definition struct {
	Info       FileInfo   `json:"info"`
	Categories []Category `json:"categories"`
	Rows       []MapRow   `json:"rows"`
}

// OpenDefinition reads a definition file for the map list.
func (a *App) OpenDefinition(path string) (Definition, error) {
	info := a.Inspect(path)
	def := Definition{Info: info, Categories: []Category{}, Rows: []MapRow{}}
	if info.Kind != "definition" {
		return def, appError{"not-definition", path}
	}
	d, err := a.loadDef(path)
	if err != nil {
		return def, err
	}
	m := d.model
	names := map[int]string{}
	counts := map[int]int{}
	for _, c := range m.Categories {
		names[c.ID] = c.Name
	}
	def.Rows = make([]MapRow, len(m.Objects))
	for i, o := range m.Objects {
		o = o.AsColumnMajor() // rows and columns as the map is shown
		cats := make([]string, 0, len(o.Categories))
		for _, id := range o.Categories {
			counts[id]++
			if n, ok := names[id]; ok {
				cats = append(cats, n)
			}
		}
		def.Rows[i] = MapRow{
			Index: i, ID: o.ID, Description: o.Description, Comment: o.Comment, Categories: cats,
			Shape: o.Shape, Rows: o.Rows, Cols: o.Cols, Address: uint32(o.Address),
			Bits: o.Data.Bits, Signed: o.Data.Signed, Float: o.Data.Float, Endian: endianOf(o.Data),
			Units: o.Value.Units, Alternates: len(o.Alternates),
		}
	}
	for _, c := range m.Categories {
		def.Categories = append(def.Categories, Category{c.ID, c.Name, counts[c.ID]})
	}
	return def, nil
}

// endianOf is the byte order of cells wider than a byte. Unset is little, as
// the engine's XDF writer reads it (C167 / ME7).
func endianOf(d model.Data) string {
	switch {
	case d.Bits <= 8:
		return ""
	case d.Endian == "big":
		return "big"
	}
	return "little"
}

// Num is a number for the page: a value that isn't finite (a reciprocal of
// zero, a float cell holding NaN) is sent as null, which JSON can carry.
type Num float64

func (n Num) MarshalJSON() ([]byte, error) {
	f := float64(n)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return []byte("null"), nil
	}
	return json.Marshal(f)
}

// AxisView is one axis of an object, with its points when they can be worked out.
type AxisView struct {
	Present     bool   `json:"present"`
	Source      string `json:"source"`
	Stored      string `json:"stored,omitempty"`
	Address     string `json:"address,omitempty"`
	Bits        int    `json:"bits,omitempty"`
	Signed      bool   `json:"signed,omitempty"`
	Float       bool   `json:"float,omitempty"`
	Endian      string `json:"endian,omitempty"`
	Description string `json:"description,omitempty"`
	Units       string `json:"units,omitempty"`
	Precision   int    `json:"precision"`
	Factor      Num    `json:"factor"`
	Offset      Num    `json:"offset"`
	Reciprocal  bool   `json:"reciprocal,omitempty"`
	Mirror      bool   `json:"mirror,omitempty"`
	Count       int    `json:"count"`
	Values      []Num  `json:"values"`
	// Note says why Values is empty: needs-image, out-of-range or unsupported.
	Note string `json:"note,omitempty"`
}

// MapView is one object in full: its definition and, with an image, its cells.
type MapView struct {
	Index            int      `json:"index"`
	ID               string   `json:"id"`
	Key              string   `json:"key,omitempty"`
	Description      string   `json:"description"`
	Comment          string   `json:"comment,omitempty"`
	Categories       []string `json:"categories"`
	Shape            string   `json:"shape"`
	Inverse          bool     `json:"inverse,omitempty"`
	Rows             int      `json:"rows"`
	Cols             int      `json:"cols"`
	Address          string   `json:"address"`
	EndAddress       string   `json:"endAddress"`
	Bits             int      `json:"bits"`
	Signed           bool     `json:"signed"`
	Float            bool     `json:"float"`
	Endian           string   `json:"endian"`
	ValueDescription string   `json:"valueDescription,omitempty"`
	Units            string   `json:"units"`
	Precision        int      `json:"precision"`
	Factor           Num      `json:"factor"`
	Offset           Num      `json:"offset"`
	Reciprocal       bool     `json:"reciprocal,omitempty"`
	Base             int      `json:"base"`
	X                AxisView `json:"x"`
	Y                AxisView `json:"y"`
	// Cells are the converted values, rows by cols; Raw the stored ones.
	Cells [][]Num `json:"cells"`
	Raw   [][]Num `json:"raw"`
	// Note says why Cells is empty: needs-image, out-of-range, too-large or unsupported.
	Note    string `json:"note,omitempty"`
	CellMin Num    `json:"cellMin"`
	CellMax Num    `json:"cellMax"`
	JSON    string `json:"json"`
}

// MapDetail returns object index of a definition, with its cells and axis
// points read from the image when one is given.
func (a *App) MapDetail(path string, index int, image string) (MapView, error) {
	d, err := a.loadDef(path)
	if err != nil {
		return MapView{}, err
	}
	if index < 0 || index >= len(d.model.Objects) {
		return MapView{}, fmt.Errorf("no object %d", index)
	}
	img, err := a.loadImage(image)
	if err != nil {
		return MapView{}, err
	}
	return viewObject(d.model, index, img), nil
}

func viewObject(m *model.Model, index int, img []byte) MapView {
	stored := m.Objects[index]
	// A 2d-inverse KP map is shown as WinOLS shows it: its stored lines are
	// the columns.
	o := stored.AsColumnMajor()
	v := MapView{
		Index: index, ID: o.ID, Key: o.Key, Description: o.Description, Comment: o.Comment,
		Categories: []string{}, Shape: o.Shape, Inverse: stored.Inverse, Rows: o.Rows, Cols: o.Cols,
		Address:    fmt.Sprintf("0x%X", uint32(o.Address)),
		EndAddress: fmt.Sprintf("0x%X", uint64(o.Address)+uint64(o.Data.Extent(o.Rows, o.Cols))),
		Bits:       o.Data.Bits, Signed: o.Data.Signed, Float: o.Data.Float, Endian: endianOf(o.Data),
		ValueDescription: o.Value.Description, Units: o.Value.Units, Precision: o.Value.Precision,
		Factor: Num(o.Value.Conversion.Factor), Offset: Num(o.Value.Conversion.Offset),
		Reciprocal: o.Value.Conversion.Reciprocal, Base: o.View.Base,
		Cells: [][]Num{}, Raw: [][]Num{},
	}
	for _, id := range o.Categories {
		for _, c := range m.Categories {
			if c.ID == id {
				v.Categories = append(v.Categories, c.Name)
			}
		}
	}
	if b, err := json.MarshalIndent(stored, "", "  "); err == nil {
		v.JSON = string(b)
	}
	if o.Shape != "value" {
		v.X = viewAxis(o.X, o.Cols, img)
		if o.Shape == "2d" {
			v.Y = viewAxis(o.Y, o.Rows, img)
		}
	}
	cells, raw, note := readCells(o, img)
	v.Note = note
	if cells == nil {
		return v
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	v.Cells, v.Raw = make([][]Num, o.Rows), make([][]Num, o.Rows)
	for r := range o.Rows {
		v.Cells[r], v.Raw[r] = make([]Num, o.Cols), make([]Num, o.Cols)
		for c := range o.Cols {
			x := cells[r*o.Cols+c]
			v.Cells[r][c], v.Raw[r][c] = Num(x), Num(raw[r*o.Cols+c])
			if !math.IsNaN(x) && !math.IsInf(x, 0) {
				lo, hi = math.Min(lo, x), math.Max(hi, x)
			}
		}
	}
	v.CellMin, v.CellMax = Num(lo), Num(hi)
	return v
}

func order(d model.Data) binary.ByteOrder {
	if d.Endian == "big" {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

// rawCell reads one stored cell: an unsigned or sign-extended integer, or an
// IEEE single.
func rawCell(p []byte, d model.Data) float64 {
	w := d.Bits / 8
	var u uint64
	switch w {
	case 1:
		u = uint64(p[0])
	case 2:
		u = uint64(order(d).Uint16(p))
	default:
		u = uint64(order(d).Uint32(p))
	}
	switch {
	case d.Float:
		return float64(math.Float32frombits(uint32(u)))
	case d.Signed:
		return float64(int64(u<<(64-8*w)) >> (64 - 8*w))
	}
	return float64(u)
}

// fits reports whether rows by cols cells of d at addr lie inside the image.
func fits(addr uint64, rows, cols int, d model.Data, img []byte) bool {
	return rows >= 0 && cols >= 0 && addr+uint64(d.Extent(rows, cols)) <= uint64(len(img))
}

func widthOK(d model.Data) bool {
	return (d.Bits == 8 || d.Bits == 16 || d.Bits == 32) && (!d.Float || d.Bits == 32)
}

// readCells returns an object's cells in row order, converted and as stored,
// the way the engine's CSV writer reads them (mapcsv). Without them, the note
// says why.
func readCells(o *model.Object, img []byte) (cells, raw []float64, note string) {
	n := o.Rows * o.Cols
	switch {
	case n <= 0 || !widthOK(o.Data):
		return nil, nil, "unsupported"
	case n > maxCells:
		return nil, nil, "too-large"
	case img == nil:
		return nil, nil, "needs-image"
	case !fits(uint64(o.Address), o.Rows, o.Cols, o.Data, img):
		return nil, nil, "out-of-range"
	}
	// Cells are where TunerPro places them: packed, or by their strides.
	cells, raw = make([]float64, n), make([]float64, n)
	for i := range n {
		at := int(o.Address) + o.Data.CellOffset(i/o.Cols, i%o.Cols, o.Rows, o.Cols)
		raw[i] = rawCell(img[at:], o.Data)
		cells[i] = o.Value.Conversion.Apply(raw[i])
	}
	return cells, raw, ""
}

func viewAxis(x *model.Axis, n int, img []byte) AxisView {
	v := AxisView{Count: n, Source: "ordinal", Factor: 1, Values: []Num{}}
	if x != nil {
		v.Present = true
		v.Source, v.Stored = x.Source, x.Stored
		v.Description, v.Units, v.Precision = x.Value.Description, x.Value.Units, x.Value.Precision
		v.Factor, v.Offset = Num(x.Value.Conversion.Factor), Num(x.Value.Conversion.Offset)
		v.Reciprocal, v.Mirror = x.Value.Conversion.Reciprocal, x.Mirror
		if x.Address != nil {
			v.Address = fmt.Sprintf("0x%X", uint32(*x.Address))
		}
		if x.Data != nil {
			v.Bits, v.Signed, v.Float, v.Endian = x.Data.Bits, x.Data.Signed, x.Data.Float, endianOf(*x.Data)
		}
	}
	pts, note := axisPoints(x, n, img)
	v.Note = note
	for _, p := range pts {
		v.Values = append(v.Values, Num(p))
	}
	return v
}

// axisPoints returns an axis's n points in display units, following the
// engine's XDF label writer (xdf.axisPoints) and docs/kp-format.md, EEPROM
// storage: an axis not read from the image shows its fixed values, else its
// converted ordinals from 0; an image axis shows its stored points, summed or
// reversed as "stored" says.
func axisPoints(x *model.Axis, n int, img []byte) (pts []float64, note string) {
	if n <= 0 || n > maxCells {
		return nil, "unsupported"
	}
	c := model.Conversion{Factor: 1}
	if x != nil {
		c = x.Value.Conversion
	}
	if x == nil || x.Source != "image" {
		pts = make([]float64, n)
		for i := range n {
			if x != nil && i < len(x.Values) {
				pts[i] = x.Values[i] // already display units
			} else if x != nil && x.Source == "editable" && len(x.Values) > 0 {
				pts[i] = 0 // a fixed list shorter than the axis ends in zeros
			} else {
				pts[i] = c.Apply(float64(i))
			}
		}
		return pts, ""
	}
	switch {
	case x.Address == nil || x.Data == nil || !widthOK(*x.Data):
		return nil, "unsupported"
	case img == nil:
		return nil, "needs-image"
	case !fits(uint64(*x.Address), 1, n, *x.Data, img):
		return nil, "out-of-range"
	}
	// The points are one row of n cells, spaced as the axis's storage says.
	d, w, start := *x.Data, x.Data.Bits/8, int(*x.Address)
	at := func(i int) float64 { return rawCell(img[start+d.CellOffset(0, i, 1, n):], d) }
	raw := make([]float64, n)
	switch x.Stored {
	case "add", "subtract", "backwards":
		if d.Float {
			return nil, "unsupported"
		}
	}
	switch x.Stored {
	case "add": // point i is the sum of raw values 0 to i
		t := 0.0
		for i := range n {
			t += at(i)
			raw[i] = t
		}
	case "subtract": // point i is 2^bits minus the sum of raw values i to n-1
		t := math.Ldexp(1, 8*w)
		for i := n - 1; i >= 0; i-- {
			t -= at(i)
			raw[i] = t
		}
	case "backwards": // point i is raw[n-1-i]
		for i := range n {
			raw[i] = at(n - 1 - i)
		}
	default:
		for i := range n {
			raw[i] = at(i)
		}
	}
	pts = make([]float64, n)
	for i := range n {
		pts[i] = c.Apply(raw[i])
	}
	return pts, ""
}
