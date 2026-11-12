package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.nyet.org/xdfkit/api"
	"go.nyet.org/xdfkit/canon"
	"go.nyet.org/xdfkit/model"
)

// latin1 spells Windows-874 bytes the way the engine's XDF reader hands them
// over: one Latin-1 character per byte.
func latin1(b ...byte) string {
	r := make([]rune, len(b))
	for i, c := range b {
		r[i] = rune(c)
	}
	return string(r)
}

func TestDecodeThai(t *testing.T) {
	// The start of a description TunerPro wrote on a Thai Windows:
	// &#225;&#161;&#185;&#225;&#185;&#199;&#185;&#205;&#185;
	got, ok := decodeThai(latin1(225, 161, 185, 225, 185, 199, 185, 205, 185), api.XDF)
	if !ok || got != "แกนแนวนอน" {
		t.Errorf("XDF text: %q, %v", got, ok)
	}
	// A KP holds the same bytes; the engine reads eight of them as other
	// characters (ISO-8859-15), here 0xB4, the Thai letter ด.
	if got, ok := decodeThai("Ž"+latin1(0xE9, 0xD2, 0xB9), api.KP); !ok || got != "ด้าน" {
		t.Errorf("KP text: %q, %v", got, ok)
	}
	// Mixed with ASCII, as map names are.
	if got, ok := decodeThai("IGN "+latin1(0xCD, 0xA7, 0xC8, 0xD2)+" 1-4", api.XDF); !ok || got != "IGN องศา 1-4" {
		t.Errorf("mixed text: %q, %v", got, ok)
	}
}

func TestWesternTextStays(t *testing.T) {
	for _, s := range []string{
		"Zündwinkelkennfeld", "Größe", "Öltemperatur für Ölüberwachung", "grad KW", "°KW", "m³/h",
		"créée", "Übertragungsfunktion", "Regelgröße ß", "Lambda λ", "plain ascii", "",
	} {
		for _, format := range []string{api.KP, api.XDF, api.A2L} {
			if got, ok := decodeThai(s, format); ok || got != s {
				t.Errorf("%s text %q read as Thai: %q", format, s, got)
			}
		}
	}
	// Text that is already Thai, or came from a format that holds Unicode, is not touched.
	if got, ok := decodeThai("องศา", api.XDF); ok || got != "องศา" {
		t.Errorf("Thai text decoded again: %q", got)
	}
	m := &model.Model{Objects: []*model.Object{{ID: latin1(225, 161, 185)}}}
	if n := readThai(m, api.JSON); n != 0 {
		t.Errorf("JSON text decoded: %d", n)
	}
}

func TestEncodeThai(t *testing.T) {
	// For an XDF the writer gets one Latin-1 character per Windows-874 byte.
	if got, lost, ok := encodeThai("ดี A", api.XDF); !ok || lost != 0 || got != latin1(0xB4, 0xD5)+" A" {
		t.Errorf("XDF: %q lost %d %v", got, lost, ok)
	}
	// For a KP, the ISO-8859-15 character of each byte: 0xB4 is Ž there.
	if got, lost, ok := encodeThai("ดี", api.KP); !ok || lost != 0 || got != "Ž"+latin1(0xD5) {
		t.Errorf("KP: %q lost %d %v", got, lost, ok)
	}
	// Windows-874 has no degree sign: it is written as "?" and counted.
	if got, lost, _ := encodeThai("องศา °", api.XDF); lost != 1 || !strings.HasSuffix(got, "?") {
		t.Errorf("lost character: %q lost %d", got, lost)
	}
	// Text without Thai is the engine's business.
	if got, _, ok := encodeThai("Zündwinkel °KW", api.KP); ok || got != "Zündwinkel °KW" {
		t.Errorf("Western text changed: %q", got)
	}
	// Every Thai character survives the trip through either format.
	for r := rune(0x0E01); r <= 0x0E5B; r++ {
		if r > 0x0E3A && r < 0x0E3F {
			continue
		}
		s := "ก" + string(r) + "ก"
		for _, format := range []string{api.KP, api.XDF} {
			enc, lost, _ := encodeThai(s, format)
			if back, ok := decodeThai(enc, format); !ok || lost != 0 || back != s {
				t.Errorf("%s: %U came back as %q", format, r, back)
			}
		}
	}
}

// TestThaiThroughFormats: Thai text written to an XDF is spelled as TunerPro
// spells it, crosses into a KP as Windows-874 bytes, and reads back as Thai
// from both.
func TestThaiThroughFormats(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	const name, desc, cat = "IGN องศา", "แกนแนวนอนของตาราง ด้านจุดระเบิด", "จูนไฟ"
	m := &model.Model{
		Schema:     model.SchemaID,
		Categories: []*model.Category{{ID: 1, Name: cat}},
		Objects: []*model.Object{{
			ID: name, Description: desc, Categories: []int{1}, Shape: "1d", Address: 0x10, Rows: 1, Cols: 4,
			Data:  model.Data{Bits: 8},
			Value: model.Value{Units: "องศา", Conversion: model.Conversion{Factor: 1}},
			X:     &model.Axis{Source: "ordinal", Value: model.Value{Units: "รอบ", Conversion: model.Conversion{Factor: 1}}},
		}},
	}
	doc, err := canon.MarshalStamped(m, "test")
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "thai.json")
	if err := os.WriteFile(src, doc, 0o644); err != nil {
		t.Fatal(err)
	}
	convert := func(in, to string) string {
		t.Helper()
		r := a.Convert(ConvertJob{Input: in, To: to, WriteMeta: true})
		if r.Error != "" {
			t.Fatalf("%s to %s: %s", filepath.Base(in), to, r.Error)
		}
		for _, w := range r.Warnings {
			if strings.HasPrefix(w, "[[thai-lost]]") {
				t.Errorf("%s to %s: %s", filepath.Base(in), to, w)
			}
		}
		return r.Output
	}

	// JSON to XDF: each Thai character is a reference of its Windows-874 byte
	// (แ is 0xE1, ก is 0xA1), never of its Unicode code point.
	xdfPath := convert(src, api.XDF)
	xdfText := string(read(t, xdfPath))
	if !strings.Contains(xdfText, "&#x00E1;&#x00A1;&#x00B9;") {
		t.Errorf("the XDF doesn't spell Thai as Windows-874 bytes:\n%s", xdfText)
	}
	if strings.Contains(xdfText, "&#x0E") {
		t.Error("the XDF holds Thai as Unicode code points, which is not how TunerPro writes it")
	}

	// XDF to KP: the step that failed before ("U+00B4 is not in ISO-8859-15").
	// The KP holds the same Windows-874 bytes.
	kpPath := filepath.Join(dir, "thai.kp")
	if r := a.Convert(ConvertJob{Input: xdfPath, To: api.KP, Output: kpPath}); r.Error != "" {
		t.Fatalf("XDF to KP: %s", r.Error)
	}
	// The engine alone reads those bytes as ISO-8859-15, which shows what the
	// KP holds: the Windows-874 bytes of the Thai text, one character each.
	raw, _, err := api.Convert(read(t, kpPath), api.ConvertRequest{To: api.JSON, Name: "thai.kp"})
	if err != nil {
		t.Fatal(err)
	}
	engine := new(model.Model)
	if err := canon.Unmarshal(raw, engine); err != nil {
		t.Fatal(err)
	}
	if want, _, _ := encodeThai(desc, api.KP); engine.Objects[0].Description != want {
		t.Errorf("the KP's bytes read %q, want the Windows-874 bytes of the Thai text, %q", engine.Objects[0].Description, want)
	}

	// Both read back as the Thai that went in.
	for _, p := range []string{xdfPath, kpPath} {
		def, err := a.OpenDefinition(p)
		if err != nil {
			t.Fatal(err)
		}
		row := def.Rows[0]
		if row.Description != desc || row.Units != "องศา" || def.Info.Thai == 0 {
			t.Errorf("%s: description %q, units %q, %d Thai texts", filepath.Base(p), row.Description, row.Units, def.Info.Thai)
		}
		if len(row.Categories) != 1 || row.Categories[0] != cat {
			t.Errorf("%s: categories %q", filepath.Base(p), row.Categories)
		}
		// And as JSON a person can read and edit.
		back := filepath.Join(dir, filepath.Base(p)+".json")
		if r := a.Convert(ConvertJob{Input: p, To: api.JSON, Output: back}); r.Error != "" {
			t.Fatalf("%s to JSON: %s", filepath.Base(p), r.Error)
		}
		if !bytes.Contains(read(t, back), []byte(desc)) {
			t.Errorf("%s: the JSON doesn't hold the Thai description", filepath.Base(p))
		}
	}

	// KP to XDF carries all of it, the eight letters the two Western sets disagree on included.
	xdf2 := filepath.Join(dir, "from-kp.xdf")
	if r := a.Convert(ConvertJob{Input: kpPath, To: api.XDF, Output: xdf2}); r.Error != "" {
		t.Fatalf("KP to XDF: %s", r.Error)
	}
	def, err := a.OpenDefinition(xdf2)
	if err != nil {
		t.Fatal(err)
	}
	if def.Rows[0].Description != desc {
		t.Errorf("KP to XDF: description %q", def.Rows[0].Description)
	}
}

// TestWesternPackUntouched: a pack without Thai converts exactly as the
// engine converts it alone.
func TestWesternPackUntouched(t *testing.T) {
	a := NewApp()
	d, err := a.loadDef(packKP)
	if err != nil {
		t.Fatal(err)
	}
	if d.thai != 0 || modelHasThai(d.model) {
		t.Fatalf("a German pack read as Thai: %d texts", d.thai)
	}
	out := filepath.Join(t.TempDir(), "pack.xdf")
	if r := a.Convert(ConvertJob{Input: packKP, To: api.XDF, Output: out}); r.Error != "" {
		t.Fatal(r.Error)
	}
	want, _, err := api.Convert(read(t, packKP), api.ConvertRequest{To: api.XDF, Name: filepath.Base(packKP), Title: "pack"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(read(t, out), want) {
		t.Error("the app's XDF differs from the engine's")
	}
}
