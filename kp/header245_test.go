package kp

import (
	"os"
	"path/filepath"
	"testing"

	"go.nyet.org/xdfkit/internal/testenv"
)

func TestHeader245Parse(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(testenv.ModuleRoot(), "testdata", "kp", "245-fewmaps.kp"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if f.Layout() != V2 || f.HeaderID != header245 || f.HeaderLen != 0 {
		t.Fatalf("layout %v header %d HeaderLen %d", f.Layout(), f.HeaderID, f.HeaderLen)
	}
	if len(f.FixedHeader) != 0x5c-0x18 {
		t.Fatalf("FixedHeader %d", len(f.FixedHeader))
	}
	m0 := f.Project.Maps[0]
	if m0.ID == "" || m0.X == nil || m0.X.SkipBytes != 0 || len(m0.Term2) != 3 {
		t.Fatalf("maps[0] id=%q x=%v", m0.ID, m0.X)
	}
	if fd := f.Project.Folders[0]; fd.Name == "" || len(fd.States) != 1 {
		t.Fatalf("folder %+v", fd)
	}
	if _, err := f.Encode(); err == nil {
		t.Fatal("Encode of header 245 should fail")
	}
}
