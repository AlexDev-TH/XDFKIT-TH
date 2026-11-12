package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go.nyet.org/xdfkit/api"
	"go.nyet.org/xdfkit/canon"
	"go.nyet.org/xdfkit/model"
)

// The XDF as a plain XML reader sees it, apart from the engine's own reader:
// what TunerPro is given.
type xdfDoc struct {
	Defaults struct {
		SigDigits *int `xml:"sigdigits,attr"`
	} `xml:"XDFHEADER>DEFAULTS"`
	Categories []struct {
		Index string `xml:"index,attr"`
		Name  string `xml:"name,attr"`
	} `xml:"XDFHEADER>CATEGORY"`
	Tables    []xdfItem `xml:"XDFTABLE"`
	Constants []xdfItem `xml:"XDFCONSTANT"`
}

type xdfItem struct {
	UID   string `xml:"uniqueid,attr"`
	Title string `xml:"title"`
	Desc  string `xml:"description"`
	Cats  []struct {
		Category int `xml:"category,attr"`
	} `xml:"CATEGORYMEM"`
	Axes []xdfAxis `xml:"XDFAXIS"`
	xdfAxis
}

type xdfAxis struct {
	ID   string `xml:"id,attr"`
	Data struct {
		Flags string `xml:"mmedtypeflags,attr"`
		Addr  string `xml:"mmedaddress,attr"`
		Bits  int    `xml:"mmedelementsizebits,attr"`
		Rows  int    `xml:"mmedrowcount,attr"`
		Cols  int    `xml:"mmedcolcount,attr"`
	} `xml:"EMBEDDEDDATA"`
	Units string `xml:"units"`
	Dec   *int   `xml:"decimalpl"`
	Count int    `xml:"indexcount"`
	Math  []struct {
		Eq  string `xml:"equation,attr"`
		Row string `xml:"row,attr"`
	} `xml:"MATH"`
	Labels []struct {
		Value string `xml:"value,attr"`
	} `xml:"LABEL"`
}

func (a *xdfAxis) axis(id string, axes []xdfAxis) *xdfAxis {
	for i := range axes {
		if axes[i].ID == id {
			return &axes[i]
		}
	}
	return nil
}

func (a *xdfAxis) equation() string {
	for _, m := range a.Math {
		if m.Row == "" {
			return m.Eq
		}
	}
	return ""
}

// calc evaluates a TunerPro equation in X with ordinary arithmetic, with no
// assumption about its shape.
type calc struct {
	s string
	i int
	x float64
}

func (c *calc) peek() byte {
	for c.i < len(c.s) && c.s[c.i] == ' ' {
		c.i++
	}
	if c.i == len(c.s) {
		return 0
	}
	return c.s[c.i]
}

func (c *calc) sum() float64 {
	v := c.product()
	for {
		switch c.peek() {
		case '+':
			c.i++
			v += c.product()
		case '-':
			c.i++
			v -= c.product()
		default:
			return v
		}
	}
}

func (c *calc) product() float64 {
	v := c.factor()
	for {
		switch c.peek() {
		case '*':
			c.i++
			v *= c.factor()
		case '/':
			c.i++
			v /= c.factor()
		default:
			return v
		}
	}
}

func (c *calc) factor() float64 {
	switch c.peek() {
	case '-':
		c.i++
		return -c.factor()
	case '(':
		c.i++
		v := c.sum()
		c.peek()
		c.i++
		return v
	case 'X':
		c.i++
		return c.x
	}
	j := c.i
	for j < len(c.s) && (strings.IndexByte("0123456789.eE", c.s[j]) >= 0 || (c.s[j] == '-' || c.s[j] == '+') && j > c.i && (c.s[j-1] == 'e' || c.s[j-1] == 'E')) {
		j++
	}
	v, err := strconv.ParseFloat(c.s[c.i:j], 64)
	if err != nil {
		return math.NaN()
	}
	c.i = j
	return v
}

func evalEq(eq string, x float64) float64 {
	if strings.TrimSpace(eq) == "" {
		return x
	}
	c := &calc{s: eq, x: x}
	v := c.sum()
	if c.peek() != 0 {
		return math.NaN()
	}
	return v
}

func sameNumber(a, b float64) bool {
	return a == b || math.Abs(a-b) <= 1e-9*math.Max(1, math.Max(math.Abs(a), math.Abs(b)))
}

func sameFormula(eq string, c model.Conversion) bool {
	for _, x := range []float64{1, 7, 100, 255, 4096} {
		if !sameNumber(evalEq(eq, x), c.Apply(x)) {
			return false
		}
	}
	return true
}

func flags(s string) uint64 {
	v, _ := strconv.ParseUint(strings.TrimPrefix(strings.TrimSpace(s), "0x"), 16, 64)
	return v
}

func oneSpace(s string) string { return strings.Join(strings.Fields(s), " ") }

// compareXDF checks every table of a written XDF against the model it was
// written from, field by field, and returns what differs.
func compareXDF(m *model.Model, xdfBytes []byte) (diffs []string, checked int) {
	var doc xdfDoc
	if err := xml.NewDecoder(bytes.NewReader(xdfBytes)).Decode(&doc); err != nil {
		return []string{"XDF is not XML: " + err.Error()}, 0
	}
	catName := map[int]string{}
	for _, c := range m.Categories {
		catName[c.ID] = c.Name
	}
	items := map[uint64]*xdfItem{}
	for _, list := range [][]xdfItem{doc.Tables, doc.Constants} {
		for i := range list {
			items[flags(list[i].UID)] = &list[i]
		}
	}
	// Thai in an XDF is Windows-874 bytes; read it as the app does.
	thai := func(s string) string {
		t, _ := decodeThai(s, api.XDF)
		return t
	}
	for i := range doc.Categories {
		doc.Categories[i].Name = thai(doc.Categories[i].Name)
	}
	for _, it := range items {
		it.Title, it.Desc, it.Units = thai(it.Title), thai(it.Desc), thai(it.Units)
		for k := range it.Axes {
			it.Axes[k].Units = thai(it.Axes[k].Units)
		}
	}
	diff := func(o *model.Object, format string, a ...any) {
		if len(diffs) < 40 {
			diffs = append(diffs, fmt.Sprintf("%q: ", o.ID)+fmt.Sprintf(format, a...))
		} else if len(diffs) == 40 {
			diffs = append(diffs, "...")
		}
	}
	for i, o := range m.Objects {
		it := items[uint64(i+1)]
		if it == nil {
			diff(o, "no table with uniqueid %#x in the XDF", i+1)
			continue
		}
		checked++
		// Names.
		want := o.ID
		if name := firstLine(o.Description); name != "" && o.Description != o.ID && o.ID != "" {
			want += titleSep + name
		}
		if o.ID != "" && oneSpace(it.Title) != oneSpace(want) {
			diff(o, "title %q, want %q", it.Title, want)
		}
		if name := firstLine(o.Description); name != "" && o.ID != "" && !strings.HasPrefix(oneSpace(it.Desc), oneSpace(name)) {
			diff(o, "description %q does not start with the name %q", it.Desc, name)
		}
		if c := strings.TrimSpace(o.Comment); c != "" && !strings.Contains(oneSpace(it.Desc), oneSpace(c)) {
			diff(o, "description %q lacks the comment %q", it.Desc, c)
		}
		if len(o.Categories) > 0 {
			got := ""
			if len(it.Cats) > 0 && it.Cats[0].Category >= 1 && it.Cats[0].Category <= len(doc.Categories) {
				got = doc.Categories[it.Cats[0].Category-1].Name
			}
			if oneSpace(got) != oneSpace(catName[o.Categories[0]]) {
				diff(o, "category %q, want %q", got, catName[o.Categories[0]])
			}
		}
		// Cells.
		z := &it.xdfAxis
		if o.Shape != "value" {
			if z = z.axis("z", it.Axes); z == nil {
				diff(o, "no z axis")
				continue
			}
			if max(z.Data.Rows, 1) != o.Rows || max(z.Data.Cols, 1) != o.Cols {
				diff(o, "cells %dx%d, want %dx%d", max(z.Data.Cols, 1), max(z.Data.Rows, 1), o.Cols, o.Rows)
			}
		}
		f := flags(z.Data.Flags)
		if flags(z.Data.Addr) != uint64(o.Address) {
			diff(o, "address %s, want %#x", z.Data.Addr, uint32(o.Address))
		}
		if z.Data.Bits != o.Data.Bits {
			diff(o, "%d bit cells, want %d", z.Data.Bits, o.Data.Bits)
		}
		if (f&1 != 0) != o.Data.Signed {
			diff(o, "signed %v, want %v", f&1 != 0, o.Data.Signed)
		}
		if o.Data.Bits > 8 && (f&2 != 0) != (o.Data.Endian != "big") {
			diff(o, "low byte first %v, want %v (%s)", f&2 != 0, o.Data.Endian != "big", o.Data.Endian)
		}
		if (f&0x10000 != 0) != o.Data.Float {
			diff(o, "float %v, want %v", f&0x10000 != 0, o.Data.Float)
		}
		if !sameFormula(z.equation(), o.Value.Conversion) {
			diff(o, "formula %q, want %+v", z.equation(), o.Value.Conversion)
		}
		// A table without its own decimal places takes the header's.
		gotDec := 2
		if doc.Defaults.SigDigits != nil {
			gotDec = *doc.Defaults.SigDigits
		}
		if z.Dec != nil {
			gotDec = *z.Dec
		}
		if wantDec := min(max(o.Value.Precision, 0), 6); gotDec != wantDec {
			diff(o, "%d decimal places, want %d", gotDec, wantDec)
		}
		if strings.TrimSpace(z.Units) != strings.TrimSpace(o.Value.Units) {
			diff(o, "units %q, want %q", z.Units, o.Value.Units)
		}
		// Axes.
		if o.Shape == "value" {
			continue
		}
		for _, ax := range []struct {
			id string
			x  *model.Axis
			n  int
		}{{"x", o.X, o.Cols}, {"y", o.Y, o.Rows}} {
			a := z.axis(ax.id, it.Axes)
			if a == nil {
				diff(o, "no %s axis", ax.id)
				continue
			}
			if a.Count != ax.n {
				diff(o, "%s axis has %d points, want %d", ax.id, a.Count, ax.n)
			}
			x := ax.x
			switch {
			case x != nil && x.Source == "image" && (x.Stored == "" || x.Stored == "absolute"):
				if x.Address == nil || flags(a.Data.Addr) != uint64(*x.Address) {
					diff(o, "%s axis address %s, want %v", ax.id, a.Data.Addr, x.Address)
				}
				if x.Data != nil && a.Data.Bits != x.Data.Bits {
					diff(o, "%s axis %d bit, want %d", ax.id, a.Data.Bits, x.Data.Bits)
				}
				if x.Data != nil && (flags(a.Data.Flags)&1 != 0) != x.Data.Signed {
					diff(o, "%s axis signed %v, want %v", ax.id, flags(a.Data.Flags)&1 != 0, x.Data.Signed)
				}
				if !sameFormula(a.equation(), x.Value.Conversion) {
					diff(o, "%s axis formula %q, want %+v", ax.id, a.equation(), x.Value.Conversion)
				}
				if strings.TrimSpace(a.Units) != strings.TrimSpace(x.Value.Units) {
					diff(o, "%s axis units %q, want %q", ax.id, a.Units, x.Value.Units)
				}
			case x != nil && x.Source == "editable" && len(x.Values) == ax.n:
				for k, l := range a.Labels {
					if v, err := strconv.ParseFloat(l.Value, 64); err != nil || k >= len(x.Values) || math.Abs(v-x.Values[k]) > 0.5*math.Pow(10, -6) {
						diff(o, "%s axis label %d is %q, want %v", ax.id, k, l.Value, x.Values[k])
						break
					}
				}
			}
		}
	}
	return diffs, checked
}

// TestKPToXDFExact: every map of a real pack reaches the XDF with its name,
// category, address, size, storage, formula, decimal places, units and axes,
// read back by a plain XML reader; and the XDF converts back to a KP that
// holds the same definitions as the pack.
func TestKPToXDFExact(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	out := filepath.Join(dir, "pack.xdf")
	if r := a.Convert(ConvertJob{Input: packKP, To: api.XDF, Output: out, Image: packBin, WriteMeta: true}); r.Error != "" {
		t.Fatal(r.Error)
	}
	d, err := a.loadDef(packKP)
	if err != nil {
		t.Fatal(err)
	}
	diffs, checked := compareXDF(d.model, read(t, out))
	for _, s := range diffs {
		t.Error(s)
	}
	if checked != len(d.model.Objects) || checked < 4000 {
		t.Errorf("%d of %d objects compared", checked, len(d.model.Objects))
	}
	// The title is what WinOLS shows: the id with the map's name.
	if !strings.Contains(string(read(t, out)), "<title>NZMSMN: untere Drehzahlgrenze f&#x00FC;r ZMS-Schutz</title>") {
		t.Error("NZMSMN is not titled with its name")
	}
	if !strings.Contains(string(read(t, out)), "<decimalpl>3</decimalpl>") {
		t.Error("no value keeps three decimal places")
	}

	// Back to a KP: the same definitions as the pack, by the engine's own reading.
	back := filepath.Join(dir, "back.kp")
	if r := a.Convert(ConvertJob{Input: out, To: api.KP, Output: back}); r.Error != "" {
		t.Fatal(r.Error)
	}
	want, got := engineRead(t, packKP), engineRead(t, back)
	if len(want.Objects) != len(got.Objects) {
		t.Fatalf("%d objects came back, the pack has %d", len(got.Objects), len(want.Objects))
	}
	different := 0
	for i := range want.Objects {
		w, _ := canon.Marshal(want.Objects[i])
		g, _ := canon.Marshal(got.Objects[i])
		if !bytes.Equal(w, g) {
			if different++; different <= 5 {
				t.Errorf("object %d differs after the round trip:\n pack %s\n back %s", i, w, g)
			}
		}
	}
	if different > 0 {
		t.Errorf("%d of %d objects differ after KP to XDF to KP", different, len(want.Objects))
	}
}
