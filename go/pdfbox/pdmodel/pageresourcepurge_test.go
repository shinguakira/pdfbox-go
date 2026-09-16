package pdmodel_test

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	// builds the pattern PDResources.GetPattern answers, and links in the shadings
	_ "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/pattern"
)

// TestRemovePageResourceFromCacheLeavesTheInheritedResources pins
// PDPage.removePageResourceFromCache, which PDFTextStripper.processPage calls
// once a page is done.
//
// Java's DefaultResourceCache holds every entry through a SoftReference, which
// the garbage collector clears when memory is short, and the method exists "to
// avoid relying on the implementation of the Cache". The port's cache holds its
// entries outright. Since the page tree hands every page the document's cache,
// the colour spaces, graphics states, shadings, patterns, property lists,
// fonts and XObjects of every page processed stayed for as long as the document
// did, with nothing to take them out.
//
// The page below has one indirect resource of each kind, a Type 1 and a Type 0
// font with their descriptors, a form XObject with a graphics state of its own,
// and a transparency group with another; the page tree's root has a colour
// space the page inherits. Every value wanted was printed by the running PDFBox,
// over the same objects read the same way: before the purge all fifteen are
// cached, after it only the inherited colour space is.
//
// The transparency group is there because the port first recursed into a
// *form.PDFormXObject only. Java's instanceof takes the PDTransparencyGroup that
// extends it; the Go type embeds it instead, so the group's resources stayed.
// geothermal.pdf in pdf.js's test PDFs kept 7 XObjects in the cache after text
// extraction that PDFBox had taken out.
func TestRemovePageResourceFromCacheLeavesTheInheritedResources(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	cache := pdmodel.NewDefaultResourceCache()
	doc.SetResourceCache(cache)

	array := func(items ...cos.Base) *cos.Array { return cos.NewArrayOf(items) }
	integer := func(value int64) cos.Base { return cos.GetInteger(value) }

	// inherited from the page tree's root
	inheritedParams := cos.NewDictionary()
	inheritedParams.SetItem(cos.WhitePoint, array(integer(1), integer(1), integer(1)))
	inheritedCS := cos.NewObject(array(cos.GetPDFName("CalGray"), inheritedParams))
	inheritedCSs := cos.NewDictionary()
	inheritedCSs.SetItem(cos.GetPDFName("CSInh"), inheritedCS)
	inheritedResources := cos.NewDictionary()
	inheritedResources.SetItem(cos.ColorSpace, inheritedCSs)
	doc.Pages().COSObject().(*cos.Dictionary).SetItem(cos.Resources, inheritedResources)

	page := pdmodel.NewPDPage()
	doc.AddPage(page)
	res := cos.NewDictionary()

	params := cos.NewDictionary()
	params.SetItem(cos.WhitePoint, array(cos.NewFloat(0.9505), integer(1), cos.NewFloat(1.089)))
	cs := cos.NewObject(array(cos.GetPDFName("CalGray"), params))
	css := cos.NewDictionary()
	css.SetItem(cos.GetPDFName("CS1"), cs)
	res.SetItem(cos.ColorSpace, css)

	gsDict := cos.NewDictionary()
	gsDict.SetItem(cos.Type, cos.ExtGState)
	gsDict.SetInt(cos.LW, 2)
	gs := cos.NewObject(gsDict)
	gsDirect := cos.NewDictionary()
	gsDirect.SetItem(cos.Type, cos.ExtGState)
	gss := cos.NewDictionary()
	gss.SetItem(cos.GetPDFName("GS1"), gs)
	gss.SetItem(cos.GetPDFName("GS3"), gsDirect)
	res.SetItem(cos.ExtGState, gss)

	function := cos.NewDictionary()
	function.SetInt(cos.FunctionType, 2)
	function.SetItem(cos.Domain, array(integer(0), integer(1)))
	function.SetItem(cos.C0, array(integer(0), integer(0), integer(0)))
	function.SetItem(cos.C1, array(integer(1), integer(1), integer(1)))
	function.SetInt(cos.N, 1)
	shadingDict := cos.NewDictionary()
	shadingDict.SetInt(cos.ShadingType, 2)
	shadingDict.SetItem(cos.ColorSpace, cos.DeviceRGB)
	shadingDict.SetItem(cos.Coords, array(integer(0), integer(0), integer(1), integer(1)))
	shadingDict.SetItem(cos.Function, function)
	sh := cos.NewObject(shadingDict)
	shs := cos.NewDictionary()
	shs.SetItem(cos.GetPDFName("Sh1"), sh)
	res.SetItem(cos.Shading, shs)

	patternDict := cos.NewDictionary()
	patternDict.SetInt(cos.PatternType, 2)
	patternDict.SetItem(cos.Shading, shadingDict)
	pattern := cos.NewObject(patternDict)
	patterns := cos.NewDictionary()
	patterns.SetItem(cos.GetPDFName("P1"), pattern)
	res.SetItem(cos.Pattern, patterns)

	ocgDict := cos.NewDictionary()
	ocgDict.SetItem(cos.Type, cos.OCG)
	ocgDict.SetString(cos.NameKey, "layer")
	ocg := cos.NewObject(ocgDict)
	properties := cos.NewDictionary()
	properties.SetItem(cos.GetPDFName("MC1"), ocg)
	res.SetItem(cos.Properties, properties)

	fdDict := cos.NewDictionary()
	fdDict.SetItem(cos.Type, cos.FontDescriptor)
	fdDict.SetName(cos.FontName, "Helvetica")
	fdDict.SetInt(cos.Flags, 32)
	fd := cos.NewObject(fdDict)
	type1Dict := cos.NewDictionary()
	type1Dict.SetItem(cos.Type, cos.Font)
	type1Dict.SetItem(cos.Subtype, cos.Type1)
	type1Dict.SetName(cos.BaseFont, "Helvetica")
	type1Dict.SetItem(cos.FontDescriptor, fd)
	type1 := cos.NewObject(type1Dict)

	cidFDDict := cos.NewDictionary()
	cidFDDict.SetItem(cos.Type, cos.FontDescriptor)
	cidFDDict.SetName(cos.FontName, "Foo")
	cidFDDict.SetInt(cos.Flags, 4)
	cidFD := cos.NewObject(cidFDDict)
	sysInfo := cos.NewDictionary()
	sysInfo.SetItem(cos.Registry, cos.NewStringObj("Adobe"))
	sysInfo.SetItem(cos.Ordering, cos.NewStringObj("Identity"))
	sysInfo.SetInt(cos.Supplement, 0)
	cidFontDict := cos.NewDictionary()
	cidFontDict.SetItem(cos.Type, cos.Font)
	cidFontDict.SetItem(cos.Subtype, cos.CIDFontType2)
	cidFontDict.SetName(cos.BaseFont, "Foo")
	cidFontDict.SetItem(cos.CIDSystemInfo, sysInfo)
	cidFontDict.SetItem(cos.FontDescriptor, cidFD)
	cidFont := cos.NewObject(cidFontDict)
	type0Dict := cos.NewDictionary()
	type0Dict.SetItem(cos.Type, cos.Font)
	type0Dict.SetItem(cos.Subtype, cos.Type0)
	type0Dict.SetName(cos.BaseFont, "Foo")
	type0Dict.SetItem(cos.Encoding, cos.IdentityH)
	type0Dict.SetItem(cos.DescendantFonts, array(cidFont))
	type0 := cos.NewObject(type0Dict)
	fonts := cos.NewDictionary()
	fonts.SetItem(cos.GetPDFName("F1"), type1)
	fonts.SetItem(cos.GetPDFName("F0"), type0)
	res.SetItem(cos.Font, fonts)

	gs2Dict := cos.NewDictionary()
	gs2Dict.SetItem(cos.Type, cos.ExtGState)
	gs2 := cos.NewObject(gs2Dict)
	formGSs := cos.NewDictionary()
	formGSs.SetItem(cos.GetPDFName("GS2"), gs2)
	formResources := cos.NewDictionary()
	formResources.SetItem(cos.ExtGState, formGSs)
	formStream := doc.Document().CreateStream()
	formStream.SetItem(cos.Type, cos.XObject)
	formStream.SetItem(cos.Subtype, cos.Form)
	formStream.SetItem(cos.BBox, array(integer(0), integer(0), integer(10), integer(10)))
	formStream.SetItem(cos.Resources, formResources)
	w, err := formStream.CreateWriter()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("/GS2 gs")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	formObject := cos.NewObject(formStream)
	xobjects := cos.NewDictionary()
	xobjects.SetItem(cos.GetPDFName("Fm1"), formObject)

	gs4Dict := cos.NewDictionary()
	gs4Dict.SetItem(cos.Type, cos.ExtGState)
	gs4 := cos.NewObject(gs4Dict)
	groupGSs := cos.NewDictionary()
	groupGSs.SetItem(cos.GetPDFName("GS4"), gs4)
	groupResources := cos.NewDictionary()
	groupResources.SetItem(cos.ExtGState, groupGSs)
	groupAttributes := cos.NewDictionary()
	groupAttributes.SetItem(cos.S, cos.Transparency)
	groupStream := doc.Document().CreateStream()
	groupStream.SetItem(cos.Type, cos.XObject)
	groupStream.SetItem(cos.Subtype, cos.Form)
	groupStream.SetItem(cos.BBox, array(integer(0), integer(0), integer(10), integer(10)))
	groupStream.SetItem(cos.Group, groupAttributes)
	groupStream.SetItem(cos.Resources, groupResources)
	w, err = groupStream.CreateWriter()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("/GS4 gs")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	groupObject := cos.NewObject(groupStream)
	xobjects.SetItem(cos.GetPDFName("Fm2"), groupObject)
	res.SetItem(cos.XObject, xobjects)

	page.COSObject().(*cos.Dictionary).SetItem(cos.Resources, res)

	// every resource read through a page the document hands out, with its cache
	read := doc.Page(0)
	r := read.Resources()
	if r.COSObject() != cos.Base(res) {
		t.Fatalf("the page's resources are not its own dictionary")
	}
	mustRead := func(what string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("reading %s: %v", what, err)
		}
	}
	_, err = r.ColorSpace(cos.GetPDFName("CS1"))
	mustRead("CS1", err)
	r.GetExtGState(cos.GetPDFName("GS1"))
	r.GetExtGState(cos.GetPDFName("GS3"))
	_, err = r.GetShading(cos.GetPDFName("Sh1"))
	mustRead("Sh1", err)
	_, err = r.GetPattern(cos.GetPDFName("P1"))
	mustRead("P1", err)
	r.GetProperties(cos.GetPDFName("MC1"))
	_, err = r.GetFont(cos.GetPDFName("F1"))
	mustRead("F1", err)
	_, err = r.GetFont(cos.GetPDFName("F0"))
	mustRead("F0", err)
	xobject, err := r.GetXObject(cos.GetPDFName("Fm1"))
	mustRead("Fm1", err)
	formXObject, ok := xobject.(*form.PDFormXObject)
	if !ok {
		t.Fatalf("Fm1 is %T, want a form XObject", xobject)
	}
	formXObject.Resources().(*pdmodel.PDResources).GetExtGState(cos.GetPDFName("GS2"))
	// a PDTransparencyGroup, which Java's removeResources takes for the
	// PDFormXObject it extends
	xobject, err = r.GetXObject(cos.GetPDFName("Fm2"))
	mustRead("Fm2", err)
	group, ok := xobject.(*form.PDTransparencyGroup)
	if !ok {
		t.Fatalf("Fm2 is %T, want a transparency group", xobject)
	}
	group.Resources().(*pdmodel.PDResources).GetExtGState(cos.GetPDFName("GS4"))
	_, err = pdmodel.NewPDResourcesOfCache(inheritedResources, cache).ColorSpace(cos.GetPDFName("CSInh"))
	mustRead("CSInh", err)

	type entry struct {
		name   string
		cached func() bool
	}
	entries := []entry{
		{"colour space", func() bool { return cache.GetColorSpace(cs) != nil }},
		{"graphics state", func() bool { return cache.GetExtGState(gs) != nil }},
		{"shading", func() bool { return cache.GetShading(sh) != nil }},
		{"pattern", func() bool { return cache.GetPattern(pattern) != nil }},
		{"property list", func() bool { return cache.GetProperties(ocg) != nil }},
		{"Type 1 font", func() bool { return cache.GetFont(type1) != nil }},
		{"its descriptor", func() bool { return cache.GetFontDescriptor(fd) != nil }},
		{"Type 0 font", func() bool { return cache.GetFont(type0) != nil }},
		{"its CID font", func() bool { return cache.GetCIDFont(cidFont) != nil }},
		{"the CID font's descriptor", func() bool { return cache.GetFontDescriptor(cidFD) != nil }},
		{"form XObject", func() bool { return cache.GetXObject(formObject) != nil }},
		{"the form's graphics state", func() bool { return cache.GetExtGState(gs2) != nil }},
		{"transparency group", func() bool { return cache.GetXObject(groupObject) != nil }},
		{"the group's graphics state", func() bool { return cache.GetExtGState(gs4) != nil }},
		{"inherited colour space", func() bool { return cache.GetColorSpace(inheritedCS) != nil }},
	}
	for _, e := range entries {
		if !e.cached() {
			t.Errorf("before the purge, %s: not cached; PDFBox has it", e.name)
		}
	}

	read.RemovePageResourceFromCache()

	for _, e := range entries {
		want := e.name == "inherited colour space"
		if got := e.cached(); got != want {
			t.Errorf("after the purge, %s: cached = %v, want %v", e.name, got, want)
		}
	}
}
