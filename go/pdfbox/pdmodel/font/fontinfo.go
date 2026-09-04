package font

import (
	"fmt"
	"strconv"

	"github.com/shinguakira/pdfbox-go/go/fontbox"
)

// FontInfo is what is known about a font on the system.
//
// Port of the abstract class org.apache.pdfbox.pdmodel.font.FontInfo. Every
// method of that class is abstract but three, and those three are final and
// read nothing but the abstract ones, so the port is an interface plus the
// three as package functions.
type FontInfo interface {
	// PostScriptName returns the PostScript name of the font.
	PostScriptName() string

	// Format returns the font's format.
	Format() FontFormat

	// CIDSystemInfo returns the CIDSystemInfo associated with the font, or nil
	// where it has none.
	CIDSystemInfo() *CIDSystemInfo

	// Font returns a new FontBox font instance for the font. Implementors of
	// this method must not cache the return value unless doing so via the
	// current FontCache.
	Font() fontbox.FontBoxFont

	// FamilyClass returns the sFamilyClass field of the "OS/2" table, or -1.
	FamilyClass() int

	// WeightClass returns the usWeightClass field of the "OS/2" table, or -1.
	WeightClass() int

	// CodePageRange1 returns the ulCodePageRange1 field of the "OS/2" table,
	// or 0.
	CodePageRange1() int

	// CodePageRange2 returns the ulCodePageRange2 field of the "OS/2" table,
	// or 0.
	CodePageRange2() int

	// MacStyle returns the macStyle field of the "head" table, or -1.
	MacStyle() int

	// Panose returns the Panose classification of the font, or nil where it
	// has none.
	Panose() *PDPanoseClassification
}

// fontInfoWeightClassAsPanose returns the usWeightClass field as a Panose
// weight.
//
// Port of the final method FontInfo.getWeightClassAsPanose.
func fontInfoWeightClassAsPanose(info FontInfo) int {
	switch info.WeightClass() {
	case -1:
		return 0
	case 0:
		return 0
	case 100:
		return 2
	case 200:
		return 3
	case 300:
		return 4
	case 400:
		return 5
	case 500:
		return 6
	case 600:
		return 7
	case 700:
		return 8
	case 800:
		return 9
	case 900:
		return 10
	default:
		return 0
	}
}

// fontInfoCodePageRange returns the ulCodePageRange1 and ulCodePageRange2
// fields of the "OS/2" table as one value, or 0.
//
// Port of the final method FontInfo.getCodePageRange.
func fontInfoCodePageRange(info FontInfo) int64 {
	range1 := int64(uint32(info.CodePageRange1()))
	range2 := int64(uint32(info.CodePageRange2()))
	return range2<<32 | range1
}

// todo: 'post' table for Italic. Also: OS/2 fsSelection for italic/bold.
// todo: ulUnicodeRange too?

// fontInfoString is what the log prints for a font.
//
// Port of FontInfo.toString. Java's string concatenation writes "null" for a
// missing CIDSystemInfo, which the port spells out because a Go String method
// on a nil pointer would panic instead.
func fontInfoString(info FontInfo) string {
	cid := "null"
	if ros := info.CIDSystemInfo(); ros != nil {
		cid = ros.String()
	}
	return fmt.Sprintf("%s (%s, mac: 0x%s, os/2: 0x%s, cid: %s)",
		info.PostScriptName(), info.Format(), toHexString(info.MacStyle()),
		toHexString(info.FamilyClass()), cid)
}

// toHexString is Java's Integer.toHexString, which reads the int as unsigned;
// -1, which both fields use for "absent", prints as "ffffffff" and not as "-1".
func toHexString(value int) string {
	return strconv.FormatUint(uint64(uint32(value)), 16)
}
