package glyphlayout_test

// A damaged positioning table is not the same as no positioning table.
//
// A font with no GPOS lays its glyphs out by their advances, which is a
// perfectly good page. A font whose GPOS cannot be read is a font this port
// does not understand, and quietly dropping every kern and every mark because
// of it leaves nobody anything to go on -- the page comes out subtly wrong and
// nothing says why.

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/glyphlayout"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// TestDamagedGPOSIsReported takes a font whose GPOS table carries a version
// the reader does not know, and checks the failure is not swallowed.
//
// It surfaces at the load rather than at the layout: `Parser.parseTables` reads
// every table of the directory when it parses the font, so a table that cannot
// be read stops the font from being loaded at all, and the layout is never
// handed one it half understands. That is where the case asserts it, because
// that is where it happens -- and it is why `position` cannot reach its own
// error path today. The path is written to return the error anyway: a table
// that cannot be read is not a table that is not there, and the two must not
// come out looking the same if the reading ever moves.
func TestDamagedGPOSIsReported(t *testing.T) {
	path := damagedGPOSFont(t)

	document := pdmodel.NewPDDocument()
	defer document.Close()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening the damaged font: %v", err)
	}
	defer file.Close()

	embedded, err := font.LoadPDType0FontSubset(document, file, false)
	if err == nil {
		// If the reading ever becomes lazy this is where the case has to
		// change, and the layout is then what has to refuse it.
		processor := glyphlayout.NewProcessorWithFeatures(
			glyphlayout.Features{Kerning: true})
		if _, err := processor.StringWidth(embedded, 12, "AVATAR"); err == nil {
			t.Fatal("a font whose GPOS cannot be read was loaded and laid out " +
				"without a word")
		} else if !strings.Contains(err.Error(), "GPOS") {
			t.Errorf("the layout's failure is %q, want it to name the table", err)
		}
		return
	}
	if !strings.Contains(err.Error(), "GPOS") {
		t.Errorf("the failure is %q, want it to name the table it could not read", err)
	}
}

// damagedGPOSFont writes a copy of DejaVuSans whose GPOS table claims a major
// version the reader does not know, and answers where it put it.
//
// Everything else about the font is untouched, so the only thing that can fail
// is reading that table.
func damagedGPOSFont(t *testing.T) string {
	t.Helper()
	program, err := os.ReadFile(layoutFonts + "DejaVuSans.ttf")
	if err != nil {
		t.Skipf("DejaVuSans is not in this repository: %v", err)
	}
	offset := tableOffset(t, program, "GPOS")
	// The major version, which Read refuses unless it is 1.
	binary.BigEndian.PutUint16(program[offset:], 2)

	path := filepath.Join(t.TempDir(), "DejaVuSans-damaged-gpos.ttf")
	if err := os.WriteFile(path, program, 0o644); err != nil {
		t.Fatalf("writing the damaged font: %v", err)
	}
	return path
}

// tableOffset answers where a table starts, out of the font's table directory:
// a 12-byte header and then one sixteen-byte record per table, each holding a
// four-byte tag, a checksum, an offset and a length.
func tableOffset(t *testing.T, program []byte, tag string) uint32 {
	t.Helper()
	if len(program) < 12 {
		t.Fatal("the font is too short to hold a table directory")
	}
	tables := binary.BigEndian.Uint16(program[4:])
	for i := 0; i < int(tables); i++ {
		record := 12 + i*16
		if record+16 > len(program) {
			break
		}
		if bytes.Equal(program[record:record+4], []byte(tag)) {
			return binary.BigEndian.Uint32(program[record+8:])
		}
	}
	t.Fatalf("the font has no %s table", tag)
	return 0
}
