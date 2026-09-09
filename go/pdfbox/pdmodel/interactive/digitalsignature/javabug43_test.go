package digitalsignature_test

// JAVA-BUGS 43: `PDSeedValue` writes `/Reasons` and `/LegalAttestation` as
// arrays of strings and reads both back as arrays of names, so either getter
// throws on anything its own setter wrote and on any conforming file.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/digitalsignature"
)

// TestSeedValueReasonsRoundTrip is the defect.
//
// The expected values are what was set: PDF 32000-1:2008 table 234 gives both
// entries as arrays of text strings, so the setters are the halves that are
// right, and the getters read strings the way `PDSeedValueCertificate.KeyUsage`
// reads the strings it writes. Java casts each entry to COSName and throws
// ClassCastException.
func TestSeedValueReasonsRoundTrip(t *testing.T) {
	seed := digitalsignature.NewPDSeedValue()
	want := []string{"I approve this document", "I am the author"}
	seed.SetReasons(want)

	got := seed.Reasons()
	if len(got) != len(want) {
		t.Fatalf("Reasons() gave %d entries, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Reasons()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestSeedValueLegalAttestationRoundTrip is the same pair on the other entry.
func TestSeedValueLegalAttestationRoundTrip(t *testing.T) {
	seed := digitalsignature.NewPDSeedValue()
	want := []string{"no legal attestation"}
	seed.SetLegalAttestation(want)

	got := seed.LegalAttestation()
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("LegalAttestation() = %q, want %q", got, want)
	}
}

// TestSeedValueReasonsOfNothing keeps the empty answer for a dictionary with
// no /Reasons, which is what the getter already gave.
func TestSeedValueReasonsOfNothing(t *testing.T) {
	if got := digitalsignature.NewPDSeedValue().Reasons(); len(got) != 0 {
		t.Errorf("Reasons() of a fresh seed value = %q, want none", got)
	}
}
