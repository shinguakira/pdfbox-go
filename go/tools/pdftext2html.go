package tools

// Wrap stripped text in simple HTML, trying to form HTML paragraphs.
// Paragraphs broken by pages, columns, or figures are not mended.
//
// Port of org.apache.pdfbox.tools.PDFText2HTML.
//
// It extends PDFTextStripper and overrides seven of its hooks. Go has no
// dispatch from a base into an embedder, so the base holds an interface and the
// constructor installs this; see text.TextStripperOverrides.

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// PDFText2HTML writes the text of a document as HTML.
type PDFText2HTML struct {
	*text.PDFTextStripper

	fontState fontState
}

// NewPDFText2HTML returns a stripper that writes HTML.
func NewPDFText2HTML() *PDFText2HTML {
	s := &PDFText2HTML{PDFTextStripper: text.NewPDFTextStripper()}
	s.SetTextOverrides(s)
	s.SetLineSeparator(text.LineSeparatorDefault)
	s.SetParagraphStart("<p>")
	s.SetParagraphEnd("</p>" + text.LineSeparatorDefault)
	s.SetPageStart("<div style=\"page-break-before:always; page-break-after:always\">")
	s.SetPageEnd("</div>" + text.LineSeparatorDefault)
	s.SetArticleStart(text.LineSeparatorDefault)
	s.SetArticleEnd(text.LineSeparatorDefault)
	return s
}

// StartDocument writes the HTML head.
func (s *PDFText2HTML) StartDocument(document *pdmodel.PDDocument) error {
	var buf strings.Builder
	buf.WriteString("<!DOCTYPE html PUBLIC \"-//W3C//DTD HTML 4.01 Transitional//EN\"" + "\n" +
		"\"http://www.w3.org/TR/html4/loose.dtd\">\n")
	buf.WriteString("<html><head>")
	buf.WriteString("<title>" + escapeHTML(s.Title()) + "</title>\n")
	buf.WriteString("<meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\">\n")
	buf.WriteString("</head>\n")
	buf.WriteString("<body>\n")
	// super.writeString, which does not escape.
	return s.PDFTextStripper.WriteStringChars(buf.String())
}

// EndDocument closes the HTML.
func (s *PDFText2HTML) EndDocument(document *pdmodel.PDDocument) error {
	return s.PDFTextStripper.WriteStringChars("</body></html>")
}

// Title guesses the title of the document from either the document properties
// or the first lines of text.
//
// Port of the protected getTitle().
func (s *PDFText2HTML) Title() string {
	titleGuess := s.Document().DocumentInformation().Title()
	if titleGuess != "" {
		return titleGuess
	}

	lastFontSize := float32(-1.0)
	var titleText strings.Builder
	for _, article := range s.CharactersByArticle() {
		for _, position := range article {
			currentFontSize := position.FontSize()
			// If we're past 64 chars we will assume that we're past the title
			// 64 is arbitrary
			//
			// Java counts with StringBuilder.length(), which is UTF-16 units.
			if compareFloat32(currentFontSize, lastFontSize) != 0 || utf16Len(titleText.String()) > 64 {
				if titleText.Len() > 0 {
					return titleText.String()
				}
				lastFontSize = currentFontSize
			}
			if currentFontSize > 13.0 { // most body text is 12pt
				titleText.WriteString(position.Unicode())
			}
		}
	}
	return ""
}

// StartArticleLTR writes out the article separator (div tag) with proper text
// direction information.
func (s *PDFText2HTML) StartArticleLTR(isLTR bool) error {
	if isLTR {
		return s.PDFTextStripper.WriteStringChars("<div>")
	}
	return s.PDFTextStripper.WriteStringChars("<div dir=\"RTL\">")
}

// EndArticle writes out the article separator.
func (s *PDFText2HTML) EndArticle() error {
	if err := s.PDFTextStripper.EndArticle(); err != nil {
		return err
	}
	return s.PDFTextStripper.WriteStringChars("</div>")
}

// WriteString writes a string to the output stream, maintaining font state and
// escaping some HTML characters. The font state is only preserved per word.
func (s *PDFText2HTML) WriteString(str string, textPositions []*text.TextPosition) error {
	return s.PDFTextStripper.WriteStringChars(s.fontState.push(str, textPositions))
}

// WriteStringChars writes a string to the output stream and escapes some HTML
// characters.
func (s *PDFText2HTML) WriteStringChars(chars string) error {
	return s.PDFTextStripper.WriteStringChars(escapeHTML(chars))
}

// WriteParagraphEnd writes the paragraph end "</p>" to the output. Furthermore,
// it will also clear the font state.
func (s *PDFText2HTML) WriteParagraphEnd() error {
	// do not escape HTML
	if err := s.PDFTextStripper.WriteStringChars(s.fontState.clear()); err != nil {
		return err
	}
	return s.PDFTextStripper.WriteParagraphEnd()
}

// escapeHTML escapes some HTML characters.
//
// Java walks the string by UTF-16 unit -- charAt -- and writes anything outside
// printable ASCII as a numeric entity, so a character outside the basic plane
// becomes two entities, one per surrogate. The port walks the units too, or the
// entities would not match.
func escapeHTML(chars string) string {
	var builder strings.Builder
	for _, unit := range utf16.Encode([]rune(chars)) {
		appendEscaped(&builder, unit)
	}
	return builder.String()
}

// appendEscaped writes one UTF-16 unit, escaped.
func appendEscaped(builder *strings.Builder, character uint16) {
	// write non-ASCII as named entities
	if character < 32 || character > 126 {
		builder.WriteString("&#")
		builder.WriteString(strconv.Itoa(int(character)))
		builder.WriteByte(';')
		return
	}
	switch character {
	case 34:
		builder.WriteString("&quot;")
	case 38:
		builder.WriteString("&amp;")
	case 60:
		builder.WriteString("&lt;")
	case 62:
		builder.WriteString("&gt;")
	default:
		builder.WriteByte(byte(character))
	}
}

// fontState maintains the current font state. Its methods emit opening and
// closing tags as needed, and in the correct order.
//
// Port of the private static nested class FontState.
type fontState struct {
	stateList []string
	stateSet  map[string]bool
}

// push pushes new TextPositions into the font state. The state is only
// preserved correctly for each letter if the number of letters in text matches
// the number of TextPosition objects. Otherwise, it's done once for the
// complete array (just by looking at its first entry).
//
// Java's `text.length()` is UTF-16 units, and the stripper builds one
// TextPosition per unit, so the comparison is over units here too.
func (f *fontState) push(str string, textPositions []*text.TextPosition) string {
	var buffer strings.Builder
	units := utf16.Encode([]rune(str))

	if len(units) == len(textPositions) {
		// There is a 1:1 mapping, and we can use the TextPositions directly
		for i, unit := range units {
			f.pushUnit(&buffer, unit, textPositions[i])
		}
	} else if len(units) != 0 {
		// The normalized text does not match the number of TextPositions, so
		// we'll just have a look at its first entry.
		if len(textPositions) == 0 {
			return str
		}
		f.pushUnit(&buffer, units[0], textPositions[0])
		buffer.WriteString(escapeHTML(string(utf16.Decode(units[1:]))))
	}
	return buffer.String()
}

// clear closes all open states.
func (f *fontState) clear() string {
	var buffer strings.Builder
	f.closeUntil(&buffer, "", false)
	f.stateList = nil
	f.stateSet = nil
	return buffer.String()
}

// pushUnit appends one character with whatever tag changes its font asks for.
func (f *fontState) pushUnit(buffer *strings.Builder, character uint16,
	textPosition *text.TextPosition) {
	bold := false
	italics := false

	if textPosition.Font() != nil {
		if descriptor := textPosition.Font().FontDescriptor(); descriptor != nil {
			bold = isBold(descriptor)
			italics = isItalic(descriptor)
		}
	}

	if bold {
		buffer.WriteString(f.open("b"))
	} else {
		buffer.WriteString(f.close("b"))
	}
	if italics {
		buffer.WriteString(f.open("i"))
	} else {
		buffer.WriteString(f.close("i"))
	}
	appendEscaped(buffer, character)
}

func (f *fontState) open(tag string) string {
	if f.stateSet[tag] {
		return ""
	}
	f.stateList = append(f.stateList, tag)
	if f.stateSet == nil {
		f.stateSet = map[string]bool{}
	}
	f.stateSet[tag] = true
	return openTag(tag)
}

func (f *fontState) close(tag string) string {
	if !f.stateSet[tag] {
		return ""
	}
	// Close all tags until (but including) the one we should close
	var tagsBuilder strings.Builder
	index := f.closeUntil(&tagsBuilder, tag, true)

	// Remove from state
	f.stateList = append(f.stateList[:index], f.stateList[index+1:]...)
	delete(f.stateSet, tag)

	// Now open the states that were closed but should remain open again
	for ; index < len(f.stateList); index++ {
		tagsBuilder.WriteString(openTag(f.stateList[index]))
	}
	return tagsBuilder.String()
}

// closeUntil closes every open tag down to endTag, answering its index.
//
// Java passes null for endTag from clear(), where no tag matches and the walk
// closes everything; hasEnd says which call this is, because Go's zero string
// is a value a tag could in principle hold.
func (f *fontState) closeUntil(tagsBuilder *strings.Builder, endTag string, hasEnd bool) int {
	for i := len(f.stateList); i > 0; {
		i--
		tag := f.stateList[i]
		tagsBuilder.WriteString(closeTag(tag))
		if hasEnd && tag == endTag {
			return i
		}
	}
	return -1
}

func openTag(tag string) string  { return "<" + tag + ">" }
func closeTag(tag string) string { return "</" + tag + ">" }

func isBold(descriptor *font.PDFontDescriptor) bool {
	if descriptor.IsForceBold() {
		return true
	}
	return strings.Contains(descriptor.FontName(), "Bold")
}

func isItalic(descriptor *font.PDFontDescriptor) bool {
	if descriptor.IsItalic() {
		return true
	}
	return strings.Contains(descriptor.FontName(), "Italic")
}

// compareFloat32 is Float.compare, which orders NaN above everything and -0.0
// below 0.0 -- neither of which Go's == does.
func compareFloat32(first, second float32) int {
	if first < second {
		return -1
	}
	if first > second {
		return 1
	}
	firstBits := float32Bits(first)
	secondBits := float32Bits(second)
	if firstBits == secondBits {
		return 0
	}
	if firstBits < secondBits {
		return -1
	}
	return 1
}

// utf16Len is String.length(), which counts UTF-16 units.
func utf16Len(s string) int { return len(utf16.Encode([]rune(s))) }

// float32Bits is Float.floatToIntBits, which Float.compare orders by.
func float32Bits(value float32) int32 { return int32(math.Float32bits(value)) }
