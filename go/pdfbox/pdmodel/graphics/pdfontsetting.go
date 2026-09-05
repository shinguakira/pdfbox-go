package graphics

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// PDFontSetting is the font and size an extended graphics state carries.
//
// Port of PDFontSetting.
type PDFontSetting struct {
	fontSetting *cos.Array
}

var _ common.COSObjectable = (*PDFontSetting)(nil)

// NewPDFontSetting builds a setting with no font and a size of one.
func NewPDFontSetting() *PDFontSetting {
	fontSetting := cos.NewArray()
	fontSetting.Add(nil)
	fontSetting.Add(cos.NewFloat(1))
	return &PDFontSetting{fontSetting: fontSetting}
}

// NewPDFontSettingOf builds one over the given array.
func NewPDFontSettingOf(fs *cos.Array) *PDFontSetting {
	return &PDFontSetting{fontSetting: fs}
}

// COSObject returns the array below this setting.
func (s *PDFontSetting) COSObject() cos.Base { return s.fontSetting }

// Font returns the font of this setting, or nil.
func (s *PDFontSetting) Font() (font.PDFont, error) {
	if dictionary, isDictionary := asDictionary(s.fontSetting.GetObject(0)); isDictionary {
		return font.CreateFont(dictionary, nil)
	}
	return nil, nil
}

// asDictionary is Java's instanceof COSDictionary, which a COSStream also
// satisfies.
func asDictionary(base cos.Base) (*cos.Dictionary, bool) {
	switch value := base.(type) {
	case *cos.Stream:
		return &value.Dictionary, true
	case *cos.Dictionary:
		return value, true
	}
	return nil, false
}

// SetFont sets the font of this setting.
func (s *PDFontSetting) SetFont(value font.PDFont) {
	if value == nil {
		s.fontSetting.Set(0, nil)
		return
	}
	s.fontSetting.Set(0, value.COSObject())
}

// FontSize returns the size of this setting.
//
// Java casts the entry to COSNumber without a check, so a setting whose second
// entry is missing or is not a number throws; the port asserts the same way.
func (s *PDFontSetting) FontSize() float32 {
	return s.fontSetting.Get(1).(cos.Number).FloatValue()
}

// SetFontSize sets the size of this setting.
func (s *PDFontSetting) SetFontSize(size float32) {
	s.fontSetting.Set(1, cos.NewFloat(size))
}
