package schema_test

// JAVA-BUGS 54: `TiffSchema.setArtist` builds a plain TextType where the field
// is declared a ProperName, and `getArtistProperty` asks for a ProperNameType,
// which a TextType is not -- so the value set through the setter is invisible
// to the getter.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
)

// TestTiffArtistRoundTrip is the defect.
//
// The expected value is what was set: a setter and the getter beside it are
// one accessor pair. The field's declared type is `Types.ProperName`, so the
// setter builds that type -- which is what `instanciateSimple` reads the
// declaration for and what every other setter of a derived text field does.
func TestTiffArtistRoundTrip(t *testing.T) {
	metadata := xmpbox.CreateXMPMetadata()
	tiff, err := schema.NewTiffSchema(metadata)
	noError(t, "NewTiffSchema", err)

	noError(t, "SetArtist", tiff.SetArtist("Hokusai"))

	if got := tiff.ArtistProperty(); got == nil {
		t.Error("ArtistProperty() answered nothing for an artist that was set")
	}
	if got := tiff.Artist(); got != "Hokusai" {
		t.Errorf("Artist() = %q, want %q", got, "Hokusai")
	}
}
