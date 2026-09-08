package filter

// JAVA-BUGS 32: `ASCII85InputStream.read()` resets `index` to 0 before it
// reads a group and returns -1 from inside the group loop when the stream ends
// part way through one, leaving `n` at the previous group's 4; the next read
// finds `index < n` and hands that group out a second time.

import (
	"bytes"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// TestASCII85TruncatedStreamDoesNotRepeatItsLastGroup is the defect.
//
// The expected value is a prefix of the original: a stream that stops part way
// through a group has decoded everything before that group and nothing after
// it, so what comes back is what the complete groups held. Java answers those
// bytes and then the last four of them again.
func TestASCII85TruncatedStreamDoesNotRepeatItsLastGroup(t *testing.T) {
	original := bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog. "), 30)

	var encoded bytes.Buffer
	if err := (ASCII85{}).Encode(&encoded, bytes.NewReader(original),
		cos.NewDictionary()); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	truncated := encoded.Bytes()[:encoded.Len()/2]
	var decoded bytes.Buffer
	_, _ = (ASCII85{}).Decode(&decoded, bytes.NewReader(truncated), cos.NewDictionary(), 0)

	got := decoded.Bytes()
	if len(got) == 0 {
		t.Fatal("a truncated stream yielded nothing; partial data must survive")
	}
	if !bytes.HasPrefix(original, got) {
		// name the repeat, since that is the shape the failure takes
		t.Errorf("the decoded %d bytes are not a prefix of the original; "+
			"the tail is %q", len(got), got[max(0, len(got)-8):])
	}
}

// TestASCII85WholeStreamIsUnchanged keeps the end that is not truncation: a
// stream that ends on a group boundary, with the terminator, decodes to all of
// its bytes and no fewer.
func TestASCII85WholeStreamIsUnchanged(t *testing.T) {
	original := []byte("the quick brown fox jumps over the lazy dog.")

	var encoded bytes.Buffer
	if err := (ASCII85{}).Encode(&encoded, bytes.NewReader(original),
		cos.NewDictionary()); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	var decoded bytes.Buffer
	if _, err := (ASCII85{}).Decode(&decoded, bytes.NewReader(encoded.Bytes()),
		cos.NewDictionary(), 0); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !bytes.Equal(decoded.Bytes(), original) {
		t.Errorf("the round trip gave %q, want %q", decoded.Bytes(), original)
	}
}
