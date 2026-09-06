package annotation_test

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/interactive/annotation/PDAnnotationTest.java,
// PDCircleAnnotationTest.java and PDSquareAnnotationTest.java.
//
// The package is annotation_test rather than annotation: PDAnnotationTest
// builds a text field, and interactive/form sits above this package.

import (
	"math"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// delta for comparing equality of float values.
const delta = 1e-4

// inDir and nameOfPDF are the IN_DIR and NAME_OF_PDF of
// PDSquareAnnotationTest.
const inDir = "../../../../../pdfbox/src/test/resources/org/apache/pdfbox/pdmodel/interactive/annotation/"
const nameOfPDF = "PDSquareAnnotationTest.pdf"

// squareRectangle is the setUp of PDSquareAnnotationTest: the location of the
// annotation.
func squareRectangle() *common.PDRectangle {
	rectangle := common.NewPDRectangle()
	rectangle.SetLowerLeftX(91.5958)
	rectangle.SetLowerLeftY(741.91)
	rectangle.SetUpperRightX(113.849)
	rectangle.SetUpperRightY(757.078)
	return rectangle
}

// assertClose is the assertEquals of Java with a delta.
func assertClose(t *testing.T, what string, got, want float32) {
	t.Helper()
	if math.Abs(float64(got-want)) > delta {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

// TestCreateDefaultWidgetAnnotation is
// PDAnnotationTest.createDefaultWidgetAnnotation.
func TestCreateDefaultWidgetAnnotation(t *testing.T) {
	annot := annotation.NewPDAnnotationWidget()
	if got := annot.AnnotationDictionary().GetItem(cos.Type); got != cos.Base(cos.Annot) {
		t.Errorf("/Type = %v, want %v", got, cos.Annot)
	}
	if got, want := annot.AnnotationDictionary().GetNameAsString(cos.Subtype, ""),
		annotation.SubTypeWidget; got != want {
		t.Errorf("/Subtype = %q, want %q", got, want)
	}
}

// TestCreateWidgetAnnotationFromField is
// PDAnnotationTest.createWidgetAnnotationFromField.
func TestCreateWidgetAnnotationFromField(t *testing.T) {
	document := pdmodel.NewPDDocument()
	acroForm := form.NewPDAcroForm(document)
	textField := form.NewPDTextField(acroForm)
	annot := textField.Widgets()[0]
	if got := annot.AnnotationDictionary().GetItem(cos.Type); got != cos.Base(cos.Annot) {
		t.Errorf("/Type = %v, want %v", got, cos.Annot)
	}
	if got, want := annot.AnnotationDictionary().GetNameAsString(cos.Subtype, ""),
		annotation.SubTypeWidget; got != want {
		t.Errorf("/Subtype = %q, want %q", got, want)
	}
}

// TestCreateDefaultCircleAnnotation is
// PDCircleAnnotationTest.createDefaultCircleAnnotation.
func TestCreateDefaultCircleAnnotation(t *testing.T) {
	annot := annotation.NewPDAnnotationCircle()
	if got := annot.AnnotationDictionary().GetItem(cos.Type); got != cos.Base(cos.Annot) {
		t.Errorf("/Type = %v, want %v", got, cos.Annot)
	}
	if got, want := annot.AnnotationDictionary().GetNameAsString(cos.Subtype, ""),
		annotation.SubTypeCircle; got != want {
		t.Errorf("/Subtype = %q, want %q", got, want)
	}
}

// TestCreateDefaultSquareAnnotation is
// PDSquareAnnotationTest.createDefaultSquareAnnotation.
func TestCreateDefaultSquareAnnotation(t *testing.T) {
	annot := annotation.NewPDAnnotationSquare()
	if got := annot.AnnotationDictionary().GetItem(cos.Type); got != cos.Base(cos.Annot) {
		t.Errorf("/Type = %v, want %v", got, cos.Annot)
	}
	if got, want := annot.AnnotationDictionary().GetNameAsString(cos.Subtype, ""),
		annotation.SubTypeSquare; got != want {
		t.Errorf("/Subtype = %q, want %q", got, want)
	}
}

// TestCreateWithAppearance is PDSquareAnnotationTest.createWithAppearance.
func TestCreateWithAppearance(t *testing.T) {
	// the width of the annotations border
	const borderWidth = 1
	document := pdmodel.NewPDDocument()
	defer document.Close()
	page := pdmodel.NewPDPage()
	document.AddPage(page)
	annotations := page.Annotations()
	annot := annotation.NewPDAnnotationSquare()
	borderThin := annotation.NewPDBorderStyleDictionary()
	borderThin.SetWidth(borderWidth)
	red := color.NewPDColorOfComponents([]float32{1, 0, 0}, color.DeviceRGB)
	annot.SetContents("Square Annotation")
	annot.SetColor(red)
	annot.SetBorderStyle(borderThin)
	annot.SetRectangle(squareRectangle())
	if err := annot.ConstructAppearances(); err != nil {
		t.Fatal(err)
	}
	annotations.Add(annot)
}

// TestValidateAppearance is PDSquareAnnotationTest.validateAppearance.
func TestValidateAppearance(t *testing.T) {
	// the width of the annotations border
	const borderWidth = 1
	rectangle := squareRectangle()
	document, err := pdfbox.LoadPDF(inDir + nameOfPDF)
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()

	page := document.Page(0)
	annotations := page.Annotations().ToSlice()
	annot := annotations[0]

	// test the correct setting of the appearance stream
	if annot.Appearance() == nil {
		t.Fatal("Appearance dictionary shall not be null")
	}
	if annot.Appearance().NormalAppearance() == nil {
		t.Fatal("Normal appearance shall not be null")
	}
	appearanceStream := annot.Appearance().NormalAppearance().AppearanceStream()
	if appearanceStream == nil {
		t.Fatal("Appearance stream shall not be null")
	}
	assertClose(t, "BBox().LowerLeftX()",
		appearanceStream.BBox().LowerLeftX(), rectangle.LowerLeftX())
	assertClose(t, "BBox().LowerLeftY()",
		appearanceStream.BBox().LowerLeftY(), rectangle.LowerLeftY())
	assertClose(t, "BBox().Width()", appearanceStream.BBox().Width(), rectangle.Width())
	assertClose(t, "BBox().Height()", appearanceStream.BBox().Height(), rectangle.Height())

	matrix := appearanceStream.Matrix()
	if matrix == nil {
		t.Fatal("Matrix shall not be null")
	}
	// should have been translated to a 0 origin
	assertClose(t, "matrix.TranslateX()", matrix.TranslateX(), -rectangle.LowerLeftX())
	assertClose(t, "matrix.TranslateY()", matrix.TranslateY(), -rectangle.LowerLeftY())

	// test the content of the appearance stream
	if appearanceStream.ContentStream() == nil {
		t.Fatal("Content stream shall not be null")
	}
	content, err := appearanceStream.ContentsForRandomAccess()
	if err != nil {
		t.Fatal(err)
	}
	parser, err := pdfparser.NewStreamTokenParserSource(content)
	if err != nil {
		t.Fatal(err)
	}
	tokens, err := parser.Parse()
	if err != nil {
		t.Fatal(err)
	}
	// the samples content stream should contain 10 tokens
	if len(tokens) != 10 {
		t.Fatalf("tokens = %d, want 10", len(tokens))
	}
	// setting of the stroking color
	for i, want := range []int{1, 0, 0} {
		integer, isInteger := tokens[i].(*cos.Integer)
		if !isInteger {
			t.Fatalf("tokens[%d] = %T, want *cos.Integer", i, tokens[i])
		}
		if got := integer.IntValue(); got != want {
			t.Errorf("tokens[%d] = %d, want %d", i, got, want)
		}
	}
	assertOperator(t, tokens, 3, "RG")
	// setting of the rectangle for the border
	// it shall be inset by the border width
	for i, want := range []float32{
		rectangle.LowerLeftX() + borderWidth,
		rectangle.LowerLeftY() + borderWidth,
		rectangle.Width() - 2*borderWidth,
		rectangle.Height() - 2*borderWidth,
	} {
		index := i + 4
		float, isFloat := tokens[index].(*cos.Float)
		if !isFloat {
			t.Fatalf("tokens[%d] = %T, want *cos.Float", index, tokens[index])
		}
		assertClose(t, "token", float.FloatValue(), want)
	}
	assertOperator(t, tokens, 8, "re")
	assertOperator(t, tokens, 9, "S")
}

// assertOperator checks that the token at the given index is the named
// operator.
func assertOperator(t *testing.T, tokens []any, index int, want string) {
	t.Helper()
	op, isOperator := tokens[index].(*operator.Operator)
	if !isOperator {
		t.Fatalf("tokens[%d] = %T, want *operator.Operator", index, tokens[index])
	}
	if got := op.Name(); got != want {
		t.Errorf("tokens[%d] = %q, want %q", index, got, want)
	}
}
