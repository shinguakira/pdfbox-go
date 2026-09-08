package encryption_test

// JAVA-BUGS 25: `getRecipientsLength` casts a missing /Recipients to a COSArray
// and dereferences it.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/encryption"
)

// TestRecipientsLengthOfADictionaryWithNone is the defect.
//
// `getItem` is documented to answer null for a key that is not there, and the
// method casts and calls `size()` on it. Every other accessor on this class
// answers a default for a missing entry.
//
// The expected value is what a count of nothing is.
func TestRecipientsLengthOfADictionaryWithNone(t *testing.T) {
	enc := encryption.NewPDEncryption()
	if got := enc.RecipientsLength(); got != 0 {
		t.Errorf("RecipientsLength() is %d for a dictionary with no /Recipients, want 0", got)
	}
}
