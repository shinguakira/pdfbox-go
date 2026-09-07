package tools

// Convert PDF text to Markdown format. Each line in the PDF is converted to a
// corresponding Markdown paragraph. Bold and italic formatting is also applied
// based on font properties.
//
// Port of org.apache.pdfbox.tools.PDFText2Markdown.
//
// Its FontState is a near-copy of PDFText2HTML's, and the port keeps it a
// second type rather than sharing one: the two differ in what a tag looks like
// opened and closed, and in how each decides a font is bold or italic -- the
// HTML one matches "Bold" and "Italic" case-sensitively, this one lower-cases
// the name first and also takes "oblique". Folding them together would hide
// exactly those three differences.

import (
	"strings"
	"unicode/utf16"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// PDFText2Markdown writes the text of a document as Markdown.
type PDFText2Markdown struct {
	*text.PDFTextStripper

	fontState markdownFontState
}

// NewPDFText2Markdown returns a stripper that writes Markdown.
func NewPDFText2Markdown() *PDFText2Markdown {
	s := &PDFText2Markdown{PDFTextStripper: text.NewPDFTextStripper()}
	s.SetTextOverrides(s)
	s.SetLineSeparator(text.LineSeparatorDefault)
	s.SetParagraphStart(text.LineSeparatorDefault)
	s.SetParagraphEnd(text.LineSeparatorDefault)
	s.SetPageStart(text.LineSeparatorDefault)
	s.SetPageEnd(text.LineSeparatorDefault)
	s.SetArticleStart(text.LineSeparatorDefault)
	s.SetArticleEnd(text.LineSeparatorDefault)
	return s
}

// StartArticleLTR writes out the article separator.
//
// Java takes isLTR and ignores it, unlike the HTML one.
func (s *PDFText2Markdown) StartArticleLTR(isLTR bool) error {
	return s.PDFTextStripper.WriteStringChars(text.LineSeparatorDefault)
}

// EndArticle writes out the article separator.
func (s *PDFText2Markdown) EndArticle() error {
	if err := s.PDFTextStripper.EndArticle(); err != nil {
		return err
	}
	return s.PDFTextStripper.WriteStringChars(text.LineSeparatorDefault)
}

// WriteString writes a string to the output stream, maintaining font state and
// escaping some Markdown characters. The font state is only preserved per word.
func (s *PDFText2Markdown) WriteString(str string, textPositions []*text.TextPosition) error {
	return s.PDFTextStripper.WriteStringChars(s.fontState.push(str, textPositions))
}

// WriteStringChars writes a string to the output stream and escapes some
// Markdown characters.
func (s *PDFText2Markdown) WriteStringChars(chars string) error {
	return s.PDFTextStripper.WriteStringChars(escapeMarkdown(chars))
}

// WriteParagraphEnd writes the Markdown paragraph end to the output.
// Furthermore, it will also clear the font state.
func (s *PDFText2Markdown) WriteParagraphEnd() error {
	// do not escape HTML
	if err := s.PDFTextStripper.WriteStringChars(s.fontState.clear()); err != nil {
		return err
	}
	return s.PDFTextStripper.WriteParagraphEnd()
}

// escapeMarkdown escapes some Markdown characters.
//
// Java walks by UTF-16 unit, as the HTML one does.
func escapeMarkdown(chars string) string {
	var builder strings.Builder
	for _, unit := range utf16.Encode([]rune(chars)) {
		appendEscapedMarkdown(&builder, unit)
	}
	return builder.String()
}

// appendEscapedMarkdown writes one UTF-16 unit, escaped.
func appendEscapedMarkdown(builder *strings.Builder, character uint16) {
	switch character {
	case '*', '+', '-', '#', '\\', '`', '[', ']', '(', ')', '!', '_':
		builder.WriteByte('\\')
		builder.WriteByte(byte(character))
	// Escape HTML special characters in inline HTML
	case '<':
		builder.WriteString("&lt;")
	case '>':
		builder.WriteString("&gt;")
	case '&':
		builder.WriteString("&amp;")
	case 178:
		builder.WriteString("<sup>2</sup>")
	case 179:
		builder.WriteString("<sup>3</sup>")
	default:
		// Java appends the char, which for a unit above 127 is that code unit;
		// writing it as a rune is the same character back.
		builder.WriteString(string(utf16.Decode([]uint16{character})))
	}
}

// markdownFontState maintains the current font state, applying Markdown
// formatting based on font properties.
//
// Port of the private static nested class FontState of PDFText2Markdown.
type markdownFontState struct {
	stateList []string
	stateSet  map[string]bool
}

// push pushes new TextPositions into the font state.
func (f *markdownFontState) push(str string, textPositions []*text.TextPosition) string {
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
		buffer.WriteString(escapeMarkdown(string(utf16.Decode(units[1:]))))
	}
	return buffer.String()
}

// clear closes all open Markdown formatting.
func (f *markdownFontState) clear() string {
	var buffer strings.Builder
	f.closeUntil(&buffer, "", false)
	f.stateList = nil
	f.stateSet = nil
	return buffer.String()
}

func (f *markdownFontState) pushUnit(buffer *strings.Builder, character uint16,
	textPosition *text.TextPosition) {
	bold := false
	italics := false

	if textPosition.Font() != nil {
		if descriptor := textPosition.Font().FontDescriptor(); descriptor != nil {
			bold = isBoldMarkdown(descriptor)
			italics = isItalicMarkdown(descriptor)
		}
	}

	if bold {
		buffer.WriteString(f.open("**"))
	} else {
		buffer.WriteString(f.close("**"))
	}
	if italics {
		buffer.WriteString(f.open("*"))
	} else {
		buffer.WriteString(f.close("*"))
	}

	appendEscapedMarkdown(buffer, character)
}

func (f *markdownFontState) open(tag string) string {
	if f.stateSet[tag] {
		return ""
	}
	f.stateList = append(f.stateList, tag)
	if f.stateSet == nil {
		f.stateSet = map[string]bool{}
	}
	f.stateSet[tag] = true
	// openTag and closeTag are both the tag itself here.
	return tag
}

func (f *markdownFontState) close(tag string) string {
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
		tagsBuilder.WriteString(f.stateList[index])
	}
	return tagsBuilder.String()
}

func (f *markdownFontState) closeUntil(tagsBuilder *strings.Builder, endTag string,
	hasEnd bool) int {
	for i := len(f.stateList); i > 0; {
		i--
		tag := f.stateList[i]
		tagsBuilder.WriteString(tag)
		if hasEnd && tag == endTag {
			return i
		}
	}
	return -1
}

func isBoldMarkdown(descriptor *font.PDFontDescriptor) bool {
	if descriptor.IsForceBold() {
		return true
	}
	return strings.Contains(strings.ToLower(descriptor.FontName()), "bold")
}

func isItalicMarkdown(descriptor *font.PDFontDescriptor) bool {
	if descriptor.IsItalic() {
		return true
	}
	fontName := strings.ToLower(descriptor.FontName())
	return strings.Contains(fontName, "italic") || strings.Contains(fontName, "oblique")
}
