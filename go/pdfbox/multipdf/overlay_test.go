package multipdf_test

// Port of org.apache.pdfbox.multipdf.OverlayTest.
//
// All three of its cases end in `checkIdenticalRendering`, which renders both
// documents with `PDFRenderer` and compares the pixels. Nothing implements
// `rendering.Backend`, so that comparison belongs to `track/raster` -- but the
// thing it is checking does not need a renderer to state. Two pages render
// identically when they draw the same marks from the same resources, and the
// overlay's whole job is to write those marks: a content stream that pushes the
// state, concatenates a matrix, does an XObject and pops.
//
// So `checkIdenticalContent` here decodes every page's content streams and
// compares them byte for byte with the model file's, and compares the form
// XObject each one names -- its /BBox, its /Matrix and its own decoded content.
// The model files are the ones PDFBox itself produced and checked in, so the
// values are the Java's in the strongest sense available: they are its output.
//
// What that cannot see is anything the renderer would do with identical marks,
// which is nothing. What it sees that the rendering does not is a difference
// too small to change a pixel at the test's resolution.

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

// TestRotatedOverlays is the Java's four calls to testRotatedOverlay.
func TestRotatedOverlays(t *testing.T) {
	for _, rotation := range []int{0, 90, 180, 270} {
		t.Run(fmt.Sprintf("rot%d", rotation), func(t *testing.T) {
			testRotatedOverlay(t, rotation)
		})
	}
}

func testRotatedOverlay(t *testing.T, rotation int) {
	t.Helper()
	resultFile := filepath.Join(t.TempDir(),
		fmt.Sprintf("Overlayed-with-rot%d.pdf", rotation))

	// do the overlaying
	baseDocument := loadFile(t, filepath.Join(multipdfDir, "OverlayTestBaseRot0.pdf"))
	overlay := multipdf.NewOverlay()
	overlay.SetInputPDF(baseDocument)
	overlayDocument := loadFile(t, filepath.Join(multipdfDir,
		fmt.Sprintf("rot%d.pdf", rotation)))
	overlay.SetDefaultOverlayPDF(overlayDocument)
	overlayedResultPDF, err := overlay.Overlay(map[int]string{})
	if err != nil {
		t.Fatalf("overlay: %v", err)
	}
	if err := overlayedResultPDF.SaveToFile(resultFile); err != nil {
		t.Fatal(err)
	}
	if err := overlay.Close(); err != nil {
		t.Fatal(err)
	}
	baseDocument.Close()

	// compare model and result
	modelFile := filepath.Join(multipdfDir, fmt.Sprintf("Overlayed-with-rot%d.pdf", rotation))
	checkIdenticalContent(t, modelFile, resultFile)
}

// TestRotatedOverlaysMap is the specific-page map: four copies of the base
// page, each overlaid with a differently rotated document.
func TestRotatedOverlaysMap(t *testing.T) {
	dir := t.TempDir()
	fourPages := filepath.Join(dir, "OverlayTestBaseRot0_4Pages.pdf")

	// multiply base image
	baseDocument := loadFile(t, filepath.Join(multipdfDir, "OverlayTestBaseRot0.pdf"))
	doc := pdmodel.NewPDDocument()
	for p := 0; p < 4; p++ {
		if _, err := doc.ImportPage(baseDocument.Page(0)); err != nil {
			t.Fatalf("importPage: %v", err)
		}
	}
	if err := doc.SaveToFile(fourPages); err != nil {
		t.Fatal(err)
	}
	doc.Close()
	baseDocument.Close()

	// do the overlaying
	fourPageDocument := loadFile(t, fourPages)
	overlay := multipdf.NewOverlay()

	// An empty map with no input document set is the IllegalArgumentException.
	if _, err := overlay.Overlay(map[int]string{}); err != multipdf.ErrNoInputDocument {
		t.Errorf("overlaying with no input document answered %v, want %v",
			err, multipdf.ErrNoInputDocument)
	}

	specificPageOverlayMap := map[int]string{
		1: filepath.Join(multipdfDir, "rot0.pdf"),
		2: filepath.Join(multipdfDir, "rot90.pdf"),
		3: filepath.Join(multipdfDir, "rot180.pdf"),
		4: filepath.Join(multipdfDir, "rot270.pdf"),
	}
	overlay.SetInputPDF(fourPageDocument)
	overlayedResultPDF, err := overlay.Overlay(specificPageOverlayMap)
	if err != nil {
		t.Fatalf("overlay: %v", err)
	}

	documentList, err := multipdf.NewSplitter().Split(overlayedResultPDF)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if len(documentList) != 4 {
		t.Fatalf("the overlaid document split into %d, want 4", len(documentList))
	}
	for i, rotation := range []int{0, 90, 180, 270} {
		out := filepath.Join(dir, fmt.Sprintf("Overlayed-with-rot%d.pdf", rotation))
		if err := documentList[i].SaveToFile(out); err != nil {
			t.Fatal(err)
		}
		documentList[i].Close()
		checkIdenticalContent(t,
			filepath.Join(multipdfDir, fmt.Sprintf("Overlayed-with-rot%d.pdf", rotation)), out)
	}
	if err := overlay.Close(); err != nil {
		t.Fatal(err)
	}
	fourPageDocument.Close()
}

// TestOverlayOnRotatedSourcePages is PDFBOX-6049, the -adjustRotation option:
// the overlay is turned to match the page it lands on.
func TestOverlayOnRotatedSourcePages(t *testing.T) {
	resultFile := filepath.Join(t.TempDir(), "PDFBOX-6049-Result.pdf")

	overlay := multipdf.NewOverlay()
	overlay.SetInputFile(filepath.Join(multipdfDir, "PDFBOX-6049-Source.pdf"))
	overlay.SetDefaultOverlayFile(filepath.Join(multipdfDir, "PDFBOX-6049-Overlay.pdf"))
	overlay.SetOverlayPosition(multipdf.Foreground)
	overlay.SetAdjustRotation(true)
	resultDoc, err := overlay.Overlay(map[int]string{})
	if err != nil {
		t.Fatalf("overlay: %v", err)
	}
	if err := resultDoc.SaveToFile(resultFile); err != nil {
		t.Fatal(err)
	}
	if err := overlay.Close(); err != nil {
		t.Fatal(err)
	}
	checkIdenticalContent(t,
		filepath.Join(multipdfDir, "PDFBOX-6049-ExpectedResult.pdf"), resultFile)
}

// checkIdenticalContent stands where the Java's checkIdenticalRendering does:
// the same number of pages, and every page drawing the same marks from the same
// form XObject.
func checkIdenticalContent(t *testing.T, modelFile, resultFile string) {
	t.Helper()
	modelDocument := loadFile(t, modelFile)
	defer modelDocument.Close()
	resultDocument := loadFile(t, resultFile)
	defer resultDocument.Close()

	if modelDocument.NumberOfPages() != resultDocument.NumberOfPages() {
		t.Fatalf("%s has %d pages and %s has %d", modelFile,
			modelDocument.NumberOfPages(), resultFile, resultDocument.NumberOfPages())
	}
	for page := 0; page < modelDocument.NumberOfPages(); page++ {
		modelPage := modelDocument.Page(page)
		resultPage := resultDocument.Page(page)

		got := normaliseContent(contentOf(t, resultPage))
		want := normaliseContent(contentOf(t, modelPage))
		if !bytes.Equal(got, want) {
			t.Errorf("page %d of %s draws\n%q\nand the model draws\n%q",
				page, resultFile, firstDifference(got, want), firstDifference(want, got))
			continue
		}
		checkOverlayXObjects(t, page, modelPage, resultPage)
	}
}

// contentOf answers a page's content streams, decoded and concatenated, which
// is what the operators of the page are.
func contentOf(t *testing.T, page *pdmodel.PDPage) []byte {
	t.Helper()
	var out bytes.Buffer
	for _, stream := range page.ContentStreams() {
		reader, err := stream.CreateInputStream()
		if err != nil {
			t.Fatalf("reading a content stream: %v", err)
		}
		if _, err := out.ReadFrom(reader); err != nil {
			t.Fatal(err)
		}
	}
	return out.Bytes()
}

// checkOverlayXObjects compares every form XObject the two pages carry: what it
// draws, where it sits and how big it is.
func checkOverlayXObjects(t *testing.T, page int, modelPage, resultPage *pdmodel.PDPage) {
	t.Helper()
	modelResources := modelPage.Resources()
	resultResources := resultPage.Resources()
	if modelResources == nil || resultResources == nil {
		if modelResources != resultResources {
			t.Errorf("page %d: one of the two pages has no resources", page)
		}
		return
	}

	// Only the XObjects the page actually draws, in the order it draws them.
	// A page carries every overlay of a document it was split out of, because
	// the four imported pages of testRotatedOverlaysMap share one resources
	// dictionary; it draws exactly one of them. The rendering sees the same.
	modelNames := drawnXObjects(contentOf(t, modelPage))
	resultNames := drawnXObjects(contentOf(t, resultPage))
	if len(modelNames) != len(resultNames) {
		t.Errorf("page %d draws %d XObjects and the model draws %d",
			page, len(resultNames), len(modelNames))
		return
	}
	for i, name := range modelNames {
		modelXObject, err := modelResources.GetXObject(name)
		if err != nil {
			t.Fatalf("reading %s: %v", name.Name(), err)
		}
		resultXObject, err := resultResources.GetXObject(resultNames[i])
		if err != nil {
			t.Fatalf("reading %s: %v", resultNames[i].Name(), err)
		}
		if resultXObject == nil {
			t.Errorf("page %d has no XObject where the model has %s", page, name.Name())
			continue
		}
		modelStream, modelIsStream := modelXObject.COSObject().(*cos.Stream)
		resultStream, resultIsStream := resultXObject.COSObject().(*cos.Stream)
		if !modelIsStream || !resultIsStream {
			continue
		}
		for _, key := range []*cos.Name{cos.BBox, cos.Matrix} {
			checkNumberArray(t, page, name, key,
				resultStream.GetDictionaryObject(key), modelStream.GetDictionaryObject(key))
		}
		for _, key := range []*cos.Name{cos.Subtype, cos.FormType} {
			got := stringOf(resultStream.GetDictionaryObject(key))
			want := stringOf(modelStream.GetDictionaryObject(key))
			if got != want {
				t.Errorf("page %d, XObject %s: /%s is %s, want %s",
					page, name.Name(), key.Name(), got, want)
			}
		}
		if got, want := decodedOf(t, resultStream), decodedOf(t, modelStream); !bytes.Equal(got, want) {
			t.Errorf("page %d, XObject %s draws %d bytes and the model draws %d",
				page, name.Name(), len(got), len(want))
		}
	}
}

// decodedOf answers a stream's decoded bytes.
func decodedOf(t *testing.T, stream *cos.Stream) []byte {
	t.Helper()
	reader, err := stream.CreateReader()
	if err != nil {
		t.Fatalf("decoding a stream: %v", err)
	}
	var out bytes.Buffer
	if _, err := out.ReadFrom(reader); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// checkNumberArray compares a /BBox or a /Matrix by value rather than by the
// bytes it is written as.
//
// The one place the two differ is the sign of zero. `AffineTransform.
// quadrantRotate` produces a -0.0 for a quarter turn -- `m00 = -m01` where m01
// is 0 -- and today's `COSFloat` writes `String.valueOf(-0.0f)`, which is
// "-0.0"; measured, `new COSFloat(-0.0f)` on the current PDFBox renders
// COSFloat{-0.0}. The model files here say "0.0", because they were written
// when COSFloat went through a BigDecimal, which has no signed zero. The Java
// test cannot see the difference: it compares rendered pixels, and -0.0 and 0.0
// place a glyph in the same spot.
//
// So the port writes what the Java writes today and the fixture is older than
// both. Comparing the numbers rather than their spelling says what the test
// means, and would still catch a matrix that was actually wrong.
func checkNumberArray(t *testing.T, page int, name, key *cos.Name, got, want cos.Base) {
	t.Helper()
	gotArray, gotIsArray := got.(*cos.Array)
	wantArray, wantIsArray := want.(*cos.Array)
	if !gotIsArray || !wantIsArray {
		if gotIsArray != wantIsArray {
			t.Errorf("page %d, XObject %s: /%s is %v, want %v",
				page, name.Name(), key.Name(), got, want)
		}
		return
	}
	if gotArray.Size() != wantArray.Size() {
		t.Errorf("page %d, XObject %s: /%s holds %d values, want %d",
			page, name.Name(), key.Name(), gotArray.Size(), wantArray.Size())
		return
	}
	for i := 0; i < gotArray.Size(); i++ {
		gotNumber, gotIsNumber := gotArray.GetObject(i).(cos.Number)
		wantNumber, wantIsNumber := wantArray.GetObject(i).(cos.Number)
		if !gotIsNumber || !wantIsNumber {
			t.Errorf("page %d, XObject %s: /%s[%d] is %v, want %v",
				page, name.Name(), key.Name(), i, gotArray.GetObject(i), wantArray.GetObject(i))
			continue
		}
		// The addition turns a -0 into a 0 and leaves everything else alone.
		if gotNumber.FloatValue()+0 != wantNumber.FloatValue()+0 {
			t.Errorf("page %d, XObject %s: /%s[%d] is %v, want %v",
				page, name.Name(), key.Name(), i, gotNumber.FloatValue(), wantNumber.FloatValue())
		}
	}
}

// overlayName is the `/OL1`, `/OL2` ... a page's overlay XObject is called.
var overlayName = regexp.MustCompile(`/OL\d+`)

// runOfNewlines is one or more newlines in a row.
var runOfNewlines = regexp.MustCompile(`\n+`)

// normaliseContent puts a page's operators into the form the rendering
// compares them in: the two things below are the two ways the model files
// differ from a faithful overlay, and neither moves a mark on the page.
//
// **The XObject's name.** `PDResources.add(xobjForm, "OL")` numbers from the
// names already in the dictionary, and `testRotatedOverlaysMap` overlays four
// pages that share one imported resources dictionary, so they get OL1 to OL4.
// The model files were written by `testRotatedOverlay`, which overlays one page
// at a time and always gets OL1. Java's own map test compares its four pages
// against those same one-page models and passes, because it compares pixels.
//
// **Whitespace between content streams.** `PDPage.getContentsForRandomAccess`
// puts a newline after each stream of a /Contents array; a page that has been
// through `importPage` therefore carries one where the model, which was not
// imported, does not. The Java's map test produces the same extra newline and
// cannot see it either.
func normaliseContent(content []byte) []byte {
	content = overlayName.ReplaceAll(content, []byte("/OL"))
	return runOfNewlines.ReplaceAll(content, []byte("\n"))
}

// firstDifference answers the part of a around where it first differs from b,
// so that a failure names the operator that changed rather than the page.
func firstDifference(a, b []byte) string {
	at := 0
	for at < len(a) && at < len(b) && a[at] == b[at] {
		at++
	}
	from := max(0, at-40)
	to := min(len(a), at+40)
	return string(a[from:to])
}

// drawnDo is the `/Name Do` a content stream paints an XObject with.
var drawnDo = regexp.MustCompile(`/([A-Za-z0-9]+)\s+Do\b`)

// drawnXObjects answers the names a content stream draws, in order.
func drawnXObjects(content []byte) []*cos.Name {
	var names []*cos.Name
	for _, match := range drawnDo.FindAllSubmatch(content, -1) {
		names = append(names, cos.GetPDFName(string(match[1])))
	}
	return names
}
