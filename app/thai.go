package main

import (
	"strings"

	"go.nyet.org/xdfkit/api"
	"go.nyet.org/xdfkit/lint"
	"go.nyet.org/xdfkit/model"
)

// Thai text in map definitions.
//
// KP and XDF hold text one byte per character. WinOLS and TunerPro on a Thai
// Windows write Thai in Windows-874: TunerPro spells each such byte as a
// numeric reference of the byte value ("&#225;&#161;" is "แก"), not of a
// Unicode code point. The engine reads those bytes as Western text
// (ISO-8859-15 in a KP, Latin-1 in an XDF), so Thai arrives as accented Latin
// letters, and eight Thai letters can't cross from one format to the other at
// all, because the two Western sets differ at exactly those bytes.
//
// The app therefore turns that text into real Thai after the engine reads a
// file, and back into the bytes each format wants before the engine writes
// one. Text that isn't Thai is left exactly as the engine made it.

// latin9 are the bytes ISO-8859-15 reads differently from Latin-1 (kp/codec.go).
var latin9 = map[byte]rune{0xA4: '€', 0xA6: 'Š', 0xA8: 'š', 0xB4: 'Ž', 0xB8: 'ž', 0xBC: 'Œ', 0xBD: 'œ', 0xBE: 'Ÿ'}

var latin9Byte = func() map[rune]byte {
	m := map[rune]byte{}
	for b, r := range latin9 {
		m[r] = b
	}
	return m
}()

// cp1252High are the Windows-1252 characters at 0x80 to 0x9F, as the engine's
// XDF reader decodes raw bytes there (xdf/read.go).
var cp1252High = []rune{
	0x20AC, 0x0081, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021,
	0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0x008D, 0x017D, 0x008F,
	0x0090, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
	0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0x009D, 0x017E, 0x0178,
}

// cp874Low are the Windows-874 characters below the Thai block.
var cp874Low = map[byte]rune{
	0x80: 0x20AC, 0x85: 0x2026, 0x91: 0x2018, 0x92: 0x2019, 0x93: 0x201C,
	0x94: 0x201D, 0x95: 0x2022, 0x96: 0x2013, 0x97: 0x2014, 0xA0: 0x00A0,
}

// thaiByte reports a byte that is a Thai character in Windows-874.
func thaiByte(b byte) bool { return b >= 0xA1 && b <= 0xDA || b >= 0xDF && b <= 0xFB }

// fromCP874 is the character a Windows-874 byte stands for.
func fromCP874(b byte) (rune, bool) {
	switch {
	case b < 0x80:
		return rune(b), true
	case b >= 0xA1 && b <= 0xDA:
		return 0x0E01 + rune(b-0xA1), true
	case b >= 0xDF && b <= 0xFB:
		return 0x0E3F + rune(b-0xDF), true
	}
	r, ok := cp874Low[b]
	return r, ok
}

// toCP874 is the Windows-874 byte of a character.
func toCP874(r rune) (byte, bool) {
	switch {
	case r < 0x80:
		return byte(r), true
	case r >= 0x0E01 && r <= 0x0E3A:
		return 0xA1 + byte(r-0x0E01), true
	case r >= 0x0E3F && r <= 0x0E5B:
		return 0xDF + byte(r-0x0E3F), true
	}
	for b, c := range cp874Low {
		if c == r {
			return b, true
		}
	}
	return 0, false
}

func isThai(r rune) bool { return r >= 0x0E01 && r <= 0x0E5B }

func hasThai(s string) bool { return strings.IndexFunc(s, isThai) >= 0 }

// sourceByte is the byte the engine read to make character r, for a format
// whose text is one byte per character.
func sourceByte(r rune, format string) (byte, bool) {
	if r < 0x80 {
		return byte(r), true
	}
	switch format {
	case api.KP:
		if b, ok := latin9Byte[r]; ok {
			return b, true
		}
		if _, differs := latin9[byte(r)]; r < 0x100 && !differs {
			return byte(r), true
		}
	case api.XDF, api.A2L:
		if r < 0x100 {
			return byte(r), true
		}
		for i, c := range cp1252High {
			if c == r {
				return 0x80 + byte(i), true
			}
		}
	}
	return 0, false
}

// writerRune is the character that makes the engine's writer for a format put
// out byte b.
func writerRune(b byte, format string) rune {
	if format == api.KP {
		if r, ok := latin9[b]; ok {
			return r
		}
	}
	return rune(b)
}

// looksThai reports bytes that read as Thai rather than Western text: three
// Thai-range bytes in a row, or two with one of them in 0xA1 to 0xBF. Western
// text has its accented letters apart ("Zündwinkel", "Größe"), and the bytes
// from 0xA1 to 0xBF are symbols there, which don't come in pairs with letters.
func looksThai(b []byte) bool {
	for i := 0; i+1 < len(b); i++ {
		if !thaiByte(b[i]) || !thaiByte(b[i+1]) {
			continue
		}
		if b[i] <= 0xBF || b[i+1] <= 0xBF || i+2 < len(b) && thaiByte(b[i+2]) {
			return true
		}
	}
	return false
}

// decodeThai turns text the engine read from a byte-per-character format into
// Thai, when its bytes are Thai in Windows-874. Other text comes back as it is.
func decodeThai(s, format string) (string, bool) {
	ascii := true
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			ascii = false
			break
		}
	}
	if ascii || hasThai(s) {
		return s, false
	}
	bytes := make([]byte, 0, len(s))
	for _, r := range s {
		b, ok := sourceByte(r, format)
		if !ok {
			return s, false
		}
		bytes = append(bytes, b)
	}
	if !looksThai(bytes) {
		return s, false
	}
	var out strings.Builder
	i := 0
	for _, r := range s {
		if c, ok := fromCP874(bytes[i]); ok {
			out.WriteRune(c)
		} else {
			out.WriteRune(r) // no such Windows-874 character: keep what the engine read
		}
		i++
	}
	return out.String(), true
}

// encodeThai turns text holding Thai into the characters that make the
// engine's writer put out its Windows-874 bytes. Text without Thai comes back
// as it is. lost counts characters Windows-874 doesn't have, written as "?".
func encodeThai(s, format string) (out string, lost int, changed bool) {
	if !hasThai(s) {
		return s, 0, false
	}
	var b strings.Builder
	for _, r := range s {
		c, ok := toCP874(r)
		if !ok {
			c, lost = '?', lost+1
		}
		b.WriteRune(writerRune(c, format))
	}
	return b.String(), lost, true
}

// eachText calls f on every text field of a model.
func eachText(m *model.Model, f func(*string)) {
	value := func(v *model.Value) {
		f(&v.Description)
		f(&v.Units)
	}
	axis := func(x *model.Axis) {
		if x != nil {
			value(&x.Value)
		}
	}
	f(&m.Project.Name)
	f(&m.Project.Version)
	for _, c := range m.Categories {
		f(&c.Name)
	}
	for _, o := range m.Objects {
		f(&o.Key)
		f(&o.ID)
		f(&o.Description)
		f(&o.Comment)
		value(&o.Value)
		axis(o.X)
		axis(o.Y)
		for i := range o.Alternates {
			alt := &o.Alternates[i]
			f(&alt.ID)
			f(&alt.Description)
			value(&alt.Value)
			if alt.X != nil {
				value(alt.X)
			}
			if alt.Y != nil {
				value(alt.Y)
			}
		}
	}
}

// byteText reports a format whose text is one byte per character.
func byteText(format string) bool { return format == api.KP || format == api.XDF || format == api.A2L }

// readThai makes the Thai text of a model read from format readable, and
// returns how many texts it changed.
func readThai(m *model.Model, format string) int {
	if !byteText(format) {
		return 0
	}
	n := 0
	eachText(m, func(s *string) {
		if t, ok := decodeThai(*s, format); ok {
			*s, n = t, n+1
		}
	})
	return n
}

// writeThai prepares a model's Thai text for the engine's KP or XDF writer.
// It returns how many texts it changed and how many characters were lost.
func writeThai(m *model.Model, format string) (changed, lost int) {
	eachText(m, func(s *string) {
		if t, l, ok := encodeThai(*s, format); ok {
			*s, changed, lost = t, changed+1, lost+l
		}
	})
	return changed, lost
}

// modelHasThai reports Thai text anywhere in a model.
func modelHasThai(m *model.Model) bool {
	found := false
	eachText(m, func(s *string) { found = found || hasThai(*s) })
	return found
}

// readableFindings makes the map names in lint findings of a KP file readable.
func readableFindings(fs []lint.Finding, format string) {
	if !byteText(format) {
		return
	}
	for i := range fs {
		for j := range fs[i].Maps {
			ref := &fs[i].Maps[j]
			ref.ID, _ = decodeThai(ref.ID, format)
			ref.Name, _ = decodeThai(ref.Name, format)
		}
	}
}
