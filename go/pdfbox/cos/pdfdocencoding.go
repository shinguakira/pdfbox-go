package cos

// PDFDocEncoding, the single-byte encoding PDF uses for text strings that are
// not UTF-16.
//
// Port of the package-private org.apache.pdfbox.cos.PDFDocEncoding. It is
// basically ISO-8859-1 with three unmapped ranges and a block of typographic
// substitutions, per the table in ISO 32000-1:2008.

const replacementRune = '�'

var (
	// codeToRune maps a byte to its character. A zero entry means the code is
	// not part of the encoding.
	codeToRune [256]rune
	// runeToCode is the reverse map, holding only codes the encoding defines.
	runeToCode map[rune]byte
)

func init() {
	runeToCode = make(map[rune]byte, 256)

	set := func(code int, r rune) {
		codeToRune[code] = r
		runeToCode[r] = byte(code)
	}

	// Start from ISO-8859-1, skipping the codes the Unicode column leaves
	// blank.
	for i := 0; i < 256; i++ {
		if i > 0x17 && i < 0x20 {
			continue
		}
		if i > 0x7E && i < 0xA1 {
			continue
		}
		if i == 0xAD {
			continue
		}
		set(i, rune(i))
	}

	// Then the deviations, from the table in ISO 32000-1:2008.
	// block 1
	set(0x18, '˘') // BREVE
	set(0x19, 'ˇ') // CARON
	set(0x1A, 'ˆ') // MODIFIER LETTER CIRCUMFLEX ACCENT
	set(0x1B, '˙') // DOT ABOVE
	set(0x1C, '˝') // DOUBLE ACUTE ACCENT
	set(0x1D, '˛') // OGONEK
	set(0x1E, '˚') // RING ABOVE
	set(0x1F, '˜') // SMALL TILDE
	// block 2
	set(0x7F, replacementRune) // undefined
	set(0x80, '•')             // BULLET
	set(0x81, '†')             // DAGGER
	set(0x82, '‡')             // DOUBLE DAGGER
	set(0x83, '…')             // HORIZONTAL ELLIPSIS
	set(0x84, '—')             // EM DASH
	set(0x85, '–')             // EN DASH
	set(0x86, 'ƒ')             // LATIN SMALL LETTER SCRIPT F
	set(0x87, '⁄')             // FRACTION SLASH (solidus)
	set(0x88, '‹')             // SINGLE LEFT-POINTING ANGLE QUOTATION MARK
	set(0x89, '›')             // SINGLE RIGHT-POINTING ANGLE QUOTATION MARK
	set(0x8A, '−')             // MINUS SIGN
	set(0x8B, '‰')             // PER MILLE SIGN
	set(0x8C, '„')             // DOUBLE LOW-9 QUOTATION MARK (quotedblbase)
	set(0x8D, '“')             // LEFT DOUBLE QUOTATION MARK (quotedblleft)
	set(0x8E, '”')             // RIGHT DOUBLE QUOTATION MARK (quotedblright)
	set(0x8F, '‘')             // LEFT SINGLE QUOTATION MARK (quoteleft)
	set(0x90, '’')             // RIGHT SINGLE QUOTATION MARK (quoteright)
	set(0x91, '‚')             // SINGLE LOW-9 QUOTATION MARK (quotesinglbase)
	set(0x92, '™')             // TRADE MARK SIGN
	set(0x93, 'ﬁ')             // LATIN SMALL LIGATURE FI
	set(0x94, 'ﬂ')             // LATIN SMALL LIGATURE FL
	set(0x95, 'Ł')             // LATIN CAPITAL LETTER L WITH STROKE
	set(0x96, 'Œ')             // LATIN CAPITAL LIGATURE OE
	set(0x97, 'Š')             // LATIN CAPITAL LETTER S WITH CARON
	set(0x98, 'Ÿ')             // LATIN CAPITAL LETTER Y WITH DIAERESIS
	set(0x99, 'Ž')             // LATIN CAPITAL LETTER Z WITH CARON
	set(0x9A, 'ı')             // LATIN SMALL LETTER DOTLESS I
	set(0x9B, 'ł')             // LATIN SMALL LETTER L WITH STROKE
	set(0x9C, 'œ')             // LATIN SMALL LIGATURE OE
	set(0x9D, 'š')             // LATIN SMALL LETTER S WITH CARON
	set(0x9E, 'ž')             // LATIN SMALL LETTER Z WITH CARON
	set(0x9F, replacementRune) // undefined
	set(0xA0, '€')             // EURO SIGN
}

// pdfDocEncodingToString decodes bytes using PDFDocEncoding.
//
// Port of PDFDocEncoding.toString. Java also guards against a code beyond the
// table, which cannot happen for a byte; the port drops that branch.
func pdfDocEncodingToString(b []byte) string {
	runes := make([]rune, len(b))
	for i, c := range b {
		runes[i] = codeToRune[c]
	}
	return string(runes)
}

// pdfDocEncodingGetBytes encodes text using PDFDocEncoding.
//
// Port of PDFDocEncoding.getBytes. Characters the encoding does not define
// become 0, as they do in Java, which is why callers check every character
// with pdfDocEncodingContainsRune first.
func pdfDocEncodingGetBytes(text string) []byte {
	out := make([]byte, 0, len(text))
	for _, r := range text {
		out = append(out, runeToCode[r])
	}
	return out
}

// pdfDocEncodingContainsRune reports whether the encoding can represent r.
//
// Port of PDFDocEncoding.containsChar. Java takes a char, a UTF-16 code unit,
// so a character outside the basic multilingual plane is tested as two
// surrogates and both fail; a Go rune fails the map lookup directly, which
// reaches the same answer.
func pdfDocEncodingContainsRune(r rune) bool {
	_, ok := runeToCode[r]
	return ok
}
