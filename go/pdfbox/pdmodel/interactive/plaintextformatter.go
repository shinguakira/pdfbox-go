package interactive

// TextContentStream is what a formatter writes its text through.
//
// Java names PDAppearanceContentStream, which lives in pdmodel; pdmodel imports
// this package through the form fields, so the dependency cannot run both ways.
// The port names the two methods the formatter uses.
type TextContentStream interface {
	// NewLineAtOffset moves to the start of the next line of text.
	NewLineAtOffset(tx, ty float32) error

	// ShowText writes the given text.
	ShowText(text string) error
}

// PlainTextFormatter writes text into an appearance, wrapping and aligning it.
//
// Port of PlainTextFormatter.
type PlainTextFormatter struct {
	appearanceStyle *AppearanceStyle
	wrapLines       bool
	width           float32
	contents        TextContentStream
	textContent     *PlainText
	textAlignment   TextAlign

	horizontalOffset float32
	verticalOffset   float32
}

// PlainTextFormatterBuilder gathers the parts of a formatter.
//
// Port of the nested class PlainTextFormatter.Builder.
type PlainTextFormatterBuilder struct {
	// required parameters
	contents TextContentStream

	// optional parameters
	appearanceStyle *AppearanceStyle
	wrapLines       bool
	width           float32
	textContent     *PlainText
	textAlignment   TextAlign

	// initial offset from where to start the position of the first line
	horizontalOffset float32
	verticalOffset   float32
}

// NewPlainTextFormatterBuilder returns a builder writing into the given
// content stream.
func NewPlainTextFormatterBuilder(contents TextContentStream) *PlainTextFormatterBuilder {
	return &PlainTextFormatterBuilder{contents: contents, textAlignment: TextAlignLeft}
}

// Style sets the font and size the text is written with.
func (b *PlainTextFormatterBuilder) Style(appearanceStyle *AppearanceStyle) *PlainTextFormatterBuilder {
	b.appearanceStyle = appearanceStyle
	return b
}

// WrapLines sets whether the text is broken into lines.
func (b *PlainTextFormatterBuilder) WrapLines(wrapLines bool) *PlainTextFormatterBuilder {
	b.wrapLines = wrapLines
	return b
}

// Width sets the width the text is laid out in.
func (b *PlainTextFormatterBuilder) Width(width float32) *PlainTextFormatterBuilder {
	b.width = width
	return b
}

// TextAlignValue sets the alignment from a /Q quadding.
func (b *PlainTextFormatterBuilder) TextAlignValue(alignment int) *PlainTextFormatterBuilder {
	b.textAlignment = TextAlignOf(alignment)
	return b
}

// TextAlign sets the alignment.
func (b *PlainTextFormatterBuilder) TextAlign(alignment TextAlign) *PlainTextFormatterBuilder {
	b.textAlignment = alignment
	return b
}

// Text sets the text to write.
func (b *PlainTextFormatterBuilder) Text(textContent *PlainText) *PlainTextFormatterBuilder {
	b.textContent = textContent
	return b
}

// InitialOffset sets where the first line starts.
func (b *PlainTextFormatterBuilder) InitialOffset(
	horizontalOffset, verticalOffset float32) *PlainTextFormatterBuilder {
	b.horizontalOffset = horizontalOffset
	b.verticalOffset = verticalOffset
	return b
}

// Build returns the formatter.
func (b *PlainTextFormatterBuilder) Build() *PlainTextFormatter {
	return &PlainTextFormatter{
		appearanceStyle:  b.appearanceStyle,
		wrapLines:        b.wrapLines,
		width:            b.width,
		contents:         b.contents,
		textContent:      b.textContent,
		textAlignment:    b.textAlignment,
		horizontalOffset: b.horizontalOffset,
		verticalOffset:   b.verticalOffset,
	}
}

// Format writes the text into the content stream.
func (f *PlainTextFormatter) Format() error {
	if f.textContent == nil || len(f.textContent.Paragraphs()) == 0 {
		return nil
	}
	isFirstParagraph := true
	for _, paragraph := range f.textContent.Paragraphs() {
		if f.wrapLines {
			lines, err := paragraph.Lines(f.appearanceStyle.Font(),
				f.appearanceStyle.FontSize(), f.width)
			if err != nil {
				return err
			}
			if err := f.processLines(lines, isFirstParagraph); err != nil {
				return err
			}
			isFirstParagraph = false
			continue
		}

		startOffset := float32(0)
		width, err := f.appearanceStyle.Font().StringWidth(paragraph.Text())
		if err != nil {
			return err
		}
		lineWidth := width * f.appearanceStyle.FontSize() / fontScale
		if lineWidth < f.width {
			switch f.textAlignment {
			case TextAlignCenter:
				startOffset = (f.width - lineWidth) / 2
			case TextAlignRight:
				startOffset = f.width - lineWidth
			default:
				// JUSTIFY and LEFT
				startOffset = 0
			}
		}
		if err := f.contents.NewLineAtOffset(f.horizontalOffset+startOffset,
			f.verticalOffset); err != nil {
			return err
		}
		if err := f.contents.ShowText(paragraph.Text()); err != nil {
			return err
		}
	}
	return nil
}

// processLines writes the lines of one wrapped paragraph. Java declares it
// private.
func (f *PlainTextFormatter) processLines(lines []*Line, isFirstParagraph bool) error {
	lastPos := float32(0)
	startOffset := float32(0)
	interWordSpacing := float32(0)

	for lineIndex, line := range lines {
		switch f.textAlignment {
		case TextAlignCenter:
			startOffset = (f.width - line.Width()) / 2
		case TextAlignRight:
			startOffset = f.width - line.Width()
		case TextAlignJustify:
			if lineIndex != len(lines)-1 {
				interWordSpacing = line.InterWordSpacing(f.width)
			}
		default:
			startOffset = 0
		}

		offset := -lastPos + startOffset + f.horizontalOffset

		if lineIndex == 0 && isFirstParagraph {
			if err := f.contents.NewLineAtOffset(offset, f.verticalOffset); err != nil {
				return err
			}
		} else {
			// keep the last position
			f.verticalOffset -= f.appearanceStyle.Leading()
			if err := f.contents.NewLineAtOffset(offset,
				-f.appearanceStyle.Leading()); err != nil {
				return err
			}
		}

		lastPos += offset

		words := line.Words()
		for wordIndex, word := range words {
			if err := f.contents.ShowText(word.Text()); err != nil {
				return err
			}
			wordWidth := word.Width()
			if wordIndex != len(words)-1 {
				if err := f.contents.NewLineAtOffset(wordWidth+interWordSpacing, 0); err != nil {
					return err
				}
				lastPos += wordWidth + interWordSpacing
			}
		}
	}
	f.horizontalOffset -= lastPos
	return nil
}
