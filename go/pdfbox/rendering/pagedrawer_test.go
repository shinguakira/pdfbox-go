package rendering

// PageDrawer against a recording backend.
//
// Java's three tests here -- TestRendering, TestQuality and TestPDFToImage --
// all compare pixels: TestRendering renders and asserts nothing threw,
// TestQuality reads back four pixels of four files from target/pdfs, and
// TestPDFToImage is disabled in Java itself because different JVMs give
// different results. None of the three can be ported: there is no rasteriser
// and, per slice 9's A5 decision, pixels are not what this port compares
// against. What is ported is the question each was asking -- did the drawer
// decide to draw the right things, in the right order, in the right colour --
// asked of the real engine over a real content stream. See migration/STATUS.md.

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/optionalcontent"
)

// pageWithContent returns a one-page document whose page is the given size and
// whose content stream is the given operators.
func pageWithContent(t *testing.T, width, height float32, content string) (
	*pdmodel.PDDocument, *pdmodel.PDPage) {
	t.Helper()
	document := pdmodel.NewPDDocument()
	page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(width, height))
	document.AddPage(page)

	stream := common.NewPDStreamOfDocument(document)
	out, err := stream.CreateOutputStream()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := out.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
	page.SetContents(stream)
	return document, page
}

// renderToRecording runs the given content stream through the real renderer and
// returns what the drawer asked the backend to do.
func renderToRecording(t *testing.T, width, height float32, content string) *recordingBackend {
	t.Helper()
	document, _ := pageWithContent(t, width, height, content)
	backend := newRecordingBackend()
	renderer := NewPDFRenderer(document)
	if err := renderer.RenderPageToBackend(0, backend, 1, 1, Export); err != nil {
		t.Fatal(err)
	}
	return backend
}

// wantDrawn compares the drawing calls with what was expected.
func wantDrawn(t *testing.T, backend *recordingBackend, want ...string) {
	t.Helper()
	got := backend.Drawn()
	if len(got) != len(want) {
		t.Fatalf("drew %d things, want %d:\ngot:\n  %s\nwant:\n  %s",
			len(got), len(want), strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("call %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestFillsARectangleInTheColourItWasGiven is the simplest whole path through
// the engine: a colour space, a colour, a rectangle, a fill.
//
// The shape reaches the backend in user space, exactly as Java hands linePath
// to Graphics2D.fill; the flip onto the device is in the transform, which
// TestDrawPageInstallsTheFlippedTransform pins separately.
func TestFillsARectangleInTheColourItWasGiven(t *testing.T) {
	backend := renderToRecording(t, 100, 100, "1 0 0 rg 10 20 30 40 re f")
	wantDrawn(t, backend, "fill [10.00 20.00 30.00 40.00] paint=color(1.000 0.000 0.000 1.000)")
}

// TestStrokeCarriesTheLineParameters pins that the stroke the backend is handed
// is the one the graphics state describes, with the CTM applied to the width.
// The CTM here is the identity, so a width of 4 stays 4.
func TestStrokeCarriesTheLineParameters(t *testing.T) {
	backend := renderToRecording(t, 100, 100, "4 w 1 J 1 j 3 M 0 0 1 RG 10 10 m 50 50 l S")
	wantDrawn(t, backend,
		"draw [10.00 10.00 40.00 40.00] paint=color(0.000 0.000 1.000 1.000) "+
			"stroke=w=4.000 cap=1 join=1 miter=3.0 dash=[] phase=0.000")
}

// TestMinimumLineWidthIsAdobes pins the one value getStroke invents: Adobe
// draws a hairline no thinner than a quarter of a point, so a zero width line
// is stroked at 0.25.
func TestMinimumLineWidthIsAdobes(t *testing.T) {
	backend := renderToRecording(t, 100, 100, "0 w 10 10 m 50 10 l S")
	if !strings.Contains(backend.Drawn()[0], "w=0.250") {
		t.Errorf("stroke = %q, want a line width of 0.25", backend.Drawn()[0])
	}
}

// TestAllZeroDashIsInvisible pins PDFBOX-5168: a dash array of nothing but
// zeros draws nothing at all, which Adobe does and a naive stroker does not.
func TestAllZeroDashIsInvisible(t *testing.T) {
	backend := renderToRecording(t, 100, 100, "[0 0] 0 d 10 10 m 50 10 l S")
	if !strings.Contains(backend.Drawn()[0], "stroke=invisible") {
		t.Errorf("stroke = %q, want the all-zero dash array to be invisible", backend.Drawn()[0])
	}
}

// TestDashArrayIsTransformedAndFloored pins the two numbers getDashArray
// invents: each entry goes through the CTM, and is then floored at 0.062,
// which is the minimum that avoids the JVM crashes of PDFBOX-2373 and its
// siblings. The scale here is 1, so 3 stays 3 and 0 becomes 0.062.
func TestDashArrayIsTransformedAndFloored(t *testing.T) {
	backend := renderToRecording(t, 100, 100, "[3 0] 1 d 10 10 m 50 10 l S")
	if !strings.Contains(backend.Drawn()[0], "dash=[3 0.062] phase=1.000") {
		t.Errorf("stroke = %q, want dash=[3 0.062] phase=1.000", backend.Drawn()[0])
	}
}

// TestMiterLimitBelowOneIsIgnored pins the other value getStroke invents: the
// specification's minimum is 1, and anything under it is replaced by the
// default of 10 rather than passed on.
func TestMiterLimitBelowOneIsIgnored(t *testing.T) {
	backend := renderToRecording(t, 100, 100, "0.5 M 10 10 m 50 10 l S")
	if !strings.Contains(backend.Drawn()[0], "miter=10.0") {
		t.Errorf("stroke = %q, want a miter limit of 10", backend.Drawn()[0])
	}
}

// TestLineCapAndJoinAreClamped pins the clamp Java writes as min(2, max(0, x)):
// the PDF's legal values are 0 to 2, and anything else is pulled into range
// rather than passed to the stroker.
func TestLineCapAndJoinAreClamped(t *testing.T) {
	backend := renderToRecording(t, 100, 100, "9 J 9 j 10 10 m 50 10 l S")
	if !strings.Contains(backend.Drawn()[0], "cap=2 join=2") {
		t.Errorf("stroke = %q, want cap and join clamped to 2", backend.Drawn()[0])
	}
}

// TestFillAndStrokeDrawsTwice pins that B fills and then strokes the same path,
// in that order, and that the fill resetting the path does not lose it.
func TestFillAndStrokeDrawsTwice(t *testing.T) {
	backend := renderToRecording(t, 100, 100, "1 0 0 rg 0 0 1 RG 10 20 30 40 re B")
	wantDrawn(t, backend,
		"fill [10.00 20.00 30.00 40.00] paint=color(1.000 0.000 0.000 1.000)",
		"draw [10.00 20.00 30.00 40.00] paint=color(0.000 0.000 1.000 1.000) "+
			"stroke=w=1.000 cap=0 join=0 miter=10.0 dash=[] phase=0.000")
}

// TestEndPathDrawsNothing pins that n ends the path without painting it, which
// is what makes "W n" a clip and nothing else.
func TestEndPathDrawsNothing(t *testing.T) {
	backend := renderToRecording(t, 100, 100, "10 20 30 40 re W n 0 0 10 10 re f")
	wantDrawn(t, backend, "fill [0.00 0.00 10.00 10.00] paint=color(0.000 0.000 0.000 1.000)")
}

// TestClipIsIntersectedBeforeTheNextPaint pins that W applies to the painting
// operator that follows it, and that the clip the backend is handed is the
// intersection rather than the last path.
func TestClipIsIntersectedBeforeTheNextPaint(t *testing.T) {
	backend := renderToRecording(t, 100, 100, "10 10 20 20 re W n 0 0 100 100 re f")
	clip := backend.Clip()
	if clip == nil {
		t.Fatal("no clip was set")
	}
	bounds := clip.Bounds2D()
	if bounds.X != 10 || bounds.Y != 10 || bounds.Width != 20 || bounds.Height != 20 {
		t.Errorf("clip bounds = %v, want [10 10 20 20]", bounds)
	}
}

// TestTransparentColorSpaceIsPaintedTransparent pins PDFBOX-5782: a colour with
// no colour space at all is drawn as transparency rather than as black, and
// PDFBOX-4900, that the separation colorant named None is too.
func TestTransparentColorSpaceIsPaintedTransparent(t *testing.T) {
	// /CS0 is a Separation whose colorant is /None
	content := "/CS0 cs 1 sc 10 10 20 20 re f"
	document, page := pageWithContent(t, 100, 100, content)

	separation := cos.NewArray()
	separation.Add(cos.Separation)
	separation.Add(cos.None)
	separation.Add(cos.DeviceGray)
	tint := cos.NewDictionary()
	tint.SetInt(cos.FunctionType, 2)
	domain := cos.NewArray()
	domain.Add(cos.NewFloat(0))
	domain.Add(cos.NewFloat(1))
	tint.SetItem(cos.Domain, domain)
	tint.SetInt(cos.N, 1)
	separation.Add(tint)

	resources := pdmodel.NewPDResources()
	colorSpaces := cos.NewDictionary()
	colorSpaces.SetItem(cos.GetPDFName("CS0"), separation)
	resources.Dictionary().SetItem(cos.ColorSpace, colorSpaces)
	page.SetResources(resources)

	backend := newRecordingBackend()
	if err := NewPDFRenderer(document).RenderPageToBackend(0, backend, 1, 1, Export); err != nil {
		t.Fatal(err)
	}
	wantDrawn(t, backend, "fill [10.00 10.00 20.00 20.00] paint=color(0.000 0.000 0.000 0.000)")
}

// TestHiddenOptionalContentIsNotDrawn pins the marked content half of
// isContentRendered: a BDC naming an optional content group that is off swallows
// everything until the matching EMC, and the drawing after it comes back.
func TestHiddenOptionalContentIsNotDrawn(t *testing.T) {
	content := "/OC /MC0 BDC 10 10 20 20 re f EMC 50 50 10 10 re f"
	document, page := pageWithContent(t, 100, 100, content)

	group := optionalcontent.NewPDOptionalContentGroup("hidden")
	properties := cos.NewDictionary()
	properties.SetItem(cos.GetPDFName("MC0"), group.COSObject())
	resources := pdmodel.NewPDResources()
	resources.Dictionary().SetItem(cos.Properties, properties)
	page.SetResources(resources)

	// turn the group off in the catalogue
	ocProperties := optionalcontent.NewPDOptionalContentProperties()
	ocProperties.AddGroup(group)
	ocProperties.SetGroupEnabled(group, false)
	document.DocumentCatalog().SetOCProperties(ocProperties)

	backend := newRecordingBackend()
	if err := NewPDFRenderer(document).RenderPageToBackend(0, backend, 1, 1, Export); err != nil {
		t.Fatal(err)
	}
	wantDrawn(t, backend, "fill [50.00 50.00 10.00 10.00] paint=color(0.000 0.000 0.000 1.000)")
}

// TestDrawPageWithoutABackendSaysSo pins what the B0 decision costs: there is
// no raster backend, so asking for an image says that rather than answering a
// blank page, which would look like a rendered one.
func TestDrawPageWithoutABackendSaysSo(t *testing.T) {
	document, _ := pageWithContent(t, 100, 100, "10 10 20 20 re f")
	err := NewPDFRenderer(document).RenderImage(0)
	if !errors.Is(err, ErrNoBackend) {
		t.Errorf("RenderImage = %v, want ErrNoBackend", err)
	}
}

// TestDrawPageInstallsTheFlippedTransform pins the mapping from PDF user space
// onto the device: y grows upwards in a PDF and downwards on a raster, so
// drawPage translates by the page height and scales y by -1, then shifts by the
// crop box's own origin.
func TestDrawPageInstallsTheFlippedTransform(t *testing.T) {
	document := pdmodel.NewPDDocument()
	page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOf(5, 7, 100, 200))
	document.AddPage(page)

	backend := newRecordingBackend()
	if err := NewPDFRenderer(document).RenderPageToBackend(0, backend, 1, 1, Export); err != nil {
		t.Fatal(err)
	}
	at := backend.Transform()
	// translate (0, 200), scale (1, -1), then translate (-5, -7) under the flip:
	// x moves by -5 and y by 200 - (-1 * -7) reversed, which is 200 + 7
	if at.ScaleX() != 1 || at.ScaleY() != -1 {
		t.Errorf("scale = (%v, %v), want (1, -1)", at.ScaleX(), at.ScaleY())
	}
	if at.TranslateX() != -5 {
		t.Errorf("translate x = %v, want -5", at.TranslateX())
	}
	if at.TranslateY() != 200+7 {
		t.Errorf("translate y = %v, want %v", at.TranslateY(), 200+7)
	}
}

// TestScaleReachesTheTransform pins that the scale renderPageToGraphics is
// given lands on the device transform, which is what a DPI other than 72 turns
// into.
func TestScaleReachesTheTransform(t *testing.T) {
	document, _ := pageWithContent(t, 100, 100, "")
	backend := newRecordingBackend()
	if err := NewPDFRenderer(document).RenderPageToBackend(0, backend, 2, 3, Export); err != nil {
		t.Fatal(err)
	}
	at := backend.Transform()
	if at.ScaleX() != 2 || at.ScaleY() != -3 {
		t.Errorf("scale = (%v, %v), want (2, -3)", at.ScaleX(), at.ScaleY())
	}
}

// TestRotatedPageTranslatesBeforeRotating pins the transform of a page that
// says /Rotate 90: the page is turned about the origin, so it has to be pushed
// back into view by its own height first.
func TestRotatedPageTranslatesBeforeRotating(t *testing.T) {
	document := pdmodel.NewPDDocument()
	page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(100, 200))
	page.SetRotation(90)
	document.AddPage(page)

	backend := newRecordingBackend()
	if err := NewPDFRenderer(document).RenderPageToBackend(0, backend, 1, 1, Export); err != nil {
		t.Fatal(err)
	}
	// scale, then translate by the crop box height, then rotate 90 degrees:
	// the rotation makes the x axis point down and the y axis point right, and
	// the flip that follows turns the y axis back.
	at := backend.Transform()
	if math.Abs(at.ShearX()-1) > 1e-9 || math.Abs(at.ShearY()-1) > 1e-9 {
		t.Errorf("shear = (%v, %v), want (1, 1) from the 90 degree rotation",
			at.ShearX(), at.ShearY())
	}
	if math.Abs(at.ScaleX()) > 1e-9 || math.Abs(at.ScaleY()) > 1e-9 {
		t.Errorf("scale = (%v, %v), want (0, 0) from the 90 degree rotation",
			at.ScaleX(), at.ScaleY())
	}
}

// TestShadingFillPaintsTheNamedShading pins the sh operator: the shading is
// looked up in the resources, the clip is replaced by the shading's own area,
// and the paint names the shading rather than a colour.
func TestShadingFillPaintsTheNamedShading(t *testing.T) {
	document, page := pageWithContent(t, 100, 100, "10 10 20 20 re W n /Sh0 sh")

	function := cos.NewDictionary()
	function.SetInt(cos.FunctionType, 2)
	domain := cos.NewArray()
	domain.Add(cos.NewFloat(0))
	domain.Add(cos.NewFloat(1))
	function.SetItem(cos.Domain, domain)
	function.SetInt(cos.N, 1)

	shadingDict := cos.NewDictionary()
	shadingDict.SetInt(cos.ShadingType, 2)
	shadingDict.SetItem(cos.ColorSpace, cos.DeviceGray)
	coords := cos.NewArray()
	for _, v := range []float32{0, 0, 100, 0} {
		coords.Add(cos.NewFloat(v))
	}
	shadingDict.SetItem(cos.Coords, coords)
	shadingDict.SetItem(cos.Function, function)

	shadings := cos.NewDictionary()
	shadings.SetItem(cos.GetPDFName("Sh0"), shadingDict)
	resources := pdmodel.NewPDResources()
	resources.Dictionary().SetItem(cos.Shading, shadings)
	page.SetResources(resources)

	backend := newRecordingBackend()
	if err := NewPDFRenderer(document).RenderPageToBackend(0, backend, 1, 1, Export); err != nil {
		t.Fatal(err)
	}
	wantDrawn(t, backend, "fill [10.00 10.00 20.00 20.00] paint=shading(type=2)")
}

// TestFormXObjectIsWalkedInto pins the Do operator over a form: its content
// stream runs with the form's own matrix concatenated, so the rectangle inside
// it lands where the matrix puts it.
func TestFormXObjectIsWalkedInto(t *testing.T) {
	document, page := pageWithContent(t, 100, 100, "/Fm0 Do")

	form := document.CreateStream()
	form.SetItem(cos.Type, cos.XObject)
	form.SetItem(cos.Subtype, cos.Form)
	bbox := cos.NewArray()
	for _, v := range []float32{0, 0, 100, 100} {
		bbox.Add(cos.NewFloat(v))
	}
	form.SetItem(cos.BBox, bbox)
	matrix := cos.NewArray()
	for _, v := range []float32{1, 0, 0, 1, 30, 40} {
		matrix.Add(cos.NewFloat(v))
	}
	form.SetItem(cos.Matrix, matrix)
	out, err := form.CreateWriter()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := out.Write([]byte("0 0 10 10 re f")); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}

	xobjects := cos.NewDictionary()
	xobjects.SetItem(cos.GetPDFName("Fm0"), form)
	resources := pdmodel.NewPDResources()
	resources.Dictionary().SetItem(cos.XObject, xobjects)
	page.SetResources(resources)

	backend := newRecordingBackend()
	if err := NewPDFRenderer(document).RenderPageToBackend(0, backend, 1, 1, Export); err != nil {
		t.Fatal(err)
	}
	wantDrawn(t, backend, "fill [30.00 40.00 10.00 10.00] paint=color(0.000 0.000 0.000 1.000)")
}

// TestTransparencyGroupIsPushedAndPopped pins that a form whose /Group is
// /Transparency becomes a compositing layer of its own, over the box the form's
// bbox and the clip intersect to.
func TestTransparencyGroupIsPushedAndPopped(t *testing.T) {
	document, page := pageWithContent(t, 100, 100, "/Fm0 Do")

	form := document.CreateStream()
	form.SetItem(cos.Type, cos.XObject)
	form.SetItem(cos.Subtype, cos.Form)
	bbox := cos.NewArray()
	for _, v := range []float32{10, 20, 40, 60} {
		bbox.Add(cos.NewFloat(v))
	}
	form.SetItem(cos.BBox, bbox)
	group := cos.NewDictionary()
	group.SetItem(cos.S, cos.Transparency)
	form.SetItem(cos.Group, group)
	out, err := form.CreateWriter()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := out.Write([]byte("20 30 5 5 re f")); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}

	xobjects := cos.NewDictionary()
	xobjects.SetItem(cos.GetPDFName("Fm0"), form)
	resources := pdmodel.NewPDResources()
	resources.Dictionary().SetItem(cos.XObject, xobjects)
	page.SetResources(resources)

	backend := newRecordingBackend()
	if err := NewPDFRenderer(document).RenderPageToBackend(0, backend, 1, 1, Export); err != nil {
		t.Fatal(err)
	}
	wantDrawn(t, backend,
		"pushGroup [10.00 20.00 30.00 40.00] softMask=false backdrop=false",
		"fill [20.00 30.00 5.00 5.00] paint=color(0.000 0.000 0.000 1.000)",
		"popGroup")
}
