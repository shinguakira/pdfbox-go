package form

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// The bits of the /Ff flags a text field carries. Java declares them private.
const (
	flagMultiline           = 1 << 12
	flagPassword            = 1 << 13
	flagFileSelect          = 1 << 20
	flagTextDoNotSpellCheck = 1 << 22
	flagDoNotScroll         = 1 << 23
	flagComb                = 1 << 24
	flagRichText            = 1 << 25
)

// PDTextField is a field the user types text into.
//
// Port of PDTextField, which Java declares final.
type PDTextField struct {
	PDVariableText
}

var _ PDField = (*PDTextField)(nil)

// NewPDTextField creates a text field in the given form.
func NewPDTextField(acroForm *PDAcroForm) *PDTextField {
	f := &PDTextField{}
	f.initVariableText(f, acroForm)
	f.FieldDictionary().SetItem(cos.FT, cos.Tx)
	return f
}

// NewPDTextFieldOf creates one over the given dictionary. Java declares the
// constructor package-private.
func NewPDTextFieldOf(acroForm *PDAcroForm, field *cos.Dictionary,
	parent *PDNonTerminalField) *PDTextField {
	f := &PDTextField{}
	f.initVariableTextOf(f, acroForm, field, parent)
	return f
}

// IsMultiline reports whether the field holds more than one line.
func (f *PDTextField) IsMultiline() bool {
	return f.FieldDictionary().GetFlag(cos.Ff, flagMultiline)
}

// SetMultiline sets whether the field holds more than one line.
func (f *PDTextField) SetMultiline(multiline bool) {
	f.FieldDictionary().SetFlag(cos.Ff, flagMultiline, multiline)
}

// IsPassword reports whether the field hides what is typed.
func (f *PDTextField) IsPassword() bool {
	return f.FieldDictionary().GetFlag(cos.Ff, flagPassword)
}

// SetPassword sets whether the field hides what is typed.
func (f *PDTextField) SetPassword(password bool) {
	f.FieldDictionary().SetFlag(cos.Ff, flagPassword, password)
}

// IsFileSelect reports whether the value is the path of a file.
func (f *PDTextField) IsFileSelect() bool {
	return f.FieldDictionary().GetFlag(cos.Ff, flagFileSelect)
}

// SetFileSelect sets whether the value is the path of a file.
func (f *PDTextField) SetFileSelect(fileSelect bool) {
	f.FieldDictionary().SetFlag(cos.Ff, flagFileSelect, fileSelect)
}

// DoNotSpellCheck reports whether the value is spell checked.
func (f *PDTextField) DoNotSpellCheck() bool {
	return f.FieldDictionary().GetFlag(cos.Ff, flagTextDoNotSpellCheck)
}

// SetDoNotSpellCheck sets whether the value is spell checked.
func (f *PDTextField) SetDoNotSpellCheck(doNotSpellCheck bool) {
	f.FieldDictionary().SetFlag(cos.Ff, flagTextDoNotSpellCheck, doNotSpellCheck)
}

// DoNotScroll reports whether the field scrolls when it is full.
func (f *PDTextField) DoNotScroll() bool {
	return f.FieldDictionary().GetFlag(cos.Ff, flagDoNotScroll)
}

// SetDoNotScroll sets whether the field scrolls when it is full.
func (f *PDTextField) SetDoNotScroll(doNotScroll bool) {
	f.FieldDictionary().SetFlag(cos.Ff, flagDoNotScroll, doNotScroll)
}

// IsComb reports whether the field is divided into equal cells.
func (f *PDTextField) IsComb() bool { return f.FieldDictionary().GetFlag(cos.Ff, flagComb) }

// SetComb sets whether the field is divided into equal cells.
func (f *PDTextField) SetComb(comb bool) { f.FieldDictionary().SetFlag(cos.Ff, flagComb, comb) }

// IsRichText reports whether the value is rich text.
func (f *PDTextField) IsRichText() bool { return f.FieldDictionary().GetFlag(cos.Ff, flagRichText) }

// SetRichText sets whether the value is rich text.
func (f *PDTextField) SetRichText(richText bool) {
	f.FieldDictionary().SetFlag(cos.Ff, flagRichText, richText)
}

// MaxLen returns the /MaxLen of the field.
func (f *PDTextField) MaxLen() int { return f.FieldDictionary().GetInt(cos.MaxLen) }

// SetMaxLen sets the /MaxLen of the field.
func (f *PDTextField) SetMaxLen(maxLen int) { f.FieldDictionary().SetInt(cos.MaxLen, maxLen) }

// SetValue sets the value of the field.
func (f *PDTextField) SetValue(value string) error {
	f.FieldDictionary().SetString(cos.V, value)
	return f.applyChange()
}

// SetDefaultValue sets the /DV of the field.
func (f *PDTextField) SetDefaultValue(value string) {
	f.FieldDictionary().SetString(cos.DV, value)
}

// Value returns the value of the field.
func (f *PDTextField) Value() string {
	return f.stringOrStream(f.inheritableAttribute(cos.V))
}

// DefaultValue returns the /DV of the field.
func (f *PDTextField) DefaultValue() string {
	return f.stringOrStream(f.inheritableAttribute(cos.DV))
}

// ValueAsString returns the value of the field as a string.
func (f *PDTextField) ValueAsString() string { return f.Value() }

// constructAppearances draws the value of the field.
func (f *PDTextField) constructAppearances() error {
	apHelper, err := newAppearanceGeneratorHelper(&f.PDVariableText)
	if err != nil {
		return err
	}
	return apHelper.setAppearanceValue(f.Value())
}
