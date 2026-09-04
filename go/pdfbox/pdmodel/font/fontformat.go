package font

import "strconv"

// FontFormat is a font file format.
//
// Port of org.apache.pdfbox.pdmodel.font.FontFormat.
//
// Java's enum has no zero value; Go's does, and it is TTF. Nothing relies on
// that: every field of this type is assigned before it is read, the same as the
// Java reference it stands for.
type FontFormat int

const (
	// FontFormatTTF is a TrueType font.
	FontFormatTTF FontFormat = iota

	// FontFormatOTF is an OpenType font.
	FontFormatOTF

	// FontFormatPFB is a Type 1 (binary) font.
	FontFormatPFB
)

// String returns the name of the constant, which is Java's Enum.toString and
// what the on-disk font cache stores.
func (f FontFormat) String() string {
	switch f {
	case FontFormatTTF:
		return "TTF"
	case FontFormatOTF:
		return "OTF"
	case FontFormatPFB:
		return "PFB"
	default:
		return "FontFormat(" + strconv.Itoa(int(f)) + ")"
	}
}

// ParseFontFormat returns the format the given name stands for.
//
// Java's FontFormat.valueOf throws IllegalArgumentException for a name it does
// not know; the second result says whether the name was one, which is what the
// single caller -- the disk cache reader -- needs to skip a damaged line.
func ParseFontFormat(name string) (FontFormat, bool) {
	switch name {
	case "TTF":
		return FontFormatTTF, true
	case "OTF":
		return FontFormatOTF, true
	case "PFB":
		return FontFormatPFB, true
	default:
		return FontFormatTTF, false
	}
}
