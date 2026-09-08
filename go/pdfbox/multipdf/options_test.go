package multipdf_test

// The public surface the Java tests do not reach.
//
// `PDFMergerUtilityTest`, `MergeAcroFormsTest` and `OverlayTest` between them
// exercise the legacy merge, the default overlay and the specific-page map;
// they never set the optimised document merge mode, the join-fields AcroForm
// mode, the destination document information or metadata, the ignore-errors
// flag, a RandomAccessRead source, or five of Overlay's six page selectors.
// Phase D of the branch asks for a test that names each function rather than a
// green suite, so this is that file: what each of those does, from the Java
// source that decides it.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter/compress"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// TestOptimizeResourcesModeMergesPagesAndResources is the other
// DocumentMergeMode. It "optimizes resource handling such as closing documents
// early. Not all document elements are merged compared to the
// PDFBOX_LEGACY_MODE. Currently supported are: page content and resources."
func TestOptimizeResourcesModeMergesPagesAndResources(t *testing.T) {
	result := filepath.Join(t.TempDir(), "optimised.pdf")
	merger := multipdf.NewPDFMergerUtility()
	merger.SetDocumentMergeMode(multipdf.OptimizeResourcesMode)
	if got := merger.DocumentMergeMode(); got != multipdf.OptimizeResourcesMode {
		t.Errorf("getDocumentMergeMode() is %v, want the optimised mode", got)
	}
	// PDFA3A.pdf is tagged, so what the two modes do differently shows.
	for i := 0; i < 2; i++ {
		if err := merger.AddSourceFile(filepath.Join(srcDir, "PDFA3A.pdf")); err != nil {
			t.Fatal(err)
		}
	}
	merger.SetDestinationFileName(result)
	if err := merger.MergeDocuments(); err != nil {
		t.Fatalf("mergeDocuments: %v", err)
	}

	source := loadFixture(t, "PDFA3A.pdf")
	if structureTreeRootOf(t, source) == nil {
		t.Fatal("the fixture is not tagged, so this test would say nothing")
	}
	wantPages := source.NumberOfPages() * 2
	source.Close()

	merged := loadFile(t, result)
	defer merged.Close()
	if got := merged.NumberOfPages(); got != wantPages {
		t.Errorf("the merged document has %d pages, want %d", got, wantPages)
	}
	// "Currently supported are: page content and resources" -- so every page
	// keeps what it drew with.
	for page := range merged.Pages().All {
		if page.Resources() == nil {
			t.Error("a merged page lost its resources")
		}
	}
	// And what the legacy mode brings across, this one does not.
	if merged.DocumentCatalog().StructureTreeRoot() != nil {
		t.Error("the optimised mode carried a structure tree, which it does not merge")
	}
}

// TestMergeDocumentsTakesCompressParameters is the overload that decides
// whether the result holds object streams.
func TestMergeDocumentsTakesCompressParameters(t *testing.T) {
	dir := t.TempDir()
	compressed := filepath.Join(dir, "compressed.pdf")
	plain := filepath.Join(dir, "plain.pdf")

	for _, c := range []struct {
		path       string
		parameters *compress.Parameters
	}{
		{compressed, compress.DefaultCompression},
		{plain, compress.NoCompression},
	} {
		merger := multipdf.NewPDFMergerUtility()
		if err := merger.AddSourceFile(filepath.Join(srcDir, "jpegrgb.pdf")); err != nil {
			t.Fatal(err)
		}
		merger.SetDestinationFileName(c.path)
		if err := merger.MergeDocumentsOfParameters(c.parameters); err != nil {
			t.Fatalf("mergeDocuments: %v", err)
		}
	}

	// An object stream is what compressing a document produces, and the
	// uncompressed one has none.
	if !bytes.Contains(readFile(t, compressed), []byte("/ObjStm")) {
		t.Error("the compressed merge holds no object stream")
	}
	if bytes.Contains(readFile(t, plain), []byte("/ObjStm")) {
		t.Error("the uncompressed merge holds an object stream")
	}
}

// TestAddSourceTakesARandomAccessRead is `addSource(RandomAccessRead)` and
// `addSources(List)`, which is what the Java's testFileDeletion uses and what
// a caller with bytes rather than a file needs.
func TestAddSourceTakesARandomAccessRead(t *testing.T) {
	result := filepath.Join(t.TempDir(), "merged.pdf")
	first := pdfio.NewReadBufferBytes(readFile(t, filepath.Join(srcDir, "jpegrgb.pdf")))
	second := pdfio.NewReadBufferBytes(readFile(t, filepath.Join(srcDir, "multitiff.pdf")))

	merger := multipdf.NewPDFMergerUtility()
	merger.AddSource(first)
	merger.AddSources([]pdfio.RandomAccessRead{second})
	merger.SetDestinationFileName(result)
	if err := merger.MergeDocuments(); err != nil {
		t.Fatalf("mergeDocuments: %v", err)
	}

	merged := loadFile(t, result)
	defer merged.Close()
	if got := merged.NumberOfPages(); got != 4 {
		t.Errorf("the merged document has %d pages, want 1 + 3", got)
	}
}

// TestDestinationDocumentInformationAndMetadata are the two the legacy merge
// puts on the result after every source has been appended. "The default is
// null, which means that it is ignored."
func TestDestinationDocumentInformationAndMetadata(t *testing.T) {
	result := filepath.Join(t.TempDir(), "merged.pdf")

	information := pdmodel.NewPDDocumentInformationEmpty()
	information.SetTitle("the destination title")

	merger := multipdf.NewPDFMergerUtility()
	merger.SetDestinationDocumentInformation(information)
	if merger.DestinationDocumentInformation() != information {
		t.Error("getDestinationDocumentInformation() is not what was set")
	}
	if merger.DestinationMetadata() != nil {
		t.Error("the destination metadata defaults to something")
	}

	// The metadata has to belong to a document, and the merge makes its own, so
	// it is built over a document of its own the way a caller would.
	holder := pdmodel.NewPDDocument()
	defer holder.Close()
	metadata, err := common.NewPDMetadataOfInput(holder,
		strings.NewReader("<?xpacket begin='' id='W5M0MpCehiHzreSzNTczkc9d'?></x:xmpmeta>"))
	if err != nil {
		t.Fatalf("NewPDMetadata: %v", err)
	}
	merger.SetDestinationMetadata(metadata)

	if err := merger.AddSourceFile(filepath.Join(srcDir, "jpegrgb.pdf")); err != nil {
		t.Fatal(err)
	}
	merger.SetDestinationFileName(result)
	if err := merger.MergeDocuments(); err != nil {
		t.Fatalf("mergeDocuments: %v", err)
	}

	merged := loadFile(t, result)
	defer merged.Close()
	if got := merged.DocumentInformation().Title(); got != "the destination title" {
		t.Errorf("the merged document's title is %q, want the one that was set", got)
	}
	if merged.DocumentCatalog().Metadata() == nil {
		t.Error("the merged document has no /Metadata")
	}
}

// TestJoinFormFieldsModeIsTheLegacyMode is `acroFormJoinFieldsMode`, whose
// whole body in Java is a call to `acroFormLegacyMode`: the mode is declared,
// documented as merging fields of the same name into one, and does not.
func TestJoinFormFieldsModeIsTheLegacyMode(t *testing.T) {
	dir := t.TempDir()
	joined := mergeAcroFormTwice(t, filepath.Join(dir, "joined.pdf"),
		multipdf.JoinFormFieldsMode)
	legacy := mergeAcroFormTwice(t, filepath.Join(dir, "legacy.pdf"),
		multipdf.PDFBoxLegacyAcroFormMode)

	if joined != legacy {
		t.Errorf("the join-fields mode gave %v root fields and the legacy mode %v; "+
			"acroFormJoinFieldsMode's body is a call to acroFormLegacyMode",
			joined, legacy)
	}
}

// mergeAcroFormTwice merges AcroFormForMerge.pdf with itself in the given mode
// and answers the field names of the result.
func mergeAcroFormTwice(t *testing.T, result string, mode multipdf.AcroFormMergeMode) string {
	t.Helper()
	merger := multipdf.NewPDFMergerUtility()
	merger.SetAcroFormMergeMode(mode)
	if got := merger.AcroFormMergeMode(); got != mode {
		t.Errorf("getAcroFormMergeMode() is %v, want %v", got, mode)
	}
	for i := 0; i < 2; i++ {
		if err := merger.AddSourceFile(filepath.Join(multipdfDir, "AcroFormForMerge.pdf")); err != nil {
			t.Fatal(err)
		}
	}
	merger.SetDestinationFileName(result)
	if err := merger.MergeDocuments(); err != nil {
		t.Fatalf("mergeDocuments: %v", err)
	}

	doc := loadFile(t, result)
	defer doc.Close()
	acroForm := doc.DocumentCatalog().COSObject().(*cos.Dictionary).GetCOSDictionary(cos.AcroForm)
	if acroForm == nil {
		t.Fatal("the merged document has no AcroForm")
	}
	fields := acroForm.GetCOSArray(cos.Fields)
	if fields == nil {
		t.Fatal("the merged AcroForm has no /Fields")
	}
	var names []string
	for i := 0; i < fields.Size(); i++ {
		field, isDictionary := fields.GetObject(i).(*cos.Dictionary)
		if !isDictionary {
			continue
		}
		names = append(names, field.GetString(cos.T, ""))
	}
	return strings.Join(names, ",")
}

// TestIgnoreAcroFormErrorsSwallowsThem is the flag `mergeAcroForm`'s catch
// asks: "if we are not ignoring exceptions, we'll re-throw this".
func TestIgnoreAcroFormErrorsSwallowsThem(t *testing.T) {
	merger := multipdf.NewPDFMergerUtility()
	if merger.IsIgnoreAcroFormErrors() {
		t.Error("acroform errors are ignored by default")
	}
	merger.SetIgnoreAcroFormErrors(true)
	if !merger.IsIgnoreAcroFormErrors() {
		t.Error("setIgnoreAcroFormErrors(true) did not take")
	}

	// The flag changes nothing for a document whose form merges cleanly, which
	// is what says the flag is a catch and not a skip.
	destination := pdmodel.NewPDDocument()
	defer destination.Close()
	source := loadFile(t, filepath.Join(multipdfDir, "AcroFormForMerge.pdf"))
	defer source.Close()
	if err := merger.AppendDocument(destination, source); err != nil {
		t.Fatalf("appendDocument: %v", err)
	}
	if got := destination.NumberOfPages(); got != source.NumberOfPages() {
		t.Errorf("the destination has %d pages and the source %d",
			got, source.NumberOfPages())
	}
	acroForm := destination.DocumentCatalog().COSObject().(*cos.Dictionary).
		GetCOSDictionary(cos.AcroForm)
	if acroForm == nil {
		t.Error("the form was not merged")
	}
}

// TestOverlaySelectorsPickTheRightPages is the five setters OverlayTest never
// uses: first, last, odd, even and all pages.
func TestOverlaySelectorsPickTheRightPages(t *testing.T) {
	overlayFile := filepath.Join(multipdfDir, "rot0.pdf")

	for _, c := range []struct {
		name  string
		set   func(*multipdf.Overlay)
		pages []bool // whether each of the four pages gets an overlay
	}{
		{"first", func(o *multipdf.Overlay) { o.SetFirstPageOverlayFile(overlayFile) },
			[]bool{true, false, false, false}},
		{"last", func(o *multipdf.Overlay) { o.SetLastPageOverlayFile(overlayFile) },
			[]bool{false, false, false, true}},
		{"odd", func(o *multipdf.Overlay) { o.SetOddPageOverlayFile(overlayFile) },
			[]bool{true, false, true, false}},
		{"even", func(o *multipdf.Overlay) { o.SetEvenPageOverlayFile(overlayFile) },
			[]bool{false, true, false, true}},
		{"all", func(o *multipdf.Overlay) { o.SetAllPagesOverlayFile(overlayFile) },
			[]bool{true, true, true, true}},
	} {
		t.Run(c.name, func(t *testing.T) {
			input := fourBlankPages(t, filepath.Join(t.TempDir(), "input.pdf"))
			overlay := multipdf.NewOverlay()
			overlay.SetInputPDF(input)
			c.set(overlay)
			result, err := overlay.Overlay(map[int]string{})
			if err != nil {
				t.Fatalf("overlay: %v", err)
			}
			for i, want := range c.pages {
				resources := result.Page(i).Resources()
				got := resources != nil && len(resources.XObjectNames()) > 0
				if got != want {
					t.Errorf("page %d carries an overlay: %v, want %v", i+1, got, want)
				}
			}
			if err := overlay.Close(); err != nil {
				t.Fatal(err)
			}
			input.Close()
		})
	}
}

// TestOverlayDocumentsTakesOpenDocuments is `overlayDocuments`, the sibling of
// `overlay` that takes documents rather than file names. "If you created the
// overlay documents with subsetted fonts, you need to save them first so that
// the subsetting gets done."
func TestOverlayDocumentsTakesOpenDocuments(t *testing.T) {
	input := fourBlankPages(t, filepath.Join(t.TempDir(), "input.pdf"))
	defer input.Close()
	overlayDoc := loadFile(t, filepath.Join(multipdfDir, "rot0.pdf"))
	defer overlayDoc.Close()

	overlay := multipdf.NewOverlay()
	overlay.SetInputPDF(input)
	result, err := overlay.OverlayDocuments(map[int]*pdmodel.PDDocument{
		2: overlayDoc,
		4: nil, // "if (doc != null)": a null entry is skipped
	})
	if err != nil {
		t.Fatalf("overlayDocuments: %v", err)
	}
	for i, want := range []bool{false, true, false, false} {
		resources := result.Page(i).Resources()
		got := resources != nil && len(resources.XObjectNames()) > 0
		if got != want {
			t.Errorf("page %d carries an overlay: %v, want %v", i+1, got, want)
		}
	}
	if err := overlay.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestWrapInSaveRestoreOfAnArray is the second arm of wrapInSaveRestore: a page
// whose /Contents is already an array gets the q at the front and the Q at the
// back, rather than a new array of three.
func TestWrapInSaveRestoreOfAnArray(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	page := pdmodel.NewPDPage()
	doc.AddPage(page)
	// Two appended streams make the /Contents an array.
	for i := 0; i < 2; i++ {
		stream, err := pdmodel.NewPDPageContentStreamCompressed(doc, page,
			pdmodel.Append, false)
		if err != nil {
			t.Fatal(err)
		}
		if err := stream.Close(); err != nil {
			t.Fatal(err)
		}
	}
	before := page.Dictionary().GetCOSArray(cos.Contents)
	if before == nil {
		t.Fatal("two appended streams did not make the /Contents an array")
	}
	sizeBefore := before.Size()

	if err := multipdf.NewLayerUtility(doc).WrapInSaveRestore(page); err != nil {
		t.Fatalf("wrapInSaveRestore: %v", err)
	}

	after := page.Dictionary().GetCOSArray(cos.Contents)
	if after != before {
		t.Error("the array was replaced; Java adds to the one that is there")
	}
	if got := after.Size(); got != sizeBefore+2 {
		t.Fatalf("the array holds %d streams, want %d", got, sizeBefore+2)
	}
	if got := decodedOf(t, after.GetObject(0).(*cos.Stream)); string(got) != "q\n" {
		t.Errorf("the first stream is %q, want %q", got, "q\n")
	}
	last := after.GetObject(after.Size() - 1).(*cos.Stream)
	if got := decodedOf(t, last); string(got) != "Q\n" {
		t.Errorf("the last stream is %q, want %q", got, "Q\n")
	}
}

// fourBlankPages saves and reloads a four-page document, which is what an
// overlay needs: a document it can write into.
func fourBlankPages(t *testing.T, path string) *pdmodel.PDDocument {
	t.Helper()
	doc := pdmodel.NewPDDocument()
	for i := 0; i < 4; i++ {
		doc.AddPage(pdmodel.NewPDPage())
	}
	if err := doc.SaveToFile(path); err != nil {
		t.Fatal(err)
	}
	doc.Close()
	reloaded, err := pdfbox.LoadPDF(path)
	if err != nil {
		t.Fatal(err)
	}
	return reloaded
}

// readFile is os.ReadFile with the test failed on an error.
func readFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return content
}
