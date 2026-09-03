package pdmodel

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Written from org.apache.pdfbox.pdmodel.PDResources. The Java suite has no
// test for the class; the tests it does have for resources go through PDPage
// and the font and colour types, which this port has not reached.

func TestPDResourcesConstruction(t *testing.T) {
	r := NewPDResources()
	if r.Dictionary() == nil {
		t.Fatal("a new set of resources has no dictionary")
	}
	if r.COSObject() != cos.Base(r.Dictionary()) {
		t.Error("COSObject and Dictionary disagree")
	}

	d := cos.NewDictionary()
	if NewPDResourcesOf(d).Dictionary() != d {
		t.Error("the resources did not keep the dictionary they were given")
	}
}

// TestPDResourcesNilDictionary pins that a missing dictionary is a caller's
// mistake, which Java reports with IllegalArgumentException.
func TestPDResourcesNilDictionary(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("no panic on a nil dictionary, want one")
		}
	}()
	NewPDResourcesOf(nil)
}

func TestPDResourcesNames(t *testing.T) {
	fonts := cos.NewDictionary()
	fonts.SetItem(cos.GetPDFName("F1"), cos.NewDictionary())
	fonts.SetItem(cos.GetPDFName("F2"), cos.NewDictionary())
	dict := cos.NewDictionary()
	dict.SetItem(cos.Font, fonts)

	r := NewPDResourcesOf(dict)
	names := r.FontNames()
	if len(names) != 2 {
		t.Fatalf("FontNames = %v, want two names", names)
	}
	// The dictionary keeps insertion order, so the names come back in the order
	// the file listed them.
	if names[0].Name() != "F1" || names[1].Name() != "F2" {
		t.Errorf("FontNames = %v, want [F1 F2]", names)
	}

	// A kind that is not in the dictionary yields nothing rather than failing.
	for _, got := range [][]*cos.Name{
		r.ColorSpaceNames(), r.XObjectNames(), r.PropertiesNames(),
		r.ShadingNames(), r.PatternNames(), r.ExtGStateNames(),
	} {
		if len(got) != 0 {
			t.Errorf("an absent resource kind yielded %v", got)
		}
	}
}

func TestPDResourcesHasColorSpace(t *testing.T) {
	r := NewPDResources()
	if r.HasColorSpace(cos.DeviceRGB) {
		t.Error("empty resources have a color space")
	}

	colorSpaces := cos.NewDictionary()
	colorSpaces.SetItem(cos.GetPDFName("CS0"), cos.NewArray())
	r.Dictionary().SetItem(cos.ColorSpace, colorSpaces)

	if !r.HasColorSpace(cos.GetPDFName("CS0")) {
		t.Error("the color space that was added is not there")
	}
}

func TestPDResourcesIsImageXObject(t *testing.T) {
	image := cos.NewStream(nil)
	image.SetItem(cos.Subtype, cos.Image)
	form := cos.NewStream(nil)
	form.SetItem(cos.Subtype, cos.Form)

	xobjects := cos.NewDictionary()
	xobjects.SetItem(cos.GetPDFName("Im0"), image)
	xobjects.SetItem(cos.GetPDFName("Fm0"), form)
	xobjects.SetItem(cos.GetPDFName("Bad"), cos.NewDictionary())
	dict := cos.NewDictionary()
	dict.SetItem(cos.XObject, xobjects)

	r := NewPDResourcesOf(dict)
	if !r.IsImageXObject(cos.GetPDFName("Im0")) {
		t.Error("the image XObject is not reported as an image")
	}
	if r.IsImageXObject(cos.GetPDFName("Fm0")) {
		t.Error("the form XObject is reported as an image")
	}
	if r.IsImageXObject(cos.GetPDFName("Bad")) {
		t.Error("a dictionary that is not a stream is reported as an image")
	}
	if r.IsImageXObject(cos.GetPDFName("Missing")) {
		t.Error("a missing XObject is reported as an image")
	}
}

// TestPDResourcesIsImageXObjectIndirect pins that an indirect reference is
// followed before the subtype is read.
func TestPDResourcesIsImageXObjectIndirect(t *testing.T) {
	image := cos.NewStream(nil)
	image.SetItem(cos.Subtype, cos.Image)

	xobjects := cos.NewDictionary()
	xobjects.SetItem(cos.GetPDFName("Im0"), cos.NewObject(image))
	dict := cos.NewDictionary()
	dict.SetItem(cos.XObject, xobjects)

	if !NewPDResourcesOf(dict).IsImageXObject(cos.GetPDFName("Im0")) {
		t.Error("an indirect image XObject is not reported as an image")
	}
}

func TestPDResourcesAddAllocatesName(t *testing.T) {
	r := NewPDResources()

	first := r.add(cos.Font, "F", cos.NewDictionary())
	if first.Name() != "F1" {
		t.Errorf("first name = %q, want F1", first.Name())
	}
	second := r.add(cos.Font, "F", cos.NewDictionary())
	if second.Name() != "F2" {
		t.Errorf("second name = %q, want F2", second.Name())
	}

	// The resource went into a sub-dictionary under its kind.
	fonts := r.Dictionary().GetCOSDictionary(cos.Font)
	if fonts == nil || fonts.Size() != 2 {
		t.Fatalf("the font dictionary holds %v", fonts)
	}
}

// TestPDResourcesAddKeepsExistingName pins that adding a resource that is
// already there gives back the name it already has rather than a second entry.
func TestPDResourcesAddKeepsExistingName(t *testing.T) {
	r := NewPDResources()
	font := cos.NewDictionary()

	first := r.add(cos.Font, "F", font)
	second := r.add(cos.Font, "F", font)

	if first != second {
		t.Errorf("the same resource went in twice, as %q and %q", first.Name(), second.Name())
	}
	if got := r.Dictionary().GetCOSDictionary(cos.Font).Size(); got != 1 {
		t.Errorf("the font dictionary holds %d entries, want 1", got)
	}
}

// TestPDResourcesAddFindsIndirectDuplicate covers PDFBOX-4509: a font taken
// from a loaded document sits in the dictionary behind an indirect reference,
// and adding it again must find it there rather than add a copy.
func TestPDResourcesAddFindsIndirectDuplicate(t *testing.T) {
	font := cos.NewDictionary()
	fonts := cos.NewDictionary()
	fonts.SetItem(cos.GetPDFName("F7"), cos.NewObject(font))
	dict := cos.NewDictionary()
	dict.SetItem(cos.Font, fonts)

	r := NewPDResourcesOf(dict)
	if got := r.add(cos.Font, "F", font); got.Name() != "F7" {
		t.Errorf("name = %q, want F7 — the indirect entry already holds it", got.Name())
	}

	// The same search is not made for other kinds of resource.
	xobject := cos.NewDictionary()
	xobjects := cos.NewDictionary()
	xobjects.SetItem(cos.GetPDFName("Im7"), cos.NewObject(xobject))
	dict.SetItem(cos.XObject, xobjects)
	if got := r.add(cos.XObject, "Im", xobject); got.Name() == "Im7" {
		t.Error("the indirect search was made for an XObject as well")
	}
}

// TestPDResourcesCreateKeySkipsTakenNames pins that a name already in use is
// stepped over rather than overwritten.
func TestPDResourcesCreateKeySkipsTakenNames(t *testing.T) {
	fonts := cos.NewDictionary()
	fonts.SetItem(cos.GetPDFName("F1"), cos.NewDictionary())
	fonts.SetItem(cos.GetPDFName("F2"), cos.NewDictionary())
	fonts.SetItem(cos.GetPDFName("F3"), cos.NewDictionary())
	dict := cos.NewDictionary()
	dict.SetItem(cos.Font, fonts)

	r := NewPDResourcesOf(dict)
	if got := r.createKey(cos.Font, "F"); got.Name() != "F4" {
		t.Errorf("createKey = %q, want F4", got.Name())
	}
}

func TestPDResourcesGetIndirect(t *testing.T) {
	direct := cos.NewDictionary()
	indirect := cos.NewObject(cos.NewDictionary())
	fonts := cos.NewDictionary()
	fonts.SetItem(cos.GetPDFName("F1"), direct)
	fonts.SetItem(cos.GetPDFName("F2"), indirect)
	dict := cos.NewDictionary()
	dict.SetItem(cos.Font, fonts)

	r := NewPDResourcesOf(dict)
	if got := r.getIndirect(cos.Font, cos.GetPDFName("F1")); got != nil {
		t.Errorf("a direct resource came back as the indirect object %v", got)
	}
	if got := r.getIndirect(cos.Font, cos.GetPDFName("F2")); got != indirect {
		t.Errorf("getIndirect = %v, want the indirect object", got)
	}
	if got := r.getIndirect(cos.XObject, cos.GetPDFName("F2")); got != nil {
		t.Errorf("an absent kind gave %v, want nil", got)
	}
}
