package kp

import "math"

// Map is one map definition. Fields are in file order (see codec for the tags).
// Undecoded fields are named as described in the package doc.
//
// The order and the header id each field appears from ("ge=N") follow the map
// serializer of WinOLS itself, read from a WinOLS 5 build: the file's header id
// is the format version it tests before each field. Every header walks the
// same record; later headers only add fields.
type Map struct {
	Index          int           `json:"index" kp:"-"`
	Key            string        `json:"-" kp:"-"`                           // model object key when the map came from model JSON; lint only
	Marked         byte          `json:"marked" kp:"hook"`                   // 0 or 1: WinOLS map-list marker (Insert/delete marker; Only marked / Only unmarked); before header 268 the record's first byte
	LinkID         int32         `json:"linkId,omitzero" kp:"ge=282"`        // -1: id of a linked map, -1 for none (unconfirmed)
	Comment        string        `json:"comment,omitzero" kp:"ge=287"`       //
	CommentText    TextExtra     `json:"commentText,omitzero" kp:"ge=330"`   //
	CommentFlag    byte          `json:"commentFlag,omitzero" kp:"hook"`     // 0 (unconfirmed); the record's first byte before header 268, kept there as Marked
	Name           string        `json:"name"`                               //
	NameText       TextExtra     `json:"nameText,omitzero" kp:"ge=330"`      //
	Organization   Org           `json:"organization"`                       //
	RightPane      int32         `json:"rightPane"`                          // the script's RWin: 0 none, 1 hex, 2 bars, 3 both (unconfirmed)
	Type           Type          `json:"type"`                               //
	Width          int32         `json:"width"`                              // element size in bytes; equals Type's width in every known pack
	Base           int32         `json:"base"`                               //
	FolderID       int32         `json:"folderId"`                           //
	ID             string        `json:"id"`                                 //
	IDPre          [2]int32      `json:"idPre,omitzero" kp:"hook"`           // header 298 and 299: 0 and 16 in every known pack (unconfirmed)
	IDPad          int32         `json:"idPad"`                              // 0 (unconfirmed)
	IDFlag         byte          `json:"idFlag"`                             // 0 (unconfirmed)
	IDState        int32         `json:"idState,omitzero" kp:"ge=123"`       // 0 or 1, mostly per pack (unconfirmed); was wrongly named Marked
	Range          [3][2]int64   `json:"range" kp:"hook"`                    // map properties value range, low and high, for 8-, 16- and 32-bit data (32-bit unconfirmed); float64 from header 300
	RangeExact     [3][2]float64 `json:"rangeExact,omitzero" kp:"-"`         // header 300 and later: a Range value that is not a whole number, as stored
	RangePad       [6]int64      `json:"rangePad,omitzero" kp:"-"`           // header 300 and later: the six former Range slots, written as zeros
	Reciprocal     bool          `json:"reciprocal"`                         // bKehrwert: physical = factor/raw + offset
	Signed         bool          `json:"signed"`                             // bVorzeichen (unconfirmed)
	Difference     bool          `json:"difference"`                         // bDelta: map window "difference" toggle (unconfirmed)
	Percent        bool          `json:"percent"`                            // bProzent: map window "percent" toggle (unconfirmed)
	Cols           int32         `json:"cols"`                               // elements per stored line
	Rows           int32         `json:"rows"`                               // stored lines
	Cursor         [2]int32      `json:"cursor"`                             // editor state, the last cursor cell (unconfirmed)
	Precision      int32         `json:"precision"`                          //
	Value          Value         `json:"value"`                              //
	Start          uint32        `json:"start"`                              //
	End            uint32        `json:"end"`                                //
	ImageSize      int32         `json:"imageSize"`                          // matches the image length in every known pack
	ImageSizePad   [2]int32      `json:"imageSizePad,omitzero" kp:"ge=264"`  // one float64, 0 or 1.0 (unconfirmed)
	Addr2          uint32        `json:"addr2"`                              // location as of the last properties edit (unconfirmed)
	Addr2Flags     [2]int32      `json:"addr2Flags"`                         // first -1 or 0 per pack, second 0 (unconfirmed)
	Addr2ImageSize int32         `json:"addr2ImageSize"`                     // the image length, or 0 (unconfirmed)
	Addr2Pad       int32         `json:"addr2Pad"`                           // 0 (unconfirmed)
	X              *Axis         `json:"x"`                                  //
	Y              *Axis         `json:"y"`                                  //
	YFlag          int32         `json:"yFlag"`                              // 0, 1 on a few maps: two flag bytes and the low half of a uint32 (unconfirmed)
	YPad           int16         `json:"yPad"`                               // 0: the high half of that uint32 (unconfirmed)
	StoredCols     int32         `json:"storedCols"`                         // columns and rows as the map window shows them: Rows and Cols on 2d-inverse maps
	StoredRows     int32         `json:"storedRows"`                         //
	ViewMode       int32         `json:"viewMode"`                           // 1 text, 2 2D, 3 3D (the script's ViewMode)
	ViewFlags      [2]byte       `json:"viewFlags"`                          // 1, 1 (unconfirmed)
	View2DScale    [2]float64    `json:"view2DScale"`                        // 2D view scaling, 1.0, 1.0 by default (unconfirmed)
	View2DRef      int32         `json:"view2DRef"`                          // -1 until the view mode is first changed, then 0 (unconfirmed meaning)
	ViewFlag       byte          `json:"viewFlag"`                           // 1, 0 on a few maps (unconfirmed)
	View3DRotation [2]float64    `json:"view3DRotation"`                     // 3D view angle and zoom, 300.0, 1.0 by default (unconfirmed)
	ViewScale      float64       `json:"viewScale"`                          // the 3D view's auto scale, 0 until first shown in 3D (unconfirmed)
	ViewScaleRef   int32         `json:"viewScaleRef"`                       // -1 exactly where ViewScale is 0 (unconfirmed)
	ViewOffset     float64       `json:"viewOffset"`                         // view state, with ViewScale (unconfirmed)
	Term2          Hex           `json:"term2" kp:"len=3"`                   // three flag bytes, 1, 1, 1 in nearly every map
	WindowRef      int32         `json:"windowRef,omitzero" kp:"ge=315"`     // -1 (unconfirmed)
	Window         [5]int32      `json:"window,omitzero" kp:"ge=383"`        // map window state, then its left, top, right and bottom (unconfirmed)
	SkipBytes      int32         `json:"skipBytes,omitzero" kp:"ge=329"`     // the script's SkipBytes: bytes passed over after each element
	SkipPad        [2]uint32     `json:"skipPad,omitzero" kp:"ge=346"`       // 0 (unconfirmed)
	LineSkipBytes  int32         `json:"lineSkipBytes,omitzero" kp:"ge=395"` // the script's LineSkipBytes: bytes passed over after each stored line
	ListRef        int32         `json:"listRef,omitzero" kp:"ge=476"`       // 0 (unconfirmed)
	ListValues     []int32       `json:"listValues,omitempty" kp:"ge=476,count"`
	ListTexts      []string      `json:"listTexts,omitempty" kp:"ge=476,count"`
	ListPad        int32         `json:"listPad,omitzero" kp:"ge=834"`   // (unconfirmed)
	TailFlags      uint32        `json:"tailFlags,omitzero" kp:"ge=503"` // 0 (unconfirmed)
	TailValue      float64       `json:"tailValue,omitzero" kp:"ge=503"` // 0 (unconfirmed)
	TailPad        uint32        `json:"tailPad,omitzero" kp:"ge=596"`   // 0 (unconfirmed)
}

// Header ids from which the fields coded by Map.hook appear.
const (
	headerFirstByte = 93  // the record's first byte
	headerMarked    = 268 // Marked moves ahead of it
	headerIDPre0    = 298
	headerIDPre1    = 299
	headerRangeF64  = 300 // Range as float64, then six unused slots
)

// hook codes the map fields whose form changes with the header.
func (m *Map) hook(c *codec, field string) {
	v := c.headerID
	switch field {
	case "Marked":
		if v >= headerMarked {
			c.u8(&m.Marked)
		}
	case "CommentFlag":
		switch {
		case v >= headerMarked:
			c.u8(&m.CommentFlag)
		case v >= headerFirstByte:
			c.u8(&m.Marked)
		}
	case "IDPre":
		if v >= headerIDPre0 {
			c.i32(&m.IDPre[0])
		}
		if v >= headerIDPre1 {
			c.i32(&m.IDPre[1])
		}
	case "Range":
		if v < headerRangeF64 {
			c.array(&m.Range)
			return
		}
		for i := range m.Range {
			for j := range m.Range[i] {
				f := float64(m.Range[i][j])
				if x := m.RangeExact[i][j]; x != 0 {
					f = x
				}
				c.f64(&f)
				if c.enc {
					continue
				}
				m.Range[i][j], m.RangeExact[i][j] = 0, 0
				if r := math.Round(f); math.Abs(r) < 1<<62 {
					m.Range[i][j] = int64(r)
					if r != f {
						m.RangeExact[i][j] = f
					}
				}
			}
		}
		for i := range m.RangePad {
			c.i64(&m.RangePad[i])
		}
	}
}

func (m *Map) check(c *codec) {
	for _, b := range m.Term2 {
		if b > 1 {
			c.pos -= len(m.Term2)
			c.fail("unexpected term2 %v", m.Term2)
		}
	}
	twoD := m.Organization == OrgTwoD || m.Organization == OrgTwoDInv
	m.X.Defined = twoD || m.Organization == OrgOneD
	m.Y.Defined = twoD
}

// TextExtra is what header 330 and later store after a map name, a map comment
// and an axis description: two ids, then other versions of the text.
// Every known pack has zeros and no other versions (unconfirmed meaning).
type TextExtra struct {
	IDs [2]int32  `json:"ids,omitzero" kp:"ge=345"`
	Alt []AltText `json:"alt,omitempty" kp:"count"`
}

// AltText is one other version of a text (TextExtra).
type AltText struct {
	Text string `json:"text" kp:"ge=345"`
	ID   int32  `json:"id" kp:"ge=345"`
}

// IsZero reports the empty extra, so JSON leaves it out.
func (t TextExtra) IsZero() bool { return t.IDs == [2]int32{} && len(t.Alt) == 0 }

// Text is a text with its TextExtra, as the axis text list stores it.
type Text struct {
	Text  string    `json:"text"`
	Extra TextExtra `json:"extra,omitzero" kp:"ge=330"`
}

// Value is the scaling block shared by maps and axes.
type Value struct {
	Description string  `json:"description"`
	Units       string  `json:"units"`
	Factor      float64 `json:"factor"`
	Offset      float64 `json:"offset"`
}

// Axis is an X or Y axis record. Every map stores both; Defined is set on the axes
// its organisation implies (X for 1D, X and Y for 2D). The unused slots (the second
// axis of a 1D map, both axes of a single value) are still parsed.
// The order and header ids follow the axis serializer of WinOLS itself (see Map).
type Axis struct {
	Defined         bool       `json:"defined" kp:"-"`
	Value           Value      `json:"value" kp:"hook"`                    // the description carries a TextExtra from header 330
	DescriptionText TextExtra  `json:"descriptionText,omitzero" kp:"-"`    //
	DataSource      DataSource `json:"dataSource"`                         //
	Addr            uint32     `json:"addr"`                               //
	Type            Type       `json:"type"`                               //
	Width           int32      `json:"width"`                              // element size in bytes; equals Type's width on all but 2 known axes
	Base            int32      `json:"base"`                               //
	Mirror          bool       `json:"mirror"`                             // bRueckwaerts: the axis and the map along it are shown in reverse order
	Reciprocal      bool       `json:"reciprocal"`                         // bKehrwert: physical = factor/raw + offset
	ReciprocalPad   float64    `json:"reciprocalPad,omitzero" kp:"ge=264"` // 0, 32.0 on two known axes (unconfirmed)
	SkipBytes       int32      `json:"skipBytes,omitzero" kp:"ge=241"`     // the script's SkipBytes: bytes passed over after each axis element
	SkipPad         int32      `json:"skipPad,omitzero" kp:"ge=834"`       // (unconfirmed)
	Precision       int32      `json:"precision"`                          // -1 on some unused slots
	Signed          bool       `json:"signed"`                             //
	Values          Hex        `json:"values" kp:"bytes"`                  // free editable values in the axis type, 4 bytes per point
	DataHeader      int32      `json:"dataHeader"`                         // the script's DataHeader: header bytes before the axis data (unconfirmed)
	SignatureByte   int32      `json:"signatureByte"`                      // the script's SignaturByte (marker before the axis), -1 for none (unconfirmed)
	ID              string     `json:"id,omitzero" kp:"ge=354"`            // the script's IdName
	Texts           []Text     `json:"texts,omitempty" kp:"ge=372,count"`  // (unconfirmed)
	IDPad           int32      `json:"idPad,omitzero" kp:"ge=440"`         // 0 (unconfirmed)
	TailPad         [2]int32   `json:"tailPad,omitzero" kp:"ge=834"`       // (unconfirmed)
	TailValues      []int32    `json:"tailValues,omitempty" kp:"ge=834,count"`
	TailTexts       []string   `json:"tailTexts,omitempty" kp:"ge=834,count"`
}

// headerTextExtra is the header id from which a TextExtra follows a text.
const headerTextExtra = 330

// hook codes Value: description, its TextExtra, units, factor and offset.
func (a *Axis) hook(c *codec, _ string) {
	c.named("description", func() { c.str(&a.Value.Description) })
	if c.headerID >= headerTextExtra {
		c.record(&a.DescriptionText)
	}
	c.named("units", func() { c.str(&a.Value.Units) })
	c.named("factor", func() { c.f64(&a.Value.Factor) })
	c.named("offset", func() { c.f64(&a.Value.Offset) })
}
