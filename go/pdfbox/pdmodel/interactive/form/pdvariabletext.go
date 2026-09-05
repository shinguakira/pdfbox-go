package form

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// The quaddings a variable text field can have.
//
// Port of the QUADDING_ constants of PDVariableText.
const (
	QuaddingLeft     = 0
	QuaddingCentered = 1
	QuaddingRight    = 2
)

// PDVariableText is a field whose value is text a reader lays out.
//
// Port of the abstract PDVariableText.
type PDVariableText struct {
	PDTerminalField
}

// initVariableText is the package-private PDVariableText(PDAcroForm)
// constructor.
func (f *PDVariableText) initVariableText(self PDField, acroForm *PDAcroForm) {
	f.initTerminalField(self, acroForm)
}

// initVariableTextOf is the package-private PDVariableText(PDAcroForm,
// COSDictionary, PDNonTerminalField) constructor.
func (f *PDVariableText) initVariableTextOf(self PDField, acroForm *PDAcroForm,
	field *cos.Dictionary, parent *PDNonTerminalField) {
	f.initTerminalFieldOf(self, acroForm, field, parent)
}

// DefaultAppearance returns the /DA string of the field, or the empty string
// where it has none, which is the null Java answers.
func (f *PDVariableText) DefaultAppearance() string {
	base, isString := f.inheritableAttribute(cos.DA).(*cos.StringObj)
	if !isString {
		return ""
	}
	return base.Value()
}

// defaultAppearanceString reads the /DA string of the field. Java declares it
// package-private.
func (f *PDVariableText) defaultAppearanceString() (*pdDefaultAppearanceString, error) {
	da, _ := f.inheritableAttribute(cos.DA).(*cos.StringObj)
	dr := f.AcroForm().DefaultResources()
	return newPDDefaultAppearanceString(da, dr)
}

// SetDefaultAppearance sets the /DA string of the field, and of the kid widgets
// that carry one of their own.
func (f *PDVariableText) SetDefaultAppearance(daValue string) {
	f.FieldDictionary().SetString(cos.DA, daValue)

	// PDFBOX-5797: Sejda files have a /DA entry in kid widgets
	if f.FieldDictionary().ContainsKey(cos.Kids) {
		for _, widget := range f.Widgets() {
			widgetDict := widget.AnnotationDictionary()
			if widgetDict.ContainsKey(cos.DA) {
				widgetDict.SetString(cos.DA, daValue)
			}
		}
	}
}

// DefaultStyleString returns the /DS style of the field.
func (f *PDVariableText) DefaultStyleString() string {
	return f.FieldDictionary().GetString(cos.DS, "")
}

// SetDefaultStyleString sets the /DS style of the field, and removes it where
// the value is empty, which is the null Java removes on.
func (f *PDVariableText) SetDefaultStyleString(defaultStyleString string) {
	if defaultStyleString != "" {
		f.FieldDictionary().SetItem(cos.DS, cos.NewStringObj(defaultStyleString))
		return
	}
	f.FieldDictionary().RemoveItem(cos.DS)
}

// Q returns the /Q quadding of the field.
//
// Java casts the entry to COSNumber without a check; the port asserts the same
// way.
func (f *PDVariableText) Q() int {
	retval := 0
	if base := f.inheritableAttribute(cos.Q); base != nil {
		retval = base.(cos.Number).IntValue()
	}
	return retval
}

// SetQ sets the /Q quadding of the field.
func (f *PDVariableText) SetQ(q int) {
	f.FieldDictionary().SetInt(cos.Q, q)
}

// RichTextValue returns the /RV rich text value of the field.
func (f *PDVariableText) RichTextValue() string {
	return f.stringOrStream(f.inheritableAttribute(cos.RV))
}

// SetRichTextValue sets the /RV rich text value of the field, and removes it
// where the value is empty.
func (f *PDVariableText) SetRichTextValue(richTextValue string) {
	if richTextValue != "" {
		f.FieldDictionary().SetItem(cos.RV, cos.NewStringObj(richTextValue))
		return
	}
	f.FieldDictionary().RemoveItem(cos.RV)
}

// stringOrStream reads a value that may be written as a string or as a stream.
// Java declares it protected and final.
func (f *PDVariableText) stringOrStream(base cos.Base) string {
	switch value := base.(type) {
	case *cos.StringObj:
		return value.Value()
	case *cos.Stream:
		return value.ToTextString()
	}
	return ""
}
