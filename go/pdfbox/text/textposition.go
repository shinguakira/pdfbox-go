// Package text extracts the text of a PDF.
//
// Port of org.apache.pdfbox.text.
package text

import (
	"math"
	"strings"
	"unicode"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"golang.org/x/text/unicode/norm"
)

// diacritics maps the non-decomposing diacritics onto their related combining
// character.
//
// These are values that the unicode spec claims are equivalent but are not
// mapped in the form NFKC normalization method. Determined by going through the
// Combining Diacritical Marks section of the Unicode spec and identifying which
// characters are not mapped to by the normalization.
var diacritics = map[rune]string{
	0x0060: "̀",
	0x02CB: "̀",
	0x0027: "́",
	0x02B9: "́",
	0x02CA: "́",
	0x005e: "̂",
	0x02C6: "̂",
	0x007E: "̃",
	0x02C9: "̄",
	0x00B0: "̊",
	0x02BA: "̋",
	0x02C7: "̌",
	0x02C8: "̍",
	0x0022: "̎",
	0x02BB: "̒",
	0x02BC: "̓",
	0x0486: "̓",
	0x055A: "̓",
	0x02BD: "̔",
	0x0485: "̔",
	0x0559: "̔",
	0x02D4: "̝",
	0x02D5: "̞",
	0x02D6: "̟",
	0x02D7: "̠",
	0x02B2: "̡",
	0x02CC: "̩",
	0x02B7: "̫",
	0x02CD: "̱",
	0x005F: "̲",
	0x204E: "͙",
}

// TextPosition is a string and the place on the page it was drawn.
//
// Port of org.apache.pdfbox.text.TextPosition.
type TextPosition struct {
	// textMatrix is the text matrix for the start of the text object,
	// coordinates are in display units and have not been adjusted
	textMatrix *util.Matrix

	// ending X and Y coordinates in display units
	endX float32
	endY float32

	maxHeight  float32 // maximum height of text, in display units
	rotation   int     // 0, 90, 180, 270 degrees of page rotation
	x          float32
	y          float32
	pageHeight float32
	pageWidth  float32

	widthOfSpace float32 // width of a space, in display units

	charCodes  []int // internal PDF character codes
	font       font.PDFont
	fontSize   float32
	fontSizePt int

	// mutable
	widths     []float32
	unicodeStr string
	direction  float32
}

// NewTextPosition returns a text position.
func NewTextPosition(pageRotation int, pageWidth, pageHeight float32, textMatrix *util.Matrix,
	endX, endY, maxHeight, individualWidth, spaceWidth float32, unicodeStr string,
	charCodes []int, f font.PDFont, fontSize float32, fontSizeInPt int) *TextPosition {
	t := &TextPosition{
		textMatrix:   textMatrix,
		endX:         endX,
		endY:         endY,
		rotation:     pageRotation,
		maxHeight:    maxHeight,
		pageHeight:   pageHeight,
		pageWidth:    pageWidth,
		widths:       []float32{individualWidth},
		widthOfSpace: spaceWidth,
		unicodeStr:   unicodeStr,
		charCodes:    charCodes,
		font:         f,
		fontSize:     fontSize,
		fontSizePt:   fontSizeInPt,
		direction:    -1,
	}
	t.x = t.xRot(float32(t.rotation))
	if t.rotation == 0 || t.rotation == 180 {
		t.y = t.pageHeight - t.yLowerLeftRot(float32(t.rotation))
	} else {
		t.y = t.pageWidth - t.yLowerLeftRot(float32(t.rotation))
	}
	return t
}

// Unicode returns the text this position holds.
func (t *TextPosition) Unicode() string { return t.unicodeStr }

// setUnicode sets the text this position holds.
func (t *TextPosition) setUnicode(unicodeStr string) { t.unicodeStr = unicodeStr }

// VisuallyOrderedUnicode returns the text in the order it is drawn, which for a
// right to left run is the reverse of the order it is stored in.
func (t *TextPosition) VisuallyOrderedUnicode() string {
	text := t.Unicode()
	runes := []rune(text)
	for index, codePoint := range runes {
		// Even if the directionality is right to left, still there is no need
		// to reverse a single code-point
		if isRightToLeft(codePoint) && (index != 0 || index+1 < len(runes)) {
			reversed := make([]rune, len(runes))
			for i, r := range runes {
				reversed[len(runes)-1-i] = r
			}
			return string(reversed)
		}
	}
	return text
}

// isRightToLeft reports whether the code point is set right to left, which is
// Java's DIRECTIONALITY_RIGHT_TO_LEFT and DIRECTIONALITY_RIGHT_TO_LEFT_ARABIC.
func isRightToLeft(codePoint rune) bool {
	return unicode.In(codePoint, unicode.Hebrew, unicode.Arabic, unicode.Syriac,
		unicode.Thaana, unicode.Nko, unicode.Samaritan, unicode.Mandaic)
}

// CharacterCodes returns the character codes the text was drawn from.
func (t *TextPosition) CharacterCodes() []int { return t.charCodes }

// TextMatrix returns the matrix the text was drawn with.
func (t *TextPosition) TextMatrix() *util.Matrix { return t.textMatrix }

// Dir returns which way the text runs, in degrees: 0, 90, 180 or 270.
func (t *TextPosition) Dir() float32 {
	if t.direction < 0 {
		a := t.textMatrix.ScaleY()
		b := t.textMatrix.ShearY()
		c := t.textMatrix.ShearX()
		d := t.textMatrix.ScaleX()

		switch {
		// 12 0   left to right
		// 0 12
		case a > 0 && abs32(b) < d && abs32(c) < a && d > 0:
			t.direction = 0
		// -12 0   right to left (upside down)
		// 0 -12
		case a < 0 && abs32(b) < abs32(d) && abs32(c) < abs32(a) && d < 0:
			t.direction = 180
		// 0  12    up
		// -12 0
		case abs32(a) < abs32(c) && b > 0 && c < 0 && abs32(d) < b:
			t.direction = 90
		// 0  -12   down
		// 12 0
		case abs32(a) < c && b < 0 && c > 0 && abs32(d) < abs32(b):
			t.direction = 270
		default:
			t.direction = 0
		}
	}
	return t.direction
}

func abs32(v float32) float32 { return float32(math.Abs(float64(v))) }

// xRot returns the x of the text as the given rotation sees it.
func (t *TextPosition) xRot(rotation float32) float32 {
	switch rotation {
	case 0:
		return t.textMatrix.TranslateX()
	case 90:
		return t.textMatrix.TranslateY()
	case 180:
		return t.pageWidth - t.textMatrix.TranslateX()
	case 270:
		return t.pageHeight - t.textMatrix.TranslateY()
	}
	return 0
}

// X returns the x of the text, as the page rotation sees it.
func (t *TextPosition) X() float32 { return t.x }

// XDirAdj returns the x of the text, as the direction of the text sees it.
func (t *TextPosition) XDirAdj() float32 { return t.xRot(t.Dir()) }

// yLowerLeftRot returns the y of the text as the given rotation sees it,
// measured from the bottom of the page.
func (t *TextPosition) yLowerLeftRot(rotation float32) float32 {
	switch rotation {
	case 0:
		return t.textMatrix.TranslateY()
	case 90:
		return t.pageWidth - t.textMatrix.TranslateX()
	case 180:
		return t.pageHeight - t.textMatrix.TranslateY()
	case 270:
		return t.textMatrix.TranslateX()
	}
	return 0
}

// Y returns the y of the text, measured from the top of the page.
func (t *TextPosition) Y() float32 { return t.y }

// YDirAdj returns the y of the text, as the direction of the text sees it.
func (t *TextPosition) YDirAdj() float32 {
	dir := t.Dir()
	// some PDFBox code assumes that the 0,0 point is in upper left, not lower
	// left
	if dir == 0 || dir == 180 {
		return t.pageHeight - t.yLowerLeftRot(dir)
	}
	return t.pageWidth - t.yLowerLeftRot(dir)
}

// widthRot returns the width of the text as the given rotation sees it.
func (t *TextPosition) widthRot(rotation float32) float32 {
	if rotation == 90 || rotation == 270 {
		return abs32(t.endY - t.textMatrix.TranslateY())
	}
	return abs32(t.endX - t.textMatrix.TranslateX())
}

// Width returns the width of the text, as the page rotation sees it.
func (t *TextPosition) Width() float32 { return t.widthRot(float32(t.rotation)) }

// WidthDirAdj returns the width of the text, as the direction of the text sees
// it.
func (t *TextPosition) WidthDirAdj() float32 { return t.widthRot(t.Dir()) }

// Height returns the height of the text.
func (t *TextPosition) Height() float32 { return t.maxHeight }

// HeightDir returns the height of the text.
//
// This is not really a rotation-dependent calculation, but this is defined for
// symmetry.
func (t *TextPosition) HeightDir() float32 { return t.maxHeight }

// FontSize returns the size the text was set at, in text space.
func (t *TextPosition) FontSize() float32 { return t.fontSize }

// FontSizeInPt returns the size the text was set at, in points.
func (t *TextPosition) FontSizeInPt() float32 { return float32(t.fontSizePt) }

// Font returns the font the text was set in.
func (t *TextPosition) Font() font.PDFont { return t.font }

// WidthOfSpace returns how wide a space is in this font, in display units.
func (t *TextPosition) WidthOfSpace() float32 { return t.widthOfSpace }

// XScale returns how far the text is scaled horizontally.
func (t *TextPosition) XScale() float32 { return t.textMatrix.ScalingFactorX() }

// YScale returns how far the text is scaled vertically.
func (t *TextPosition) YScale() float32 { return t.textMatrix.ScalingFactorY() }

// IndividualWidths returns the width of each character of the text.
func (t *TextPosition) IndividualWidths() []float32 { return t.widths }

// Contains reports whether the given text overlaps this one enough to be
// treated as sitting on top of it.
func (t *TextPosition) Contains(tp2 *TextPosition) bool {
	thisXstart := float64(t.XDirAdj())
	thisWidth := float64(t.WidthDirAdj())
	thisXend := thisXstart + thisWidth

	tp2Xstart := float64(tp2.XDirAdj())
	tp2Xend := tp2Xstart + float64(tp2.WidthDirAdj())

	// no X overlap at all so return as soon as possible
	if tp2Xend <= thisXstart || tp2Xstart >= thisXend {
		return false
	}

	// no Y overlap at all so return as soon as possible. Note: 0.0 is in the
	// upper left and y-coordinate is top of TextPosition
	thisYstart := float64(t.YDirAdj())
	tp2Ystart := float64(tp2.YDirAdj())
	if tp2Ystart+float64(tp2.HeightDir()) < thisYstart ||
		tp2Ystart > thisYstart+float64(t.HeightDir()) {
		return false
	}

	// we're going to calculate the percentage of overlap, if its less than a
	// 15% x-coordinate overlap then we'll return false because its negligible,
	// .15 was determined by trial and error in the regression test files
	if tp2Xstart > thisXstart && tp2Xend > thisXend {
		overlap := thisXend - tp2Xstart
		overlapPercent := overlap / thisWidth
		return overlapPercent > .15
	}
	if tp2Xstart < thisXstart && tp2Xend < thisXend {
		overlap := tp2Xend - thisXstart
		overlapPercent := overlap / thisWidth
		return overlapPercent > .15
	}
	return true
}

// CompletelyContains reports whether the given text sits entirely inside this
// one.
func (t *TextPosition) CompletelyContains(tp2 *TextPosition) bool {
	//  Note: (0, 0) is in the upper left and y-coordinate is top of TextPosition
	//      +---thisTop------------+
	//      |    +--tp2Top---+     |
	//      |    |           |     |
	//  thisLeft |       tp2Right  |
	//      | tp2Left        | thisRight
	//      |    |           |     |
	//      |    +-tp2Bottom-+     |
	//      +---------thisBottom---+
	thisLeft := t.XDirAdj()
	thisWidth := t.WidthDirAdj()
	thisRight := thisLeft + thisWidth
	tp2Left := tp2.XDirAdj()
	tp2Width := tp2.WidthDirAdj()
	tp2Right := tp2Left + tp2Width
	if thisLeft > tp2Left || tp2Right > thisRight {
		return false
	}

	thisTop := t.YDirAdj()
	thisHeight := t.HeightDir()
	thisBottom := thisTop + thisHeight
	tp2Top := tp2.YDirAdj()
	tp2Height := tp2.HeightDir()
	tp2Bottom := tp2Top + tp2Height
	return thisTop <= tp2Top && tp2Bottom <= thisBottom
}

// MergeDiacritic folds the given diacritic into whichever character of this
// text it sits over.
func (t *TextPosition) MergeDiacritic(diacritic *TextPosition) {
	if utf16Length(diacritic.Unicode()) > 1 {
		return
	}

	diacXStart := diacritic.XDirAdj()
	diacXEnd := diacXStart + diacritic.widths[0]

	currCharXStart := t.XDirAdj()

	// Java indexes the string by UTF-16 code unit; the port indexes runes, and
	// the surrogate handling insertDiacritic does falls out of that.
	runes := []rune(t.unicodeStr)
	strLen := len(runes)
	wasAdded := false

	for i := 0; i < strLen && !wasAdded; i++ {
		if i >= len(t.widths) {
			// a diacritic on a ligature is not supported yet and is ignored
			// (PDFBOX-2831)
			break
		}
		currCharXEnd := currCharXStart + t.widths[i]

		// this is the case where there is an overlap of the diacritic character
		// with the current character and the previous character. If no previous
		// character, just append the diacritic after the current one
		switch {
		case diacXStart < currCharXStart && diacXEnd <= currCharXEnd:
			if i == 0 {
				t.insertDiacritic(i, diacritic)
			} else {
				distanceOverlapping1 := diacXEnd - currCharXStart
				percentage1 := distanceOverlapping1 / t.widths[i]

				distanceOverlapping2 := currCharXStart - diacXStart
				percentage2 := distanceOverlapping2 / t.widths[i-1]

				if percentage1 >= percentage2 {
					t.insertDiacritic(i, diacritic)
				} else {
					t.insertDiacritic(i-1, diacritic)
				}
			}
			wasAdded = true
		// diacritic completely covers this character and therefore we assume
		// that this is the character the diacritic belongs to
		case diacXStart < currCharXStart:
			t.insertDiacritic(i, diacritic)
			wasAdded = true
		// otherwise, The diacritic modifies this character because its
		// completely contained by the character width
		case diacXEnd <= currCharXEnd:
			t.insertDiacritic(i, diacritic)
			wasAdded = true
		// last character in the TextPosition so we add diacritic to the end
		case i == strLen-1:
			t.insertDiacritic(i, diacritic)
			wasAdded = true
		}

		// couldn't find anything useful so we go to the next character in the
		// TextPosition
		currCharXStart += t.widths[i]
	}
}

// insertDiacritic puts the diacritic after the given character.
func (t *TextPosition) insertDiacritic(i int, diacritic *TextPosition) {
	runes := []rune(t.unicodeStr)

	widths2 := make([]float32, len(t.widths)+1)
	copy(widths2, t.widths[:i])
	// First we add a zero-width entry for the diacritic in the widths array
	widths2[i] = t.widths[i]
	widths2[i+1] = 0
	copy(widths2[i+2:], t.widths[i+1:])

	var sb strings.Builder
	sb.WriteString(string(runes[:i]))
	// Unicode combining diacritics always go after the base character,
	// regardless of whether the string is in presentation order or logical
	// order
	sb.WriteRune(runes[i])
	sb.WriteString(combineDiacritic(diacritic.Unicode()))
	// get the rest of the string
	sb.WriteString(string(runes[i+1:]))

	t.unicodeStr = sb.String()
	t.widths = widths2
}

// combineDiacritic returns the combining form of the given diacritic.
func combineDiacritic(str string) string {
	// Unicode contains special combining forms of the diacritic characters
	// which we want to use
	codePoint := []rune(str)[0]

	// convert the characters not defined in the Unicode spec
	if combining, ok := diacritics[codePoint]; ok {
		return combining
	}
	return strings.TrimSpace(norm.NFKC.String(str))
}

// IsDiacritic reports whether this text is a diacritic, which belongs on top of
// the character before it rather than beside it.
func (t *TextPosition) IsDiacritic() bool {
	text := t.Unicode()
	runes := []rune(text)
	if utf16Length(text) != 1 {
		return false
	}
	if text == "ー" {
		// PDFBOX-3833: ー is not a real diacritic like ¨ or ˆ, it just changes
		// the pronunciation of the previous sound, and is printed after the
		// previous glyph
		// http://www.japanesewithanime.com/2017/04/prolonged-sound-mark.html
		// Ignoring it as diacritic avoids trouble if it slightly overlaps with
		// the next glyph.
		return false
	}
	// Java asks for NON_SPACING_MARK, MODIFIER_SYMBOL or MODIFIER_LETTER
	c := runes[0]
	return unicode.Is(unicode.Mn, c) || unicode.Is(unicode.Sk, c) || unicode.Is(unicode.Lm, c)
}

// String returns the text this position holds.
func (t *TextPosition) String() string { return t.Unicode() }

// EndX returns where the text ends horizontally, in display units.
func (t *TextPosition) EndX() float32 { return t.endX }

// EndY returns where the text ends vertically, in display units.
func (t *TextPosition) EndY() float32 { return t.endY }

// Rotation returns how far the page is rotated.
func (t *TextPosition) Rotation() int { return t.rotation }

// PageHeight returns how tall the page is.
func (t *TextPosition) PageHeight() float32 { return t.pageHeight }

// PageWidth returns how wide the page is.
func (t *TextPosition) PageWidth() float32 { return t.pageWidth }

// Equals reports whether two text positions were drawn the same way in the same
// place.
//
// Port of equals. The mutable fields are left out on purpose (PDFBOX-4701).
func (t *TextPosition) Equals(that *TextPosition) bool {
	if t == that {
		return true
	}
	if that == nil {
		return false
	}
	if t.endX != that.endX || t.endY != that.endY || t.maxHeight != that.maxHeight ||
		t.rotation != that.rotation || t.x != that.x || t.y != that.y ||
		t.pageHeight != that.pageHeight || t.pageWidth != that.pageWidth ||
		t.widthOfSpace != that.widthOfSpace || t.fontSize != that.fontSize ||
		t.fontSizePt != that.fontSizePt {
		return false
	}
	if !matrixEquals(t.textMatrix, that.textMatrix) {
		return false
	}
	if len(t.charCodes) != len(that.charCodes) {
		return false
	}
	for i := range t.charCodes {
		if t.charCodes[i] != that.charCodes[i] {
			return false
		}
	}
	return t.font == that.font
}

// matrixEquals reports whether two matrices hold the same values, either both
// being absent.
func matrixEquals(a, b *util.Matrix) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Values() == b.Values()
}

// CompareTextPositions orders two text positions the way the text stripper
// reads them: down the page, then across it.
//
// Port of org.apache.pdfbox.text.TextPositionComparator. It is not a total
// order -- the tolerance in the middle is not transitive -- which is why the
// stripper sorts with IterativeMergeSort.
func CompareTextPositions(pos1, pos2 *TextPosition) int {
	// only compare text that is in the same direction
	if cmp1 := compareFloat32(pos1.Dir(), pos2.Dir()); cmp1 != 0 {
		return cmp1
	}

	// get the text direction adjusted coordinates
	x1 := pos1.XDirAdj()
	x2 := pos2.XDirAdj()

	pos1YBottom := pos1.YDirAdj()
	pos2YBottom := pos2.YDirAdj()
	// note that the coordinates have been adjusted so 0,0 is in upper left
	pos1YTop := pos1YBottom - pos1.HeightDir()
	pos2YTop := pos2YBottom - pos2.HeightDir()

	yDifference := abs32(pos1YBottom - pos2YBottom)
	// we will do a simple tolerance comparison
	if yDifference < .1 ||
		pos2YBottom >= pos1YTop && pos2YBottom <= pos1YBottom ||
		pos1YBottom >= pos2YTop && pos1YBottom <= pos2YBottom {
		return compareFloat32(x1, x2)
	}
	if pos1YBottom < pos2YBottom {
		return -1
	}
	return 1
}

// compareFloat32 orders two floats, standing in for Java's Float.compare.
func compareFloat32(a, b float32) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// utf16Length returns how many UTF-16 code units the string takes, which is
// what Java's String.length() counts.
//
// Every place the Java compares a length against a number is comparing code
// units, so a character outside the basic plane counts twice there and once in
// a Go range loop. The port counts the way Java does wherever the number is
// then compared.
func utf16Length(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			n += 2
		} else {
			n++
		}
	}
	return n
}
