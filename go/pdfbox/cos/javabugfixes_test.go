package cos_test

// The `cos` half of `track/java-bug-fixes`: what the Go now does that the Java
// does not, with the entry of `migration/JAVA-BUGS.md` that each one closes.
//
// Every expected value here comes from the specification or from arithmetic,
// never from the Java — the whole point of these is that the Java is wrong.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// TestIntegerEqualsDoesNotTruncate is JAVA-BUGS 1.
//
// `COSInteger.equals` compares `intValue()`, which is `(int) value`, so it
// drops everything above bit 31: Java says `COSInteger.get(0)` and
// `COSInteger.get(4294967296)` are equal. They are not, and `equals` is what
// `COSArray.indexOf`, `removeObject` and every dictionary comparison route
// through.
func TestIntegerEqualsDoesNotTruncate(t *testing.T) {
	zero := cos.GetInteger(0)
	twoToThe32 := cos.GetInteger(4294967296)

	if zero.Equals(twoToThe32) {
		t.Error("0 and 4294967296 compare equal; the comparison truncates to 32 bits")
	}
	if twoToThe32.Equals(zero) {
		t.Error("4294967296 and 0 compare equal; the comparison truncates to 32 bits")
	}
	// The two whose low 32 bits differ were never equal, and still are not.
	if cos.GetInteger(1).Equals(cos.GetInteger(2)) {
		t.Error("1 and 2 compare equal")
	}
	// And equal values still are.
	if !cos.GetInteger(4294967296).Equals(cos.GetInteger(4294967296)) {
		t.Error("4294967296 is not equal to itself")
	}
	// IntValue still narrows: it is `intValue()`, and a caller who asks for
	// Java's int wants Java's answer.
	if got := twoToThe32.IntValue(); got != 0 {
		t.Errorf("IntValue() is %d, want 0: it is still the (int) cast", got)
	}
}
