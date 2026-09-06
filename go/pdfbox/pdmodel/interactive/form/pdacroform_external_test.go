package form_test

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/interactive/form/PDAcroFormTest.java.
//
// testIllegalFieldsDefinition and testPDFBox3347 download a PDF from the issue
// tracker, and testPDFBox5797 embeds a TrueType font with PDType0Font.load,
// which needs the font embedders a later slice brings. None of the three is
// here. See migration/STATUS.md.

import (
	"bytes"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// newAcroFormFixture is PDAcroFormTest.setUp.
func newAcroFormFixture(t *testing.T) (*pdmodel.PDDocument, *form.PDAcroForm) {
	t.Helper()
	document := pdmodel.NewPDDocument()
	t.Cleanup(func() { document.Close() })
	acroForm := form.NewPDAcroForm(document)
	form.SetAcroFormOfCatalog(document.DocumentCatalog(), acroForm)
	return document, acroForm
}

// TestFieldsEntry is PDAcroFormTest.testFieldsEntry.
func TestFieldsEntry(t *testing.T) {
	_, acroForm := newAcroFormFixture(t)

	// the /Fields entry has been created with the AcroForm as this is a
	// required entry
	if acroForm.Fields() == nil {
		t.Error("Fields() = nil, want an empty list")
	}
	if got := len(acroForm.Fields()); got != 0 {
		t.Errorf("Fields() size = %d, want 0", got)
	}
	// there shouldn't be an exception if there is no such field
	if got := acroForm.Field("foo"); got != nil {
		t.Errorf("Field(\"foo\") = %v, want nil", got)
	}

	// remove the required entry which is the case for some PDFs (see
	// PDFBOX-2965)
	acroForm.Dictionary().RemoveItem(cos.Fields)

	// ensure there is always an empty collection returned
	if acroForm.Fields() == nil {
		t.Error("Fields() = nil, want an empty list")
	}
	if got := len(acroForm.Fields()); got != 0 {
		t.Errorf("Fields() size = %d, want 0", got)
	}
	// there shouldn't be an exception if there is no such field
	if got := acroForm.Field("foo"); got != nil {
		t.Errorf("Field(\"foo\") = %v, want nil", got)
	}
}

// TestAcroFormProperties is PDAcroFormTest.testAcroFormProperties.
func TestAcroFormProperties(t *testing.T) {
	_, acroForm := newAcroFormFixture(t)
	if got := acroForm.DefaultAppearance(); got != "" {
		t.Errorf("DefaultAppearance() = %q, want empty", got)
	}
	acroForm.SetDefaultAppearance("/Helv 0 Tf 0 g")
	if got, want := acroForm.DefaultAppearance(), "/Helv 0 Tf 0 g"; got != want {
		t.Errorf("DefaultAppearance() = %q, want %q", got, want)
	}
}

// TestFlatten is PDAcroFormTest.testFlatten; the rendering comparison it ends
// with is left out.
func TestFlatten(t *testing.T) {
	testPdf, err := pdfbox.LoadPDF(formFixture + "AlignmentTests.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer testPdf.Close()
	acroForm := form.AcroFormOfCatalog(testPdf.DocumentCatalog())
	if err := acroForm.Flatten(); err != nil {
		t.Fatal(err)
	}
	if got := form.AcroFormOfCatalog(testPdf.DocumentCatalog()).Fields(); len(got) != 0 {
		t.Errorf("Fields() = %v, want empty", got)
	}
	saveTo(t, testPdf, "AlignmentTests-flattened.pdf")
}

// TestFlattenWidgetNoRef is PDAcroFormTest.testFlattenWidgetNoRef.
func TestFlattenWidgetNoRef(t *testing.T) {
	testPdf, err := pdfbox.LoadPDF(formFixture + "AlignmentTests.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer testPdf.Close()
	acroFormToTest := form.AcroFormOfCatalog(testPdf.DocumentCatalog())
	for field := range acroFormToTest.FieldTree().All() {
		for _, widget := range field.Widgets() {
			widget.AnnotationDictionary().RemoveItem(cos.P)
		}
	}
	if err := acroFormToTest.Flatten(); err != nil {
		t.Fatal(err)
	}
	// 36 non widget annotations shall not be flattened
	if got := testPdf.Page(0).Annotations().Size(); got != 36 {
		t.Errorf("annotations = %d, want 36", got)
	}
	if got := acroFormToTest.Fields(); len(got) != 0 {
		t.Errorf("Fields() = %v, want empty", got)
	}
	saveTo(t, testPdf, "AlignmentTests-flattened-noRef.pdf")
}

// TestFlattenSpecificFieldsOnly is PDAcroFormTest.testFlattenSpecificFieldsOnly.
func TestFlattenSpecificFieldsOnly(t *testing.T) {
	testPdf, err := pdfbox.LoadPDF(formFixture + "AlignmentTests.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer testPdf.Close()
	acroFormToFlatten := form.AcroFormOfCatalog(testPdf.DocumentCatalog())
	numFieldsBeforeFlatten := len(acroFormToFlatten.Fields())
	numWidgetsBeforeFlatten := countWidgets(testPdf)

	fieldsToFlatten := []form.PDField{}
	for _, name := range []string{
		"AlignLeft-Border_Small-Filled",
		"AlignLeft-Border_Medium-Filled",
		"AlignLeft-Border_Wide-Filled",
		"AlignLeft-Border_Wide_Clipped-Filled",
	} {
		fieldsToFlatten = append(fieldsToFlatten, acroFormToFlatten.Field(name))
	}
	if err := acroFormToFlatten.FlattenFields(fieldsToFlatten, true); err != nil {
		t.Fatal(err)
	}

	numFieldsAfterFlatten := len(acroFormToFlatten.Fields())
	numWidgetsAfterFlatten := countWidgets(testPdf)
	if got, want := numFieldsAfterFlatten+len(fieldsToFlatten), numFieldsBeforeFlatten; got != want {
		t.Errorf("fields after flatten = %d, want %d", got, want)
	}
	if got, want := numWidgetsAfterFlatten+len(fieldsToFlatten), numWidgetsBeforeFlatten; got != want {
		t.Errorf("widgets after flatten = %d, want %d", got, want)
	}
	saveTo(t, testPdf, "AlignmentTests-flattened-specificFields.pdf")
}

// countWidgets is the private countWidgets of the Java test.
func countWidgets(documentToTest *pdmodel.PDDocument) int {
	count := 0
	for page := range documentToTest.Pages().All {
		for _, annot := range page.Annotations().ToSlice() {
			if _, isWidget := annot.(*annotation.PDAnnotationWidget); isWidget {
				count++
			}
		}
	}
	return count
}

// createAcroFormWithMissingResourceInformation is the private helper of the Java
// test, which builds a working PDF whose form has neither /DA nor /DR.
func createAcroFormWithMissingResourceInformation(t *testing.T) []byte {
	t.Helper()
	tmpDocument := pdmodel.NewPDDocument()
	defer tmpDocument.Close()
	page := pdmodel.NewPDPage()
	tmpDocument.AddPage(page)
	newAcroForm := form.NewPDAcroForm(tmpDocument)
	form.SetAcroFormOfCatalog(tmpDocument.DocumentCatalog(), newAcroForm)
	textBox := form.NewPDTextField(newAcroForm)
	textBox.SetPartialName("SampleField")
	newAcroForm.SetFields(append(newAcroForm.Fields(), textBox))
	widget := textBox.Widgets()[0]
	widget.SetRectangle(common.NewPDRectangleOf(50, 750, 200, 20))
	widget.SetPage(page)
	page.Annotations().Add(widget)
	baos := &bytes.Buffer{}
	if err := tmpDocument.Save(baos); err != nil { // this is a working PDF
		t.Fatal(err)
	}
	return baos.Bytes()
}

// TestDontAddMissingInformationOnDocumentLoad is
// PDAcroFormTest.testDontAddMissingInformationOnDocumentLoad.
func TestDontAddMissingInformationOnDocumentLoad(t *testing.T) {
	pdfBytes := createAcroFormWithMissingResourceInformation(t)
	pdfDocument, err := pdfbox.LoadPDFBytes(pdfBytes)
	if err != nil {
		t.Fatal(err)
	}
	defer pdfDocument.Close()

	// do a low level access to the AcroForm to avoid the generation of missing
	// entries
	catalogDictionary := pdfDocument.DocumentCatalog().Dictionary()
	acroFormDictionary := catalogDictionary.GetDictionaryObject(cos.AcroForm).(*cos.Dictionary)

	// ensure that the missing information has not been generated
	if got := acroFormDictionary.GetDictionaryObject(cos.DA); got != nil {
		t.Errorf("/DA = %v, want nil", got)
	}
	if got := acroFormDictionary.GetDictionaryObject(cos.Resources); got != nil {
		t.Errorf("/Resources = %v, want nil", got)
	}
}

// TestAddMissingInformationOnAcroFormAccess is
// PDAcroFormTest.testAddMissingInformationOnAcroFormAccess.
func TestAddMissingInformationOnAcroFormAccess(t *testing.T) {
	pdfBytes := createAcroFormWithMissingResourceInformation(t)
	pdfDocument, err := pdfbox.LoadPDFBytes(pdfBytes)
	if err != nil {
		t.Fatal(err)
	}
	defer pdfDocument.Close()

	// this call shall trigger the generation of missing information
	theAcroForm := form.AcroFormOfCatalog(pdfDocument.DocumentCatalog())

	// ensure that the missing information has been generated
	// DA entry
	if got, want := theAcroForm.DefaultAppearance(), "/Helv 0 Tf 0 g "; got != want {
		t.Errorf("DefaultAppearance() = %q, want %q", got, want)
	}
	acroFormResources := theAcroForm.DefaultResources()
	if acroFormResources == nil {
		t.Fatal("DefaultResources() = nil, want resources")
	}

	// DR entry
	for _, c := range []struct{ name, want string }{
		{"Helv", "Helvetica"},
		{"ZaDb", "ZapfDingbats"},
	} {
		resourceFont, err := acroFormResources.GetFont(cos.GetPDFName(c.name))
		if err != nil {
			t.Fatal(err)
		}
		if resourceFont == nil {
			t.Fatalf("GetFont(%q) = nil, want a font", c.name)
		}
		if got := resourceFont.Name(); got != c.want {
			t.Errorf("GetFont(%q).Name() = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestBadDA is PDAcroFormTest.testBadDA. Java asserts IllegalArgumentException,
// which is unchecked, so the port panics.
func TestBadDA(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	page := pdmodel.NewPDPage()
	doc.AddPage(page)
	theAcroForm := form.NewPDAcroForm(doc)
	form.SetAcroFormOfCatalog(doc.DocumentCatalog(), theAcroForm)
	theAcroForm.SetDefaultResources(pdmodel.NewPDResources())
	textBox := form.NewPDTextField(theAcroForm)
	textBox.SetPartialName("SampleField")

	// https://stackoverflow.com/questions/50609478/
	// "tf" is a typo, should have been "Tf" and this results that no font is
	// chosen
	textBox.SetDefaultAppearance("/Helv 0 tf 0 g")
	theAcroForm.SetFields(append(theAcroForm.Fields(), textBox))
	widget := textBox.Widgets()[0]
	widget.SetRectangle(common.NewPDRectangleOf(50, 750, 200, 20))
	widget.SetPage(page)
	page.Annotations().Add(widget)

	defer func() {
		if recover() == nil {
			t.Error("IllegalArgumentException should have been thrown")
		}
	}()
	textBox.SetValue("huhu") //nolint:errcheck // it panics
}

// TestAcroFormDefaultFonts is PDAcroFormTest.testAcroFormDefaultFonts.
func TestAcroFormDefaultFonts(t *testing.T) {
	baos := &bytes.Buffer{}
	func() {
		doc := pdmodel.NewPDDocument()
		defer doc.Close()
		page := pdmodel.NewPDPageOfSize(common.A4)
		doc.AddPage(page)
		acroForm2 := form.NewPDAcroForm(doc)
		form.SetAcroFormOfCatalog(doc.DocumentCatalog(), acroForm2)
		if got := acroForm2.DefaultResources(); got != nil {
			t.Errorf("DefaultResources() = %v, want nil", got)
		}
		defaultResources := pdmodel.NewPDResources()
		acroForm2.SetDefaultResources(defaultResources)
		assertNoFont(t, defaultResources, cos.Helv)
		assertNoFont(t, defaultResources, cos.ZaDb)

		// getting AcroForm sets the two fonts
		acroForm2 = form.AcroFormOfCatalog(doc.DocumentCatalog())
		defaultResources = acroForm2.DefaultResources()
		assertFont(t, defaultResources, cos.Helv)
		assertFont(t, defaultResources, cos.ZaDb)

		// repeat with a new AcroForm (to delete AcroForm cache) and thus missing
		// /DR
		form.SetAcroFormOfCatalog(doc.DocumentCatalog(), form.NewPDAcroForm(doc))
		acroForm2 = form.AcroFormOfCatalog(doc.DocumentCatalog())
		defaultResources = acroForm2.DefaultResources()
		assertFont(t, defaultResources, cos.Helv)
		assertFont(t, defaultResources, cos.ZaDb)

		if err := doc.Save(baos); err != nil {
			t.Fatal(err)
		}
	}()

	doc, err := pdfbox.LoadPDFBytes(baos.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	acroForm2 := form.AcroFormOfCatalog(doc.DocumentCatalog())
	defaultResources := acroForm2.DefaultResources()
	helv := assertFont(t, defaultResources, cos.Helv)
	zadb := assertFont(t, defaultResources, cos.ZaDb)

	// make sure that font wasn't overwritten
	helvType1, isType1 := helv.(*font.PDType1Font)
	if !isType1 {
		t.Fatalf("Helv = %T, want *font.PDType1Font", helv)
	}
	zadbType1, isType1 := zadb.(*font.PDType1Font)
	if !isType1 {
		t.Fatalf("ZaDb = %T, want *font.PDType1Font", zadb)
	}
	if got, want := helv.Name(), string(font.Helvetica); got != want {
		t.Errorf("Helv.Name() = %q, want %q", got, want)
	}
	if got, want := zadb.Name(), string(font.ZapfDingbatsFontName); got != want {
		t.Errorf("ZaDb.Name() = %q, want %q", got, want)
	}
	if got := helvType1.Type1Font(); got != nil {
		t.Errorf("Helv.Type1Font() = %v, want nil", got)
	}
	if got := zadbType1.Type1Font(); got != nil {
		t.Errorf("ZaDb.Type1Font() = %v, want nil", got)
	}
}

// assertFont checks that the resources hold a font under the given name and
// answers it.
func assertFont(t *testing.T, resources *pdmodel.PDResources, name *cos.Name) font.PDFont {
	t.Helper()
	resourceFont, err := resources.GetFont(name)
	if err != nil {
		t.Fatal(err)
	}
	if resourceFont == nil {
		t.Fatalf("GetFont(%v) = nil, want a font", name)
	}
	return resourceFont
}

// assertNoFont checks that the resources hold no font under the given name.
func assertNoFont(t *testing.T, resources *pdmodel.PDResources, name *cos.Name) {
	t.Helper()
	resourceFont, err := resources.GetFont(name)
	if err != nil {
		t.Fatal(err)
	}
	if resourceFont != nil {
		t.Errorf("GetFont(%v) = %v, want nil", name, resourceFont)
	}
}

// TestCycle is PDAcroFormTest.testCycle.
func TestCycle(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	page := pdmodel.NewPDPage()
	doc.AddPage(page)
	theAcroForm := form.NewPDAcroForm(doc)
	theAcroForm.SetNeedAppearances(true)
	form.SetAcroFormOfCatalog(doc.DocumentCatalog(), theAcroForm)
	theAcroForm.SetDefaultResources(pdmodel.NewPDResources())
	textBox := form.NewPDTextField(theAcroForm)
	textBox.SetPartialName("SampleField")
	textBox.SetDefaultAppearance("/Helv 0 Ff 0 g")
	// field not added to force repair
	widget := textBox.Widgets()[0]
	rect := common.NewPDRectangleOf(50, 750, 200, 20)
	func() {
		defer func() {
			if recover() == nil {
				t.Error("SetWidgetParent() did not panic")
			}
		}()
		form.SetWidgetParent(widget, textBox)
	}()
	widget.AnnotationDictionary().SetItem(cos.Parent, widget.AnnotationDictionary())
	widget.SetRectangle(rect)
	widget.SetPage(page)
	page.Annotations().Add(widget)
	if got := form.AcroFormOfCatalog(doc.DocumentCatalog()).Fields(); len(got) != 0 {
		t.Errorf("Fields() = %v, want empty", got)
	}
}
