package multipdf_test

// Port of org.apache.pdfbox.multipdf.TestLayerUtility.
//
// Its one case builds both documents itself and asserts about the object graph
// -- the document version, the optional content group, the properties resource
// that names it -- so all of it ports. The task file expected a rendering half
// to hold back for `track/raster`; there is none.

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter/compress"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/optionalcontent"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// TestLayerImport tests layer import.
func TestLayerImport(t *testing.T) {
	dir := t.TempDir()
	mainPDF := createMainPDF(t, dir)
	overlay1 := createOverlay1(t, dir)
	targetFile := filepath.Join(dir, "text-with-form-overlay.pdf")

	targetDoc := loadFile(t, mainPDF)
	overlay1Doc := loadFile(t, overlay1)
	if got := targetDoc.Version(); got != 1.4 {
		t.Errorf("the target document is version %v, want 1.4", got)
	}
	layerUtil := multipdf.NewLayerUtility(targetDoc)
	formXObject, err := layerUtil.ImportPageAsForm(overlay1Doc, 0)
	if err != nil {
		t.Fatalf("importPageAsForm: %v", err)
	}
	targetPage := targetDoc.Page(0)
	if err := layerUtil.WrapInSaveRestore(targetPage); err != nil {
		t.Fatalf("wrapInSaveRestore: %v", err)
	}
	at := geom.NewIdentityTransform()
	if _, err := layerUtil.AppendFormAsLayer(targetPage, formXObject, at, "overlay"); err != nil {
		t.Fatalf("appendFormAsLayer: %v", err)
	}

	// OCGs require PDF 1.5 or later, and appendFormAsLayer raises the version.
	if got := targetDoc.Version(); got != 1.5 {
		t.Errorf("the target document is version %v, want 1.5", got)
	}
	// save with no compression to avoid version going up to 1.6
	if err := targetDoc.SaveToFileOfParameters(targetFile, compress.NoCompression); err != nil {
		t.Fatal(err)
	}
	if got := targetDoc.Version(); got != 1.5 {
		t.Errorf("after saving, the target document is version %v, want 1.5", got)
	}
	targetDoc.Close()
	overlay1Doc.Close()

	doc := loadFile(t, targetFile)
	defer doc.Close()
	catalog := doc.DocumentCatalog()

	// OCGs require PDF 1.5 or later
	if got := doc.Version(); got != 1.5 {
		t.Errorf("the reloaded document is version %v, want 1.5", got)
	}

	page := doc.Page(0)
	properties := page.Resources().GetProperties(cos.GetPDFName("oc1"))
	if properties == nil {
		t.Fatal("the page has no /oc1 properties resource")
	}
	ocg, isGroup := properties.(*optionalcontent.PDOptionalContentGroup)
	if !isGroup {
		t.Fatalf("the /oc1 properties resource is %T, want an optional content group",
			properties)
	}
	if got := ocg.Name(); got != "overlay" {
		t.Errorf("the group is named %q, want %q", got, "overlay")
	}

	ocgs := catalog.OCProperties()
	if ocgs == nil {
		t.Fatal("the catalog has no /OCProperties")
	}
	overlay := ocgs.Group("overlay")
	if overlay == nil {
		t.Fatal("the /OCProperties holds no group named overlay")
	}
	if overlay.Name() != ocg.Name() {
		t.Errorf("the group in /OCProperties is named %q and the one on the page %q",
			overlay.Name(), ocg.Name())
	}

	// test PDFBOX-5232 (never ended)
	if _, err := multipdf.NewLayerUtility(doc).ImportPageAsForm(doc, 0); err != nil {
		t.Fatalf("importing a page of the target document into itself: %v", err)
	}
}

// createMainPDF is "Simple test document with text."
func createMainPDF(t *testing.T, dir string) string {
	t.Helper()
	targetFile := filepath.Join(dir, "text-doc.pdf")
	doc := pdmodel.NewPDDocument()
	defer doc.Close()

	// Create new page
	page := pdmodel.NewPDPage()
	doc.AddPage(page)
	if page.Resources() == nil {
		page.SetResources(pdmodel.NewPDResources())
	}

	text := []string{
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Integer fermentum lacus in eros",
		"condimentum eget tristique risus viverra. Sed ac sem et lectus ultrices placerat. Nam",
		"fringilla tincidunt nulla id euismod. Vivamus eget mauris dui. Mauris luctus ullamcorper",
		"leo, et laoreet diam suscipit et. Nulla viverra commodo sagittis. Integer vitae rhoncus velit.",
		"Mauris porttitor ipsum in est sagittis non luctus purus molestie. Sed placerat aliquet",
		"vulputate.",
	}

	contentStream, err := pdmodel.NewPDPageContentStreamCompressed(doc, page,
		pdmodel.Overwrite, false)
	if err != nil {
		t.Fatalf("NewPDPageContentStream: %v", err)
	}
	// Setup page content stream and paint background/title
	bold := standard14(t, font.HelveticaBold)
	must(t, contentStream.BeginText())
	must(t, contentStream.NewLineAtOffset(50, 720))
	must(t, contentStream.SetFont(bold, 14))
	must(t, contentStream.ShowText("Simple test document with text."))
	must(t, contentStream.EndText())

	plain := standard14(t, font.Helvetica)
	must(t, contentStream.BeginText())
	const fontSize = 12
	must(t, contentStream.SetFont(plain, fontSize))
	must(t, contentStream.NewLineAtOffset(50, 700))
	for _, line := range text {
		must(t, contentStream.NewLineAtOffset(0, -fontSize*1.2))
		must(t, contentStream.ShowText(line))
	}
	must(t, contentStream.EndText())
	must(t, contentStream.Close())

	// save with no compression to avoid version going up to 1.6
	if err := doc.SaveToFileOfParameters(targetFile, compress.NoCompression); err != nil {
		t.Fatal(err)
	}
	return targetFile
}

// createOverlay1 is the word OVERLAY across the page at 45 degrees.
func createOverlay1(t *testing.T, dir string) string {
	t.Helper()
	targetFile := filepath.Join(dir, "overlay1.pdf")
	doc := pdmodel.NewPDDocument()
	defer doc.Close()

	// Create new page
	page := pdmodel.NewPDPage()
	doc.AddPage(page)
	if page.Resources() == nil {
		page.SetResources(pdmodel.NewPDResources())
	}

	contentStream, err := pdmodel.NewPDPageContentStreamCompressed(doc, page,
		pdmodel.Overwrite, false)
	if err != nil {
		t.Fatalf("NewPDPageContentStream: %v", err)
	}
	// Setup page content stream and paint background/title
	bold := standard14(t, font.HelveticaBold)
	// Color.LIGHT_GRAY is 192, 192, 192.
	must(t, contentStream.SetNonStrokingColorRGB255(192, 192, 192))
	must(t, contentStream.BeginText())
	const fontSize = 96
	must(t, contentStream.SetFont(bold, fontSize))
	const text = "OVERLAY"
	// float sw = font.getStringWidth(text);
	// Too bad, base 14 fonts don't return character metrics.
	crop := page.CropBox()
	cx := crop.Width() / 2
	cy := crop.Height() / 2
	transform := util.NewMatrix()
	transform.Translate(cx, cy)
	transform.Rotate(math.Pi / 4)
	transform.Translate(-190 /* sw/2 */, 0)
	must(t, contentStream.SetTextMatrix(transform))
	must(t, contentStream.ShowText(text))
	must(t, contentStream.EndText())
	must(t, contentStream.Close())

	if err := doc.SaveToFile(targetFile); err != nil {
		t.Fatal(err)
	}
	return targetFile
}

// standard14 answers one of the base fourteen fonts.
func standard14(t *testing.T, name font.FontName) *font.PDType1Font {
	t.Helper()
	f, err := font.NewPDType1FontStandard14(name)
	if err != nil {
		t.Fatalf("NewPDType1FontStandard14: %v", err)
	}
	return f
}

// must fails the test where a content stream operator answered an error.
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
