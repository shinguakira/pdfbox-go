package pdfparser_test

// JAVA-BUGS 10: a truncated inline image loses its last two bytes.

import (
	"bytes"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
)

// TestTruncatedInlineImageKeepsItsLastTwoBytes is the defect.
//
// The two bytes held in `lastByte` and `currentByte` are written only on the
// next turn of the loop. When the source runs out -- an inline image with no
// closing EI -- the loop ends on the end-of-input check and both are dropped.
//
// The expected value is the arithmetic: four bytes went in after the ID, so
// four bytes come out. What a truncated stream should answer is a judgement,
// but silently answering two bytes fewer than are on disk is not one of the
// choices.
func TestTruncatedInlineImageKeepsItsLastTwoBytes(t *testing.T) {
	// `ID ` then four data bytes and nothing else: no EI at all.
	content := []byte("BI /W 2 /H 2 ID \x01\x02\x03\x04")

	parser, err := pdfparser.NewStreamTokenParser(content)
	if err != nil {
		t.Fatalf("NewStreamTokenParser: %v", err)
	}
	tokens, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	var data []byte
	for _, token := range tokens {
		if op, isOperator := token.(*operator.Operator); isOperator {
			if len(op.ImageData()) > 0 {
				data = op.ImageData()
			}
		}
	}
	want := []byte{1, 2, 3, 4}
	if !bytes.Equal(data, want) {
		t.Errorf("the image data is % x, want % x: the two bytes the loop was "+
			"holding when the input ran out were dropped", data, want)
	}
}
