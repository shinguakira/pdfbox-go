package ttf

import (
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// The platform IDs a name record can carry.
const (
	PlatformUnicode   = 0
	PlatformMacintosh = 1
	PlatformISO       = 2
	PlatformWindows   = 3
)

// The encoding IDs of the Unicode platform.
const (
	EncodingUnicode10     = 0
	EncodingUnicode11     = 1
	EncodingUnicode20BMP  = 3
	EncodingUnicode20Full = 4
)

// LanguageUnicode is the language ID of the Unicode platform.
const LanguageUnicode = 0

// The encoding IDs of the Windows platform.
const (
	EncodingWindowsSymbol      = 0
	EncodingWindowsUnicodeBMP  = 1
	EncodingWindowsUnicodeUCS4 = 10
)

// LanguageWindowsENUS is the language ID of US English on Windows.
const LanguageWindowsENUS = 0x0409

// EncodingMacintoshRoman is the Roman encoding ID of the Macintosh platform.
const EncodingMacintoshRoman = 0

// LanguageMacintoshEnglish is the English language ID of the Macintosh platform.
const LanguageMacintoshEnglish = 0

// The name IDs a name record can carry.
const (
	NameCopyright         = 0
	NameFontFamilyName    = 1
	NameFontSubFamilyName = 2
	NameUniqueFontID      = 3
	NameFullFontName      = 4
	NameVersion           = 5
	NamePostScriptName    = 6
	NameTrademark         = 7
)

// NameRecord is one entry of the name table.
//
// Port of org.apache.fontbox.ttf.NameRecord.
type NameRecord struct {
	platformID         int
	platformEncodingID int
	languageID         int
	nameID             int
	stringLength       int
	stringOffset       int
	str                string
}

// initData reads the fixed-size part of the record.
func (r *NameRecord) initData(ttf *TrueTypeFont, data DataStream) error {
	rd := newReader(data)
	r.platformID = rd.unsignedShort()
	r.platformEncodingID = rd.unsignedShort()
	r.languageID = rd.unsignedShort()
	r.nameID = rd.unsignedShort()
	r.stringLength = rd.unsignedShort()
	r.stringOffset = rd.unsignedShort()
	return rd.err
}

// PlatformID returns which platform the record is for.
func (r *NameRecord) PlatformID() int { return r.platformID }

// SetPlatformID sets which platform the record is for.
func (r *NameRecord) SetPlatformID(platformID int) { r.platformID = platformID }

// PlatformEncodingID returns how the string of the record is encoded.
func (r *NameRecord) PlatformEncodingID() int { return r.platformEncodingID }

// SetPlatformEncodingID sets how the string of the record is encoded.
func (r *NameRecord) SetPlatformEncodingID(platformEncodingID int) {
	r.platformEncodingID = platformEncodingID
}

// LanguageID returns which language the record is in.
func (r *NameRecord) LanguageID() int { return r.languageID }

// SetLanguageID sets which language the record is in.
func (r *NameRecord) SetLanguageID(languageID int) { r.languageID = languageID }

// NameID returns which of the names the record carries.
func (r *NameRecord) NameID() int { return r.nameID }

// SetNameID sets which of the names the record carries.
func (r *NameRecord) SetNameID(nameID int) { r.nameID = nameID }

// StringLength returns the length of the string in bytes.
func (r *NameRecord) StringLength() int { return r.stringLength }

// SetStringLength sets the length of the string in bytes.
func (r *NameRecord) SetStringLength(stringLength int) { r.stringLength = stringLength }

// StringOffset returns where the string is, from the start of the storage area.
func (r *NameRecord) StringOffset() int { return r.stringOffset }

// SetStringOffset sets where the string is.
func (r *NameRecord) SetStringOffset(stringOffset int) { r.stringOffset = stringOffset }

// String returns the decoded string of the record.
func (r *NameRecord) String() string { return r.str }

// SetString sets the decoded string of the record.
func (r *NameRecord) SetString(str string) { r.str = str }

// NamingTable is the name table: every name the font carries, in every
// platform, encoding and language it carries them for.
//
// Port of org.apache.fontbox.ttf.NamingTable.
type NamingTable struct {
	Table

	nameRecords []*NameRecord
	// lookupTable is the four-deep map Java builds, keyed name / platform /
	// encoding / language. Go can key one map on a struct, which is the same
	// lookup in one level rather than four.
	lookupTable map[nameKey]string

	fontFamily    string
	fontSubFamily string
	psName        string
}

// nameKey is what a name is looked up by.
type nameKey struct {
	nameID     int
	platformID int
	encodingID int
	languageID int
}

var _ TableReader = (*NamingTable)(nil)

// Read reads the name table.
func (t *NamingTable) Read(ttf *TrueTypeFont, data DataStream) error {
	if err := t.read(ttf, data, false); err != nil {
		return err
	}
	t.SetInitialized(true)
	return nil
}

func (t *NamingTable) read(ttf *TrueTypeFont, data DataStream, onlyHeaders bool) error {
	r := newReader(data)
	_ = r.unsignedShort() // formatSelector
	numberOfNameRecords := r.unsignedShort()
	_ = r.unsignedShort() // offsetToStartOfStringStorage
	if r.err != nil {
		return r.err
	}

	t.nameRecords = make([]*NameRecord, 0, numberOfNameRecords)
	for i := 0; i < numberOfNameRecords; i++ {
		nr := &NameRecord{}
		if err := nr.initData(ttf, data); err != nil {
			return err
		}
		if !onlyHeaders || isUsefulForOnlyHeaders(nr) {
			t.nameRecords = append(t.nameRecords, nr)
		}
	}

	for _, nr := range t.nameRecords {
		// don't try to read invalid offsets, see PDFBOX-2608
		if int64(nr.StringOffset()) > t.Length() {
			nr.SetString("")
			continue
		}
		r.seek(t.Offset() + 2*3 + int64(numberOfNameRecords)*2*6 + int64(nr.StringOffset()))
		raw := r.bytes(nr.StringLength())
		if r.err != nil {
			return r.err
		}
		nr.SetString(decodeString(raw, charsetOf(nr)))
	}

	t.lookupTable = make(map[nameKey]string, len(t.nameRecords))
	t.fillLookupTable()
	t.readInterestingStrings()
	return nil
}

// isUsefulForOnlyHeaders reports whether a record is one of the few the
// header-only read keeps.
func isUsefulForOnlyHeaders(nr *NameRecord) bool {
	nameID := nr.NameID()
	if nameID == NameFontFamilyName || nameID == NameFontSubFamilyName ||
		nameID == NamePostScriptName {
		languageID := nr.LanguageID()
		return languageID == LanguageUnicode ||
			languageID == LanguageMacintoshEnglish ||
			languageID == LanguageWindowsENUS
	}
	return false
}

// The charsets a name string is decoded with.
const (
	charsetISO88591 = iota
	charsetUTF16
	charsetUSASCII
	charsetUTF16BE
)

// charsetOf returns which charset the string of the record is in.
func charsetOf(nr *NameRecord) int {
	platform := nr.PlatformID()
	encoding := nr.PlatformEncodingID()
	charset := charsetISO88591
	if platform == PlatformWindows &&
		(encoding == EncodingWindowsSymbol || encoding == EncodingWindowsUnicodeBMP) {
		charset = charsetUTF16
	} else if platform == PlatformUnicode {
		charset = charsetUTF16
	} else if platform == PlatformISO {
		switch encoding {
		case 0:
			charset = charsetUSASCII
		case 1:
			// not sure is this is correct??
			charset = charsetUTF16BE
		}
	}
	return charset
}

// decodeString decodes the bytes of a name string.
//
// Java hands the bytes to new String(byte[], Charset), which replaces anything
// malformed with U+FFFD rather than failing; the port does the same.
func decodeString(data []byte, charset int) string {
	switch charset {
	case charsetUSASCII:
		var b strings.Builder
		for _, c := range data {
			if c <= 0x7F {
				b.WriteByte(c)
			} else {
				b.WriteRune(utf8.RuneError)
			}
		}
		return b.String()
	case charsetUTF16:
		// Java's UTF_16 honours a byte order mark and falls back to big endian.
		if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
			return decodeUTF16(data[2:], true)
		}
		if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
			return decodeUTF16(data[2:], false)
		}
		return decodeUTF16(data, true)
	case charsetUTF16BE:
		return decodeUTF16(data, true)
	default:
		var b strings.Builder
		for _, c := range data {
			b.WriteRune(rune(c))
		}
		return b.String()
	}
}

// decodeUTF16 decodes UTF-16 in the given byte order.
func decodeUTF16(data []byte, bigEndian bool) string {
	units := make([]uint16, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		if bigEndian {
			units = append(units, uint16(data[i])<<8|uint16(data[i+1]))
		} else {
			units = append(units, uint16(data[i+1])<<8|uint16(data[i]))
		}
	}
	decoded := string(utf16.Decode(units))
	if len(data)%2 != 0 {
		// A truncated string ends with the replacement character, as it does in
		// Java.
		decoded += string(utf8.RuneError)
	}
	return decoded
}

// fillLookupTable indexes the records by what they can be looked up on.
func (t *NamingTable) fillLookupTable() {
	for _, nr := range t.nameRecords {
		t.lookupTable[nameKey{
			nameID:     nr.NameID(),
			platformID: nr.PlatformID(),
			encodingID: nr.PlatformEncodingID(),
			languageID: nr.LanguageID(),
		}] = nr.String()
	}
}

// readInterestingStrings picks out the three names the rest of the library asks
// for by name rather than by ID.
func (t *NamingTable) readInterestingStrings() {
	t.fontFamily, _ = t.englishName(NameFontFamilyName)
	t.fontSubFamily, _ = t.englishName(NameFontSubFamilyName)

	// extract PostScript name, only these two formats are valid
	psName, ok := t.GetName(NamePostScriptName, PlatformMacintosh,
		EncodingMacintoshRoman, LanguageMacintoshEnglish)
	if !ok {
		psName, ok = t.GetName(NamePostScriptName, PlatformWindows,
			EncodingWindowsUnicodeBMP, LanguageWindowsENUS)
	}
	if ok {
		psName = strings.TrimSpace(psName)
	}
	t.psName = psName
}

// englishName returns an English name by best effort.
func (t *NamingTable) englishName(nameID int) (string, bool) {
	// Unicode, Full, BMP, 1.1, 1.0
	for i := 4; i >= 0; i-- {
		if nameUni, ok := t.GetName(nameID, PlatformUnicode, i, LanguageUnicode); ok {
			return nameUni, true
		}
	}

	// Windows, Unicode BMP, EN-US
	if nameWin, ok := t.GetName(nameID, PlatformWindows,
		EncodingWindowsUnicodeBMP, LanguageWindowsENUS); ok {
		return nameWin, true
	}

	// Macintosh, Roman, English
	return t.GetName(nameID, PlatformMacintosh,
		EncodingMacintoshRoman, LanguageMacintoshEnglish)
}

// GetName returns a name from the table. The second result is false where the
// table has no such name, which is the null Java returns.
func (t *NamingTable) GetName(nameID, platformID, encodingID, languageID int) (string, bool) {
	name, ok := t.lookupTable[nameKey{
		nameID:     nameID,
		platformID: platformID,
		encodingID: encodingID,
		languageID: languageID,
	}]
	return name, ok
}

// NameRecords returns every record of the table.
func (t *NamingTable) NameRecords() []*NameRecord {
	return t.nameRecords
}

// FontFamily returns the family the font belongs to.
func (t *NamingTable) FontFamily() string { return t.fontFamily }

// FontSubFamily returns which member of the family the font is.
func (t *NamingTable) FontSubFamily() string { return t.fontSubFamily }

// PostScriptName returns the PostScript name of the font.
func (t *NamingTable) PostScriptName() string { return t.psName }
