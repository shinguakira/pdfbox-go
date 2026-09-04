package font

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// The font program headers createFont sniffs a damaged /Subtype against.
const (
	fontType1C        = "Type1C"
	fontOpenType      = "OTTO"
	fontTTFCollection = "ttcf"
	fontTrueType      = "true"
)

var ttfHeader = []byte{0, 1, 0, 0}

// fontType is what the font program says it is, paired with the /Subtype a
// CIDFont of that kind would carry.
//
// Port of the private static class PDFontFactory.FontType.
type fontType struct {
	fontTypeName *cos.Name
	subtype      *cos.Name
}

var (
	cidType0Types = []string{cos.Type1.Name(), fontType1C}
	cidType2Types = []string{cos.TrueType.Name(), cos.OpenType.Name()}
)

// newFontTypeOfSubtypeString is Java's FontType(COSName, String).
func newFontTypeOfSubtypeString(name *cos.Name, subtypeString string) fontType {
	t := fontType{fontTypeName: name}
	switch {
	case containsString(cidType0Types, subtypeString):
		t.subtype = cos.CIDFontType0
	case containsString(cidType2Types, subtypeString):
		t.subtype = cos.CIDFontType2
	}
	return t
}

// newFontTypeOf is Java's FontType(COSName), whose subtype is null.
func newFontTypeOf(name *cos.Name) fontType { return fontType{fontTypeName: name} }

// Subtype returns the CIDFont /Subtype this font program would carry.
func (t fontType) Subtype() *cos.Name { return t.subtype }

// isCIDSubtype reports whether this is a composite font of the given CIDFont
// subtype.
func (t fontType) isCIDSubtype(cidSubtype *cos.Name) bool {
	if !cos.Type0.Equals(t.fontTypeName) {
		return false
	}
	return t.subtype != nil && t.subtype.Equals(cidSubtype)
}

func containsString(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

// CreateFont returns the font the given dictionary describes.
//
// Port of org.apache.pdfbox.pdmodel.font.PDFontFactory.createFont.
func CreateFont(dictionary *cos.Dictionary, resourceCache ResourceCache) (PDFont, error) {
	fontTypeName := dictionary.GetCOSNameDefault(cos.Type, cos.Font)
	if !cos.Font.Equals(fontTypeName) {
		slog.Error("Expected 'Font' dictionary but found other type",
			"type", fontTypeName.Name())
	}

	subType := dictionary.GetCOSName(cos.Subtype)
	switch {
	case cos.Type1.Equals(subType):
		fd := dictionary.GetCOSDictionary(cos.FontDescriptor)
		if fd != nil && fd.ContainsKey(cos.FontFile3) {
			return NewPDType1CFontFromDictionary(dictionary, resourceCache)
		}
		return NewPDType1FontFromDictionary(dictionary, resourceCache)
	case cos.MMType1.Equals(subType):
		fd := dictionary.GetCOSDictionary(cos.FontDescriptor)
		if fd != nil && fd.ContainsKey(cos.FontFile3) {
			return NewPDType1CFontFromDictionary(dictionary, resourceCache)
		}
		return NewPDMMType1Font(dictionary)
	case cos.TrueType.Equals(subType):
		return NewPDTrueTypeFontFromDictionary(dictionary, resourceCache)
	case cos.Type3.Equals(subType):
		return NewPDType3Font(dictionary, resourceCache)
	case cos.Type0.Equals(subType):
		fontDescriptor := getFontDescriptor(dictionary)
		fontTypeFromFont, err := getFontTypeFromFont(fontDescriptor, subType)
		if err != nil {
			return nil, err
		}
		if fontTypeFromFont != nil {
			descendantFont := getDescendantFont(dictionary)
			var descFontType *cos.Name
			if descendantFont != nil {
				descFontType = descendantFont.GetCOSName(cos.Subtype)
			}
			if descFontType != nil && !fontTypeFromFont.isCIDSubtype(descFontType) {
				fixType0Subtype(descendantFont, fontDescriptor, fontTypeFromFont.Subtype())
			}
		}
		return NewPDType0Font(dictionary, resourceCache)
	case cos.CIDFontType0.Equals(subType):
		return nil, fmt.Errorf("Type 0 descendant font not allowed")
	case cos.CIDFontType2.Equals(subType):
		return nil, fmt.Errorf("Type 2 descendant font not allowed")
	default:
		// assuming Type 1 font (see PDFBOX-1988) because it seems that Adobe
		// Reader does this however, we may need more sophisticated logic perhaps
		// looking at the FontFile
		slog.Warn("Invalid font subtype", "subtype", subType)
		return NewPDType1FontFromDictionary(dictionary, resourceCache)
	}
}

// fixType0Subtype rewrites the descendant font's /Subtype to the one the
// embedded font program actually is, moving the font file entry with it.
func fixType0Subtype(descendantFont, fontDescriptor *cos.Dictionary, newSubType *cos.Name) {
	slog.Warn("Try to fix different descendant font types for font",
		"font", fontDescriptor.GetNameAsString(cos.FontName, ""))
	if cos.CIDFontType0.Equals(newSubType) &&
		!fontDescriptor.ContainsKey(cos.FontFile3) &&
		fontDescriptor.ContainsKey(cos.FontFile2) {
		fontDescriptor.SetItem(cos.FontFile3, fontDescriptor.GetItem(cos.FontFile2))
		fontDescriptor.RemoveItem(cos.FontFile2)
	}
	if cos.CIDFontType2.Equals(newSubType) &&
		fontDescriptor.ContainsKey(cos.FontFile3) &&
		!fontDescriptor.ContainsKey(cos.FontFile2) {
		fontDescriptor.SetItem(cos.FontFile2, fontDescriptor.GetItem(cos.FontFile3))
		fontDescriptor.RemoveItem(cos.FontFile3)
	}
	descendantFont.SetItem(cos.Subtype, newSubType)
}

// getFontTypeFromFont returns what the embedded font program is, or nil where
// there is none or its header says nothing.
func getFontTypeFromFont(fontDescriptor *cos.Dictionary,
	fontTypeName *cos.Name) (*fontType, error) {
	fontHeader, err := getFontHeader(fontDescriptor)
	if err != nil {
		return nil, err
	}
	if fontHeader == nil {
		return nil, nil
	}
	isComposite := cos.Type0.Equals(fontTypeName)
	if isTrueTypeFile(fontHeader) || isTrueTypeCollectionFile(fontHeader) {
		if isComposite {
			t := newFontTypeOfSubtypeString(cos.Type0, cos.TrueType.Name())
			return &t, nil
		}
		t := newFontTypeOf(cos.TrueType)
		return &t, nil
	}
	if isOpenTypeFile(fontHeader) {
		if isComposite {
			t := newFontTypeOfSubtypeString(cos.Type0, cos.OpenType.Name())
			return &t, nil
		}
		t := newFontTypeOf(cos.OpenType)
		return &t, nil
	}
	if isType1File(fontHeader) || isPfbFile(fontHeader) {
		if isComposite {
			t := newFontTypeOfSubtypeString(cos.Type0, cos.Type1.Name())
			return &t, nil
		}
		if fontTypeName.Equals(cos.MMType1) {
			t := newFontTypeOfSubtypeString(cos.MMType1, cos.Type1.Name())
			return &t, nil
		}
		t := newFontTypeOf(cos.Type1)
		return &t, nil
	}
	// CFF fonts have a more or less variable header so that the check should be
	// done after all others to avoid wrong classifications
	if isCFFFile(fontHeader) {
		if isComposite {
			t := newFontTypeOfSubtypeString(cos.Type0, fontType1C)
			return &t, nil
		}
		if fontTypeName.Equals(cos.MMType1) {
			t := newFontTypeOfSubtypeString(cos.MMType1, fontType1C)
			return &t, nil
		}
		t := newFontTypeOfSubtypeString(cos.Type1, fontType1C)
		return &t, nil
	}
	return nil, nil
}

func isTrueTypeFile(header []byte) bool {
	return string(header) == string(ttfHeader) || fontTrueType == string(header)
}

func isTrueTypeCollectionFile(header []byte) bool {
	return fontTTFCollection == string(header)
}

func isOpenTypeFile(header []byte) bool {
	return fontOpenType == string(header)
}

func isType1File(header []byte) bool {
	// All Type1 font programs must begin with the comment '%!' (0x25 + 0x21).
	return header[0] == 0x25 && header[1] == 0x21
}

func isPfbFile(header []byte) bool {
	// all PFB fonts start with 0x80 followed by either 0x01 or 0x02
	return header[0] == 0x80 && (header[1] == 0x01 || header[1] == 0x02)
}

func isCFFFile(header []byte) bool {
	// the header consist of 4 values
	// major version, minor version, header size, offset size
	// the major version must be >= 1 and the offset size >= 1 and <= 4
	return header[0] >= 1 && header[3] >= 1 && header[3] <= 4
}

// getFontDescriptor returns the font descriptor of the dictionary, falling back
// on the descendant font's where the dictionary has none of its own.
func getFontDescriptor(dictionary *cos.Dictionary) *cos.Dictionary {
	fontDescriptor := dictionary.GetCOSDictionary(cos.FontDescriptor)
	if fontDescriptor == nil {
		if descendantFont := getDescendantFont(dictionary); descendantFont != nil {
			fontDescriptor = descendantFont.GetCOSDictionary(cos.FontDescriptor)
		}
	}
	return fontDescriptor
}

func getDescendantFont(dictionary *cos.Dictionary) *cos.Dictionary {
	descendantFonts := dictionary.GetCOSArray(cos.DescendantFonts)
	if descendantFonts != nil && descendantFonts.Size() != 0 {
		if descendantFontDict, ok := descendantFonts.GetObject(0).(*cos.Dictionary); ok {
			return descendantFontDict
		}
	}
	return nil
}

// getFontHeader returns the first four bytes of the embedded font program, or
// nil where there is none.
func getFontHeader(fontDescriptor *cos.Dictionary) ([]byte, error) {
	if fontDescriptor == nil {
		return nil, nil
	}
	// COSDictionary.getCOSStream is not ported -- see migration/STATUS.md -- so
	// the resolution and the type test are written out, which is what it does.
	streamAt := func(key *cos.Name) *cos.Stream {
		stream, _ := fontDescriptor.GetDictionaryObject(key).(*cos.Stream)
		return stream
	}
	fontFile := streamAt(cos.FontFile)
	if fontFile == nil {
		fontFile = streamAt(cos.FontFile2)
	}
	if fontFile == nil {
		fontFile = streamAt(cos.FontFile3)
	}
	if fontFile == nil {
		return nil, nil
	}
	const headerLength = 4
	header := make([]byte, headerLength)
	fontView, err := fontFile.CreateView()
	if err != nil {
		// Java catches IOException around the whole read and logs it, leaving
		// the header as far as it got.
		slog.Error("Could not read the font header", "err", err)
		return header, nil
	}
	defer pdfio.CloseQuietly(fontView)
	remainingBytes := headerLength
	for remainingBytes > 0 {
		amountRead, err := fontView.Read(header[headerLength-remainingBytes:])
		if amountRead <= 0 {
			if err != nil && err != io.EOF {
				slog.Error("Could not read the font header", "err", err)
			}
			break
		}
		remainingBytes -= amountRead
	}
	return header, nil
}

// CreateDescendantFont returns the CIDFont the given dictionary describes,
// which is the descendant of a Type 0 font.
//
// Port of org.apache.pdfbox.pdmodel.font.PDFontFactory.createDescendantFont.
func CreateDescendantFont(dictionary *cos.Dictionary, resourceCache ResourceCache) (PDCIDFont, error) {
	fontTypeName := dictionary.GetCOSNameDefault(cos.Type, cos.Font)
	if !cos.Font.Equals(fontTypeName) {
		return nil, fmt.Errorf("Expected 'Font' dictionary but found '%s'", fontTypeName.Name())
	}
	subType := dictionary.GetCOSName(cos.Subtype)
	if cos.CIDFontType0.Equals(subType) {
		return NewPDCIDFontType0(dictionary, resourceCache)
	}
	if cos.CIDFontType2.Equals(subType) {
		return NewPDCIDFontType2(dictionary, resourceCache)
	}
	// Java names the /Type here, not the /Subtype it just failed to match, and
	// concatenates the COSName rather than its name, so the message reads
	// "Invalid font type: COSName{Font}". Ported as written.
	return nil, fmt.Errorf("Invalid font type: %s", fontTypeName)
}
