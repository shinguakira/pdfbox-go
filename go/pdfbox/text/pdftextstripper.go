package text

import (
	"bufio"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/markedcontent"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/resources"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"golang.org/x/text/unicode/norm"
)

// lineSeparatorDefault is what a line of text ends with.
//
// Java reads System.lineSeparator, which is "\r\n" on Windows; the port uses
// "\n" outright, so that the text a document yields does not depend on where it
// was extracted. A caller wanting the platform separator sets it.
const lineSeparatorDefault = "\n"

// The values the per-line running state is reset to.
const (
	endOfLastTextXResetValue           = -1
	maxYForLineResetValue              = -math.MaxFloat32
	expectedStartOfNextWordXResetValue = -math.MaxFloat32
	maxHeightForLineResetValue         = -1
	minYTopForLineResetValue           = math.MaxFloat32
	lastWordSpacingResetValue          = -1
)

// listItemExpressions are the shapes a list marker takes, which the paragraph
// detection looks for.
var listItemExpressions = []string{
	`\.`, `\d+\.`, `\[\d+\]`, `\d+\)`, `[A-Z]\.`, `[a-z]\.`, `[A-Z]\)`, `[a-z]\)`,
	`[IVXL]+\.`, `[ivxl]+\.`,
}

// PDFTextStripper walks the pages of a document and writes out their text.
//
// Port of org.apache.pdfbox.text.PDFTextStripper.
//
// The document-level entry points -- getText(PDDocument) and writeText -- are
// not here: PDDocument is the file-opening path, which this slice does not
// carry. ProcessPages takes the page tree instead, which is what processPages
// walks. Bookmarks go with PDDocument, since a bookmark is resolved against the
// document catalogue. See migration/STATUS.md.
type PDFTextStripper struct {
	*LegacyPDFStreamEngine

	lineSeparator  string
	wordSeparator  string
	paragraphStart string
	paragraphEnd   string
	pageStart      string
	pageEnd        string
	articleStart   string
	articleEnd     string

	currentPageNo int
	startPage     int
	endPage       int

	suppressDuplicateOverlappingText bool
	shouldSeparateByBeads            bool
	sortByPosition                   bool
	addMoreFormatting                bool
	ignoreContentStreamSpaceGlyphs   bool

	indentThreshold float32
	dropThreshold   float32

	spacingTolerance     float32
	averageCharTolerance float32

	beadRectangles []*common.PDRectangle

	// currentMarkedContents is a stack, so we don't get confused if another BDC
	// within "/ActualText... BDC" block
	currentMarkedContents []*markedcontent.PDMarkedContent

	// to replace the unicode of the first TextPosition and empty the others
	firstActualTextPosition bool
	actualText              string
	hasActualText           bool

	charactersByArticle  [][]*TextPosition
	characterListMapping map[string]map[float32]map[float32]bool

	output io.Writer

	inParagraph bool

	listOfPatterns []*regexp.Regexp
}

// The defaults Java gives the two paragraph thresholds.
const (
	defaultIndentThreshold = 2.0
	defaultDropThreshold   = 2.5
)

// NewPDFTextStripper returns a stripper with the defaults Java gives it.
func NewPDFTextStripper() *PDFTextStripper {
	s := &PDFTextStripper{
		LegacyPDFStreamEngine: NewLegacyPDFStreamEngine(),

		lineSeparator:  lineSeparatorDefault,
		wordSeparator:  " ",
		paragraphStart: "",
		paragraphEnd:   "",
		pageStart:      "",
		pageEnd:        lineSeparatorDefault,
		articleStart:   "",
		articleEnd:     "",

		currentPageNo: 1,
		startPage:     1,
		endPage:       math.MaxInt32,

		suppressDuplicateOverlappingText: true,
		shouldSeparateByBeads:            true,

		indentThreshold: defaultIndentThreshold,
		dropThreshold:   defaultDropThreshold,

		spacingTolerance:     .5,
		averageCharTolerance: .3,

		characterListMapping: map[string]map[float32]map[float32]bool{},
	}
	s.SetOverrides(s)
	s.SetProcessTextPosition(s.ProcessTextPosition)
	return s
}

// Output returns where the text is written.
func (s *PDFTextStripper) Output() io.Writer { return s.output }

// SetOutput sets where the text is written.
func (s *PDFTextStripper) SetOutput(output io.Writer) { s.output = output }

// GetTextOfPages returns the text of the given pages.
//
// This stands in for getText(PDDocument) while the loader is unported: it is
// the same walk, given the page tree rather than the document.
func (s *PDFTextStripper) GetTextOfPages(pages *pdmodel.PDPageTree) (string, error) {
	var out strings.Builder
	s.output = &out
	if err := s.ProcessPages(pages); err != nil {
		return "", err
	}
	return out.String(), nil
}

// ProcessPages walks the pages, writing out the text of each.
//
// Port of processPages. The bookmark range Java also honours needs PDDocument.
func (s *PDFTextStripper) ProcessPages(pages *pdmodel.PDPageTree) error {
	for i := 0; i < pages.Count(); i++ {
		page := pages.Get(i)
		if page == nil {
			continue
		}
		if err := s.ProcessPage(page); err != nil {
			return err
		}
		s.currentPageNo++
	}
	return nil
}

// ProcessPage walks one page, writing out its text.
func (s *PDFTextStripper) ProcessPage(page *pdmodel.PDPage) error {
	if s.currentPageNo < s.startPage || s.currentPageNo > s.endPage {
		return nil
	}

	if err := s.StartPage(page); err != nil {
		return err
	}

	numberOfArticleSections := 1
	if s.shouldSeparateByBeads {
		s.fillBeadRectangles(page)
		numberOfArticleSections += len(s.beadRectangles) * 2
	}
	originalSize := len(s.charactersByArticle)
	lastIndex := max(numberOfArticleSections, originalSize)
	for i := 0; i < lastIndex; i++ {
		if i < originalSize {
			s.charactersByArticle[i] = s.charactersByArticle[i][:0]
		} else if numberOfArticleSections < originalSize {
			//TODO Looks like decrement (--i) needed because next value will be
			// ignored. This segment is never reached in tests?!
			s.charactersByArticle = append(s.charactersByArticle[:i], s.charactersByArticle[i+1:]...)
		} else {
			s.charactersByArticle = append(s.charactersByArticle, nil)
		}
	}
	s.characterListMapping = map[string]map[float32]map[float32]bool{}

	if err := s.LegacyPDFStreamEngine.ProcessPage(page); err != nil {
		return err
	}
	if err := s.WritePage(); err != nil {
		return err
	}
	if err := s.EndPage(page); err != nil {
		return err
	}
	// Java calls page.removePageResourceFromCache() here; the port's PDPage has
	// no resource cache yet. See migration/STATUS.md.
	return nil
}

// fillBeadRectangles works out where the article beads of the page sit, in the
// coordinates the glyphs are in.
func (s *PDFTextStripper) fillBeadRectangles(page *pdmodel.PDPage) {
	// The thread beads of a page need PDThreadBead, which is a slice this port
	// has not reached; without them every glyph falls into one article, which
	// is what a page with no beads does anyway. See migration/STATUS.md.
	s.beadRectangles = nil
}

// StartArticle writes whatever opens an article.
func (s *PDFTextStripper) StartArticle() error { return s.StartArticleLTR(true) }

// StartArticleLTR writes whatever opens an article, isLTR saying which way it
// runs.
func (s *PDFTextStripper) StartArticleLTR(isLTR bool) error {
	return s.write(s.ArticleStart())
}

// EndArticle writes whatever closes an article.
func (s *PDFTextStripper) EndArticle() error { return s.write(s.ArticleEnd()) }

// StartPage is called before a page is walked. It does nothing; a caller
// wanting more overrides it by wrapping the stripper.
func (s *PDFTextStripper) StartPage(page *pdmodel.PDPage) error { return nil }

// EndPage is called after a page is walked. It does nothing.
func (s *PDFTextStripper) EndPage(page *pdmodel.PDPage) error { return nil }

// WritePage writes out the text of the page the engine has just walked.
func (s *PDFTextStripper) WritePage() error {
	maxYForLine := float32(maxYForLineResetValue)
	minYTopForLine := float32(minYTopForLineResetValue)
	endOfLastTextX := float32(endOfLastTextXResetValue)
	lastWordSpacing := float32(lastWordSpacingResetValue)
	maxHeightForLine := float32(maxHeightForLineResetValue)
	var lastPosition *positionWrapper
	var lastLineStartPosition *positionWrapper
	startOfPage := true // flag to indicate start of page
	startOfArticle := false

	if len(s.charactersByArticle) != 0 {
		if err := s.WritePageStart(); err != nil {
			return err
		}
	}

	for articleIndex := range s.charactersByArticle {
		textList := s.charactersByArticle[articleIndex]
		if s.SortByPosition() {
			// because the TextPositionComparator is not transitive, but JDK7+
			// enforces transitivity on comparators, we need to use a custom
			// mergesort implementation (which is slower, unfortunately). Go's
			// sort does not check, so the port sorts with the mergesort
			// outright rather than trying the fast one first.
			util.IterativeMergeSort(textList, CompareTextPositions)

			// PDFBOX-5487: Remove all space characters if contained within the
			// adjacent letters
			// Java removes through the iterator, so the list the article holds
			// shrinks too; the port writes the shortened slice back.
			textList = removeContainedSpaces(textList)
			s.charactersByArticle[articleIndex] = textList
		}

		if err := s.StartArticle(); err != nil {
			return err
		}
		startOfArticle = true

		// Now cycle through to print the text.
		// We queue up a line at a time before we print so that we can convert
		// the line from presentation form to logical form (if needed).
		var line []lineItem

		// PDF files don't always store spaces. We will need to guess where we
		// should add spaces based on the distances between TextPositions.
		// Historically, this was done based on the size of the space character
		// provided by the font. In general, this worked but there were cases
		// where it did not work. Calculating the average character width and
		// using that as a metric works better in some cases but fails in some
		// cases where the spacing worked. So we use both. NOTE: Adobe reader
		// also fails on some of these examples.

		// Keeps track of the previous average character width
		previousAveCharWidth := float32(-1)

		for _, position := range textList {
			current := &positionWrapper{position: position}
			characterValue := position.Unicode()

			// PDFBOX-3774: conditionally ignore spaces from the content stream
			if characterValue == " " && s.IgnoreContentStreamSpaceGlyphs() {
				continue
			}

			// Resets the average character width when we see a change in font
			// or a change in the font size
			if lastPosition != nil && hasFontOrSizeChanged(position, lastPosition.position) {
				previousAveCharWidth = -1
			}

			var positionX, positionY, positionWidth, positionHeight float32

			// If we are sorting, then we need to use the text direction
			// adjusted coordinates, because they were used in the sorting.
			if s.SortByPosition() {
				positionX = position.XDirAdj()
				positionY = position.YDirAdj()
				positionWidth = position.WidthDirAdj()
				positionHeight = position.HeightDir()
			} else {
				positionX = position.X()
				positionY = position.Y()
				positionWidth = position.Width()
				positionHeight = position.Height()
			}

			// The current amount of characters in a word
			wordCharCount := len(position.IndividualWidths())

			// Estimate the expected width of the space based on the
			// space character with some margin.
			wordSpacing := position.WidthOfSpace()
			var deltaSpace float32
			if wordSpacing == 0 || isNaN32(wordSpacing) {
				deltaSpace = math.MaxFloat32
			} else if lastWordSpacing < 0 {
				deltaSpace = wordSpacing * s.SpacingTolerance()
			} else {
				deltaSpace = (wordSpacing + lastWordSpacing) / 2 * s.SpacingTolerance()
			}

			// Estimate the expected width of the space based on the average
			// character width with some margin. This calculation does not make
			// a true average (average of averages) but we found that it gave
			// the best results after numerous experiments. Based on experiments
			// we also found that .3 worked well.
			var averageCharWidth float32
			if previousAveCharWidth < 0 {
				averageCharWidth = positionWidth / float32(wordCharCount)
			} else {
				averageCharWidth = (previousAveCharWidth + positionWidth/float32(wordCharCount)) / 2
			}
			deltaCharWidth := averageCharWidth * s.AverageCharTolerance()

			// Compares the values obtained by the average method and the
			// wordSpacing method and picks the smaller number.
			expectedStartOfNextWordX := float32(expectedStartOfNextWordXResetValue)
			if endOfLastTextX != endOfLastTextXResetValue {
				expectedStartOfNextWordX = endOfLastTextX + min32(deltaSpace, deltaCharWidth)
			}

			if lastPosition != nil {
				if startOfArticle {
					lastPosition.isArticleStart = true
					startOfArticle = false
				}
				// RDD - Here we determine whether this text object is on the
				// current line. We use the lastBaselineFontSize to handle the
				// superscript case, and the size of the current font to handle
				// the subscript case. Text must overlap with the last rendered
				// baseline text by at least a small amount in order to be
				// considered as being on the same line.

				// XXX BC: In theory, this check should really check if the next
				// char is in full range seen in this line. This is what I tried
				// to do with minYTopForLine, but this caused a lot of
				// regression test failures. So, I'm leaving it be for now
				if !overlap(positionY, positionHeight, maxYForLine, maxHeightForLine) {
					if err := s.writeLine(s.normalizeLine(line)); err != nil {
						return err
					}
					line = line[:0]
					var err error
					lastLineStartPosition, err = s.handleLineSeparation(current, lastPosition,
						lastLineStartPosition, maxHeightForLine)
					if err != nil {
						return err
					}
					expectedStartOfNextWordX = expectedStartOfNextWordXResetValue
					maxYForLine = maxYForLineResetValue
					maxHeightForLine = maxHeightForLineResetValue
					minYTopForLine = minYTopForLineResetValue
				}

				// test if our TextPosition starts after a new word would be
				// expected to start
				if expectedStartOfNextWordX != expectedStartOfNextWordXResetValue &&
					expectedStartOfNextWordX < positionX &&
					// only bother adding a word separator if the last character
					// was not a word separator
					(s.wordSeparator == "" ||
						!strings.HasSuffix(lastPosition.position.Unicode(), s.wordSeparator)) {
					line = append(line, wordSeparatorItem)
				}

				// if there is at least the equivalent of one space between the
				// last character and the current one, reset the max line height
				// as the font size may have completely changed.
				if abs32(position.X()-lastPosition.position.X()) > wordSpacing+deltaSpace {
					maxYForLine = maxYForLineResetValue
					maxHeightForLine = maxHeightForLineResetValue
					minYTopForLine = minYTopForLineResetValue
				}
			}

			if positionY >= maxYForLine {
				maxYForLine = positionY
			}

			// RDD - endX is what PDF considers to be the x coordinate of the
			// end position of the text. We use it in computing our metrics
			// below.
			endOfLastTextX = positionX + positionWidth

			// add it to the list
			if startOfPage && lastPosition == nil {
				if err := s.WriteParagraphStart(); err != nil { // not sure this is correct for RTL?
					return err
				}
			}
			line = append(line, lineItem{position: position})

			maxHeightForLine = max32(maxHeightForLine, positionHeight)
			minYTopForLine = min32(minYTopForLine, positionY-positionHeight)

			lastPosition = current
			if startOfPage {
				lastPosition.isParagraphStart = true
				lastPosition.isLineStart = true
				lastLineStartPosition = lastPosition
				startOfPage = false
			}
			lastWordSpacing = wordSpacing
			previousAveCharWidth = averageCharWidth
		}

		// print the final line
		if len(line) != 0 {
			if err := s.writeLine(s.normalizeLine(line)); err != nil {
				return err
			}
			if err := s.WriteParagraphEnd(); err != nil {
				return err
			}
		}
		if err := s.EndArticle(); err != nil {
			return err
		}
	}
	// minYTopForLine is computed and never read, in Java as here: the comment in
	// writePage says the check it was meant for caused regression failures and
	// was left out. Kept so the port does not quietly drop a variable the Java
	// still maintains.
	_ = minYTopForLine
	return s.WritePageEnd()
}

// hasFontOrSizeChanged reports whether the two positions were set in different
// fonts or at different sizes.
func hasFontOrSizeChanged(current, last *TextPosition) bool {
	if last == nil {
		return false
	}
	// compare font sizes
	if current.FontSize() != last.FontSize() {
		return true
	}
	// compare font instances, may not work if the resource cache is disabled
	if current.Font() == last.Font() {
		return false
	}
	currentFontName := current.Font().Name()
	lastFontName := last.Font().Name()
	if currentFontName != "" {
		// compare font names
		return currentFontName != lastFontName
	}
	if lastFontName != "" {
		// currentFontName is null but lastFontName isn't -> font changes
		return true
	}
	// both fonts don't have a name -> compare hashes
	//
	// Java compares PDFont.hashCode, which is the hash of the font dictionary;
	// the port compares the dictionaries, which is what that hash stands for.
	// A Type 3 font with no /Name is the case this reaches.
	return current.Font().Dictionary() != last.Font().Dictionary()
}

// overlap reports whether two lines of text share any vertical extent.
func overlap(y1, height1, y2, height2 float32) bool {
	return within(y1, y2, .1) || y2 <= y1 && y2 >= y1-height1 ||
		y1 <= y2 && y1 >= y2-height2
}

// removeContainedSpaces drops a space that sits entirely inside the character
// before it.
func removeContainedSpaces(textList []*TextPosition) []*TextPosition {
	if len(textList) == 0 {
		return textList
	}
	out := textList[:1]
	previousPosition := textList[0]
	for _, position := range textList[1:] {
		if position.Unicode() == " " && previousPosition.CompletelyContains(position) {
			continue
		}
		out = append(out, position)
		previousPosition = position
	}
	return out
}

// WriteLineSeparator writes whatever ends a line.
func (s *PDFTextStripper) WriteLineSeparator() error { return s.write(s.LineSeparator()) }

// WriteWordSeparator writes whatever goes between two words.
func (s *PDFTextStripper) WriteWordSeparator() error { return s.write(s.WordSeparator()) }

// WriteCharacters writes the text of one position.
func (s *PDFTextStripper) WriteCharacters(text *TextPosition) error {
	return s.write(text.Unicode())
}

// WriteString writes one word, with the positions it was built from.
func (s *PDFTextStripper) WriteString(text string, textPositions []*TextPosition) error {
	return s.write(text)
}

// write writes to the output.
func (s *PDFTextStripper) write(text string) error {
	if s.output == nil || text == "" {
		return nil
	}
	_, err := io.WriteString(s.output, text)
	return err
}

// within reports whether two values are no further apart than the variance.
func within(first, second, variance float32) bool {
	return second < first+variance && second > first-variance
}

// BeginMarkedContentSequence handles the BDC and BMC operators, picking up the
// /ActualText a sequence may carry.
func (s *PDFTextStripper) BeginMarkedContentSequence(tag *cos.Name, properties *cos.Dictionary) {
	markedContent := markedcontent.Create(tag, properties)
	s.currentMarkedContents = append(s.currentMarkedContents, markedContent)
	actualText, ok := markedContent.ActualText()
	s.actualText = actualText
	s.hasActualText = ok
	if ok {
		s.actualText = strings.ReplaceAll(s.actualText, "­", "") // remove soft hyphens
		s.firstActualTextPosition = true
	}
	s.LegacyPDFStreamEngine.BeginMarkedContentSequence(tag, properties)
}

// EndMarkedContentSequence handles the EMC operator.
func (s *PDFTextStripper) EndMarkedContentSequence() {
	if n := len(s.currentMarkedContents); n != 0 {
		markedContent := s.currentMarkedContents[n-1]
		if _, ok := markedContent.ActualText(); ok {
			s.actualText = ""
			s.hasActualText = false
		}
		s.currentMarkedContents = s.currentMarkedContents[:n-1]
	}
	s.LegacyPDFStreamEngine.EndMarkedContentSequence()
}

// ProcessTextPosition files one glyph under the article it belongs to, dropping
// it where it duplicates one already there and folding it into the character
// before it where it is a diacritic.
func (s *PDFTextStripper) ProcessTextPosition(text *TextPosition) error {
	if s.hasActualText {
		if s.firstActualTextPosition {
			text.setUnicode(s.actualText)
			s.firstActualTextPosition = false
		} else {
			text.setUnicode("")
		}
	}

	showCharacter := true
	if s.suppressDuplicateOverlappingText && !s.hasActualText {
		showCharacter = false
		textCharacter := text.Unicode()
		textX := text.X()
		textY := text.Y()
		sameTextCharacters, ok := s.characterListMapping[textCharacter]
		if !ok {
			sameTextCharacters = map[float32]map[float32]bool{}
			s.characterListMapping[textCharacter] = sameTextCharacters
		}

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
		// Java walks a sorted map's submap; the port walks the keys it holds
		// and takes the same range.
		for x, xMatch := range sameTextCharacters {
			if x < textX-tolerance || x >= textX+tolerance {
				continue
			}
			for y := range xMatch {
				if y >= textY-tolerance && y < textY+tolerance {
					suppressCharacter = true
					break
				}
			}
			if suppressCharacter {
				break
			}
		}
		if !suppressCharacter {
			ySet, ok := sameTextCharacters[textX]
			if !ok {
				ySet = map[float32]bool{}
				sameTextCharacters[textX] = ySet
			}
			ySet[textY] = true
			showCharacter = true
		}
	}

	if !showCharacter {
		return nil
	}

	// if we are showing the character then we need to determine which article
	// it belongs to
	foundArticleDivisionIndex := -1
	notFoundButFirstLeftAndAboveArticleDivisionIndex := -1
	notFoundButFirstLeftArticleDivisionIndex := -1
	notFoundButFirstAboveArticleDivisionIndex := -1
	x := text.X()
	y := text.Y()
	if s.shouldSeparateByBeads {
		for i := 0; i < len(s.beadRectangles) && foundArticleDivisionIndex == -1; i++ {
			rect := s.beadRectangles[i]
			if rect == nil {
				foundArticleDivisionIndex = 0
				continue
			}
			switch {
			case rect.Contains(x, y):
				foundArticleDivisionIndex = i*2 + 1
			case (x < rect.LowerLeftX() || y < rect.UpperRightY()) &&
				notFoundButFirstLeftAndAboveArticleDivisionIndex == -1:
				notFoundButFirstLeftAndAboveArticleDivisionIndex = i * 2
			case x < rect.LowerLeftX() && notFoundButFirstLeftArticleDivisionIndex == -1:
				notFoundButFirstLeftArticleDivisionIndex = i * 2
			case y < rect.UpperRightY() && notFoundButFirstAboveArticleDivisionIndex == -1:
				notFoundButFirstAboveArticleDivisionIndex = i * 2
			}
		}
	} else {
		foundArticleDivisionIndex = 0
	}

	var articleDivisionIndex int
	switch {
	case foundArticleDivisionIndex != -1:
		articleDivisionIndex = foundArticleDivisionIndex
	case notFoundButFirstLeftAndAboveArticleDivisionIndex != -1:
		articleDivisionIndex = notFoundButFirstLeftAndAboveArticleDivisionIndex
	case notFoundButFirstLeftArticleDivisionIndex != -1:
		articleDivisionIndex = notFoundButFirstLeftArticleDivisionIndex
	case notFoundButFirstAboveArticleDivisionIndex != -1:
		articleDivisionIndex = notFoundButFirstAboveArticleDivisionIndex
	default:
		articleDivisionIndex = len(s.charactersByArticle) - 1
	}
	if articleDivisionIndex < 0 || articleDivisionIndex >= len(s.charactersByArticle) {
		return nil
	}

	textList := s.charactersByArticle[articleDivisionIndex]

	// In the wild, some PDF encoded documents put diacritics (accents on top of
	// characters) into a separate Tj element. When displaying them graphically,
	// the two chunks get overlaid. With text output though, we need to do the
	// overlay. This code recombines the diacritic with its associated character
	// if the two are consecutive.
	if len(textList) == 0 {
		textList = append(textList, text)
	} else {
		// test if we overlap the previous entry.
		// Note that we are making an assumption that we need to only look back
		// one TextPosition to find what we are overlapping.
		// This may not always be true.
		previousTextPosition := textList[len(textList)-1]
		switch {
		case text.IsDiacritic() && previousTextPosition.Contains(text):
			previousTextPosition.MergeDiacritic(text)
		// If the previous TextPosition was the diacritic, merge it into this one
		// and remove it from the list.
		case previousTextPosition.IsDiacritic() && text.Contains(previousTextPosition):
			text.MergeDiacritic(previousTextPosition)
			textList[len(textList)-1] = text
		default:
			textList = append(textList, text)
		}
	}
	s.charactersByArticle[articleDivisionIndex] = textList
	return nil
}

// handleLineSeparation ends the line and works out whether the next one starts
// a paragraph.
func (s *PDFTextStripper) handleLineSeparation(current, lastPosition,
	lastLineStartPosition *positionWrapper, maxHeightForLine float32) (*positionWrapper, error) {
	current.isLineStart = true
	s.isParagraphSeparation(current, lastPosition, lastLineStartPosition, maxHeightForLine)
	lastLineStartPosition = current
	if current.isParagraphStart {
		if lastPosition.isArticleStart {
			if lastPosition.isLineStart {
				if err := s.WriteLineSeparator(); err != nil {
					return nil, err
				}
			}
			if err := s.WriteParagraphStart(); err != nil {
				return nil, err
			}
		} else {
			if err := s.WriteLineSeparator(); err != nil {
				return nil, err
			}
			if err := s.WriteParagraphSeparator(); err != nil {
				return nil, err
			}
		}
	} else if err := s.WriteLineSeparator(); err != nil {
		return nil, err
	}
	return lastLineStartPosition, nil
}

// isParagraphSeparation decides whether the position starts a new paragraph.
func (s *PDFTextStripper) isParagraphSeparation(position, lastPosition,
	lastLineStartPosition *positionWrapper, maxHeightForLine float32) {
	result := false
	if lastLineStartPosition == nil {
		result = true
	} else {
		yGap := abs32(position.position.YDirAdj() - lastPosition.position.YDirAdj())
		newYVal := multiplyFloat(s.DropThreshold(), maxHeightForLine)
		// do we need to flip this for rtl?
		xGap := position.position.XDirAdj() - lastLineStartPosition.position.XDirAdj()
		newXVal := multiplyFloat(s.IndentThreshold(), position.position.WidthOfSpace())
		positionWidth := multiplyFloat(0.25, position.position.Width())

		switch {
		case yGap > newYVal:
			result = true
		case xGap > newXVal:
			// text is indented, but try to screen for hanging indent
			if !lastLineStartPosition.isParagraphStart {
				result = true
			} else {
				position.isHangingIndent = true
			}
		case xGap < -position.position.WidthOfSpace():
			// text is left of previous line. Was it a hanging indent?
			if !lastLineStartPosition.isParagraphStart {
				result = true
			}
		case abs32(xGap) < positionWidth:
			// current horizontal position is within 1/4 a char of the last
			// linestart. We'll treat them as lined up.
			if lastLineStartPosition.isHangingIndent {
				position.isHangingIndent = true
			} else if lastLineStartPosition.isParagraphStart {
				// check to see if the previous line looks like any of a number
				// of standard list item formats
				if liPattern := s.matchListItemPattern(lastLineStartPosition); liPattern != nil {
					if currentPattern := s.matchListItemPattern(position); liPattern == currentPattern {
						result = true
					}
				}
			}
		}
	}
	if result {
		position.isParagraphStart = true
	}
}

// multiplyFloat multiplies two floats and truncates the resulting value to 3
// decimal places to avoid wrong results when comparing with another float.
//
// Java multiplies in float and then calls Math.round(float), which is
// floor(x + 0.5); the port multiplies in float32 for the same reason, since
// widening first would round differently at the boundary.
func multiplyFloat(value1, value2 float32) float32 {
	return float32(math.Floor(float64(value1*value2*1000)+0.5)) / 1000
}

// WriteParagraphSeparator writes whatever goes between two paragraphs.
func (s *PDFTextStripper) WriteParagraphSeparator() error {
	if err := s.WriteParagraphEnd(); err != nil {
		return err
	}
	return s.WriteParagraphStart()
}

// WriteParagraphStart writes whatever opens a paragraph.
func (s *PDFTextStripper) WriteParagraphStart() error {
	if s.inParagraph {
		if err := s.WriteParagraphEnd(); err != nil {
			return err
		}
		s.inParagraph = false
	}
	if err := s.write(s.ParagraphStart()); err != nil {
		return err
	}
	s.inParagraph = true
	return nil
}

// WriteParagraphEnd writes whatever closes a paragraph.
func (s *PDFTextStripper) WriteParagraphEnd() error {
	if !s.inParagraph {
		if err := s.WriteParagraphStart(); err != nil {
			return err
		}
	}
	if err := s.write(s.ParagraphEnd()); err != nil {
		return err
	}
	s.inParagraph = false
	return nil
}

// WritePageStart writes whatever opens a page.
func (s *PDFTextStripper) WritePageStart() error { return s.write(s.PageStart()) }

// WritePageEnd writes whatever closes a page.
func (s *PDFTextStripper) WritePageEnd() error { return s.write(s.PageEnd()) }

// matchListItemPattern returns which list marker the text of the position looks
// like, or nil where it looks like none of them.
func (s *PDFTextStripper) matchListItemPattern(pw *positionWrapper) *regexp.Regexp {
	return matchPattern(pw.position.Unicode(), s.ListItemPatterns())
}

// SetListItemPatterns sets the shapes a list marker may take.
func (s *PDFTextStripper) SetListItemPatterns(patterns []*regexp.Regexp) {
	s.listOfPatterns = patterns
}

// ListItemPatterns returns the shapes a list marker may take.
func (s *PDFTextStripper) ListItemPatterns() []*regexp.Regexp {
	if s.listOfPatterns == nil {
		s.listOfPatterns = make([]*regexp.Regexp, 0, len(listItemExpressions))
		for _, expression := range listItemExpressions {
			// Java's Matcher.matches anchors at both ends; Go's MatchString does
			// not, so the anchors are written out.
			s.listOfPatterns = append(s.listOfPatterns, regexp.MustCompile(`\A(?:`+expression+`)\z`))
		}
	}
	return s.listOfPatterns
}

// matchPattern returns the first pattern the string matches, or nil.
func matchPattern(str string, patterns []*regexp.Regexp) *regexp.Regexp {
	for _, p := range patterns {
		if p.MatchString(str) {
			return p
		}
	}
	return nil
}

// writeLine writes one line of words, separated by the word separator.
func (s *PDFTextStripper) writeLine(line []wordWithTextPositions) error {
	for i, word := range line {
		if err := s.WriteString(word.text, word.textPositions); err != nil {
			return err
		}
		if i < len(line)-1 {
			if err := s.WriteWordSeparator(); err != nil {
				return err
			}
		}
	}
	return nil
}

// normalizeLine turns a line of positions into the words it holds.
func (s *PDFTextStripper) normalizeLine(line []lineItem) []wordWithTextPositions {
	var normalized []wordWithTextPositions
	var lineBuilder strings.Builder
	var wordPositions []*TextPosition

	for _, item := range line {
		if item.isWordSeparator() {
			normalized = append(normalized, createWord(lineBuilder.String(), wordPositions))
			lineBuilder.Reset()
			wordPositions = nil
		} else {
			text := item.position
			lineBuilder.WriteString(text.VisuallyOrderedUnicode())
			wordPositions = append(wordPositions, text)
		}
	}
	if lineBuilder.Len() > 0 {
		normalized = append(normalized, createWord(lineBuilder.String(), wordPositions))
	}
	return normalized
}

// createWord returns one word, normalised.
func createWord(word string, wordPositions []*TextPosition) wordWithTextPositions {
	positions := make([]*TextPosition, len(wordPositions))
	copy(positions, wordPositions)
	return wordWithTextPositions{text: normalizeWord(word), textPositions: positions}
}

// normalizeWord folds the presentation forms of a word back into the letters
// they stand for.
func normalizeWord(word string) string {
	var builder *strings.Builder
	runes := []rune(word)
	p := 0
	q := 0
	for ; q < len(runes); q++ {
		// We only normalize if the codepoint is in a given range. Otherwise,
		// NFKC converts too many things that would cause confusion. For
		// example, it converts the micro symbol in extended Latin to the value
		// in the Greek script. We normalize the Unicode Alphabetic and Arabic
		// A&B Presentation forms.
		c := runes[q]
		if 0xFB00 <= c && c <= 0xFDFF || 0xFE70 <= c && c <= 0xFEFF {
			if builder == nil {
				builder = &strings.Builder{}
			}
			builder.WriteString(string(runes[p:q]))

			// Some fonts map U+FDF2 differently than the Unicode spec. They add
			// an extra U+0627 character to compensate. This removes the extra
			// character for those fonts.
			if c == 0xFDF2 && q > 0 && (runes[q-1] == 0x0627 || runes[q-1] == 0xFE8D) {
				builder.WriteString("لله")
			} else {
				// Trim because some decompositions have an extra space, such as
				// U+FC5E
				normalized := strings.TrimSpace(norm.NFKC.String(string(c)))
				// Hebrew in Alphabetic Presentation Forms from FB1D to FB4F and
				// Arabic Presentation Forms-A from FB50 to FDFF and
				// Arabic Presentation Forms-B from FE70 to FEFF
				if 0xFB1D <= c && utf16Length(normalized) > 1 {
					// Reverse the order of decomposed Hebrew and Arabic letters
					normalized = reverseString(normalized)
				}
				builder.WriteString(normalized)
			}
			p = q + 1
		}
	}
	if builder == nil {
		return handleDirection(word)
	}
	builder.WriteString(string(runes[p:q]))
	return handleDirection(builder.String())
}

// reverseString returns the string with its runes in the opposite order.
func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// mirroringCharMap maps a character onto the one that mirrors it, which is what
// a right to left run swaps brackets for.
var mirroringCharMap = func() map[rune]rune {
	m := map[rune]rune{}
	input, err := resources.Open("text/BidiMirroring.txt")
	if err != nil {
		// Could not parse BidiMirroring.txt, mirroring char map will be empty
		return m
	}
	defer input.Close()
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		s := scanner.Text()
		if comment := strings.IndexByte(s, '#'); comment != -1 { // ignore comments
			s = s[:comment]
		}
		if len(s) < 2 {
			continue
		}
		fields := strings.Split(s, ";")
		if len(fields) != 2 {
			continue
		}
		from, err1 := strconv.ParseInt(strings.TrimSpace(fields[0]), 16, 32)
		to, err2 := strconv.ParseInt(strings.TrimSpace(fields[1]), 16, 32)
		if err1 != nil || err2 != nil {
			continue
		}
		m[rune(from)] = rune(to)
	}
	return m
}()

// lineItem is one entry of a line: either a position, or the gap between two
// words.
type lineItem struct {
	position *TextPosition
}

// wordSeparatorItem stands for the gap between two words.
var wordSeparatorItem = lineItem{}

// isWordSeparator reports whether the item is the gap between two words.
func (l lineItem) isWordSeparator() bool { return l.position == nil }

// wordWithTextPositions is one word and the positions it was built from.
type wordWithTextPositions struct {
	text          string
	textPositions []*TextPosition
}

// positionWrapper is one position and what the line and paragraph detection has
// worked out about it.
type positionWrapper struct {
	isLineStart      bool
	isParagraphStart bool
	isPageBreak      bool
	isHangingIndent  bool
	isArticleStart   bool
	position         *TextPosition
}

func isNaN32(v float32) bool { return v != v }
func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// --- the settings, as Java's getters and setters ---

// StartPageNo returns the first page the stripper writes out, counting from 1.
func (s *PDFTextStripper) StartPageNo() int { return s.startPage }

// SetStartPage sets the first page the stripper writes out.
func (s *PDFTextStripper) SetStartPage(startPageValue int) { s.startPage = startPageValue }

// EndPageNo returns the last page the stripper writes out.
func (s *PDFTextStripper) EndPageNo() int { return s.endPage }

// SetEndPage sets the last page the stripper writes out.
func (s *PDFTextStripper) SetEndPage(endPageValue int) { s.endPage = endPageValue }

// SetLineSeparator sets what ends a line.
func (s *PDFTextStripper) SetLineSeparator(separator string) { s.lineSeparator = separator }

// LineSeparator returns what ends a line.
func (s *PDFTextStripper) LineSeparator() string { return s.lineSeparator }

// WordSeparator returns what goes between two words.
func (s *PDFTextStripper) WordSeparator() string { return s.wordSeparator }

// SetWordSeparator sets what goes between two words.
func (s *PDFTextStripper) SetWordSeparator(separator string) { s.wordSeparator = separator }

// SuppressDuplicateOverlappingText reports whether text drawn twice in one
// place is written out once.
func (s *PDFTextStripper) SuppressDuplicateOverlappingText() bool {
	return s.suppressDuplicateOverlappingText
}

// SetSuppressDuplicateOverlappingText sets whether text drawn twice in one
// place is written out once.
func (s *PDFTextStripper) SetSuppressDuplicateOverlappingText(value bool) {
	s.suppressDuplicateOverlappingText = value
}

// CurrentPageNo returns which page is being written out, counting from 1.
func (s *PDFTextStripper) CurrentPageNo() int { return s.currentPageNo }

// CharactersByArticle returns the positions of the current page, one list per
// article.
func (s *PDFTextStripper) CharactersByArticle() [][]*TextPosition {
	return s.charactersByArticle
}

// SeparateByBeads reports whether the text is split by the article beads of the
// page.
func (s *PDFTextStripper) SeparateByBeads() bool { return s.shouldSeparateByBeads }

// SetShouldSeparateByBeads sets whether the text is split by the article beads.
func (s *PDFTextStripper) SetShouldSeparateByBeads(value bool) {
	s.shouldSeparateByBeads = value
}

// AddMoreFormatting reports whether the paragraph and article markers are
// written out.
func (s *PDFTextStripper) AddMoreFormatting() bool { return s.addMoreFormatting }

// SetAddMoreFormatting sets whether the paragraph and article markers are
// written out.
func (s *PDFTextStripper) SetAddMoreFormatting(newAddMoreFormatting bool) {
	s.addMoreFormatting = newAddMoreFormatting
}

// SortByPosition reports whether the text is sorted by where it sits rather
// than by the order it was drawn in.
func (s *PDFTextStripper) SortByPosition() bool { return s.sortByPosition }

// SetSortByPosition sets whether the text is sorted by where it sits.
func (s *PDFTextStripper) SetSortByPosition(newSortByPosition bool) {
	s.sortByPosition = newSortByPosition
}

// IgnoreContentStreamSpaceGlyphs reports whether a space the content stream
// draws is dropped in favour of one worked out from the spacing.
func (s *PDFTextStripper) IgnoreContentStreamSpaceGlyphs() bool {
	return s.ignoreContentStreamSpaceGlyphs
}

// SetIgnoreContentStreamSpaceGlyphs sets whether a space the content stream
// draws is dropped.
func (s *PDFTextStripper) SetIgnoreContentStreamSpaceGlyphs(value bool) {
	s.ignoreContentStreamSpaceGlyphs = value
}

// SpacingTolerance returns how much of a space's width a gap must be before it
// counts as a word break.
func (s *PDFTextStripper) SpacingTolerance() float32 { return s.spacingTolerance }

// SetSpacingTolerance sets how much of a space's width a gap must be.
func (s *PDFTextStripper) SetSpacingTolerance(spacingToleranceValue float32) {
	s.spacingTolerance = spacingToleranceValue
}

// AverageCharTolerance returns how much of the average character width a gap
// must be before it counts as a word break.
func (s *PDFTextStripper) AverageCharTolerance() float32 { return s.averageCharTolerance }

// SetAverageCharTolerance sets how much of the average character width a gap
// must be.
func (s *PDFTextStripper) SetAverageCharTolerance(averageCharToleranceValue float32) {
	s.averageCharTolerance = averageCharToleranceValue
}

// IndentThreshold returns how many space widths a line must be indented by
// before it starts a paragraph.
func (s *PDFTextStripper) IndentThreshold() float32 { return s.indentThreshold }

// SetIndentThreshold sets how many space widths a line must be indented by.
func (s *PDFTextStripper) SetIndentThreshold(indentThresholdValue float32) {
	s.indentThreshold = indentThresholdValue
}

// DropThreshold returns how many line heights the gap between two lines must be
// before the second starts a paragraph.
func (s *PDFTextStripper) DropThreshold() float32 { return s.dropThreshold }

// SetDropThreshold sets how many line heights the gap between two lines must
// be.
func (s *PDFTextStripper) SetDropThreshold(dropThresholdValue float32) {
	s.dropThreshold = dropThresholdValue
}

// ParagraphStart returns what opens a paragraph.
func (s *PDFTextStripper) ParagraphStart() string { return s.paragraphStart }

// SetParagraphStart sets what opens a paragraph.
func (s *PDFTextStripper) SetParagraphStart(s2 string) { s.paragraphStart = s2 }

// ParagraphEnd returns what closes a paragraph.
func (s *PDFTextStripper) ParagraphEnd() string { return s.paragraphEnd }

// SetParagraphEnd sets what closes a paragraph.
func (s *PDFTextStripper) SetParagraphEnd(s2 string) { s.paragraphEnd = s2 }

// PageStart returns what opens a page.
func (s *PDFTextStripper) PageStart() string { return s.pageStart }

// SetPageStart sets what opens a page.
func (s *PDFTextStripper) SetPageStart(pageStartValue string) { s.pageStart = pageStartValue }

// PageEnd returns what closes a page.
func (s *PDFTextStripper) PageEnd() string { return s.pageEnd }

// SetPageEnd sets what closes a page.
func (s *PDFTextStripper) SetPageEnd(pageEndValue string) { s.pageEnd = pageEndValue }

// ArticleStart returns what opens an article.
func (s *PDFTextStripper) ArticleStart() string { return s.articleStart }

// SetArticleStart sets what opens an article.
func (s *PDFTextStripper) SetArticleStart(articleStartValue string) {
	s.articleStart = articleStartValue
}

// ArticleEnd returns what closes an article.
func (s *PDFTextStripper) ArticleEnd() string { return s.articleEnd }

// SetArticleEnd sets what closes an article.
func (s *PDFTextStripper) SetArticleEnd(articleEndValue string) { s.articleEnd = articleEndValue }
