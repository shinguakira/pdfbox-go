package fdf

import (
	"fmt"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/action"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"github.com/shinguakira/pdfbox-go/go/w3c/dom"
)

// FDFField is one field of an FDF document.
//
// Port of FDFField.
type FDFField struct {
	field *cos.Dictionary
}

var _ common.COSObjectable = (*FDFField)(nil)

// NewFDFField returns an empty field.
func NewFDFField() *FDFField { return &FDFField{field: cos.NewDictionary()} }

// NewFDFFieldOf returns the field the given dictionary holds.
func NewFDFFieldOf(f *cos.Dictionary) *FDFField { return &FDFField{field: f} }

// NewFDFFieldOfXML returns the field the given XFDF element describes.
func NewFDFFieldOfXML(fieldXML *dom.Element) (*FDFField, error) {
	f := NewFDFField()
	f.SetPartialFieldName(fieldXML.GetAttribute("name"))
	nodeList := fieldXML.ChildNodes()
	kids := []*FDFField{}
	for i := 0; i < nodeList.Length(); i++ {
		node := nodeList.Item(i)
		child, isElement := node.(*dom.Element)
		if !isElement {
			continue
		}
		switch child.TagName() {
		case "value":
			if err := f.SetValue(util.XMLNodeValue(child)); err != nil {
				return nil, err
			}
		case "value-richtext":
			f.SetRichText(cos.NewStringObj(util.XMLNodeValue(child)))
		case "field":
			kid, err := NewFDFFieldOfXML(child)
			if err != nil {
				return nil, err
			}
			kids = append(kids, kid)
		}
	}
	if len(kids) != 0 {
		f.SetKids(kids)
	}
	return f, nil
}

// WriteXML writes this field out as XFDF.
//
// Java throws IllegalStateException where the field has no name, which is
// unchecked, so the port panics.
func (f *FDFField) WriteXML(output io.Writer) error {
	partialFieldName := f.PartialFieldName()
	if partialFieldName == "" {
		panic("Field name is missing")
	}
	w := &xmlWriter{out: output}
	w.write("<field name=\"")
	w.write(escapeXML10(partialFieldName))
	w.write("\">\n")
	value, err := f.Value()
	if err != nil {
		return err
	}
	switch typed := value.(type) {
	case string:
		w.write("<value>")
		w.write(escapeXML10(typed))
		w.write("</value>\n")
	case []string:
		for _, item := range typed {
			w.write("<value>")
			w.write(escapeXML10(item))
			w.write("</value>\n")
		}
	}
	rt := f.RichText()
	if rt != "" {
		w.write("<value-richtext>")
		w.write(escapeXML10(rt))
		w.write("</value-richtext>\n")
	}
	if w.err != nil {
		return w.err
	}
	kids := f.Kids()
	if kids != nil {
		for _, kid := range kids.ToSlice() {
			if err := kid.WriteXML(output); err != nil {
				return err
			}
		}
	}
	w.write("</field>\n")
	return w.err
}

// COSObject returns the dictionary.
func (f *FDFField) COSObject() cos.Base { return f.field }

// Dictionary returns the dictionary, typed.
func (f *FDFField) Dictionary() *cos.Dictionary { return f.field }

// Kids returns the kids of the field, or nil where it has none.
//
// The list is backed by the kids array, so adding to it or deleting from it
// changes the document too.
func (f *FDFField) Kids() *common.COSArrayList[*FDFField] {
	kids := f.field.GetCOSArray(cos.Kids)
	if kids != nil {
		actuals := make([]*FDFField, 0, kids.Size())
		for i := 0; i < kids.Size(); i++ {
			actuals = append(actuals, NewFDFFieldOf(kids.GetObject(i).(*cos.Dictionary)))
		}
		return common.NewCOSArrayListOf(actuals, kids)
	}
	return nil
}

// SetKids sets the kids of the field.
func (f *FDFField) SetKids(kids []*FDFField) {
	f.field.SetItem(cos.Kids, common.NewCOSArrayOfObjectables(kids))
}

// PartialFieldName returns the /T of the field, or the empty string where it
// has none, which is the null Java answers.
func (f *FDFField) PartialFieldName() string { return f.field.GetString(cos.T, "") }

// SetPartialFieldName sets the /T of the field.
func (f *FDFField) SetPartialFieldName(partial string) { f.field.SetString(cos.T, partial) }

// Value returns the value of the field: a string for a name, a string or a
// stream, a []string for an array, and nil where there is none.
//
// Java answers Object and the caller switches on it, so the port answers any and
// the caller does the same.
func (f *FDFField) Value() (any, error) {
	switch value := f.field.GetDictionaryObject(cos.V).(type) {
	case *cos.Name:
		return value.Name(), nil
	case *cos.Array:
		return stringStringList(value), nil
	case *cos.StringObj:
		return value.Value(), nil
	case *cos.Stream:
		return value.ToTextString(), nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("Error: Unknown type for field import: %v", value)
	}
}

// stringStringList is COSArray.toCOSStringStringList, which casts every entry to
// COSString.
//
// Java raises ClassCastException on an entry that is not one; the port panics,
// which is the same unchecked failure.
func stringStringList(array *cos.Array) []string {
	out := make([]string, 0, array.Size())
	for _, item := range array.ToList() {
		str, isString := item.(*cos.StringObj)
		if !isString {
			panic(fmt.Sprintf("fdf: %T cannot be cast to COSString", item))
		}
		out = append(out, str.Value())
	}
	return out
}

// COSValue returns the value of the field as the COS object it is written as,
// or nil where there is none.
func (f *FDFField) COSValue() (cos.Base, error) {
	value := f.field.GetDictionaryObject(cos.V)
	switch value.(type) {
	case *cos.Name, *cos.Array, *cos.StringObj, *cos.Stream:
		return value, nil
	case nil:
		return nil, nil
	}
	return nil, fmt.Errorf("Error: Unknown type for field import: %v", value)
}

// SetValue sets the value of the field from a string, a []string or a
// COSObjectable, and clears it for nil.
//
// Port of setValue(Object).
func (f *FDFField) SetValue(value any) error {
	var base cos.Base
	switch typed := value.(type) {
	case []string:
		base = cos.ArrayOfStrings(typed)
	case string:
		base = cos.NewStringObj(typed)
	case common.COSObjectable:
		base = typed.COSObject()
	case nil:
	default:
		return fmt.Errorf("Error: Unknown type for field import: %v", value)
	}
	f.field.SetItem(cos.V, base)
	return nil
}

// SetCOSValue sets the value of the field to the given COS object.
//
// Java names it setValue(COSBase), overloading setValue(Object).
func (f *FDFField) SetCOSValue(value cos.Base) { f.field.SetItem(cos.V, value) }

// FieldFlags returns the /Ff of the field, and reports whether it has one --
// Java answers a nullable Integer.
func (f *FDFField) FieldFlags() (int, bool) { return intEntryOf(f.field, cos.Ff) }

// SetFieldFlagsOrNil sets the /Ff of the field, and removes it where the flags
// are nil.
//
// Port of setFieldFlags(Integer).
func (f *FDFField) SetFieldFlagsOrNil(ff *int) { setIntEntryOrNil(f.field, cos.Ff, ff) }

// SetFieldFlags sets the /Ff of the field.
//
// Port of setFieldFlags(int).
func (f *FDFField) SetFieldFlags(ff int) { f.field.SetInt(cos.Ff, ff) }

// SetFieldFlagsValue returns the /SetFf of the field, and reports whether it
// has one.
//
// Java names the getter getSetFieldFlags.
func (f *FDFField) SetFieldFlagsEntry() (int, bool) { return intEntryOf(f.field, cos.SetFf) }

// SetSetFieldFlagsOrNil sets the /SetFf of the field, and removes it where the
// flags are nil.
func (f *FDFField) SetSetFieldFlagsOrNil(ff *int) { setIntEntryOrNil(f.field, cos.SetFf, ff) }

// SetSetFieldFlags sets the /SetFf of the field.
func (f *FDFField) SetSetFieldFlags(ff int) { f.field.SetInt(cos.SetFf, ff) }

// ClearFieldFlags returns the /ClrFf of the field, and reports whether it has
// one.
func (f *FDFField) ClearFieldFlags() (int, bool) { return intEntryOf(f.field, cos.ClrFf) }

// SetClearFieldFlagsOrNil sets the /ClrFf of the field, and removes it where the
// flags are nil.
func (f *FDFField) SetClearFieldFlagsOrNil(ff *int) { setIntEntryOrNil(f.field, cos.ClrFf, ff) }

// SetClearFieldFlags sets the /ClrFf of the field.
func (f *FDFField) SetClearFieldFlags(ff int) { f.field.SetInt(cos.ClrFf, ff) }

// WidgetFieldFlags returns the /F of the field, and reports whether it has one.
func (f *FDFField) WidgetFieldFlags() (int, bool) { return intEntryOf(f.field, cos.F) }

// SetWidgetFieldFlagsOrNil sets the /F of the field, and removes it where the
// flags are nil.
func (f *FDFField) SetWidgetFieldFlagsOrNil(fl *int) { setIntEntryOrNil(f.field, cos.F, fl) }

// SetWidgetFieldFlags sets the /F of the field.
func (f *FDFField) SetWidgetFieldFlags(fl int) { f.field.SetInt(cos.F, fl) }

// SetWidgetFieldFlagsEntry returns the /SetF of the field, and reports whether
// it has one.
//
// Java names the getter getSetWidgetFieldFlags.
func (f *FDFField) SetWidgetFieldFlagsEntry() (int, bool) { return intEntryOf(f.field, cos.SetF) }

// SetSetWidgetFieldFlagsOrNil sets the /SetF of the field, and removes it where
// the flags are nil.
func (f *FDFField) SetSetWidgetFieldFlagsOrNil(ff *int) { setIntEntryOrNil(f.field, cos.SetF, ff) }

// SetSetWidgetFieldFlags sets the /SetF of the field.
func (f *FDFField) SetSetWidgetFieldFlags(ff int) { f.field.SetInt(cos.SetF, ff) }

// ClearWidgetFieldFlags returns the /ClrF of the field, and reports whether it
// has one.
func (f *FDFField) ClearWidgetFieldFlags() (int, bool) { return intEntryOf(f.field, cos.ClrF) }

// SetClearWidgetFieldFlagsOrNil sets the /ClrF of the field, and removes it
// where the flags are nil.
func (f *FDFField) SetClearWidgetFieldFlagsOrNil(ff *int) {
	setIntEntryOrNil(f.field, cos.ClrF, ff)
}

// SetClearWidgetFieldFlags sets the /ClrF of the field.
func (f *FDFField) SetClearWidgetFieldFlags(ff int) { f.field.SetInt(cos.ClrF, ff) }

// intEntryOf reads a flags entry that Java holds as a nullable Integer, and
// reports whether the dictionary has one.
//
// Java casts the entry to COSNumber without a check, so an entry that is not a
// number raises ClassCastException; the port panics.
func intEntryOf(dictionary *cos.Dictionary, key *cos.Name) (int, bool) {
	base := dictionary.GetDictionaryObject(key)
	if base == nil {
		return 0, false
	}
	number, isNumber := base.(cos.Number)
	if !isNumber {
		panic(fmt.Sprintf("fdf: %T cannot be cast to COSNumber", base))
	}
	return number.IntValue(), true
}

// setIntEntryOrNil writes a flags entry that Java takes as a nullable Integer,
// and removes it for a nil one.
func setIntEntryOrNil(dictionary *cos.Dictionary, key *cos.Name, value *int) {
	var entry cos.Base
	if value != nil {
		entry = cos.GetInteger(int64(*value))
	}
	dictionary.SetItem(key, entry)
}

// AppearanceDictionary returns the /AP of the field, or nil where it has none.
func (f *FDFField) AppearanceDictionary() *annotation.PDAppearanceDictionary {
	dict := f.field.GetCOSDictionary(cos.AP)
	if dict != nil {
		return annotation.NewPDAppearanceDictionaryOf(dict)
	}
	return nil
}

// SetAppearanceDictionary sets the /AP of the field.
func (f *FDFField) SetAppearanceDictionary(ap *annotation.PDAppearanceDictionary) {
	f.field.SetItem(cos.AP, common.COSObjectOrNil(ap))
}

// AppearanceStreamReference returns the /APRef of the field, or nil where it
// has none.
func (f *FDFField) AppearanceStreamReference() *FDFNamedPageReference {
	ref := f.field.GetCOSDictionary(cos.APRef)
	if ref != nil {
		return NewFDFNamedPageReferenceOf(ref)
	}
	return nil
}

// SetAppearanceStreamReference sets the /APRef of the field.
func (f *FDFField) SetAppearanceStreamReference(ref *FDFNamedPageReference) {
	f.field.SetItem(cos.APRef, common.COSObjectOrNil(ref))
}

// IconFit returns the /IF of the field, or nil where it has none.
func (f *FDFField) IconFit() *FDFIconFit {
	dic := f.field.GetCOSDictionary(cos.IF)
	if dic != nil {
		return NewFDFIconFitOf(dic)
	}
	return nil
}

// SetIconFit sets the /IF of the field.
func (f *FDFField) SetIconFit(fit *FDFIconFit) {
	f.field.SetItem(cos.IF, common.COSObjectOrNil(fit))
}

// Options returns the /Opt of the field, or nil where it has none. Each entry
// is a string or an *FDFOptionElement.
//
// The list is backed by the options array, so adding to it or deleting from it
// changes the document too.
//
// Java casts an entry that is not a COSString to COSArray without a check, so a
// number there raises ClassCastException; the port panics.
func (f *FDFField) Options() *common.COSArrayList[any] {
	array := f.field.GetCOSArray(cos.Opt)
	if array != nil {
		objects := make([]any, 0, array.Size())
		for i := 0; i < array.Size(); i++ {
			next := array.GetObject(i)
			if str, isString := next.(*cos.StringObj); isString {
				objects = append(objects, str.Value())
				continue
			}
			value, isArray := next.(*cos.Array)
			if !isArray {
				panic(fmt.Sprintf("fdf: %T cannot be cast to COSArray", next))
			}
			objects = append(objects, NewFDFOptionElementOf(value))
		}
		return common.NewCOSArrayListOf(objects, array)
	}
	return nil
}

// SetOptions sets the /Opt of the field. Each entry is a string or an
// *FDFOptionElement.
func (f *FDFField) SetOptions(options []any) {
	value := common.ConverterToCOSArray(options)
	f.field.SetItem(cos.Opt, value)
}

// Action returns the /A of the field, or nil where it has none.
func (f *FDFField) Action() action.Action {
	return action.CreateAction(f.field.GetCOSDictionary(cos.A))
}

// SetAction sets the /A of the field.
func (f *FDFField) SetAction(a action.Action) {
	f.field.SetItem(cos.A, common.COSObjectOrNil(a))
}

// AdditionalActions returns the /AA of the field, or nil where it has none.
func (f *FDFField) AdditionalActions() *action.PDAdditionalActions {
	dict := f.field.GetCOSDictionary(cos.AA)
	if dict != nil {
		return action.NewPDAdditionalActionsOf(dict)
	}
	return nil
}

// SetAdditionalActions sets the /AA of the field.
func (f *FDFField) SetAdditionalActions(aa *action.PDAdditionalActions) {
	f.field.SetItem(cos.AA, common.COSObjectOrNil(aa))
}

// RichText returns the /RV of the field, or the empty string where it has none,
// which is the null Java answers.
//
// Java casts an /RV that is neither a string nor a stream to COSStream without a
// check; the port panics, which is the same unchecked failure.
func (f *FDFField) RichText() string {
	rv := f.field.GetDictionaryObject(cos.RV)
	if rv == nil {
		return ""
	}
	if str, isString := rv.(*cos.StringObj); isString {
		return str.Value()
	}
	stream, isStream := rv.(*cos.Stream)
	if !isStream {
		panic(fmt.Sprintf("fdf: %T cannot be cast to COSStream", rv))
	}
	return stream.ToTextString()
}

// SetRichText sets the /RV of the field to a string.
//
// Port of setRichText(COSString).
func (f *FDFField) SetRichText(rv *cos.StringObj) { f.field.SetItem(cos.RV, rv) }

// SetRichTextStream sets the /RV of the field to a stream.
//
// Port of setRichText(COSStream).
func (f *FDFField) SetRichTextStream(rv *cos.Stream) { f.field.SetItem(cos.RV, rv) }

// xmlWriter is the sticky-error helper the writeXML methods write through, so
// that each of them reads like the Java it is ported from.
type xmlWriter struct {
	out io.Writer
	err error
}

// write writes the given text unless a previous write failed.
func (w *xmlWriter) write(text string) {
	if w.err != nil {
		return
	}
	_, w.err = io.WriteString(w.out, text)
}
