package filter

// PDFBOX-6317, Apache d219d4140, in the sync of 2026-11-21, and the entry that
// was JAVA-BUGS.md 32.
//
// A stream that stops part way through a group used to return -1 from inside
// the group loop with `n` still holding the previous group's 4, so the previous
// group was handed out a second time. The port's answer to that, on
// track/java-bug-fixes, was to set `n` to 0 beside `eof`: the repeat went, and
// so did the part group, so a truncated stream decoded to its whole groups and
// stopped.
//
// Upstream took the other road the entry named. A read of -1 inside a group is
// now taken for the terminator, "a missing end-of-data marker ends the last
// group like ~>", so the part group decodes under the padding rule and one to
// three more of the original's bytes come back. Neither tree repeats anything.
// This commit follows the Java.
//
// The padding rule is why the table below is exact: k characters of a group
// decode to k-1 bytes, so cutting one character off the last group loses one
// byte, cutting four leaves a single character, which is no group at all.

import (
	"bytes"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// TestASCII85GroupCutShortDecodesLikeTheTerminator is the change. Twenty bytes
// encode as four whole groups and one more; each case cuts characters off that
// last group and says how much of the original survives.
func TestASCII85GroupCutShortDecodesLikeTheTerminator(t *testing.T) {
	original := []byte("the quick brown fox!") // 20 bytes, five whole groups

	var encoded bytes.Buffer
	if err := (ASCII85{}).Encode(&encoded, bytes.NewReader(original),
		cos.NewDictionary()); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	// the encoder ends the stream with ~> and a line break
	stream := bytes.TrimRight(encoded.Bytes(), "\r\n")
	withoutTerminator := bytes.TrimSuffix(stream, []byte("~>"))
	if len(withoutTerminator) == len(stream) {
		t.Fatalf("the encoder did not end with ~>: %q", encoded.Bytes())
	}

	cases := []struct {
		cut  int // characters taken off the end of the last group
		want int // bytes of the original that survive
	}{
		{1, 19},
		{2, 18},
		{3, 17},
		{4, 16}, // one character left is not a group, so the whole five go
	}
	for _, c := range cases {
		truncated := withoutTerminator[:len(withoutTerminator)-c.cut]
		var decoded bytes.Buffer
		if _, err := (ASCII85{}).Decode(&decoded, bytes.NewReader(truncated),
			cos.NewDictionary(), 0); err != nil {
			t.Errorf("cut %d: Decode: %v", c, err)
			continue
		}
		if got := decoded.Bytes(); !bytes.Equal(got, original[:c.want]) {
			t.Errorf("cut %d decoded %d bytes, %q; want the first %d, %q",
				c.cut, len(got), got, c.want, original[:c.want])
		}
	}
}

// TestASCII85StreamThatEndsOnAGroupBoundaryIsUnchanged is the other arm, and
// the one the port already had right: a source that runs out exactly between
// groups ends the stream with nothing left over, and that path still sets n to
// 0 rather than handing the last group out twice.
func TestASCII85StreamThatEndsOnAGroupBoundaryIsUnchanged(t *testing.T) {
	original := []byte("the quick brown fox!")

	var encoded bytes.Buffer
	if err := (ASCII85{}).Encode(&encoded, bytes.NewReader(original),
		cos.NewDictionary()); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	withoutTerminator := bytes.TrimSuffix(
		bytes.TrimRight(encoded.Bytes(), "\r\n"), []byte("~>"))

	var decoded bytes.Buffer
	if _, err := (ASCII85{}).Decode(&decoded, bytes.NewReader(withoutTerminator),
		cos.NewDictionary(), 0); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !bytes.Equal(decoded.Bytes(), original) {
		t.Errorf("decoded %q, want all %d bytes of the original",
			decoded.Bytes(), len(original))
	}
}
