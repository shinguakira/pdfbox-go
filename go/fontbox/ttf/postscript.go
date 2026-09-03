package ttf

import (
	"errors"
	"io"
)

// PostScriptTable is the post table: the PostScript names of the glyphs, and
// the metrics a PostScript driver wants.
//
// Port of org.apache.fontbox.ttf.PostScriptTable.
type PostScriptTable struct {
	Table

	formatType         float32
	italicAngle        float32
	underlinePosition  int16
	underlineThickness int16
	isFixedPitch       int64
	minMemType42       int64
	maxMemType42       int64
	mimMemType1        int64
	maxMemType1        int64
	glyphNames         []string
}

var _ TableReader = (*PostScriptTable)(nil)

// Read reads the post table.
func (t *PostScriptTable) Read(ttf *TrueTypeFont, data DataStream) error {
	r := newReader(data)
	t.formatType = r.fixed()
	t.italicAngle = r.fixed()
	t.underlinePosition = r.signedShort()
	t.underlineThickness = r.signedShort()
	t.isFixedPitch = r.unsignedInt()
	t.minMemType42 = r.unsignedInt()
	t.maxMemType42 = r.unsignedInt()
	t.mimMemType1 = r.unsignedInt()
	t.maxMemType1 = r.unsignedInt()
	if r.err != nil {
		return r.err
	}

	switch {
	case data.CurrentPosition() == data.OriginalDataSize():
		// No PostScript name data is provided for the font.
	case t.formatType == 1.0:
		t.glyphNames = AllGlyphNames()
	case t.formatType == 2.0:
		if err := t.readFormat2(r, data); err != nil {
			return err
		}
	case t.formatType == 2.5:
		if err := t.readFormat25(r, ttf); err != nil {
			return err
		}
	case t.formatType == 3.0:
		// No PostScript name information is provided for the font.
	}
	t.SetInitialized(true)
	return nil
}

// readFormat2 reads the names of a format 2.0 table: an index per glyph into
// either the standard Macintosh ordering or the names that follow.
func (t *PostScriptTable) readFormat2(r *reader, data DataStream) error {
	numGlyphs := r.unsignedShort()
	if r.err != nil {
		return r.err
	}
	glyphNameIndex := make([]int, numGlyphs)
	t.glyphNames = make([]string, numGlyphs)
	maxIndex := minInt32
	for i := 0; i < numGlyphs; i++ {
		index := r.unsignedShort()
		if r.err != nil {
			return r.err
		}
		glyphNameIndex[i] = index
		if index <= 32767 {
			maxIndex = max(maxIndex, index)
		}
	}

	var nameArray []string
	if maxIndex >= NumberOfMacGlyphs {
		nameArray = make([]string, maxIndex-NumberOfMacGlyphs+1)
		for i := range nameArray {
			numberOfChars, err := readUnsignedByte(data)
			if err == nil {
				nameArray[i], err = readString(data, numberOfChars)
			}
			if err != nil {
				// Error reading names in the PostScript table; the remaining
				// entries are set to .notdef.
				for j := i; j < len(nameArray); j++ {
					nameArray[j] = ".notdef"
				}
				break
			}
		}
	}

	for i := 0; i < numGlyphs; i++ {
		index := glyphNameIndex[i]
		switch {
		case index >= 0 && index < NumberOfMacGlyphs:
			t.glyphNames[i] = GlyphName(index)
		case index >= NumberOfMacGlyphs && index <= 32767 && nameArray != nil:
			t.glyphNames[i] = nameArray[index-NumberOfMacGlyphs]
		default:
			t.glyphNames[i] = ".undefined"
		}
	}
	return nil
}

// readFormat25 reads the names of a format 2.5 table: a signed offset per glyph
// into the standard Macintosh ordering.
func (t *PostScriptTable) readFormat25(r *reader, ttf *TrueTypeFont) error {
	numberOfGlyphs, err := ttf.NumberOfGlyphs()
	if err != nil {
		return err
	}
	glyphNameIndex := make([]int, numberOfGlyphs)
	for i := range glyphNameIndex {
		offset := r.signedByte()
		if r.err != nil {
			return r.err
		}
		glyphNameIndex[i] = i + 1 + offset
	}
	t.glyphNames = make([]string, len(glyphNameIndex))
	for i := range t.glyphNames {
		index := glyphNameIndex[i]
		if index >= 0 && index < NumberOfMacGlyphs {
			// GlyphName returns the empty string where Java returns null, and
			// the guard above already rules that out; the assignment is the same
			// either way.
			t.glyphNames[i] = GlyphName(index)
		}
		// an incorrect glyph name index leaves the entry as it was
	}
	return nil
}

// minInt32 is Java's Integer.MIN_VALUE, which the format 2.0 read starts its
// running maximum at.
const minInt32 = -1 << 31

// FormatType returns which of the post table formats this is.
func (t *PostScriptTable) FormatType() float32 { return t.formatType }

// SetFormatType sets which of the post table formats this is.
func (t *PostScriptTable) SetFormatType(formatType float32) { t.formatType = formatType }

// IsFixedPitch returns whether the font is monospaced, as a flag rather than a
// boolean, which is how the table stores it.
func (t *PostScriptTable) IsFixedPitch() int64 { return t.isFixedPitch }

// SetIsFixedPitch sets whether the font is monospaced.
func (t *PostScriptTable) SetIsFixedPitch(isFixedPitch int64) { t.isFixedPitch = isFixedPitch }

// ItalicAngle returns how far the font leans, in degrees counterclockwise.
func (t *PostScriptTable) ItalicAngle() float32 { return t.italicAngle }

// SetItalicAngle sets how far the font leans.
func (t *PostScriptTable) SetItalicAngle(italicAngle float32) { t.italicAngle = italicAngle }

// MaxMemType1 returns the maximum memory a Type 1 font needs.
func (t *PostScriptTable) MaxMemType1() int64 { return t.maxMemType1 }

// SetMaxMemType1 sets the maximum memory a Type 1 font needs.
func (t *PostScriptTable) SetMaxMemType1(maxMemType1 int64) { t.maxMemType1 = maxMemType1 }

// MaxMemType42 returns the maximum memory a Type 42 font needs.
func (t *PostScriptTable) MaxMemType42() int64 { return t.maxMemType42 }

// SetMaxMemType42 sets the maximum memory a Type 42 font needs.
func (t *PostScriptTable) SetMaxMemType42(maxMemType42 int64) { t.maxMemType42 = maxMemType42 }

// MinMemType1 returns the minimum memory a Type 1 font needs.
func (t *PostScriptTable) MinMemType1() int64 { return t.mimMemType1 }

// SetMinMemType1 sets the minimum memory a Type 1 font needs.
func (t *PostScriptTable) SetMinMemType1(mimMemType1 int64) { t.mimMemType1 = mimMemType1 }

// MinMemType42 returns the minimum memory a Type 42 font needs.
func (t *PostScriptTable) MinMemType42() int64 { return t.minMemType42 }

// SetMinMemType42 sets the minimum memory a Type 42 font needs.
func (t *PostScriptTable) SetMinMemType42(minMemType42 int64) { t.minMemType42 = minMemType42 }

// UnderlinePosition returns where the underline sits.
func (t *PostScriptTable) UnderlinePosition() int16 { return t.underlinePosition }

// SetUnderlinePosition sets where the underline sits.
func (t *PostScriptTable) SetUnderlinePosition(underlinePosition int16) {
	t.underlinePosition = underlinePosition
}

// UnderlineThickness returns how thick the underline is.
func (t *PostScriptTable) UnderlineThickness() int16 { return t.underlineThickness }

// SetUnderlineThickness sets how thick the underline is.
func (t *PostScriptTable) SetUnderlineThickness(underlineThickness int16) {
	t.underlineThickness = underlineThickness
}

// GlyphNames returns the name of every glyph, or nil where the table carries
// none.
func (t *PostScriptTable) GlyphNames() []string { return t.glyphNames }

// SetGlyphNames sets the name of every glyph.
func (t *PostScriptTable) SetGlyphNames(glyphNames []string) { t.glyphNames = glyphNames }

// GetName returns the name of the given glyph, or the empty string where the
// table has no name for it. Java returns null there.
func (t *PostScriptTable) GetName(gid int) string {
	if gid < 0 || t.glyphNames == nil || gid >= len(t.glyphNames) {
		return ""
	}
	return t.glyphNames[gid]
}

// isEOF reports whether an error is the end of the stream, which several tables
// treat as the end of the data they read rather than as a failure.
func isEOF(err error) bool {
	return errors.Is(err, io.ErrUnexpectedEOF)
}
