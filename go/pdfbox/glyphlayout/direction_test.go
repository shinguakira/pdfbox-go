package glyphlayout_test

// A mark has to sit over its letter whichever way the text runs.
//
// The reference comparison covers the left-to-right half of this thoroughly --
// 40 lines of DIN 91379 -- and cannot cover the right-to-left half, because the
// Arabic it would compare against is shaped by the platform and this port has
// no Arabic shaper. So the property is asserted directly instead: the layout is
// driven through the interface a content stream implements, and the ink
// positions are worked out from what it was asked to write.
//
// This is what `resolveAttachments` is for. A mark's anchor is a distance from
// the origin of the letter it hangs off, and in a right-to-left run the letter
// is drawn *after* the mark, so the distance has to be measured the other way.
// Getting it wrong put the mark a letter's width away, and no test that only
// checked "something was written" would notice.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/glyphlayout"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// recordingStream is a content stream that keeps what it was written.
//
// It implements pdmodel.ContentStreamForGlyphLayout, which is the whole of what
// a glyph layout processor writes through.
type recordingStream struct {
	// items holds an int for a glyph id and a float32 for an adjustment, in
	// the order they were written.
	items []any
	rises []float32
}

var _ pdmodel.ContentStreamForGlyphLayout = (*recordingStream)(nil)

func (s *recordingStream) ShowGlyphsWithPositioning(ga *pdmodel.GlyphsAndPositions) error {
	for _, element := range ga.Array() {
		switch value := element.(type) {
		case *pdmodel.GlyphSubList:
			for _, glyph := range value.IntArray() {
				s.items = append(s.items, glyph)
			}
		case float32:
			s.items = append(s.items, value)
		}
	}
	return nil
}

func (s *recordingStream) ShowGlyphCodes(glyphCodes []int) error {
	for _, glyph := range glyphCodes {
		s.items = append(s.items, glyph)
	}
	return nil
}

func (s *recordingStream) SetTextRise(rise float32) error {
	s.rises = append(s.rises, rise)
	return nil
}

// TestMarkSitsOverItsLetterInBothDirections lays out one letter and one mark,
// once in a left-to-right script and once in a right-to-left one, and checks
// the mark's ink lands on the letter both times.
func TestMarkSitsOverItsLetterInBothDirections(t *testing.T) {
	for _, run := range []struct {
		name, file, text string
		rightToLeft      bool
	}{
		// C and a combining ogonek, which is the DIN 91379 page's `C̨`.
		{"latin", "Arimo-Regular.ttf", "C̨", false},
		// Arabic lam and a fatha, the vowel mark that sits over it.
		{"arabic", "NotoSansArabic-Regular.ttf", "لَ", true},
	} {
		t.Run(run.name, func(t *testing.T) {
			document := pdmodel.NewPDDocument()
			defer document.Close()
			embedded := openLayoutFont(t, document, layoutFonts+run.file)

			stream := &recordingStream{}
			processor := glyphlayout.NewProcessor()
			if err := processor.ShowText(stream, embedded, 12, run.text); err != nil {
				t.Fatalf("ShowText: %v", err)
			}

			glyphs, ink := inkPositions(t, stream, embedded)
			if len(glyphs) != 2 {
				t.Fatalf("the run came out as %d glyphs, want the letter and the mark",
					len(glyphs))
			}

			hmtx, err := embedded.TrueTypeFont().HorizontalMetrics()
			if err != nil {
				t.Fatalf("HorizontalMetrics: %v", err)
			}
			// Which of the two is the letter: the one with an advance. A
			// combining mark has none, which is what makes it combining.
			letter, mark := 0, 1
			if hmtx.AdvanceWidth(glyphs[0]) == 0 {
				letter, mark = 1, 0
			}
			if hmtx.AdvanceWidth(glyphs[letter]) == 0 {
				t.Fatal("neither glyph has an advance, so neither is the letter")
			}
			// The mark is drawn first in a right-to-left run and second in a
			// left-to-right one, which is the whole point of the case.
			if drawnFirstIsMark := mark == 0; drawnFirstIsMark != run.rightToLeft {
				t.Errorf("the mark is glyph %d of the run and the run %s",
					mark, directionName(run.rightToLeft))
			}

			// The mark has to land inside the letter, with a tenth of its width
			// of slack for a mark that hangs over the edge. Both sit close to
			// the middle: the ogonek 900 units across a C that is 1479 wide,
			// the fatha 396 across a lam that is 695.
			//
			// The first of those two numbers is the Java's. The AWT reference
			// writes `C̨` as an adjustment of 282.71484, which in Arimo's 2048
			// units to the em is 579 back from the pen, and the C is 1479 wide:
			// 1479 - 579 = 900. The second has no Java counterpart, because the
			// Arabic AWT laid out is shaped and this port's is not; what the
			// case asserts there is the property, and the defect it guards
			// against moves the mark by a whole advance.
			width := hmtx.AdvanceWidth(glyphs[letter])
			slack := width / 10
			if ink[mark] < ink[letter]-slack || ink[mark] > ink[letter]+width+slack {
				t.Errorf("the letter is drawn at %d and is %d wide, and the mark is "+
					"at %d: it is not on the letter", ink[letter], width, ink[mark])
			}
		})
	}
}

// inkPositions answers each glyph and where its ink lands, in font design
// units, worked out the way a viewer works it out: the pen moves by the
// advance the font declares, and every number in a TJ array moves it back.
func inkPositions(t *testing.T, stream *recordingStream,
	embedded *font.PDType0Font) ([]int, []int) {
	t.Helper()
	header, err := embedded.TrueTypeFont().Header()
	if err != nil {
		t.Fatalf("Header: %v", err)
	}
	hmtx, err := embedded.TrueTypeFont().HorizontalMetrics()
	if err != nil {
		t.Fatalf("HorizontalMetrics: %v", err)
	}
	unitsPerEm := header.UnitsPerEm()

	var glyphs, ink []int
	pen := 0
	for _, item := range stream.items {
		switch value := item.(type) {
		case float32:
			// A TJ number is in thousandths of the font size and is
			// subtracted from the position.
			pen -= int(float64(value) * float64(unitsPerEm) / 1000)
		case int:
			glyphs = append(glyphs, value)
			ink = append(ink, pen)
			pen += hmtx.AdvanceWidth(value)
		}
	}
	return glyphs, ink
}

// directionName is for a failure message.
func directionName(rightToLeft bool) string {
	if rightToLeft {
		return "runs right to left"
	}
	return "runs left to right"
}
