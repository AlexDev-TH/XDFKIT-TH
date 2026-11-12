package kp

import (
	"bufio"
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestRealPacks reads every pack listed in the file named by XDFKIT_KP_LIST
// (one path per line) and checks that parsing and encoding gives the same map
// block, and the same file outside the zip. It is skipped without the list.
func TestRealPacks(t *testing.T) {
	list := os.Getenv("XDFKIT_KP_LIST")
	if list == "" {
		t.Skip("XDFKIT_KP_LIST is not set")
	}
	lf, err := os.Open(list)
	if err != nil {
		t.Fatal(err)
	}
	defer lf.Close()
	var read, same, block, skipMaps, skipAxes int
	headers := map[int32]int{}
	sc := bufio.NewScanner(lf)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		path := strings.TrimSpace(strings.TrimPrefix(sc.Text(), "\ufeff"))
		if path == "" {
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if !bytes.HasPrefix(b, []byte("\x0b\x00\x00\x00WinOLS File\x00")) {
			continue // not a map pack, whatever its name
		}
		f, err := Parse(b)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		read++
		headers[f.HeaderID]++
		for _, m := range f.Project.Maps {
			if m.SkipBytes != 0 || m.LineSkipBytes != 0 {
				skipMaps++
			}
			for _, a := range []*Axis{m.X, m.Y} {
				if a.SkipBytes != 0 {
					skipAxes++
				}
			}
		}
		if f.HeaderID == header245 {
			continue
		}
		out, err := f.Encode()
		if err != nil {
			t.Errorf("%s: encode: %v", path, err)
			continue
		}
		if bytes.Equal(out, b) {
			same++
			continue
		}
		// A fresh zip differs; the map block and the rest must not.
		g, err := Parse(out)
		if err != nil {
			t.Errorf("%s: reparse: %v", path, err)
			continue
		}
		if !bytes.Equal(g.Project.intern, f.Project.intern) {
			t.Errorf("%s: map block differs after encode", path)
			continue
		}
		// A pack written with a stale EndOffset differs in that field alone.
		if loc, err := f.Offset("endOffset"); err == nil && len(out) == len(b) {
			copy(out[loc.Off:loc.Off+4], b[loc.Off:loc.Off+4])
		}
		if !bytes.Equal(out, b) && bytes.Equal(g.Project.zip, f.Project.zip) {
			for _, d := range ByteDiff(b, out, 6) {
				t.Errorf("%s: %s", path, d)
			}
		}
		block++
	}
	t.Logf("read %d (headers %v), byte-identical %d, other %d, maps with skip bytes %d, axes with skip bytes %d", read, headers, same, block, skipMaps, skipAxes)
}
