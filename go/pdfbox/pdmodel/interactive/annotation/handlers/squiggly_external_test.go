package handlers_test

// The squiggly annotation's appearance, against PDFBox's own.
//
// PDSquigglyAppearanceHandler is the one appearance handler that fills with a
// tiling pattern, and the port could not write it until `track/raster` brought
// PDTilingPattern, PDPatternContentStream and the PDPattern colour space.
//
// `testdata/squiggly.pdf` is what PDFBox generated for two squiggly
// annotations -- see `testdata/SquigglyDrv.java`, which is checked in beside
// it. The test throws the appearances away, regenerates them with the port,
// and compares the content streams token for token, which is what
// AppearanceGenerationTest.checkAnnotationTokens does.
//
// The comparison reaches further than that one does, because the squiggle's
// appearance is three streams deep: the annotation's, the form it draws, and
// the tiling pattern the form fills with. All three are compared.

import (
	"math"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	_ "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/pattern"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// appearanceDelta is AppearanceGenerationTest's DELTA, the tolerance two
// numbers in a content stream may differ by.
const appearanceDelta = 1e-4

// TestSquigglyAppearanceMatchesPDFBox regenerates both annotations and
// compares every stream that goes into them.
func TestSquigglyAppearanceMatchesPDFBox(t *testing.T) {
	document, err := pdfbox.LoadPDF("testdata/squiggly.pdf")
	if err != nil {
		t.Fatalf("loading the reference: %v", err)
	}
	defer document.Close()

	annotations := document.Page(0).Annotations().ToSlice()
	if len(annotations) != 2 {
		t.Fatalf("the reference has %d annotations, want 2", len(annotations))
	}

	for index, annot := range annotations {
		// What PDFBox generated.
		wanted := streamsOf(t, annot)

		// What the port generates for the same annotation.
		annot.AnnotationDictionary().RemoveItem(cos.AP)
		if err := annot.ConstructAppearances(); err != nil {
			t.Fatalf("annotation %d: ConstructAppearances: %v", index, err)
		}
		got := streamsOf(t, annot)

		if len(got) != len(wanted) {
			t.Fatalf("annotation %d: the port wrote %d streams and PDFBox wrote %d",
				index, len(got), len(wanted))
		}
		// The appearance, the form it draws and the pattern that form fills
		// with. A squiggle that lost its pattern would still compare equal on
		// the first two.
		if len(wanted) != 3 {
			t.Fatalf("annotation %d is made of %d streams, want 3", index, len(wanted))
		}
		for stream := range wanted {
			compareTokens(t, index, stream, wanted[stream], got[stream])
		}
	}
}

// streamsOf is every content stream the annotation's normal appearance is made
// of, tokenised: the appearance itself, then each form it draws, then each
// tiling pattern those fill with.
//
// They come back in a fixed order -- appearance, forms in resource-name order,
// patterns in resource-name order -- so that two runs line up.
func streamsOf(t *testing.T, annot annotation.PDAnnotation) [][]any {
	t.Helper()
	appearance := annot.NormalAppearanceStream()
	if appearance == nil {
		t.Fatal("the annotation has no normal appearance")
	}
	streams := [][]any{tokensOf(t, appearance.ContentsForRandomAccess)}

	resources, isResources := appearance.Resources().(*pdmodel.PDResources)
	if !isResources {
		return streams
	}
	xobjects := resources.Dictionary().GetCOSDictionary(cos.XObject)
	if xobjects == nil {
		return streams
	}
	for _, name := range sortedNames(xobjects) {
		xobject := xobjects.GetCOSStream(name)
		if xobject == nil {
			continue
		}
		streams = append(streams, tokensOf(t, xobject.CreateView))
		// The pattern the form fills with lives in the form's own resources.
		formResources := xobject.Dictionary.GetCOSDictionary(cos.Resources)
		if formResources == nil {
			continue
		}
		patterns := formResources.GetCOSDictionary(cos.Pattern)
		if patterns == nil {
			continue
		}
		for _, patternName := range sortedNames(patterns) {
			if p := patterns.GetCOSStream(patternName); p != nil {
				streams = append(streams, tokensOf(t, p.CreateView))
			}
		}
	}
	return streams
}

// sortedNames is the dictionary's keys in a fixed order.
func sortedNames(dictionary *cos.Dictionary) []*cos.Name {
	names := dictionary.KeySet()
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j].Name() < names[j-1].Name(); j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
	return names
}

// tokensOf parses one content stream, the way the appearance test does.
func tokensOf(t *testing.T, contents func() (pdfio.RandomAccessRead, error)) []any {
	t.Helper()
	content, err := contents()
	if err != nil {
		t.Fatalf("reading a content stream: %v", err)
	}
	parser, err := pdfparser.NewStreamTokenParserSource(content)
	if err != nil {
		t.Fatalf("parsing a content stream: %v", err)
	}
	tokens, err := parser.Parse()
	if err != nil {
		t.Fatalf("parsing a content stream: %v", err)
	}
	return tokens
}

// compareTokens is checkAnnotationTokens' comparison, for one stream.
func compareTokens(t *testing.T, annotationIndex, stream int, wanted, got []any) {
	t.Helper()
	if len(got) != len(wanted) {
		t.Errorf("annotation %d stream %d: the port wrote %d tokens and PDFBox wrote %d",
			annotationIndex, stream, len(got), len(wanted))
		return
	}
	for i, want := range wanted {
		switch expected := want.(type) {
		case *operator.Operator:
			actual, isOperator := got[i].(*operator.Operator)
			if !isOperator || actual.Name() != expected.Name() {
				t.Errorf("annotation %d stream %d token %d: want the operator %s, got %v",
					annotationIndex, stream, i, expected.Name(), got[i])
			}
		case *cos.Float:
			actual, isFloat := got[i].(*cos.Float)
			if !isFloat ||
				math.Abs(float64(actual.FloatValue()-expected.FloatValue())) >= appearanceDelta {
				t.Errorf("annotation %d stream %d token %d: want %v, got %v",
					annotationIndex, stream, i, expected.FloatValue(), got[i])
			}
		case *cos.Integer:
			actual, isInteger := got[i].(*cos.Integer)
			if !isInteger || actual.IntValue() != expected.IntValue() {
				t.Errorf("annotation %d stream %d token %d: want %v, got %v",
					annotationIndex, stream, i, expected.IntValue(), got[i])
			}
		case *cos.Name:
			// The name is a resource name the writer chose, so only its shape
			// is compared -- both sides number from the same prefix.
			actual, isName := got[i].(*cos.Name)
			if !isName || actual.Name() != expected.Name() {
				t.Errorf("annotation %d stream %d token %d: want the name %s, got %v",
					annotationIndex, stream, i, expected.Name(), got[i])
			}
		}
	}
}
