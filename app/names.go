package main

import (
	"strings"

	"go.nyet.org/xdfkit/model"
)

// Names between TunerPro and WinOLS.
//
// A TunerPro table has a title, its name, and a description, its help text.
// A WinOLS map has an id, a name and a comment, and WinOLS lists maps by name.
// The engine keeps the title as the id and the description as the name, which
// suits packs exported from Bosch data ("KFZW", "Zündwinkelkennfeld") but
// leaves a map made in TunerPro listed in WinOLS by its help text, or by
// nothing at all when it has none.
//
// So for a definition that comes from TunerPro itself, the app writes a KP
// with the title as both id and name and the help text as the comment, and
// writing an XDF puts them back where TunerPro keeps them.

// tunerProNames arranges the names of a model read from a TunerPro XDF for a KP.
func tunerProNames(m *model.Model) {
	for _, o := range m.Objects {
		if o.ID == "" {
			continue
		}
		if help := strings.TrimSpace(o.Description); help != "" && help != o.ID {
			if o.Comment != "" {
				help += "\n" + o.Comment
			}
			o.Comment = help
		}
		o.Description = o.ID
	}
}

// echoesTitle reports an object whose name only repeats its id, as
// tunerProNames leaves it.
func echoesTitle(m *model.Model) bool {
	for _, o := range m.Objects {
		if o.ID != "" && o.Description == o.ID {
			return true
		}
	}
	return false
}

// plainTitles prepares such a model for an XDF: the name that repeats the
// title is left out, so the description is the help text alone.
func plainTitles(m *model.Model) {
	for _, o := range m.Objects {
		if o.ID != "" && o.Description == o.ID {
			o.Description = ""
		}
	}
}

// A map exported from Bosch data has a short id ("KFZW") and a name
// ("Zuendwinkelkennfeld"). WinOLS lists such maps by name; TunerPro lists a
// table by its title alone. So that TunerPro shows what WinOLS shows, the
// title of such a table is "id: name". The description still holds the name,
// which is how the id is found again when the XDF is read.
const titleSep = ": "

// boschNames reports a model whose ids are short identifiers with separate
// names: one that came from a KP, an A2L, a DAMOS file or an image.
func boschNames(m *model.Model) bool {
	if m.Provenance == nil {
		return false
	}
	switch m.Provenance.Format {
	case "kp", "a2l", "damos", "image":
		return true
	}
	return false
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(line)
}

// listedTitles prepares a model with Bosch names for an XDF: each id gets its
// name after it. A name that only repeats the id is left out (plainTitles).
func listedTitles(m *model.Model) {
	for _, o := range m.Objects {
		name := firstLine(o.Description)
		switch {
		case o.ID == "" || name == "":
		case o.Description == o.ID:
			o.Description = ""
		case !strings.HasSuffix(o.ID, titleSep+name):
			if o.Key == "" {
				o.Key = o.ID
			}
			o.ID += titleSep + name
		}
	}
}

// unlistTitles takes the name off the ids of a model read from such an XDF.
func unlistTitles(m *model.Model) {
	for _, o := range m.Objects {
		name := firstLine(o.Description)
		if name == "" {
			continue
		}
		if id, ok := strings.CutSuffix(o.ID, titleSep+name); ok && id != "" {
			if o.Key == o.ID {
				o.Key = id
			}
			o.ID = id
		}
	}
}
