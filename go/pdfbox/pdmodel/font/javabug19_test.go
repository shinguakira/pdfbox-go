package font

// JAVA-BUGS 19: `writeFontInfo` sign-extends a Panose byte before writing it as
// hex. In-package, because the writer and its record are.

import (
	"bufio"
	"strings"
	"testing"
)

// TestPanoseHexIsTwoDigits is the defect.
//
// Java writes each Panose byte with `Integer.toHexString(bytes[i])`, and
// `bytes[i]` is a signed Java byte, so 0x80 widens to 0xFFFFFF80 and writes
// eight hex digits where the reader takes two. The reader's own `& 0xff` says
// what the writer meant.
//
// The expected value is the format's: ten bytes, two hex digits each.
func TestPanoseHexIsTwoDigits(t *testing.T) {
	panose := make([]byte, PanoseClassificationLength)
	for i := range panose {
		panose[i] = 0x80 | byte(i) // every byte with the high bit set
	}
	info := newFSFontInfo("f.ttf", FontFormatTTF, "Test", nil, 0, 0, 0, 0, 0,
		panose, nil, "", 0)

	var out strings.Builder
	writer := bufio.NewWriter(&out)
	if err := writeFontInfo(writer, info); err != nil {
		t.Fatalf("writeFontInfo: %v", err)
	}
	if err := writer.Flush(); err != nil {
		t.Fatal(err)
	}

	line := out.String()
	if strings.Contains(line, "ffffff") {
		t.Errorf("a Panose byte was written sign-extended:\n%s", line)
	}
	if !strings.Contains(line, "80818283848586878889") {
		t.Errorf("the ten Panose bytes are not twenty hex characters:\n%s", line)
	}
}
