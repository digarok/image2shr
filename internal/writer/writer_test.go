package writer

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/digarok/image2shr/shr"
)

func TestRawEncode(t *testing.T) {
	f := &shr.Frame{}
	f.Pixels[0] = 0xAB
	f.SCB[0] = 0x0F
	f.Palettes[0][1] = shr.RGB12{R: 15, G: 8, B: 0}

	raw, err := Lookup("raw")
	if err != nil {
		t.Fatal(err)
	}
	ft, at := raw.ProDOS()
	if ft != 0xC1 || at != 0x0000 {
		t.Errorf("raw ProDOS = $%02X/$%04X, want $C1/$0000", ft, at)
	}

	var buf bytes.Buffer
	if err := raw.Encode(&buf, f); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != shr.FrameSize {
		t.Fatalf("raw output = %d bytes, want %d", buf.Len(), shr.FrameSize)
	}
	if !bytes.Equal(buf.Bytes(), f.EncodeRaw()) {
		t.Error("raw writer output differs from Frame.EncodeRaw")
	}
}

func TestStubFormats(t *testing.T) {
	expect := map[string][2]uint16{ // fileType, auxType
		"packed": {0xC0, 0x0001},
		"apf":    {0xC0, 0x0002},
	}
	for name, want := range expect {
		f, err := Lookup(name)
		if err != nil {
			t.Fatalf("stub %q not registered: %v", name, err)
		}
		ft, at := f.ProDOS()
		if uint16(ft) != want[0] || at != want[1] {
			t.Errorf("%s ProDOS = $%02X/$%04X, want $%02X/$%04X", name, ft, at, want[0], want[1])
		}
		if err := f.Encode(&bytes.Buffer{}, &shr.Frame{}); !errors.Is(err, ErrNotImplemented) {
			t.Errorf("%s Encode err = %v, want ErrNotImplemented", name, err)
		}
	}
}

func TestBrooksEncode(t *testing.T) {
	brooks, err := Lookup("brooks")
	if err != nil {
		t.Fatal(err)
	}
	ft, at := brooks.ProDOS()
	if ft != 0xC1 || at != 0x0002 {
		t.Errorf("brooks ProDOS = $%02X/$%04X, want $C1/$0002", ft, at)
	}
	var buf bytes.Buffer
	if err := brooks.Encode(&buf, &shr.Frame{}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != shr.BrooksSize {
		t.Errorf("brooks output = %d bytes, want %d", buf.Len(), shr.BrooksSize)
	}
}

// TestRawRejects3200: a per-line-palette frame cannot fit the raw container.
func TestRawRejects3200(t *testing.T) {
	raw, _ := Lookup("raw")
	f := &shr.Frame{LinePalettes: &[shr.Height][16]shr.RGB12{}}
	if err := raw.Encode(&bytes.Buffer{}, f); err == nil {
		t.Error("raw accepted a 3200-color frame")
	}
}

func TestLookupUnknown(t *testing.T) {
	if _, err := Lookup("tiff"); err == nil {
		t.Error("expected error for unknown format")
	}
}

func TestFormatsSorted(t *testing.T) {
	fs := Formats()
	for i := 1; i < len(fs); i++ {
		if fs[i-1].Name() >= fs[i].Name() {
			t.Fatalf("Formats() not sorted: %q before %q", fs[i-1].Name(), fs[i].Name())
		}
	}
}

func TestWriteSidecar(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "pic.shr")
	raw, _ := Lookup("raw")
	if err := WriteSidecar(out, raw, "v1.2.3"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out + ".meta.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc Sidecar
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	want := Sidecar{
		File: "pic.shr", Format: "raw",
		FileType: 0xC1, FileTypeHex: "$C1",
		AuxType: 0, AuxTypeHex: "$0000",
		Tool: "image2shr", Version: "v1.2.3",
	}
	if doc != want {
		t.Errorf("sidecar = %+v, want %+v", doc, want)
	}
}
