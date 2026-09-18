package pdmodel_test

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/pattern"
)

// A /Pattern colour space is not put in the resource cache.
//
// Java says why on the line that skips it: "we can't cache PDPattern, because it
// holds page resources, see PDFBOX-2370". A PDPattern carries the resources it
// was created from, and it looks a pattern name up in them when a colour asks
// for one; handed back to another page from the cache, it looks the name up in
// the first page's resources and does not find it. The port cached it, because
// the comment beside its cache still said /Pattern was not ported.
//
// Found by the corpus comparison of rendered pages: 37 pages of `itextsharp`'s
// `PdfCopyTest/cmp_copyLargeFile.pdf` -- which PDFBox draws -- failed with
// "pattern COSName{P1} was not found", and only after an earlier page of the
// same document had been drawn.
//
// The expected values are PDFBox's, printed for two PDResources sharing one
// cache and one indirect [/Pattern] colour space: two different PDPattern
// objects, nothing in the cache under that object, and each page seeing its own
// pattern.
func TestAPatternColourSpaceIsNotCached(t *testing.T) {
	cache := pdmodel.NewDefaultResourceCache()
	key, err := cos.NewObjectKey(42, 0)
	if err != nil {
		t.Fatalf("the object key: %v", err)
	}
	space := cos.NewObjectWithKey(patternArray(), key)

	first := patternResources(space, "P1", cache)
	second := patternResources(space, "P2", cache)

	a, err := first.ColorSpace(cos.GetPDFName("CS0"))
	if err != nil {
		t.Fatalf("the first page's colour space: %v", err)
	}
	b, err := second.ColorSpace(cos.GetPDFName("CS0"))
	if err != nil {
		t.Fatalf("the second page's colour space: %v", err)
	}
	if _, isPattern := a.(*pattern.PDPattern); !isPattern {
		t.Fatalf("the first page's colour space is %T, want a pattern", a)
	}
	if a == b {
		t.Error("both pages were handed the same pattern colour space, and PDFBox makes one each")
	}
	if cached := cache.GetColorSpace(space); cached != nil {
		t.Errorf("the cache holds %T under the colour space's object, and PDFBox holds nothing", cached)
	}

	// Each page finds its own pattern, which is what the cache would break.
	for _, c := range []struct {
		resources *pdmodel.PDResources
		name      string
	}{{first, "P1"}, {second, "P2"}} {
		found, err := c.resources.GetPattern(cos.GetPDFName(c.name))
		if err != nil {
			t.Errorf("looking %s up: %v", c.name, err)
			continue
		}
		if found == nil {
			t.Errorf("%s was not found, and PDFBox finds it", c.name)
		}
	}
}

// patternArray is the colour space [/Pattern].
func patternArray() *cos.Array {
	array := cos.NewArray()
	array.Add(cos.Pattern)
	return array
}

// patternResources answers resources naming that colour space as /CS0 and
// holding one tiling pattern under the given name.
func patternResources(space cos.Base, name string, cache pdmodel.ResourceCache) *pdmodel.PDResources {
	spaces := cos.NewDictionary()
	spaces.SetItem(cos.GetPDFName("CS0"), space)

	tiling := cos.NewStream(nil)
	tiling.SetItem(cos.Type, cos.GetPDFName("Pattern"))
	tiling.SetInt(cos.PatternType, 1)
	box := cos.NewArray()
	for _, value := range []int64{0, 0, 10, 10} {
		box.Add(cos.GetInteger(value))
	}
	tiling.SetItem(cos.BBox, box)
	tiling.SetInt(cos.XStep, 10)
	tiling.SetInt(cos.YStep, 10)
	patterns := cos.NewDictionary()
	patterns.SetItem(cos.GetPDFName(name), tiling)

	dictionary := cos.NewDictionary()
	dictionary.SetItem(cos.ColorSpace, spaces)
	dictionary.SetItem(cos.Pattern, patterns)
	return pdmodel.NewPDResourcesOfCache(dictionary, cache)
}
