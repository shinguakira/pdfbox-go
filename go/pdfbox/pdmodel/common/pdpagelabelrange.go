package common

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// The page label dictionary keys, PDF32000-1:2008 Section 12.4.2, Table 159.
var (
	keyStart  = cos.St
	keyPrefix = cos.P
	keyStyle  = cos.S
)

// The /S entry values, PDF32000-1:2008 Section 12.4.2, Table 159.
const (
	// StyleDecimal numbers pages with decimal arabic numerals.
	StyleDecimal = "D"
	// StyleRomanUpper numbers pages with upper case roman numerals.
	StyleRomanUpper = "R"
	// StyleRomanLower numbers pages with lower case roman numerals.
	StyleRomanLower = "r"
	// StyleLettersUpper numbers pages with upper case letters.
	StyleLettersUpper = "A"
	// StyleLettersLower numbers pages with lower case letters.
	StyleLettersLower = "a"
)

// PDPageLabelRange is a page label range of a document.
//
// Port of org.apache.pdfbox.pdmodel.common.PDPageLabelRange.
type PDPageLabelRange struct {
	root *cos.Dictionary
}

var _ COSObjectable = (*PDPageLabelRange)(nil)

// NewPDPageLabelRange creates a new empty page label range.
func NewPDPageLabelRange() *PDPageLabelRange {
	return NewPDPageLabelRangeOf(cos.NewDictionary())
}

// NewPDPageLabelRangeOf creates a page label range over the given dictionary.
func NewPDPageLabelRangeOf(dict *cos.Dictionary) *PDPageLabelRange {
	return &PDPageLabelRange{root: dict}
}

// COSObject returns the dictionary.
func (r *PDPageLabelRange) COSObject() cos.Base { return r.root }

// Dictionary returns the dictionary, typed.
func (r *PDPageLabelRange) Dictionary() *cos.Dictionary { return r.root }

// Style returns the numbering style of this range, or "" where it has none.
func (r *PDPageLabelRange) Style() string {
	return r.root.GetNameAsString(keyStyle, "")
}

// SetStyle sets the numbering style of this range. The empty string is Java's
// null, which removes the entry.
func (r *PDPageLabelRange) SetStyle(style string) {
	if style != "" {
		r.root.SetName(keyStyle, style)
	} else {
		r.root.RemoveItem(keyStyle)
	}
}

// Start returns the page number the range starts at, which defaults to 1.
func (r *PDPageLabelRange) Start() int {
	return r.root.GetIntDefault(keyStart, 1)
}

// SetStart sets the page number the range starts at.
//
// Java throws IllegalArgumentException for a value that is not positive, which
// is unchecked, so the port panics.
func (r *PDPageLabelRange) SetStart(start int) {
	if start <= 0 {
		panic("The page numbering start value must be a positive integer")
	}
	r.root.SetInt(keyStart, start)
}

// Prefix returns the label prefix of this range, or "" where it has none.
func (r *PDPageLabelRange) Prefix() string {
	return r.root.GetString(keyPrefix, "")
}

// SetPrefix sets the label prefix of this range. The empty string is Java's
// null, which removes the entry.
func (r *PDPageLabelRange) SetPrefix(prefix string) {
	if prefix != "" {
		r.root.SetString(keyPrefix, prefix)
	} else {
		r.root.RemoveItem(keyPrefix)
	}
}
