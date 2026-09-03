package font

import (
	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font/encoding"
)

// PDSimpleFont is a font whose character codes are one byte each.
//
// Port of the abstract org.apache.pdfbox.pdmodel.font.PDSimpleFont.
type PDSimpleFont interface {
	PDFont

	// Encoding returns how the character codes map onto glyph names.
	Encoding() encoding.Encoding

	// GlyphList returns the list the glyph names are read through.
	GlyphList() *encoding.GlyphList

	// IsSymbolic reports whether the font uses an encoding of its own rather
	// than the standard Latin one.
	IsSymbolic() bool

	// GetPathByName returns the outline of the named glyph.
	GetPathByName(name string) (*geom.Path2D, error)

	// HasGlyphByName reports whether the font has the named glyph.
	HasGlyphByName(name string) (bool, error)

	// FontBoxFont returns the font program the glyphs are drawn from.
	FontBoxFont() fontbox.FontBoxFont

	// readEncodingFromFont returns the encoding built into the font program.
	// Java's protected abstract readEncodingFromFont.
	readEncodingFromFont() (encoding.Encoding, error)

	// isFontSymbolic reports whether the font is symbolic, the second result
	// being false where the font cannot say. Java returns a Boolean that may be
	// null.
	isFontSymbolic() (bool, bool)

	// simple returns the shared part of a simple font.
	simple() *pdSimpleFont
}

// pdSimpleFont is the state every simple font carries.
type pdSimpleFont struct {
	pdFont

	encoding  encoding.Encoding
	glyphList *encoding.GlyphList

	isSymbolic    bool
	isSymbolicSet bool

	// self is the simple font this state belongs to, kept alongside pdFont.self
	// so that the shared methods can reach the simple-font members too.
	selfSimple PDSimpleFont
}

// simple returns the shared part of a simple font.
func (f *pdSimpleFont) simple() *pdSimpleFont { return f }

// readEncoding works out how the character codes of the font map onto glyph
// names.
func (f *pdSimpleFont) readEncoding() error {
	encodingBase := f.dict.GetDictionaryObject(cos.Encoding)
	switch value := encodingBase.(type) {
	case *cos.Name:
		if ZapfDingbatsFontName.Name() == f.selfSimple.Name() && !f.selfSimple.IsEmbedded() {
			// PDFBOX- and PDF.js issue 16464: ignore other encodings
			// this segment will work only if readEncoding() is called after the
			// data for getName() and isEmbedded() is available
			f.encoding = encoding.ZapfDingbatsEncodingInstance
		} else {
			f.encoding = encoding.GetInstance(value)
			if f.encoding == nil {
				// Unknown encoding
				builtIn, err := f.selfSimple.readEncodingFromFont() // fallback
				if err != nil {
					return err
				}
				f.encoding = builtIn
			}
		}
	case *cos.Dictionary:
		var builtIn encoding.Encoding
		symbolic, hasSymbolic := f.symbolicFlag()
		baseEncoding := value.GetCOSName(cos.BaseEncoding)
		hasValidBaseEncoding := baseEncoding != nil && encoding.GetInstance(baseEncoding) != nil
		if !hasValidBaseEncoding && hasSymbolic && symbolic {
			var err error
			builtIn, err = f.selfSimple.readEncodingFromFont()
			if err != nil {
				return err
			}
		}
		if !hasSymbolic {
			symbolic = false
		}
		dictEncoding, err := encoding.NewDictionaryEncodingWithBuiltIn(value, !symbolic, builtIn)
		if err != nil {
			return err
		}
		f.encoding = dictEncoding
	default:
		builtIn, err := f.selfSimple.readEncodingFromFont()
		if err != nil {
			return err
		}
		f.encoding = builtIn
	}

	// normalise the standard 14 name, e.g "Symbol,Italic" -> "Symbol"
	standard14Name, _ := GetMappedFontName(f.selfSimple.Name())
	f.assignGlyphList(standard14Name)
	return nil
}

// Encoding returns how the character codes map onto glyph names.
func (f *pdSimpleFont) Encoding() encoding.Encoding { return f.encoding }

// GlyphList returns the list the glyph names are read through.
func (f *pdSimpleFont) GlyphList() *encoding.GlyphList { return f.glyphList }

// IsSymbolic reports whether the font uses an encoding of its own.
func (f *pdSimpleFont) IsSymbolic() bool {
	if !f.isSymbolicSet {
		result, ok := f.selfSimple.isFontSymbolic()
		if ok {
			f.isSymbolic = result
		} else {
			// unless we can prove that the font is non-symbolic, we assume that
			// it is not
			f.isSymbolic = true
		}
		f.isSymbolicSet = true
	}
	return f.isSymbolic
}

// isFontSymbolic reports whether the font is symbolic, the second result being
// false where the font cannot say.
func (f *pdSimpleFont) isFontSymbolic() (bool, bool) {
	if result, ok := f.symbolicFlag(); ok {
		return result, true
	}
	if f.selfSimple.IsStandard14() {
		mappedName, _ := GetMappedFontName(f.selfSimple.Name())
		return mappedName == SymbolFontName || mappedName == ZapfDingbatsFontName, true
	}
	if f.encoding == nil {
		// check, should never happen
		if _, isTrueType := f.selfSimple.(*PDTrueTypeFont); !isTrueType {
			panic("PDFBox bug: encoding should not be null!")
		}
		// TTF without its non-symbolic flag set must be symbolic
		return true, true
	}
	switch enc := f.encoding.(type) {
	case *encoding.WinAnsiEncoding, *encoding.MacRomanEncoding, *encoding.StandardEncoding:
		return false, true
	case *encoding.DictionaryEncoding:
		// each name in Differences array must also be in the latin character set
		for _, name := range enc.Differences() {
			if name == ".notdef" {
				// skip
			} else if !(encoding.WinAnsiEncodingInstance.ContainsName(name) &&
				encoding.MacRomanEncodingInstance.ContainsName(name) &&
				encoding.StandardEncodingInstance.ContainsName(name)) {
				return true, true
			}
		}
		return false, true
	default:
		// we don't know
		return false, false
	}
}

// symbolicFlag returns the symbolic flag of the font descriptor, the second
// result being false where the font has no descriptor.
func (f *pdSimpleFont) symbolicFlag() (bool, bool) {
	if fd := f.selfSimple.FontDescriptor(); fd != nil {
		// fixme: isSymbolic() defaults to false if the flag is missing so we
		// can't trust this
		return fd.IsSymbolic(), true
	}
	return false, false
}

// ToUnicode returns what the given character code stands for.
func (f *pdSimpleFont) ToUnicode(code int) (string, error) {
	return f.ToUnicodeWithGlyphList(code, encoding.AdobeGlyphList())
}

// ToUnicodeWithGlyphList returns what the given character code stands for,
// reading unknown glyph names through the given list.
func (f *pdSimpleFont) ToUnicodeWithGlyphList(code int, customGlyphList *encoding.GlyphList) (string, error) {
	// allow the glyph list to be overridden for the purpose of extracting
	// Unicode we only do this when the font's glyph list is the AGL, to avoid
	// breaking Zapf Dingbats
	var unicodeGlyphList *encoding.GlyphList
	if f.glyphList == encoding.AdobeGlyphList() {
		unicodeGlyphList = customGlyphList
	} else {
		unicodeGlyphList = f.glyphList
	}

	// first try to use a ToUnicode CMap
	unicode, err := f.pdFont.ToUnicode(code)
	if err != nil {
		return "", err
	}
	if unicode != "" {
		return unicode, nil
	}

	// if the font is a "simple font" and uses MacRoman/MacExpert/WinAnsi
	// [Encoding] or has Differences with names from only Adobe Standard and/or
	// Symbol, then:
	//
	//    a) Map the character codes to names
	//    b) Look up the name in the Adobe Glyph List to obtain the Unicode value
	if f.encoding != nil {
		name := f.encoding.Name(code)
		unicode = unicodeGlyphList.ToUnicode(name)
		if unicode != "" {
			return unicode, nil
		}
	}

	// if no value has been produced, there is no way to obtain Unicode for the
	// character. Java logs it once per code.
	return "", nil
}

// IsVertical reports that a simple font is never set vertically.
func (f *pdSimpleFont) IsVertical() bool { return false }

// standard14Width returns the width the metrics of a standard 14 font give for
// the glyph.
//
// It panics where the font is not one of the fourteen, as Java's
// IllegalStateException does.
func (f *pdSimpleFont) standard14Width(code int) float32 {
	afmStandard14 := f.standard14AFM()
	if afmStandard14 == nil {
		panic("No AFM")
	}
	nameInAFM := f.selfSimple.Encoding().Name(code)

	// the Adobe AFMs don't include .notdef, but Acrobat uses 250, test with
	// PDFBOX-2334
	if nameInAFM == ".notdef" {
		return 250
	}
	if nameInAFM == "nbspace" {
		// PDFBOX-4944: nbspace is missing in AFM files,
		// but PDF specification tells "it shall be typographically the same as
		// SPACE"
		nameInAFM = "space"
	} else if nameInAFM == "sfthyphen" {
		// PDFBOX-5115: sfthyphen is missing in AFM files,
		// but PDF specification tells "it shall be typographically the same as
		// hyphen"
		nameInAFM = "hyphen"
	}
	return afmStandard14.CharacterWidth(nameInAFM)
}

// IsStandard14 reports whether the font is one of the fourteen every reader
// has.
func (f *pdSimpleFont) IsStandard14() bool {
	// this logic is based on Acrobat's behaviour, see PDFBOX-2372
	// the Encoding entry cannot have Differences if we want "standard 14" font
	// handling
	if dictionary, ok := f.selfSimple.Encoding().(*encoding.DictionaryEncoding); ok {
		if len(dictionary.Differences()) != 0 {
			// we also require that the differences are actually different, see
			// PDFBOX-1900 with the file from PDFBOX-2192 on Windows
			baseEncoding := dictionary.BaseEncoding()
			for code, name := range dictionary.Differences() {
				if name != baseEncoding.Name(code) {
					return false
				}
			}
		}
	}
	return f.pdFont.IsStandard14()
}

// isNonZeroBoundingBox reports whether the given box has any extent at all.
func isNonZeroBoundingBox(bbox *common.PDRectangle) bool {
	return bbox != nil && (bbox.LowerLeftX() != 0 ||
		bbox.LowerLeftY() != 0 ||
		bbox.UpperRightX() != 0 ||
		bbox.UpperRightY() != 0)
}

// AddToSubset panics: only TTF subsetting through a Type 0 font is supported.
func (f *pdSimpleFont) AddToSubset(codePoint int) {
	panic("font: AddToSubset is not supported for a simple font")
}

// Subset panics: only TTF subsetting through a Type 0 font is supported.
func (f *pdSimpleFont) Subset() error {
	// only TTF subsetting via PDType0Font is currently supported
	panic("font: Subset is not supported for a simple font")
}

// WillBeSubset reports that a simple font is never subset.
func (f *pdSimpleFont) WillBeSubset() bool { return false }

// HasExplicitWidth reports whether the PDF gives a width for the glyph itself.
func (f *pdSimpleFont) HasExplicitWidth(code int) (bool, error) {
	if f.dict.ContainsKey(cos.Widths) {
		firstChar := f.dict.GetIntDefault(cos.FirstChar, -1)
		if code >= firstChar && code-firstChar < len(f.getWidths()) {
			return true, nil
		}
	}
	return false, nil
}

// assignGlyphList picks the glyph list the font's names are read through.
func (f *pdSimpleFont) assignGlyphList(fontName FontName) {
	// assign the glyph list based on the font
	if fontName == ZapfDingbatsFontName {
		f.glyphList = encoding.ZapfDingbats()
	} else {
		f.glyphList = encoding.AdobeGlyphList()
	}
}
