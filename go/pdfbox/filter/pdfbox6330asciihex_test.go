package filter

// PDFBOX-6330, Apache 46f9e442d, in the sync of 2026-11-21.
//
// The decoder skipped white space before the first digit of a pair and not
// before the second, so `4 1>` read the space as a digit the table holds -1
// for: it logged, took the space as the low nibble and wrote 0x40, then read
// `1` as the next pair's high nibble and wrote 0x10. Two bytes where the stream
// says one. Upstream added the same skip loop before the second digit.
//
// The -1 arithmetic that JAVA-BUGS.md 30 is about is a separate matter and
// stands: the Java still adds the table's -1, and the port still reads a bad
// digit as zero. TestASCIIHexStillTreatsABadDigitAsZero is here to say that the
// skip did not quietly take that over.

import (
	"bytes"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// decodeASCIIHex returns what the decoder made of the given text.
func decodeASCIIHex(t *testing.T, encoded string) ([]byte, error) {
	t.Helper()
	var out bytes.Buffer
	_, err := ASCIIHex{}.Decode(&out, bytes.NewReader([]byte(encoded)),
		cos.NewDictionary(), 0)
	return out.Bytes(), err
}

// TestASCIIHexSkipsWhitespaceBetweenTheDigits is the fix, over the six
// white-space characters of ISO 32000-1 table 1. Each case is one byte, 0x41,
// written with its digits apart; before the fix every one of them decoded to
// two bytes, 0x40 and 0x10.
func TestASCIIHexSkipsWhitespaceBetweenTheDigits(t *testing.T) {
	for _, c := range []byte{0, '\t', '\n', '\f', '\r', ' '} {
		decoded, err := decodeASCIIHex(t, "4"+string(c)+"1>")
		if err != nil {
			t.Errorf("white space %d: %v", c, err)
			continue
		}
		if !bytes.Equal(decoded, []byte{0x41}) {
			t.Errorf("white space %d decoded to %v, want [65]", c, decoded)
		}
	}
}

// TestASCIIHexSkipsSeveralWhitespaceCharactersBetweenTheDigits is the loop
// rather than a single test: the Java reads again while it is white space.
func TestASCIIHexSkipsSeveralWhitespaceCharactersBetweenTheDigits(t *testing.T) {
	decoded, err := decodeASCIIHex(t, "4 \t\r\n1 65>")
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !bytes.Equal(decoded, []byte{0x41, 0x65}) {
		t.Errorf("decoded to %v, want [65 101]", decoded)
	}
}

// TestASCIIHexStillTreatsABadDigitAsZero is the guard, and it passes either
// way: a byte that is not a hexadecimal digit and not white space is read as
// zero, which is JAVA-BUGS.md 30 and not what this commit is about.
func TestASCIIHexStillTreatsABadDigitAsZero(t *testing.T) {
	decoded, err := decodeASCIIHex(t, "4Z65>")
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !bytes.Equal(decoded, []byte{0x40, 0x65}) {
		t.Errorf("decoded to %v, want [64 101]", decoded)
	}
}

// TestASCIIHexWhitespaceBeforeTheEndOfDataStillWritesTheHighNibble pins the arm
// the skip reaches the end of data through: `4 >` is a half byte, and the
// method's own comment says the second value behaves like 0 for an EOD.
func TestASCIIHexWhitespaceBeforeTheEndOfDataStillWritesTheHighNibble(t *testing.T) {
	decoded, err := decodeASCIIHex(t, "4 >")
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !bytes.Equal(decoded, []byte{0x40}) {
		t.Errorf("decoded to %v, want [64]", decoded)
	}
}
