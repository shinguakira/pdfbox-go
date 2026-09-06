// Package taggedpdf holds the standard attribute objects of a tagged PDF: the
// layout, list, table and print field attributes a structure element carries,
// and the standard structure types it can have.
//
// Port of org.apache.pdfbox.pdmodel.documentinterchange.taggedpdf.
package taggedpdf

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
)

// Unspecified is the default that says a number attribute has no value.
//
// Port of the protected PDStandardAttributeObject.UNSPECIFIED.
const Unspecified float32 = -1

// PDStandardAttributeObject carries the typed reading and writing every
// standard attribute object shares.
//
// Port of PDStandardAttributeObject, which Java declares abstract. The names it
// takes are Java's bare strings, which COSDictionary interns.
type PDStandardAttributeObject struct {
	logicalstructure.PDAttributeObjectBase
}

// InitStandardAttributeObject is the protected PDStandardAttributeObject()
// constructor. A concrete attribute object calls it from its own constructor
// with itself as self, since Go embedding does not dispatch.
func (a *PDStandardAttributeObject) InitStandardAttributeObject(
	self logicalstructure.PDAttributeObject) {
	a.InitAttributeObject(self)
}

// InitStandardAttributeObjectOf is the protected
// PDStandardAttributeObject(COSDictionary) constructor.
func (a *PDStandardAttributeObject) InitStandardAttributeObjectOf(
	self logicalstructure.PDAttributeObject, dictionary *cos.Dictionary) {
	a.InitAttributeObjectOf(self, dictionary)
}

// IsSpecified reports whether the given attribute is in the dictionary.
func (a *PDStandardAttributeObject) IsSpecified(name string) bool {
	return a.Dictionary().GetDictionaryObject(cos.GetPDFName(name)) != nil
}

// GetString returns a string attribute. Java declares it protected.
func (a *PDStandardAttributeObject) GetString(name string) string {
	return a.Dictionary().GetString(cos.GetPDFName(name), "")
}

// SetString sets a string attribute. Java declares it protected.
func (a *PDStandardAttributeObject) SetString(name string, value string) {
	key := cos.GetPDFName(name)
	oldBase := a.Dictionary().GetDictionaryObject(key)
	a.Dictionary().SetString(key, value)
	newBase := a.Dictionary().GetDictionaryObject(key)
	a.PotentiallyNotifyChanged(oldBase, newBase)
}

// GetArrayOfString returns an array attribute of strings, or nil.
//
// JAVA BUG: it reads each entry as a name, while SetArrayOfString writes
// strings, so a round trip throws. See migration/JAVA-BUGS.md entry 40. The
// port keeps it: the type assertion panics on anything but a name.
func (a *PDStandardAttributeObject) GetArrayOfString(name string) []string {
	v := a.Dictionary().GetDictionaryObject(cos.GetPDFName(name))
	array, isArray := v.(*cos.Array)
	if !isArray {
		return nil
	}
	strings := make([]string, array.Size())
	for i := 0; i < array.Size(); i++ {
		strings[i] = array.GetObject(i).(*cos.Name).Name()
	}
	return strings
}

// SetArrayOfString sets an array attribute of strings. Java declares it
// protected.
func (a *PDStandardAttributeObject) SetArrayOfString(name string, values []string) {
	key := cos.GetPDFName(name)
	oldBase := a.Dictionary().GetDictionaryObject(key)
	array := cos.NewArray()
	for _, value := range values {
		array.Add(cos.NewStringObj(value))
	}
	a.Dictionary().SetItem(key, array)
	newBase := a.Dictionary().GetDictionaryObject(key)
	a.PotentiallyNotifyChanged(oldBase, newBase)
}

// GetName returns a name attribute. Java declares it protected.
func (a *PDStandardAttributeObject) GetName(name string) string {
	return a.Dictionary().GetNameAsString(cos.GetPDFName(name), "")
}

// GetNameDefault returns a name attribute, or defaultValue. Java declares it
// protected.
func (a *PDStandardAttributeObject) GetNameDefault(name string, defaultValue string) string {
	return a.Dictionary().GetNameAsString(cos.GetPDFName(name), defaultValue)
}

// GetNameOrArrayOfName returns a name attribute as a string, an array of them
// as a slice of strings, or defaultValue. Java declares it protected.
//
// An entry of the array that is not a name leaves the empty string in its
// place, which is the null Java leaves.
func (a *PDStandardAttributeObject) GetNameOrArrayOfName(
	name string, defaultValue string) any {
	v := a.Dictionary().GetDictionaryObject(cos.GetPDFName(name))
	if array, isArray := v.(*cos.Array); isArray {
		names := make([]string, array.Size())
		for i := 0; i < array.Size(); i++ {
			if item, isName := array.GetObject(i).(*cos.Name); isName {
				names[i] = item.Name()
			}
		}
		return names
	}
	if item, isName := v.(*cos.Name); isName {
		return item.Name()
	}
	return defaultValue
}

// SetName sets a name attribute. Java declares it protected.
func (a *PDStandardAttributeObject) SetName(name string, value string) {
	key := cos.GetPDFName(name)
	oldBase := a.Dictionary().GetDictionaryObject(key)
	a.Dictionary().SetName(key, value)
	newBase := a.Dictionary().GetDictionaryObject(key)
	a.PotentiallyNotifyChanged(oldBase, newBase)
}

// SetArrayOfName sets an array attribute of names. Java declares it protected.
func (a *PDStandardAttributeObject) SetArrayOfName(name string, values []string) {
	key := cos.GetPDFName(name)
	oldBase := a.Dictionary().GetDictionaryObject(key)
	array := cos.NewArray()
	for _, value := range values {
		array.Add(cos.GetPDFName(value))
	}
	a.Dictionary().SetItem(key, array)
	newBase := a.Dictionary().GetDictionaryObject(key)
	a.PotentiallyNotifyChanged(oldBase, newBase)
}

// GetNumberOrName returns a number attribute as a float32, a name attribute as
// a string, or defaultValue. Java declares it protected.
func (a *PDStandardAttributeObject) GetNumberOrName(name string, defaultValue string) any {
	value := a.Dictionary().GetDictionaryObject(cos.GetPDFName(name))
	if number, isNumber := value.(cos.Number); isNumber {
		return number.FloatValue()
	}
	if item, isName := value.(*cos.Name); isName {
		return item.Name()
	}
	return defaultValue
}

// GetInteger returns an integer attribute, or defaultValue. Java declares it
// protected.
func (a *PDStandardAttributeObject) GetInteger(name string, defaultValue int) int {
	return a.Dictionary().GetIntDefault(cos.GetPDFName(name), defaultValue)
}

// SetInteger sets an integer attribute. Java declares it protected.
func (a *PDStandardAttributeObject) SetInteger(name string, value int) {
	key := cos.GetPDFName(name)
	oldBase := a.Dictionary().GetDictionaryObject(key)
	a.Dictionary().SetInt(key, value)
	newBase := a.Dictionary().GetDictionaryObject(key)
	a.PotentiallyNotifyChanged(oldBase, newBase)
}

// GetNumberDefault returns a number attribute, or defaultValue. Java declares
// it protected.
func (a *PDStandardAttributeObject) GetNumberDefault(name string, defaultValue float32) float32 {
	return a.Dictionary().GetFloat(cos.GetPDFName(name), defaultValue)
}

// GetNumber returns a number attribute, or -1. Java declares it protected.
func (a *PDStandardAttributeObject) GetNumber(name string) float32 {
	return a.Dictionary().GetFloat(cos.GetPDFName(name), -1)
}

// GetNumberOrArrayOfNumber returns a number attribute as a float32, an array of
// them as a slice, or defaultValue -- and nil where defaultValue is Unspecified.
// Java declares it protected.
//
// An entry of the array that is not a number leaves a zero in its place, which
// is the zero Java leaves in the float array.
func (a *PDStandardAttributeObject) GetNumberOrArrayOfNumber(
	name string, defaultValue float32) any {
	v := a.Dictionary().GetDictionaryObject(cos.GetPDFName(name))
	if array, isArray := v.(*cos.Array); isArray {
		values := make([]float32, array.Size())
		for i := 0; i < array.Size(); i++ {
			if item, isNumber := array.GetObject(i).(cos.Number); isNumber {
				values[i] = item.FloatValue()
			}
		}
		return values
	}
	if number, isNumber := v.(cos.Number); isNumber {
		return number.FloatValue()
	}
	if defaultValue == Unspecified {
		return nil
	}
	return defaultValue
}

// SetNumber sets a number attribute. Java declares it protected.
func (a *PDStandardAttributeObject) SetNumber(name string, value float32) {
	key := cos.GetPDFName(name)
	oldBase := a.Dictionary().GetDictionaryObject(key)
	a.Dictionary().SetFloat(key, value)
	newBase := a.Dictionary().GetDictionaryObject(key)
	a.PotentiallyNotifyChanged(oldBase, newBase)
}

// SetNumberInt sets a number attribute from an integer, which writes an integer
// rather than a real. Java tells the two apart by the argument type.
func (a *PDStandardAttributeObject) SetNumberInt(name string, value int) {
	key := cos.GetPDFName(name)
	oldBase := a.Dictionary().GetDictionaryObject(key)
	a.Dictionary().SetInt(key, value)
	newBase := a.Dictionary().GetDictionaryObject(key)
	a.PotentiallyNotifyChanged(oldBase, newBase)
}

// SetArrayOfNumber sets an array attribute of numbers. Java declares it
// protected.
func (a *PDStandardAttributeObject) SetArrayOfNumber(name string, values []float32) {
	key := cos.GetPDFName(name)
	array := cos.NewArray()
	for _, value := range values {
		array.Add(cos.NewFloat(value))
	}
	oldBase := a.Dictionary().GetDictionaryObject(key)
	a.Dictionary().SetItem(key, array)
	newBase := a.Dictionary().GetDictionaryObject(key)
	a.PotentiallyNotifyChanged(oldBase, newBase)
}

// GetColor returns a colour attribute, or nil. Java declares it protected.
//
// Java casts the entry to COSArray without a check, so anything else throws;
// the port asserts the same way.
func (a *PDStandardAttributeObject) GetColor(name string) *color.PDGamma {
	c := a.Dictionary().GetDictionaryObject(cos.GetPDFName(name))
	if c == nil {
		return nil
	}
	return color.NewPDGammaOfArray(c.(*cos.Array))
}

// GetColorOrFourColors returns one colour, four of them, or nil where the entry
// is missing or is an array of another length. Java declares it protected.
func (a *PDStandardAttributeObject) GetColorOrFourColors(name string) any {
	c := a.Dictionary().GetDictionaryObject(cos.GetPDFName(name))
	if c == nil {
		return nil
	}
	array := c.(*cos.Array)
	switch array.Size() {
	case 3:
		// only one colour
		return color.NewPDGammaOfArray(array)
	case 4:
		return NewPDFourColoursOfArray(array)
	}
	return nil
}

// SetColor sets a colour attribute. Java declares it protected.
func (a *PDStandardAttributeObject) SetColor(name string, value *color.PDGamma) {
	key := cos.GetPDFName(name)
	oldValue := a.Dictionary().GetDictionaryObject(key)
	var newValue cos.Base
	if value != nil {
		newValue = value.COSObject()
	}
	a.Dictionary().SetItem(key, newValue)
	a.PotentiallyNotifyChanged(oldValue, newValue)
}

// SetFourColors sets a four colour attribute. Java declares it protected.
func (a *PDStandardAttributeObject) SetFourColors(name string, value *PDFourColours) {
	key := cos.GetPDFName(name)
	oldValue := a.Dictionary().GetDictionaryObject(key)
	var newValue cos.Base
	if value != nil {
		newValue = value.COSObject()
	}
	a.Dictionary().SetItem(key, newValue)
	a.PotentiallyNotifyChanged(oldValue, newValue)
}
