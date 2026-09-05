package optionalcontent_test

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/graphics/optionalcontent/TestOptionalContentGroups.java.
//
// The package is optionalcontent_test rather than optionalcontent: the tests
// build a document, and pdmodel sits above this package.
//
// testOCGsWithSameNameCanHaveDifferentVisibility and
// testOCGGenerationSameNameCanHaveSameVisibilityOff are not here: both render
// the page with PDFRenderer and read pixels out of the image, which waits for
// slice 9. See migration/STATUS.md.
//
// Java writes the generated file into target/test-output and the second test
// reads it back from there, generating it first if it is missing; the port
// writes into the test's own temporary directory and the second test generates
// its own copy, which is the same file.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt"
	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/markedcontent"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/optionalcontent"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// TestOCGGeneration is TestOptionalContentGroups.testOCGGeneration.
func TestOCGGeneration(t *testing.T) {
	ocgGeneration(t, filepath.Join(t.TempDir(), "ocg-generation.pdf"))
}

// ocgGeneration is testOCGGeneration, taking the file to write, so that
// testOCGConsumption can call it the way Java does.
func ocgGeneration(t *testing.T, targetFile string) {
	t.Helper()
	doc := pdmodel.NewPDDocument()
	defer doc.Close()

	// Create new page
	page := pdmodel.NewPDPage()
	doc.AddPage(page)
	resources := page.Resources()
	if resources == nil {
		resources = pdmodel.NewPDResources()
		page.SetResources(resources)
	}

	// Prepare OCG functionality
	ocprops := optionalcontent.NewPDOptionalContentProperties()
	doc.DocumentCatalog().SetOCProperties(ocprops)
	// ocprops.SetBaseState(optionalcontent.BaseStateON) // ON=default

	// Create OCG for background
	background := optionalcontent.NewPDOptionalContentGroup("background")
	ocprops.AddGroup(background)
	if !ocprops.IsGroupEnabledNamed("background") {
		t.Error(`IsGroupEnabledNamed("background") = false, want true`)
	}

	// Create OCG for enabled
	enabled := optionalcontent.NewPDOptionalContentGroup("enabled")
	ocprops.AddGroup(enabled)
	if ocprops.SetGroupEnabledNamed("enabled", true) {
		t.Error(`SetGroupEnabledNamed("enabled", true) = true, want false`)
	}
	if !ocprops.IsGroupEnabledNamed("enabled") {
		t.Error(`IsGroupEnabledNamed("enabled") = false, want true`)
	}

	// Create OCG for disabled
	disabled := optionalcontent.NewPDOptionalContentGroup("disabled")
	ocprops.AddGroup(disabled)
	if ocprops.SetGroupEnabledNamed("disabled", true) {
		t.Error(`SetGroupEnabledNamed("disabled", true) = true, want false`)
	}
	if !ocprops.IsGroupEnabledNamed("disabled") {
		t.Error(`IsGroupEnabledNamed("disabled") = false, want true`)
	}
	if !ocprops.SetGroupEnabledNamed("disabled", false) {
		t.Error(`SetGroupEnabledNamed("disabled", false) = false, want true`)
	}
	if ocprops.IsGroupEnabledNamed("disabled") {
		t.Error(`IsGroupEnabledNamed("disabled") = true, want false`)
	}

	// Setup page content stream and paint background/title
	contentStream, err := pdmodel.NewPDPageContentStreamCompressed(doc, page, pdmodel.Overwrite, false)
	if err != nil {
		t.Fatal(err)
	}
	helveticaBold, err := font.NewPDType1FontStandard14(font.HelveticaBold)
	if err != nil {
		t.Fatal(err)
	}
	helvetica, err := font.NewPDType1FontStandard14(font.Helvetica)
	if err != nil {
		t.Fatal(err)
	}
	steps := []func() error{
		func() error { return contentStream.BeginMarkedContentWithProperties(cos.OC, background) },
		contentStream.BeginText,
		func() error { return contentStream.SetFont(helveticaBold, 14) },
		func() error { return contentStream.NewLineAtOffset(80, 700) },
		func() error { return contentStream.ShowText("PDF 1.5: Optional Content Groups") },
		contentStream.EndText,
		contentStream.BeginText,
		func() error { return contentStream.SetFont(helvetica, 12) },
		func() error { return contentStream.NewLineAtOffset(80, 680) },
		func() error {
			return contentStream.ShowText("You should see a green textline, but no red text line.")
		},
		contentStream.EndText,
		contentStream.EndMarkedContent,

		// Paint enabled layer
		func() error { return contentStream.BeginMarkedContentWithProperties(cos.OC, enabled) },
		func() error { return setNonStrokingColor(contentStream, awt.Green) },
		contentStream.BeginText,
		func() error { return contentStream.SetFont(helvetica, 12) },
		func() error { return contentStream.NewLineAtOffset(80, 600) },
		func() error {
			return contentStream.ShowText(
				"This is from an enabled layer. If you see this, that's good.")
		},
		contentStream.EndText,
		contentStream.EndMarkedContent,

		// Paint disabled layer
		func() error { return contentStream.BeginMarkedContentWithProperties(cos.OC, disabled) },
		func() error { return setNonStrokingColor(contentStream, awt.Red) },
		contentStream.BeginText,
		func() error { return contentStream.SetFont(helvetica, 12) },
		func() error { return contentStream.NewLineAtOffset(80, 500) },
		func() error {
			return contentStream.ShowText(
				"This is from a disabled layer. If you see this, that's NOT good!")
		},
		contentStream.EndText,
		contentStream.EndMarkedContent,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	if err := contentStream.Close(); err != nil {
		t.Fatal(err)
	}

	if err := doc.SaveToFile(targetFile); err != nil {
		t.Fatal(err)
	}
}

// setNonStrokingColor is contentStream.setNonStrokingColor(Color), which Java
// declares over java.awt.Color; the port's abstract content stream takes a
// PDColor, and SetNonStrokingColorRGB255 is the overload Java's Color one
// forwards to.
func setNonStrokingColor(contentStream *pdmodel.PDPageContentStream, c awt.Color) error {
	return contentStream.SetNonStrokingColorRGB255(c.Red(), c.Green(), c.Blue())
}

// TestOCGConsumption is TestOptionalContentGroups.testOCGConsumption.
func TestOCGConsumption(t *testing.T) {
	pdfFile := filepath.Join(t.TempDir(), "ocg-generation.pdf")
	if _, err := os.Stat(pdfFile); err != nil {
		ocgGeneration(t, pdfFile)
	}

	doc, err := pdfbox.LoadPDF(pdfFile)
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	if got := doc.Version(); got != 1.6 {
		t.Errorf("Version() = %v, want 1.6", got)
	}
	catalog := doc.DocumentCatalog()

	page := doc.Page(0)
	resources := page.Resources()

	mc0 := cos.GetPDFName("oc1")
	properties := resources.GetProperties(mc0)
	if properties == nil {
		t.Fatal("GetProperties(oc1) = nil, want the background group")
	}
	ocg, isGroup := properties.(*optionalcontent.PDOptionalContentGroup)
	if !isGroup {
		t.Fatalf("GetProperties(oc1) = %T, want *PDOptionalContentGroup", properties)
	}
	if got, want := ocg.Name(), "background"; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}

	if got := resources.GetProperties(cos.GetPDFName("inexistent")); got != nil {
		t.Errorf("GetProperties(inexistent) = %v, want nil", got)
	}

	ocgs := catalog.OCProperties()
	if got, want := ocgs.BaseState(), optionalcontent.BaseStateON; got != want {
		t.Errorf("BaseState() = %v, want %v", got, want)
	}
	names := map[string]bool{}
	for _, name := range ocgs.GroupNames() {
		names[name] = true
	}
	if got := len(names); got != 3 {
		t.Errorf("GroupNames() distinct = %d, want 3", got)
	}
	if !names["background"] {
		t.Error(`GroupNames() has no "background"`)
	}

	if !ocgs.IsGroupEnabledNamed("background") {
		t.Error(`IsGroupEnabledNamed("background") = false, want true`)
	}
	if !ocgs.IsGroupEnabledNamed("enabled") {
		t.Error(`IsGroupEnabledNamed("enabled") = false, want true`)
	}
	if ocgs.IsGroupEnabledNamed("disabled") {
		t.Error(`IsGroupEnabledNamed("disabled") = true, want false`)
	}

	ocgs.SetGroupEnabledNamed("background", false)
	if ocgs.IsGroupEnabledNamed("background") {
		t.Error(`IsGroupEnabledNamed("background") = true, want false`)
	}

	background := ocgs.Group("background")
	if background == nil {
		t.Fatal(`Group("background") = nil, want the background group`)
	}
	if got, want := background.Name(), ocg.Name(); got != want {
		t.Errorf("Group(background).Name() = %q, want %q", got, want)
	}
	if got := ocgs.Group("inexistent"); got != nil {
		t.Errorf(`Group("inexistent") = %v, want nil`, got)
	}

	coll := ocgs.OptionalContentGroups()
	if got := len(coll); got != 3 {
		t.Fatalf("OptionalContentGroups() = %d, want 3", got)
	}
	nameSet := map[string]bool{}
	for _, group := range coll {
		nameSet[group.Name()] = true
	}
	for _, want := range []string{"background", "enabled", "disabled"} {
		if !nameSet[want] {
			t.Errorf("OptionalContentGroups() has no %q", want)
		}
	}

	extractor := text.NewPDFMarkedContentExtractor()
	if err := extractor.ProcessPage(page); err != nil {
		t.Fatal(err)
	}
	markedContents := extractor.MarkedContents()
	if got := len(markedContents); got < 3 {
		t.Fatalf("MarkedContents() = %d, want at least 3", got)
	}
	for i, want := range []struct {
		name string
		text string
	}{
		{"background", "PDF 1.5: Optional Content Groups" +
			"You should see a green textline, but no red text line."},
		{"enabled", "This is from an enabled layer. If you see this, that's good."},
		{"disabled", "This is from a disabled layer. If you see this, that's NOT good!"},
	} {
		tag, _ := markedContents[i].Tag()
		if tag != "OC" {
			t.Errorf("MarkedContents()[%d].Tag() = %q, want %q", i, tag, "OC")
		}
		created := markedcontent.CreatePropertyList(markedContents[i].Properties())
		group, isGroup := created.(*optionalcontent.PDOptionalContentGroup)
		if !isGroup {
			t.Fatalf("CreatePropertyList(%d) = %T, want *PDOptionalContentGroup", i, created)
		}
		if got := group.Name(); got != want.name {
			t.Errorf("MarkedContents()[%d] group = %q, want %q", i, got, want.name)
		}
		if got := textPositionListToString(markedContents[i].Contents()); got != want.text {
			t.Errorf("MarkedContents()[%d] text = %q, want %q", i, got, want.text)
		}
	}
}

// textPositionListToString converts a list of TextPosition objects to a string.
func textPositionListToString(contents []any) string {
	sb := ""
	for _, o := range contents {
		tp := o.(*text.TextPosition)
		sb += tp.Unicode()
	}
	return sb
}
