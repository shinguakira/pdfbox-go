package text

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/markedcontent"
)

// PDFMarkedContentExtractor collects the marked content of a page, with the
// text of each sequence in it.
//
// Port of org.apache.pdfbox.text.PDFMarkedContentExtractor.
//
// The DrawObject processor Java also registers walks into an XObject, which is
// a slice this port has not reached; the xobject method is here because the
// extractor's callers use it. See migration/STATUS.md.
type PDFMarkedContentExtractor struct {
	*LegacyPDFStreamEngine

	suppressDuplicateOverlappingText bool
	markedContents                   []*markedcontent.PDMarkedContent
	currentMarkedContents            []*markedcontent.PDMarkedContent
	characterListMapping             map[string][]*TextPosition
}

// NewPDFMarkedContentExtractor returns an extractor with nothing collected yet.
func NewPDFMarkedContentExtractor() *PDFMarkedContentExtractor {
	e := &PDFMarkedContentExtractor{
		LegacyPDFStreamEngine:            NewLegacyPDFStreamEngine(),
		suppressDuplicateOverlappingText: true,
		characterListMapping:             map[string][]*TextPosition{},
	}
	e.SetOverrides(e)
	e.SetProcessTextPosition(e.ProcessTextPosition)
	return e
}

// IsSuppressDuplicateOverlappingText reports whether text drawn twice in one
// place is collected once.
func (e *PDFMarkedContentExtractor) IsSuppressDuplicateOverlappingText() bool {
	return e.suppressDuplicateOverlappingText
}

// SetSuppressDuplicateOverlappingText sets whether text drawn twice in one
// place is collected once.
func (e *PDFMarkedContentExtractor) SetSuppressDuplicateOverlappingText(value bool) {
	e.suppressDuplicateOverlappingText = value
}

// BeginMarkedContentSequence handles the BDC and BMC operators, opening a
// sequence.
func (e *PDFMarkedContentExtractor) BeginMarkedContentSequence(tag *cos.Name, properties *cos.Dictionary) {
	content := markedcontent.Create(tag, properties)
	if len(e.currentMarkedContents) == 0 {
		e.markedContents = append(e.markedContents, content)
	} else {
		currentMarkedContent := e.currentMarkedContents[len(e.currentMarkedContents)-1]
		currentMarkedContent.AddMarkedContent(content)
	}
	e.currentMarkedContents = append(e.currentMarkedContents, content)
}

// EndMarkedContentSequence handles the EMC operator, closing a sequence.
func (e *PDFMarkedContentExtractor) EndMarkedContentSequence() {
	if n := len(e.currentMarkedContents); n != 0 {
		e.currentMarkedContents = e.currentMarkedContents[:n-1]
	}
}

// MarkedContentPoint handles the MP and DP operators.
func (e *PDFMarkedContentExtractor) MarkedContentPoint(tag *cos.Name, properties *cos.Dictionary) {
	// Nothing happens here yet. If you know anything useful that should happen,
	// please tell us.
	e.LegacyPDFStreamEngine.MarkedContentPoint(tag, properties)
}

// XObject adds an XObject to the sequence being collected.
func (e *PDFMarkedContentExtractor) XObject(xobject any) {
	if n := len(e.currentMarkedContents); n != 0 {
		e.currentMarkedContents[n-1].AddXObject(xobject)
	}
}

// ProcessTextPosition adds the position to the sequence being collected,
// dropping it where it duplicates one already there.
func (e *PDFMarkedContentExtractor) ProcessTextPosition(text *TextPosition) error {
	showCharacter := true
	if e.suppressDuplicateOverlappingText {
		showCharacter = false
		textCharacter := text.Unicode()
		textX := text.X()
		textY := text.Y()
		sameTextCharacters := e.characterListMapping[textCharacter]

		// RDD - Here we compute the value that represents the end of the
		// rendered text. This value is used to determine whether subsequent
		// text rendered on the same line overwrites the current text.
		//
		// We subtract any positive padding to handle cases where extreme
		// amounts of padding are applied, then backed off (not sure why this is
		// done, but there are cases where the padding is on the order of 10x
		// the character width, and the TJ just backs up to compensate after
		// each character). Also, we subtract an amount to allow for kerning (a
		// percentage of the width of the last character).
		suppressCharacter := false
		tolerance := text.Width() / float32(utf16Length(textCharacter)) / 3.0
		for _, sameTextCharacter := range sameTextCharacters {
			charX := sameTextCharacter.X()
			charY := sameTextCharacter.Y()
			// only want to suppress
			if withinExtractor(charX, textX, tolerance) &&
				withinExtractor(charY, textY, tolerance) {
				suppressCharacter = true
				break
			}
		}
		if !suppressCharacter {
			e.characterListMapping[textCharacter] = append(sameTextCharacters, text)
			showCharacter = true
		}
	}

	if showCharacter && len(e.currentMarkedContents) != 0 {
		e.currentMarkedContents[len(e.currentMarkedContents)-1].AddText(text)
	}
	return nil
}

// withinExtractor reports whether two values are no further apart than the
// variance.
//
// Java's PDFMarkedContentExtractor has its own copy of within, with the two
// comparisons the other way round from PDFTextStripper's; both mean the same
// thing.
func withinExtractor(first, second, variance float32) bool {
	return second > first-variance && second < first+variance
}

// MarkedContents returns the sequences the extractor has collected.
func (e *PDFMarkedContentExtractor) MarkedContents() []*markedcontent.PDMarkedContent {
	return e.markedContents
}
