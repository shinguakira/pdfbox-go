package font

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// CreateFont returns the font the given dictionary describes.
//
// Port of org.apache.pdfbox.pdmodel.font.PDFontFactory.createFont.
//
// Type 0 and the two CID font types are read by a later step of this slice; a
// dictionary naming one of them is reported rather than read as something else.
// The /Subtype repair Java does for a Type 0 font goes with them, since it only
// matters to the font it repairs. See migration/STATUS.md.
func CreateFont(dictionary *cos.Dictionary, resourceCache ResourceCache) (PDFont, error) {
	fontType := dictionary.GetCOSNameDefault(cos.Type, cos.Font)
	if !cos.Font.Equals(fontType) {
		// Expected 'Font' dictionary but found something else; Java logs and
		// carries on.
		_ = fontType
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
		return NewPDType0Font(dictionary, resourceCache)
	case cos.CIDFontType0.Equals(subType):
		return nil, fmt.Errorf("font: Type 0 descendant font not allowed")
	case cos.CIDFontType2.Equals(subType):
		return nil, fmt.Errorf("font: Type 2 descendant font not allowed")
	default:
		// assuming Type 1 font (see PDFBOX-1988) because it seems that Adobe
		// Reader does this however, we may need more sophisticated logic perhaps
		// looking at the FontFile
		return NewPDType1FontFromDictionary(dictionary, resourceCache)
	}
}

// CreateDescendantFont returns the CIDFont the given dictionary describes,
// which is the descendant of a Type 0 font.
//
// Port of org.apache.pdfbox.pdmodel.font.PDFontFactory.createDescendantFont.
func CreateDescendantFont(dictionary *cos.Dictionary, resourceCache ResourceCache) (PDCIDFont, error) {
	fontType := dictionary.GetCOSNameDefault(cos.Type, cos.Font)
	if !cos.Font.Equals(fontType) {
		return nil, fmt.Errorf("Expected 'Font' dictionary but found '%s'", fontType.Name())
	}
	subType := dictionary.GetCOSName(cos.Subtype)
	if cos.CIDFontType0.Equals(subType) {
		return NewPDCIDFontType0(dictionary, resourceCache)
	}
	if cos.CIDFontType2.Equals(subType) {
		return NewPDCIDFontType2(dictionary, resourceCache)
	}
	return nil, fmt.Errorf("Invalid font type: %s", fontType.Name())
}
