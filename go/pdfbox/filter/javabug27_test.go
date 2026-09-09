package filter

// JAVA-BUGS 27: `ASCII85InputStream` narrows what `in.read()` returned to a
// `byte` before testing it for -1, so a data byte 0xFF ends the stream instead
// of being rejected.

import (
	"bytes"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// TestASCII85RejectsAnFFByte is the defect.
//
// 0xFF is not in the ASCII85 alphabet, which runs from '!' (0x21) to 'u'
// (0x75), so the decoder must report it the way it reports every other byte
// outside the alphabet: `Invalid data in Ascii85 stream`. The comparison in
// the same method, `z < 0 || z > 93`, is the one that says so -- 0xFF reaches
// it as -34 -- and 0x80 already takes that path, which
// `TestASCII85Invalid` covers.
func TestASCII85RejectsAnFFByte(t *testing.T) {
	var out bytes.Buffer
	// a full group with 0xFF in the middle: without the fix the decoder stops
	// at it and reports success with nothing decoded
	_, err := ASCII85{}.Decode(&out, bytes.NewReader([]byte{'!', '!', 0xFF, '!', '!'}),
		cos.NewDictionary(), 0)
	if err == nil {
		t.Fatalf("a 0xFF byte decoded to %v with no error, want the invalid-data error",
			out.Bytes())
	}
	if err.Error() != "Invalid data in Ascii85 stream" {
		t.Errorf("the error is %q, want the invalid-data error", err)
	}
}

// TestASCII85StillEndsAtTheEndOfItsSource keeps the end the narrowing was
// standing in for: a source that runs out mid-group ends the stream quietly,
// which is what Java's `in.read()` returning a real -1 does.
func TestASCII85StillEndsAtTheEndOfItsSource(t *testing.T) {
	var out bytes.Buffer
	// two characters and then nothing -- not a group, and not an error
	if _, err := (ASCII85{}).Decode(&out, bytes.NewReader([]byte{'!', '!'}),
		cos.NewDictionary(), 0); err != nil {
		t.Fatalf("a truncated source should end quietly: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("a truncated group decoded to %v, want nothing", out.Bytes())
	}
}
