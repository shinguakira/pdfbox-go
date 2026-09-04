package ttf

// NumberOfMacGlyphs is how many names the standard Macintosh ordering has.
const NumberOfMacGlyphs = 258

// macGlyphNames is the standard Macintosh ordering of glyph names, which a post
// table of format 1.0 uses whole and a format 2.0 table indexes into.
//
// Port of the MAC_GLYPH_NAMES table of org.apache.fontbox.ttf.WGL4Names.
var macGlyphNames = [NumberOfMacGlyphs]string{
	".notdef", ".null", "nonmarkingreturn", "space", "exclam", "quotedbl",
	"numbersign", "dollar", "percent", "ampersand", "quotesingle",
	"parenleft", "parenright", "asterisk", "plus", "comma", "hyphen",
	"period", "slash", "zero", "one", "two", "three", "four", "five",
	"six", "seven", "eight", "nine", "colon", "semicolon", "less",
	"equal", "greater", "question", "at", "A", "B", "C", "D", "E", "F",
	"G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S",
	"T", "U", "V", "W", "X", "Y", "Z", "bracketleft", "backslash",
	"bracketright", "asciicircum", "underscore", "grave", "a", "b",
	"c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o",
	"p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z", "braceleft",
	"bar", "braceright", "asciitilde", "Adieresis", "Aring",
	"Ccedilla", "Eacute", "Ntilde", "Odieresis", "Udieresis", "aacute",
	"agrave", "acircumflex", "adieresis", "atilde", "aring",
	"ccedilla", "eacute", "egrave", "ecircumflex", "edieresis",
	"iacute", "igrave", "icircumflex", "idieresis", "ntilde", "oacute",
	"ograve", "ocircumflex", "odieresis", "otilde", "uacute", "ugrave",
	"ucircumflex", "udieresis", "dagger", "degree", "cent", "sterling",
	"section", "bullet", "paragraph", "germandbls", "registered",
	"copyright", "trademark", "acute", "dieresis", "notequal", "AE",
	"Oslash", "infinity", "plusminus", "lessequal", "greaterequal",
	"yen", "mu", "partialdiff", "summation", "product", "pi",
	"integral", "ordfeminine", "ordmasculine", "Omega", "ae", "oslash",
	"questiondown", "exclamdown", "logicalnot", "radical", "florin",
	"approxequal", "Delta", "guillemotleft", "guillemotright",
	"ellipsis", "nonbreakingspace", "Agrave", "Atilde", "Otilde", "OE",
	"oe", "endash", "emdash", "quotedblleft", "quotedblright",
	"quoteleft", "quoteright", "divide", "lozenge", "ydieresis",
	"Ydieresis", "fraction", "currency", "guilsinglleft",
	"guilsinglright", "fi", "fl", "daggerdbl", "periodcentered",
	"quotesinglbase", "quotedblbase", "perthousand", "Acircumflex",
	"Ecircumflex", "Aacute", "Edieresis", "Egrave", "Iacute",
	"Icircumflex", "Idieresis", "Igrave", "Oacute", "Ocircumflex",
	"apple", "Ograve", "Uacute", "Ucircumflex", "Ugrave", "dotlessi",
	"circumflex", "tilde", "macron", "breve", "dotaccent", "ring",
	"cedilla", "hungarumlaut", "ogonek", "caron", "Lslash", "lslash",
	"Scaron", "scaron", "Zcaron", "zcaron", "brokenbar", "Eth", "eth",
	"Yacute", "yacute", "Thorn", "thorn", "minus", "multiply",
	"onesuperior", "twosuperior", "threesuperior", "onehalf",
	"onequarter", "threequarters", "franc", "Gbreve", "gbreve",
	"Idotaccent", "Scedilla", "scedilla", "Cacute", "cacute", "Ccaron",
	"ccaron", "dcroat",
}

// macGlyphNamesIndices is macGlyphNames the other way round.
var macGlyphNamesIndices = func() map[string]int {
	indices := make(map[string]int, NumberOfMacGlyphs)
	for i := 0; i < NumberOfMacGlyphs; i++ {
		indices[macGlyphNames[i]] = i
	}
	return indices
}()

// GlyphIndex returns where a name sits in the standard Macintosh ordering. The
// second result is false where the name is not one of them, which is the null
// Java returns.
func GlyphIndex(name string) (int, bool) {
	index, ok := macGlyphNamesIndices[name]
	return index, ok
}

// GlyphName returns the name at the given place in the standard Macintosh
// ordering, or the empty string where there is no such place. Java returns null
// there, and its one caller null-checks the result.
func GlyphName(index int) string {
	if index >= 0 && index < NumberOfMacGlyphs {
		return macGlyphNames[index]
	}
	return ""
}

// AllGlyphNames returns a copy of the standard Macintosh ordering.
func AllGlyphNames() []string {
	glyphNames := make([]string, NumberOfMacGlyphs)
	copy(glyphNames, macGlyphNames[:])
	return glyphNames
}
