package font

// The bytes of /FontFile2, checked against the running Java.
//
// A subsetted font that parses is not a font that is right. The stream this
// branch writes is TTFSubsetter's output, and the subset tag in the font's name
// is computed from a Java HashMap's hashCode -- 32-bit arithmetic that wraps,
// summed over entries whose hash is key ^ value. Neither is something a Go test
// can check against itself.
//
// Every wanted value below was printed by the running Java: fontbox compiled
// with javac, driven by a program that calls TTFSubsetter exactly as
// TrueTypeEmbedder.subset does -- the same ten tables, the same four
// forceInvisible calls, the same getTag.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// TestSubsetBytesMatchJava subsets LiberationSans for the string the
// CIDFontType2 cases write and checks the result against what Java produced for
// the same input.
func TestSubsetBytesMatchJava(t *testing.T) {
	const text = "Unicode русский язык Tiếng Việt"

	// Printed by the Java run.
	const (
		wantTag    = "AALHKC+"
		wantHash   = 21410
		wantLength = 8332
		wantSHA    = "72fcedd104476342f5f9f234b7db06165fff7b58da445e57e6a16fec7573ec9b"
	)
	wantGIDToCID := map[int]int{
		0: 0, 1: 3, 2: 55, 3: 56, 4: 57, 5: 70, 6: 71, 7: 72, 8: 74, 9: 76,
		10: 81, 11: 82, 12: 83, 13: 87, 14: 92, 15: 648, 16: 1000, 17: 1001,
		18: 1002, 19: 1003, 20: 1009, 21: 1010, 22: 1012, 23: 1020, 24: 1024,
		25: 1705, 26: 1713, 27: 2331, 28: 2355, 29: 2597,
	}

	source, err := pdfio.OpenBufferedFile(
		"../../../../pdfbox/src/main/resources/org/apache/pdfbox/resources/ttf/" +
			"LiberationSans-Regular.ttf")
	if err != nil {
		t.Fatalf("opening LiberationSans-Regular.ttf: %v", err)
	}
	font, err := ttf.NewParser().Parse(source)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// What PDType0Font.encode adds to the subset, one code point at a time in
	// first-occurrence order.
	seen := map[int]bool{}
	codePoints := []int{}
	for _, r := range text {
		if !seen[int(r)] {
			seen[int(r)] = true
			codePoints = append(codePoints, int(r))
		}
	}

	sub, err := ttf.NewTTFSubsetterTables(font, subsetTables)
	if err != nil {
		t.Fatalf("NewTTFSubsetterTables: %v", err)
	}
	sub.AddAll(codePoints)
	sub.ForceInvisible(0x200B) // ZWSP
	sub.ForceInvisible(0x200C) // ZWNJ
	sub.ForceInvisible(0x2060) // WJ
	sub.ForceInvisible(0xFEFF) // ZWNBSP

	gidToCid, err := sub.GetGIDMap()
	if err != nil {
		t.Fatalf("GetGIDMap: %v", err)
	}
	if len(gidToCid) != len(wantGIDToCID) {
		t.Fatalf("the subset kept %d glyphs, want %d", len(gidToCid), len(wantGIDToCID))
	}
	for gid, want := range wantGIDToCID {
		if got, ok := gidToCid[gid]; !ok || got != want {
			t.Errorf("gidToCid[%d] = %d (present %v), want %d", gid, got, ok, want)
		}
	}

	// The tag is part of the font's name in the file, so it has to be Java's
	// byte for byte.
	var hash int32
	for gid, cid := range gidToCid {
		hash += int32(gid) ^ int32(cid)
	}
	if hash != wantHash {
		t.Errorf("the map hash is %d, want Java's %d", hash, wantHash)
	}
	tag := subsetTag(gidToCid)
	if tag != wantTag {
		t.Errorf("subsetTag = %q, want %q", tag, wantTag)
	}
	sub.SetPrefix(tag)

	var out bytes.Buffer
	if err := sub.WriteToStream(&out); err != nil {
		t.Fatalf("WriteToStream: %v", err)
	}
	program := out.Bytes()

	if len(program) != wantLength {
		t.Errorf("the subset is %d bytes, want %d", len(program), wantLength)
	}
	sum := sha256.Sum256(program)
	if got := hex.EncodeToString(sum[:]); got != wantSHA {
		t.Errorf("the subset hashes to %s, want %s -- the bytes differ from Java's", got, wantSHA)
	}
}
