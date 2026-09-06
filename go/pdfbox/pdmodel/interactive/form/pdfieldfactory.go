package form

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// The /FT field types. Java declares them private.
const (
	fieldTypeText      = "Tx"
	fieldTypeButton    = "Btn"
	fieldTypeChoice    = "Ch"
	fieldTypeSignature = "Sig"
)

// CreateField builds the field the given dictionary describes, and answers nil
// where it describes none.
//
// Port of the static PDFieldFactory.createField.
func CreateField(form *PDAcroForm, field *cos.Dictionary,
	parent *PDNonTerminalField) PDField {
	// Test if we have a non terminal field first as it might have
	// properties which do apply to other fields
	// A non terminal fields has Kids entries which do have
	// a field name (other than annotations)
	if field.ContainsKey(cos.Kids) {
		kids := field.GetCOSArray(cos.Kids)
		if kids != nil && !kids.IsEmpty() {
			for i := 0; i < kids.Size(); i++ {
				kid, isDictionary := kids.GetObject(i).(*cos.Dictionary)
				if isDictionary && kid.GetString(cos.T, "") != "" {
					return NewPDNonTerminalFieldOf(form, field, parent)
				}
			}
		}
	}

	switch findFieldType(field, map[*cos.Dictionary]bool{}) {
	case fieldTypeChoice:
		return createChoiceSubType(form, field, parent)
	case fieldTypeText:
		return NewPDTextFieldOf(form, field, parent)
	case fieldTypeSignature:
		return NewPDSignatureFieldOf(form, field, parent)
	case fieldTypeButton:
		return createButtonSubType(form, field, parent)
	}
	// an erroneous non-field object, see PDFBOX-2885
	return nil
}

// createChoiceSubType builds the choice field the flags describe. Java declares
// it private.
func createChoiceSubType(form *PDAcroForm, field *cos.Dictionary,
	parent *PDNonTerminalField) PDField {
	flags := field.GetIntDefault(cos.Ff, 0)
	if flags&flagCombo != 0 {
		return NewPDComboBoxOf(form, field, parent)
	}
	return NewPDListBoxOf(form, field, parent)
}

// createButtonSubType builds the button the flags describe. Java declares it
// private.
func createButtonSubType(form *PDAcroForm, field *cos.Dictionary,
	parent *PDNonTerminalField) PDField {
	flags := field.GetIntDefault(cos.Ff, 0)
	// BJL: I have found that the radio flag bit is not always set
	// and that sometimes there is just a kids dictionary.
	// so, if there is a kids dictionary then it must be a radio button group.
	switch {
	case flags&flagRadio != 0:
		return NewPDRadioButtonOf(form, field, parent)
	case flags&flagPushButton != 0:
		return NewPDPushButtonOf(form, field, parent)
	}
	return NewPDCheckBoxOf(form, field, parent)
}

// findFieldType returns the /FT of the dictionary or of the nearest parent that
// has one. Java declares it private.
func findFieldType(dic *cos.Dictionary, seen map[*cos.Dictionary]bool) string {
	if seen[dic] {
		// PDFBOX-5896: avoid endless recursion
		return ""
	}
	seen[dic] = true
	retval := dic.GetNameAsString(cos.FT, "")
	if retval == "" {
		if base := dic.GetCOSDictionary2(cos.Parent, cos.P); base != nil {
			return findFieldType(base, seen)
		}
		return ""
	}
	delete(seen, dic)
	return retval
}
