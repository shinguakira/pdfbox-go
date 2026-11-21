package filter

// PDFBOX-6330, Apache fdfeaa44e, in the sync of 2026-11-21.
//
// The decoder skipped a line feed, a carriage return and a space between the
// characters of a group, and ISO 32000-1 table 1 gives six white-space
// characters: NUL, HT, LF, FF, CR and SP. A producer that wraps its ASCII85
// with tabs, or pads it with NUL, reached the alphabet check with the
// white-space byte and the stream was refused as invalid data. Upstream moved
// the three-way test into an isWhitespace of all six.
//
// `!!!!!` is one group of five zero digits, so it decodes to four zero bytes;
// every case below is that group with one white-space character pushed into
// the middle of it.

import (
	"bytes"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// decodeASCII85 returns what the decoder made of the given bytes.
func decodeASCII85(t *testing.T, encoded []byte) ([]byte, error) {
	t.Helper()
	var out bytes.Buffer
	_, err := ASCII85{}.Decode(&out, bytes.NewReader(encoded), cos.NewDictionary(), 0)
	return out.Bytes(), err
}

// TestASCII85SkipsEveryPDFWhitespaceCharacter is the fix. Before it, the three
// the decoder did not know -- NUL, HT and FF -- each ended the decode with
// `Invalid data in Ascii85 stream`.
func TestASCII85SkipsEveryPDFWhitespaceCharacter(t *testing.T) {
	for _, c := range []byte{0, '\t', '\n', '\f', '\r', ' '} {
		encoded := []byte{'!', '!', c, '!', '!', '!', '~', '>'}
		decoded, err := decodeASCII85(t, encoded)
		if err != nil {
			t.Errorf("white space %d: %v", c, err)
			continue
		}
		if !bytes.Equal(decoded, []byte{0, 0, 0, 0}) {
			t.Errorf("white space %d decoded to %v, want four zero bytes", c, decoded)
		}
	}
}

// TestASCII85SkipsWhitespaceBeforeAGroup is the other place the same test is
// made: the first character of a group, rather than the rest of it.
func TestASCII85SkipsWhitespaceBeforeAGroup(t *testing.T) {
	decoded, err := decodeASCII85(t, []byte{'\t', 0, '\f', '!', '!', '!', '!', '!', '~', '>'})
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !bytes.Equal(decoded, []byte{0, 0, 0, 0}) {
		t.Errorf("decoded to %v, want four zero bytes", decoded)
	}
}

// TestASCII85StillRefusesAByteOutsideTheAlphabet is the guard: widening the
// white-space test must not widen what counts as a digit. 0x80 is neither, and
// `TestASCII85Invalid` in fromsource_test.go holds the Java's answer for it.
func TestASCII85StillRefusesAByteOutsideTheAlphabet(t *testing.T) {
	_, err := decodeASCII85(t, []byte{'!', '!', 0x80, '!', '!', '!', '~', '>'})
	if err == nil {
		t.Fatal("a 0x80 byte was accepted as a digit")
	}
	if err.Error() != "Invalid data in Ascii85 stream" {
		t.Errorf("the error is %q, want the invalid-data error", err)
	}
}
