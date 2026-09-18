package pdmodel

import (
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
)

// An ICCBased colour space's /Alternate is built without the resources the
// ICCBased space was found in.
//
// Java's PDICCBased.getAlternateColorSpace wraps the entry in an array and hands
// it to the one-argument PDColorSpace.create, so the name there is never looked
// up. Looking it up can lead straight back: where the resources map /DefaultRGB
// to an ICCBased space whose alternate is /DeviceRGB, /DeviceRGB resolves to
// /DefaultRGB, which is the same ICCBased space again. The Go version did that,
// and died of a stack overflow on pdfjs/issue9285.pdf, whose two pages carry
// exactly those resources and whose images PDFBox reads.
//
// The expected values are PDFBox's, printed for these dictionaries with a
// profile stream whose bytes are not a profile. That sends Java down its
// fallback to the alternate, which is the path the Go version always takes.

// iccBasedOf answers [/ICCBased stream] for a stream with the given /N and
// /Alternate, leaving /Alternate out where it is nil.
func iccBasedOf(n int, alternate cos.Base) *cos.Array {
	stream := cos.NewStream(nil)
	stream.SetInt(cos.N, n)
	if alternate != nil {
		stream.SetItem(cos.Alternate, alternate)
	}
	array := cos.NewArray()
	array.Add(cos.ICCBased)
	array.Add(stream)
	return array
}

func resourcesWithColorSpaces(spaces map[*cos.Name]cos.Base) *PDResources {
	dictionary := cos.NewDictionary()
	for name, space := range spaces {
		dictionary.SetItem(name, space)
	}
	r := NewPDResources()
	r.Dictionary().SetItem(cos.ColorSpace, dictionary)
	return r
}

// alternateOf answers the alternate of an ICCBased space, failing the test for
// anything else.
func alternateOf(t *testing.T, what string, space color.PDColorSpace) color.PDColorSpace {
	t.Helper()
	icc, ok := space.(*color.PDICCBased)
	if !ok {
		t.Fatalf("%s is %T, want ICCBased", what, space)
	}
	return icc.AlternateColorSpace()
}

// TestICCBasedAlternateDefaultRGBLoop is the shape of issue9285.pdf.
//
// PDFBox prints, for the three ways in:
//
//	[/Indexed /DeviceRGB 1 <000000FFFFFF>]  Indexed(base=ICCBased(alternate=DeviceRGB))
//	/DeviceRGB                              ICCBased(alternate=DeviceRGB)
//	[/ICCBased stream]                      ICCBased(alternate=DeviceRGB)
func TestICCBasedAlternateDefaultRGBLoop(t *testing.T) {
	space := iccBasedOf(3, cos.DeviceRGB)
	r := resourcesWithColorSpaces(map[*cos.Name]cos.Base{
		cos.DefaultRGB: space,
		cos.DeviceRGB:  space,
	})

	indexed := cos.NewArray()
	indexed.Add(cos.Indexed)
	indexed.Add(cos.DeviceRGB)
	indexed.Add(cos.IntegerOne)
	indexed.Add(cos.NewStringObjBytes([]byte{0, 0, 0, 255, 255, 255}))
	got, err := color.CreateOfResources(indexed, r)
	if err != nil {
		t.Fatalf("the Indexed space: %v", err)
	}
	base := got.(*color.PDIndexed).BaseColorSpace()
	if alternate := alternateOf(t, "the Indexed space's base", base); alternate != color.PDColorSpace(color.DeviceRGB) {
		t.Errorf("the Indexed space's base has the alternate %v, want DeviceRGB", alternate)
	}

	got, err = color.CreateOfResources(cos.DeviceRGB, r)
	if err != nil {
		t.Fatalf("/DeviceRGB: %v", err)
	}
	if alternate := alternateOf(t, "/DeviceRGB", got); alternate != color.PDColorSpace(color.DeviceRGB) {
		t.Errorf("/DeviceRGB has the alternate %v, want DeviceRGB", alternate)
	}

	got, err = color.CreateOfResources(space, r)
	if err != nil {
		t.Fatalf("the ICCBased array: %v", err)
	}
	if alternate := alternateOf(t, "the ICCBased array", got); alternate != color.PDColorSpace(color.DeviceRGB) {
		t.Errorf("the ICCBased array has the alternate %v, want DeviceRGB", alternate)
	}
}

// TestICCBasedAlternateDefaultGrayLoop is the same through /DefaultGray. PDFBox
// prints ICCBased(alternate=DeviceGray).
func TestICCBasedAlternateDefaultGrayLoop(t *testing.T) {
	r := resourcesWithColorSpaces(map[*cos.Name]cos.Base{
		cos.DefaultGray: iccBasedOf(1, cos.DeviceGray),
	})
	got, err := color.CreateOfResources(cos.DeviceGray, r)
	if err != nil {
		t.Fatalf("/DeviceGray: %v", err)
	}
	if alternate := alternateOf(t, "/DeviceGray", got); alternate != color.PDColorSpace(color.DeviceGray) {
		t.Errorf("/DeviceGray has the alternate %v, want DeviceGray", alternate)
	}
}

// TestICCBasedAlternateResourceName checks that an /Alternate naming a colour
// space resource is not looked up. The resources map /CS0 to /DeviceRGB, and
// PDFBox still refuses it: "Invalid color space kind: COSName{CS0}".
func TestICCBasedAlternateResourceName(t *testing.T) {
	r := resourcesWithColorSpaces(map[*cos.Name]cos.Base{
		cos.GetPDFName("CS0"): cos.DeviceRGB,
	})
	_, err := color.CreateOfResources(iccBasedOf(3, cos.GetPDFName("CS0")), r)
	if err == nil || !strings.Contains(err.Error(), "Invalid color space kind") {
		t.Errorf("an /Alternate naming a resource gave %v, want PDFBox's invalid color space kind", err)
	}
}
