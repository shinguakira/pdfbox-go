package form

import (
	"bytes"
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/fdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"github.com/shinguakira/pdfbox-go/go/w3c/dom"
)

// ImportFDF sets the value and the flags of the given field from the given FDF
// field.
//
// Port of the package-private PDField.importFDF, along with the two overrides
// that call super first: it is a function over a PDField rather than a method,
// because this package cannot put a method naming FDFField on the PDField
// interface without every implementation of that interface naming it too.
func ImportFDF(field PDField, fdfField *fdf.FDFField) error {
	if err := importFDFBase(field, fdfField); err != nil {
		return err
	}
	switch typed := field.(type) {
	case *PDNonTerminalField:
		return importFDFNonTerminal(typed, fdfField)
	}
	// Every other field is a terminal one.
	return importFDFTerminal(field, fdfField)
}

// importFDFBase is the body of PDField.importFDF.
func importFDFBase(field PDField, fdfField *fdf.FDFField) error {
	fieldValue, err := fdfField.COSValue()
	if err != nil {
		return err
	}

	_, isNonTerminal := field.(*PDNonTerminalField)
	if fieldValue != nil && !isNonTerminal {
		// Java tests `this instanceof PDTerminalField`; every field that is not
		// a non-terminal one is a terminal one.
		switch value := fieldValue.(type) {
		case *cos.Name:
			if err := field.SetValue(value.Name()); err != nil {
				return err
			}
		case *cos.StringObj:
			if err := field.SetValue(value.Value()); err != nil {
				return err
			}
		case *cos.Stream:
			if err := field.SetValue(value.ToTextString()); err != nil {
				return err
			}
		case *cos.Array:
			choice := asChoice(field)
			if choice == nil {
				return fmt.Errorf("Error:Unknown type for field import%v", fieldValue)
			}
			if err := choice.SetValues(stringStringList(value)); err != nil {
				return err
			}
		default:
			return fmt.Errorf("Error:Unknown type for field import%v", fieldValue)
		}
	} else if fieldValue != nil {
		field.FieldDictionary().SetItem(cos.V, fieldValue)
	}

	if ff, hasFf := fdfField.FieldFlags(); hasFf {
		setFieldFlags(field, ff)
	} else {
		// these are suppose to be ignored if the Ff is set.
		setFf, hasSetFf := fdfField.SetFieldFlagsEntry()
		fieldFlags := field.FieldFlags()

		if hasSetFf {
			fieldFlags = fieldFlags | setFf
			setFieldFlags(field, fieldFlags)
		}

		if clrFf, hasClrFf := fdfField.ClearFieldFlags(); hasClrFf {
			// we have to clear the bits of the document fields for every bit that is
			// set in this field.
			//
			// Example:
			// docFf = 1011
			// clrFf = 1101
			// clrFfValue = 0010;
			// newValue = 1011 & 0010 which is 0010
			clrFfValue := clrFf
			clrFfValue ^= -1
			fieldFlags = fieldFlags & clrFfValue
			setFieldFlags(field, fieldFlags)
		}
	}
	return nil
}

// importFDFTerminal is the body of PDTerminalField.importFDF after its call to
// super.
func importFDFTerminal(field PDField, fdfField *fdf.FDFField) error {
	f, hasF := fdfField.WidgetFieldFlags()
	for _, widget := range field.Widgets() {
		if hasF {
			widget.SetAnnotationFlags(f)
			continue
		}
		// these are supposed to be ignored if the F is set.
		setF, hasSetF := fdfField.SetWidgetFieldFlagsEntry()
		annotFlags := widget.AnnotationFlags()
		if hasSetF {
			annotFlags = annotFlags | setF
			widget.SetAnnotationFlags(annotFlags)
		}
		if clrF, hasClrF := fdfField.ClearWidgetFieldFlags(); hasClrF {
			// we have to clear the bits of the document fields for every bit that is
			// set in this field.
			//
			// Example:
			// docF = 1011
			// clrF = 1101
			// clrFValue = 0010;
			// newValue = 1011 & 0010 which is 0010
			//
			// Java writes the mask as a long, 0xFFFFFFFFL, so the xor widens
			// clrF to a long and the & narrows it back; the result is the same
			// as xoring with -1, which is what the port does.
			clrFValue := clrF
			clrFValue ^= -1
			annotFlags = annotFlags & clrFValue
			widget.SetAnnotationFlags(annotFlags)
		}
	}
	return nil
}

// importFDFNonTerminal is the body of PDNonTerminalField.importFDF after its
// call to super.
func importFDFNonTerminal(field *PDNonTerminalField, fdfField *fdf.FDFField) error {
	fdfKids := fdfField.Kids()
	if fdfKids == nil {
		return nil
	}
	children := field.Children()
	for i := 0; i < fdfKids.Size(); i++ {
		for _, pdChild := range children {
			fdfChild := fdfKids.Get(i)
			fdfName := fdfChild.PartialFieldName()
			if fdfName != "" && fdfName == pdChild.PartialName() {
				if err := ImportFDF(pdChild, fdfChild); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// ExportFDF returns the FDF field that holds the value of the given field.
//
// Port of the package-private PDField.exportFDF, which Java declares abstract
// and implements in the terminal and non-terminal fields; see ImportFDF for why
// it is a function.
func ExportFDF(field PDField) (*fdf.FDFField, error) {
	if nonTerminal, isNonTerminal := field.(*PDNonTerminalField); isNonTerminal {
		fdfField := fdf.NewFDFField()
		fdfField.SetPartialFieldName(nonTerminal.PartialName())
		if err := fdfField.SetValue(nonTerminal.Value()); err != nil {
			return nil, err
		}
		children := nonTerminal.Children()
		fdfChildren := make([]*fdf.FDFField, 0, len(children))
		for _, child := range children {
			fdfChild, err := ExportFDF(child)
			if err != nil {
				return nil, err
			}
			fdfChildren = append(fdfChildren, fdfChild)
		}
		fdfField.SetKids(fdfChildren)
		return fdfField, nil
	}

	// PDTerminalField.exportFDF.
	fdfField := fdf.NewFDFField()
	fdfField.SetPartialFieldName(field.PartialName())
	fdfField.SetCOSValue(field.FieldDictionary().GetDictionaryObject(cos.V))
	// fixme: the old code which was here assumed that Kids were PDField instances,
	//        which is never true. They're annotation widgets.
	return fdfField, nil
}

// setFieldFlags writes the /Ff of the given field, which PDField declares and
// PDTerminalField and PDNonTerminalField both inherit.
func setFieldFlags(field PDField, flags int) {
	field.FieldDictionary().SetInt(cos.Ff, flags)
}

// asChoice returns the choice half of the given field, and nil where it is not
// one, which is the `this instanceof PDChoice` of Java.
func asChoice(field PDField) *PDChoice {
	if choice, isChoice := field.(interface{ choice() *PDChoice }); isChoice {
		return choice.choice()
	}
	return nil
}

// stringStringList is COSArray.toCOSStringStringList, which casts every entry to
// COSString and so panics on an entry that is not one.
func stringStringList(array *cos.Array) []string {
	out := make([]string, 0, array.Size())
	for _, item := range array.ToList() {
		str, isString := item.(*cos.StringObj)
		if !isString {
			panic(fmt.Sprintf("form: %T cannot be cast to COSString", item))
		}
		out = append(out, str.Value())
	}
	return out
}

// ImportFDFDocument sets the values of the fields of the given form from the
// given FDF document.
//
// Port of PDAcroForm.importFDF; see ImportFDF for why it is a function.
func ImportFDFDocument(acroForm *PDAcroForm, fdfDocument *fdf.FDFDocument) error {
	fields := fdfDocument.Catalog().FDF().Fields()
	if fields == nil {
		return nil
	}
	for _, fdfField := range fields.ToSlice() {
		docField := acroForm.Field(fdfField.PartialFieldName())
		if docField != nil {
			if err := ImportFDF(docField, fdfField); err != nil {
				return err
			}
		}
	}
	return nil
}

// ExportFDFDocument returns the FDF document that holds the values of the
// fields of the given form.
//
// Port of PDAcroForm.exportFDF; see ImportFDF for why it is a function.
func ExportFDFDocument(acroForm *PDAcroForm) (*fdf.FDFDocument, error) {
	fdfDocument := fdf.NewFDFDocument()
	catalog := fdfDocument.Catalog()
	fdfDict := fdf.NewFDFDictionary()
	catalog.SetFDF(fdfDict)
	fields := acroForm.Fields()
	fdfFields := make([]*fdf.FDFField, 0, len(fields))
	for _, field := range fields {
		fdfField, err := ExportFDF(field)
		if err != nil {
			fdfDocument.Close()
			return nil, err
		}
		fdfFields = append(fdfFields, fdfField)
	}
	fdfDict.SetID(acroForm.Document().Document().DocumentID())
	if len(fdfFields) != 0 {
		fdfDict.SetFields(fdfFields)
	}
	return fdfDocument, nil
}

// XFADocument returns the XFA of the given resource, read into a DOM tree with
// the parser namespace aware.
//
// Port of PDXFAResource.getDocument, which is a function here because it names
// the DOM and every other accessor of that class works on bytes.
func XFADocument(resource *PDXFAResource) (*dom.Document, error) {
	b, err := resource.Bytes()
	if err != nil {
		return nil, err
	}
	return util.XMLParseNamespaceAware(bytes.NewReader(b), true)
}
