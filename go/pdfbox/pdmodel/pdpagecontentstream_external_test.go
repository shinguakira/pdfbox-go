package pdmodel_test

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/TestPDPageContentStream.java.
//
// The package is pdmodel_test rather than pdmodel: the tokens are read back
// with pdfparser.StreamTokenParser, and pdfparser sits below pdmodel, so an
// in-package test would be fine -- but the file also uses the shared helpers of
// the other pdmodel_test files, and Go allows a package only one test package
// of each kind per directory anyway.
//
// The Java asserts IllegalArgumentException for a colour component outside 0..1
// and IllegalStateException for a path or image operator inside a text block;
// both are unchecked, so the port panics and the assertions recover.
//
// testGeneralGraphicStateOperatorTextMode's shadingFill assertion is not here:
// shadingFill takes a PDShading, which is slice 9's, and is one of the
// PDAbstractContentStream methods this slice left out. See migration/STATUS.md.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// TestSetCmykColors is TestPDPageContentStream.testSetCmykColors.
func TestSetCmykColors(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	page := pdmodel.NewPDPage()
	doc.AddPage(page)

	contentStream, err := pdmodel.NewPDPageContentStreamCompressed(doc, page, pdmodel.Overwrite, true)
	if err != nil {
		t.Fatal(err)
	}
	// pass a non-stroking color in CMYK color space
	if err := contentStream.SetNonStrokingColorCMYK(0.1, 0.2, 0.3, 0.4); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][4]float32{{1.1, 0, 0, 0}, {0, 1.1, 0, 0}, {0, 0, 1.1, 0}, {0, 0, 0, 1.1}} {
		assertPanics(t, "SetNonStrokingColorCMYK", func() {
			_ = contentStream.SetNonStrokingColorCMYK(bad[0], bad[1], bad[2], bad[3])
		})
	}
	if err := contentStream.Close(); err != nil {
		t.Fatal(err)
	}

	// now read the PDF stream and verify that the CMYK values are correct
	// expected five tokens :
	// [0] = COSFloat{0.1} .. [3] = COSFloat{0.4}
	// [4] = PDFOperator{"k"}
	assertTokens(t, page, []float32{0.1, 0.2, 0.3, 0.4}, operator.NonStrokingCmyk)

	// same as above but for PDPageContentStream#setStrokingColor
	page = pdmodel.NewPDPage()
	doc.AddPage(page)

	contentStream, err = pdmodel.NewPDPageContentStreamCompressed(doc, page, pdmodel.Overwrite, false)
	if err != nil {
		t.Fatal(err)
	}
	// pass a stroking color in CMYK color space
	if err := contentStream.SetStrokingColorCMYK(0.5, 0.6, 0.7, 0.8); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][4]float32{{1.1, 0, 0, 0}, {0, 1.1, 0, 0}, {0, 0, 1.1, 0}, {0, 0, 0, 1.1}} {
		assertPanics(t, "SetStrokingColorCMYK", func() {
			_ = contentStream.SetStrokingColorCMYK(bad[0], bad[1], bad[2], bad[3])
		})
	}
	if err := contentStream.Close(); err != nil {
		t.Fatal(err)
	}

	assertTokens(t, page, []float32{0.5, 0.6, 0.7, 0.8}, operator.StrokingColorCmyk)
}

// TestSetRGBandGColors is TestPDPageContentStream.testSetRGBandGColors.
func TestSetRGBandGColors(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	page := pdmodel.NewPDPage()
	doc.AddPage(page)

	contentStream, err := pdmodel.NewPDPageContentStreamCompressed(doc, page, pdmodel.Overwrite, true)
	if err != nil {
		t.Fatal(err)
	}
	// pass a non-stroking color in RGB and Gray color space
	if err := contentStream.SetNonStrokingColorRGB(0.1, 0.2, 0.3); err != nil {
		t.Fatal(err)
	}
	if err := contentStream.SetNonStrokingColorGray(0.8); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][3]float32{{1.1, 0, 0}, {0, 1.1, 0}, {0, 0, 1.1}} {
		assertPanics(t, "SetNonStrokingColorRGB", func() {
			_ = contentStream.SetNonStrokingColorRGB(bad[0], bad[1], bad[2])
		})
	}
	assertPanics(t, "SetNonStrokingColorGray", func() {
		_ = contentStream.SetNonStrokingColorGray(1.1)
	})
	if err := contentStream.Close(); err != nil {
		t.Fatal(err)
	}

	// now read the PDF stream and verify that the values are correct
	assertTokens(t, page, []float32{0.1, 0.2, 0.3}, operator.NonStrokingRgb,
		tokenTail{value: 0.8, name: operator.NonStrokingGray})

	// same as above but for PDPageContentStream#setStrokingColor
	page = pdmodel.NewPDPage()
	doc.AddPage(page)

	contentStream, err = pdmodel.NewPDPageContentStreamCompressed(doc, page, pdmodel.Overwrite, false)
	if err != nil {
		t.Fatal(err)
	}
	// pass a stroking color in RGB and Gray color space
	if err := contentStream.SetStrokingColorRGB(0.5, 0.6, 0.7); err != nil {
		t.Fatal(err)
	}
	if err := contentStream.SetStrokingColorGray(0.8); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][3]float32{{1.1, 0, 0}, {0, 1.1, 0}, {0, 0, 1.1}} {
		assertPanics(t, "SetStrokingColorRGB", func() {
			_ = contentStream.SetStrokingColorRGB(bad[0], bad[1], bad[2])
		})
	}
	assertPanics(t, "SetStrokingColorGray", func() {
		_ = contentStream.SetStrokingColorGray(1.1)
	})
	if err := contentStream.Close(); err != nil {
		t.Fatal(err)
	}

	assertTokens(t, page, []float32{0.5, 0.6, 0.7}, operator.StrokingColorRgb,
		tokenTail{value: 0.8, name: operator.StrokingColorGray})
}

// tokenTail is one more operand-and-operator pair to check after the first.
type tokenTail struct {
	value float32
	name  string
}

// assertTokens reads the page's content stream back and checks that it holds
// the given operands followed by the given operator, and then each tail pair.
func assertTokens(t *testing.T, page *pdmodel.PDPage, operands []float32,
	name string, tails ...tokenTail) {
	t.Helper()
	pageTokens := parsePage(t, page)
	want := len(operands) + 1
	for range tails {
		want += 2
	}
	if got := len(pageTokens); got != want {
		t.Fatalf("tokens = %d, want %d: %v", got, want, pageTokens)
	}
	at := 0
	check := func(value float32) {
		number, isNumber := pageTokens[at].(cos.Number)
		if !isNumber {
			t.Fatalf("tokens[%d] = %T, want a number", at, pageTokens[at])
		}
		if got := number.FloatValue(); got != value {
			t.Errorf("tokens[%d] = %v, want %v", at, got, value)
		}
		at++
	}
	checkOperator := func(want string) {
		op, isOperator := pageTokens[at].(*operator.Operator)
		if !isOperator {
			t.Fatalf("tokens[%d] = %T, want an operator", at, pageTokens[at])
		}
		if got := op.Name(); got != want {
			t.Errorf("tokens[%d] = %q, want %q", at, got, want)
		}
		at++
	}
	for _, value := range operands {
		check(value)
	}
	checkOperator(name)
	for _, tail := range tails {
		check(tail.value)
		checkOperator(tail.name)
	}
}

// parsePage is `new PDFStreamParser(page).parse()`.
func parsePage(t *testing.T, page *pdmodel.PDPage) []any {
	t.Helper()
	contents, err := page.ContentsForStreamParsing()
	if err != nil {
		t.Fatal(err)
	}
	parser, err := pdfparser.NewStreamTokenParserSource(contents)
	if err != nil {
		t.Fatal(err)
	}
	tokens, err := parser.Parse()
	if err != nil {
		t.Fatal(err)
	}
	return tokens
}

// TestMissingContentStream is TestPDPageContentStream.testMissingContentStream:
// PDFBOX-3510, a missing content stream should not fail.
func TestMissingContentStream(t *testing.T) {
	page := pdmodel.NewPDPage()
	if got := len(parsePage(t, page)); got != 0 {
		t.Errorf("tokens = %d, want 0", got)
	}
}

// TestCloseContract is TestPDPageContentStream.testCloseContract: check that
// Close can be called twice.
func TestCloseContract(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	page := pdmodel.NewPDPage()
	doc.AddPage(page)
	contentStream, err := pdmodel.NewPDPageContentStreamCompressed(doc, page, pdmodel.Overwrite, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := contentStream.Close(); err != nil {
		t.Fatal(err)
	}
	if err := contentStream.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestGeneralGraphicStateOperatorTextMode is
// TestPDPageContentStream.testGeneralGraphicStateOperatorTextMode: check that
// general graphics state operators are allowed in text mode, and that the path
// and image ones are not.
func TestGeneralGraphicStateOperatorTextMode(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	page := pdmodel.NewPDPage()
	doc.AddPage(page)
	contentStream, err := pdmodel.NewPDPageContentStream(doc, page)
	if err != nil {
		t.Fatal(err)
	}
	if err := contentStream.BeginText(); err != nil {
		t.Fatal(err)
	}

	// Java's PDImageXObject(PDDocument) is `this(new PDStream(document), null)`.
	img1 := image.NewPDImageXObject(common.NewPDStreamOfDocument(doc.Document()), nil)
	img2, err := image.NewPDInlineImage(cos.NewDictionary(), []byte{}, pdmodel.NewPDResources())
	if err != nil {
		t.Fatal(err)
	}
	for _, refused := range []struct {
		what string
		call func() error
	}{
		{"DrawImageSized", func() error { return contentStream.DrawImageSized(img1, 0, 0, 1, 1) }},
		{"DrawImageWithMatrix", func() error {
			return contentStream.DrawImageWithMatrix(img1, util.NewMatrix())
		}},
		{"DrawInlineImageSized", func() error { return contentStream.DrawInlineImageSized(img2, 0, 0, 1, 1) }},
		{"AddRect", func() error { return contentStream.AddRect(0, 0, 1, 1) }},
		{"CurveTo", func() error { return contentStream.CurveTo(0, 0, 1, 1, 2, 2) }},
		{"CurveTo1", func() error { return contentStream.CurveTo1(0, 0, 1, 1) }},
		{"CurveTo2", func() error { return contentStream.CurveTo2(0, 0, 1, 1) }},
		{"MoveTo", func() error { return contentStream.MoveTo(0, 0) }},
		{"LineTo", func() error { return contentStream.LineTo(1, 1) }},
		{"Stroke", contentStream.Stroke},
		{"CloseAndStroke", contentStream.CloseAndStroke},
		{"CloseAndFillAndStroke", contentStream.CloseAndFillAndStroke},
		{"CloseAndFillAndStrokeEvenOdd", contentStream.CloseAndFillAndStrokeEvenOdd},
		{"Fill", contentStream.Fill},
		{"FillAndStroke", contentStream.FillAndStroke},
		{"FillAndStrokeEvenOdd", contentStream.FillAndStrokeEvenOdd},
		{"FillEvenOdd", contentStream.FillEvenOdd},
		{"ClosePath", contentStream.ClosePath},
		{"Clip", contentStream.Clip},
		{"ClipEvenOdd", contentStream.ClipEvenOdd},
	} {
		assertPanics(t, refused.what, func() { _ = refused.call() })
	}

	for _, allowed := range []struct {
		what string
		call func() error
	}{
		// J
		{"SetLineCapStyle", func() error { return contentStream.SetLineCapStyle(0) }},
		// j
		{"SetLineJoinStyle", func() error { return contentStream.SetLineJoinStyle(0) }},
		// w
		{"SetLineWidth", func() error { return contentStream.SetLineWidth(10) }},
		// d
		{"SetLineDashPattern", func() error {
			return contentStream.SetLineDashPattern([]float32{2, 1}, 0)
		}},
		// M
		{"SetMiterLimit", func() error { return contentStream.SetMiterLimit(1.0) }},
		// gs
		{"SetGraphicsStateParameters", func() error {
			return contentStream.SetGraphicsStateParameters(state.NewPDExtendedGraphicsState())
		}},
		// ri, i are not supported with a specific setter
		{"EndText", contentStream.EndText},
		{"Close", contentStream.Close},
	} {
		if err := allowed.call(); err != nil {
			t.Errorf("%s() = %v, want no error", allowed.what, err)
		}
	}
}

// assertPanics is the assertThrows of Java, which the port raises as a panic.
func assertPanics(t *testing.T, what string, call func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s did not panic", what)
		}
	}()
	call()
}
