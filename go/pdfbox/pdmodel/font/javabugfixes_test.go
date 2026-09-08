package font_test

// The `pdmodel/font` half of `track/java-bug-fixes`.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// TestPanoseOfAStyleWithoutOne is JAVA-BUGS 14.
//
// `getPanose` checks that the /Style dictionary is there and then casts
// `style.getDictionaryObject(PANOSE)` without checking it, so a /Style with no
// /Panose throws NullPointerException. The method answers null for a descriptor
// with no /Style at all, one line above, and for a /Panose shorter than 12
// bytes, two lines below.
func TestPanoseOfAStyleWithoutOne(t *testing.T) {
	descriptor := font.NewPDFontDescriptor()
	style := cos.NewDictionary()
	style.SetString(cos.GetPDFName("Something"), "else")
	descriptor.COSObject().(*cos.Dictionary).SetItem(cos.Style, style)

	if got := descriptor.Panose(); got != nil {
		t.Errorf("Panose() answered %v for a /Style with no /Panose, want nil", got)
	}
}
