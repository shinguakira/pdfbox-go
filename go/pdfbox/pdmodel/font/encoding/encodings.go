package encoding

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// StandardEncoding is the Adobe standard encoding.
//
// Port of org.apache.pdfbox.pdmodel.font.encoding.StandardEncoding.
type StandardEncoding struct {
	encodingBase
}

// StandardEncodingInstance is the one instance of the standard encoding.
var StandardEncodingInstance = newStandardEncoding()

func newStandardEncoding() *StandardEncoding {
	return &StandardEncoding{encodingBase: fromTable(standardEncodingTable)}
}

// COSObject returns the name of the standard encoding.
func (e *StandardEncoding) COSObject() cos.Base { return cos.StandardEncoding }

// EncodingName returns the name of this encoding.
func (e *StandardEncoding) EncodingName() string { return "StandardEncoding" }

// WinAnsiEncoding is the Windows ANSI encoding.
//
// Port of org.apache.pdfbox.pdmodel.font.encoding.WinAnsiEncoding.
type WinAnsiEncoding struct {
	encodingBase
}

// WinAnsiEncodingInstance is the one instance of the Windows ANSI encoding.
var WinAnsiEncodingInstance = newWinAnsiEncoding()

func newWinAnsiEncoding() *WinAnsiEncoding {
	e := &WinAnsiEncoding{encodingBase: fromTable(winAnsiEncodingTable)}
	// From the PDF specification:
	// In WinAnsiEncoding, all unused codes greater than 40 map to the bullet
	// character.
	for i := 0o41; i <= 255; i++ {
		if _, ok := e.codeToName[i]; !ok {
			e.add(i, "bullet")
		}
	}
	return e
}

// COSObject returns the name of the Windows ANSI encoding.
func (e *WinAnsiEncoding) COSObject() cos.Base { return cos.WinAnsiEncoding }

// EncodingName returns the name of this encoding.
func (e *WinAnsiEncoding) EncodingName() string { return "WinAnsiEncoding" }

// MacRomanEncoding is the Mac Roman encoding.
//
// Port of org.apache.pdfbox.pdmodel.font.encoding.MacRomanEncoding.
type MacRomanEncoding struct {
	encodingBase
}

// MacRomanEncodingInstance is the one instance of the Mac Roman encoding.
var MacRomanEncodingInstance = newMacRomanEncoding()

func newMacRomanEncoding() *MacRomanEncoding {
	return &MacRomanEncoding{encodingBase: fromTable(macRomanEncodingTable)}
}

// COSObject returns the name of the Mac Roman encoding.
func (e *MacRomanEncoding) COSObject() cos.Base { return cos.MacRomanEncoding }

// EncodingName returns the name of this encoding.
func (e *MacRomanEncoding) EncodingName() string { return "MacRomanEncoding" }

// MacOSRomanEncoding is the Mac Roman encoding as Mac OS itself uses it, which
// adds sixteen glyphs to it.
//
// Port of org.apache.pdfbox.pdmodel.font.encoding.MacOSRomanEncoding. Java
// extends MacRomanEncoding and its constructor adds the extra rows on top; the
// port reads both tables in the same order.
type MacOSRomanEncoding struct {
	MacRomanEncoding
}

// MacOSRomanEncodingInstance is the one instance of the Mac OS Roman encoding.
var MacOSRomanEncodingInstance = newMacOSRomanEncoding()

func newMacOSRomanEncoding() *MacOSRomanEncoding {
	e := &MacOSRomanEncoding{
		MacRomanEncoding: MacRomanEncoding{encodingBase: fromTable(macRomanEncodingTable)},
	}
	// differences and additions to MacRomanEncoding
	for _, entry := range macOSRomanEncodingTable {
		e.add(entry.code, entry.name)
	}
	return e
}

// COSObject returns nothing: the Mac OS Roman encoding has no name of its own.
func (e *MacOSRomanEncoding) COSObject() cos.Base { return nil }

// MacExpertEncoding is the Mac Expert encoding.
//
// Port of org.apache.pdfbox.pdmodel.font.encoding.MacExpertEncoding.
type MacExpertEncoding struct {
	encodingBase
}

// MacExpertEncodingInstance is the one instance of the Mac Expert encoding.
var MacExpertEncodingInstance = newMacExpertEncoding()

func newMacExpertEncoding() *MacExpertEncoding {
	return &MacExpertEncoding{encodingBase: fromTable(macExpertEncodingTable)}
}

// COSObject returns the name of the Mac Expert encoding.
func (e *MacExpertEncoding) COSObject() cos.Base { return cos.MacExpertEncoding }

// EncodingName returns the name of this encoding.
func (e *MacExpertEncoding) EncodingName() string { return "MacExpertEncoding" }

// SymbolEncoding is the built-in encoding of the Symbol font.
//
// Port of org.apache.pdfbox.pdmodel.font.encoding.SymbolEncoding.
type SymbolEncoding struct {
	encodingBase
}

// SymbolEncodingInstance is the one instance of the Symbol encoding.
var SymbolEncodingInstance = newSymbolEncoding()

func newSymbolEncoding() *SymbolEncoding {
	return &SymbolEncoding{encodingBase: fromTable(symbolEncodingTable)}
}

// COSObject returns the name of the Symbol encoding.
func (e *SymbolEncoding) COSObject() cos.Base { return cos.GetPDFName("SymbolEncoding") }

// EncodingName returns the name of this encoding.
func (e *SymbolEncoding) EncodingName() string { return "SymbolEncoding" }

// ZapfDingbatsEncoding is the built-in encoding of the Zapf Dingbats font.
//
// Port of org.apache.pdfbox.pdmodel.font.encoding.ZapfDingbatsEncoding.
type ZapfDingbatsEncoding struct {
	encodingBase
}

// ZapfDingbatsEncodingInstance is the one instance of the Zapf Dingbats
// encoding.
var ZapfDingbatsEncodingInstance = newZapfDingbatsEncoding()

func newZapfDingbatsEncoding() *ZapfDingbatsEncoding {
	return &ZapfDingbatsEncoding{encodingBase: fromTable(zapfDingbatsEncodingTable)}
}

// COSObject returns the name of the Zapf Dingbats encoding.
func (e *ZapfDingbatsEncoding) COSObject() cos.Base {
	return cos.GetPDFName("ZapfDingbatsEncoding")
}

// EncodingName returns the name of this encoding.
func (e *ZapfDingbatsEncoding) EncodingName() string { return "ZapfDingbatsEncoding" }

// The seven static encodings are all encodings.
var (
	_ Encoding = (*StandardEncoding)(nil)
	_ Encoding = (*WinAnsiEncoding)(nil)
	_ Encoding = (*MacRomanEncoding)(nil)
	_ Encoding = (*MacOSRomanEncoding)(nil)
	_ Encoding = (*MacExpertEncoding)(nil)
	_ Encoding = (*SymbolEncoding)(nil)
	_ Encoding = (*ZapfDingbatsEncoding)(nil)
)
