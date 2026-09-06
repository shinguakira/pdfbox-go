package form

import (
	"slices"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// The bits of the /Ff flags a choice field carries. Java declares FLAG_COMBO
// package-private and the rest private.
const (
	flagCombo             = 1 << 17
	flagSort              = 1 << 19
	flagMultiSelect       = 1 << 21
	flagDoNotSpellCheck   = 1 << 22
	flagCommitOnSelChange = 1 << 26
)

// PDChoice is a field whose value is chosen from a list.
//
// Port of the abstract PDChoice.
type PDChoice struct {
	PDVariableText
}

// initChoice is the protected PDChoice(PDAcroForm) constructor.
func (f *PDChoice) initChoice(self PDField, acroForm *PDAcroForm) {
	f.initVariableText(self, acroForm)
	f.FieldDictionary().SetItem(cos.FT, cos.Ch)
}

// initChoiceOf is the package-private PDChoice(PDAcroForm, COSDictionary,
// PDNonTerminalField) constructor.
func (f *PDChoice) initChoiceOf(self PDField, acroForm *PDAcroForm, field *cos.Dictionary,
	parent *PDNonTerminalField) {
	f.initVariableTextOf(self, acroForm, field, parent)
}

// Options returns the /Opt export values of the field.
func (f *PDChoice) Options() []string {
	values := f.FieldDictionary().GetDictionaryObject(cos.Opt)
	return getPairableItems(values, 0)
}

// SetOptions sets the /Opt options of the field, sorting them where the sort
// flag is set, and removes them where the list is empty.
func (f *PDChoice) SetOptions(displayValues []string) {
	if len(displayValues) == 0 {
		f.FieldDictionary().RemoveItem(cos.Opt)
		return
	}
	if f.IsSort() {
		slices.Sort(displayValues)
	}
	f.FieldDictionary().SetItem(cos.Opt, cos.ArrayOfStrings(displayValues))
}

// SetOptionsWithDisplayValues sets the /Opt options of the field as export and
// display value pairs.
//
// Java names this setOptions(List, List), overloading the one above. It throws
// IllegalArgumentException where the two lists differ in length, which is
// unchecked, so the port panics.
func (f *PDChoice) SetOptionsWithDisplayValues(exportValues, displayValues []string) {
	if len(exportValues) == 0 || len(displayValues) == 0 {
		f.FieldDictionary().RemoveItem(cos.Opt)
		return
	}
	if len(exportValues) != len(displayValues) {
		panic("The number of entries for exportValue and displayValue shall be the same.")
	}
	keyValuePairs := toKeyValueList(exportValues, displayValues)
	if f.IsSort() {
		sortByValue(keyValuePairs)
	}
	options := cos.NewArray()
	for i := 0; i < len(exportValues); i++ {
		entry := cos.NewArray()
		pair := keyValuePairs[i]
		entry.Add(cos.NewStringObj(pair.Key()))
		entry.Add(cos.NewStringObj(pair.Value()))
		options.Add(entry)
	}
	f.FieldDictionary().SetItem(cos.Opt, options)
}

// OptionsDisplayValues returns the display half of the /Opt options.
func (f *PDChoice) OptionsDisplayValues() []string {
	values := f.FieldDictionary().GetDictionaryObject(cos.Opt)
	return getPairableItems(values, 1)
}

// OptionsExportValues returns the export half of the /Opt options.
func (f *PDChoice) OptionsExportValues() []string { return f.Options() }

// HasSeparateExportAndDisplayValues reports whether the two halves of the
// options differ.
func (f *PDChoice) HasSeparateExportAndDisplayValues() bool {
	return !slices.Equal(f.OptionsExportValues(), f.OptionsDisplayValues())
}

// SelectedOptionsIndex returns the /I indices of the chosen options.
func (f *PDChoice) SelectedOptionsIndex() []int {
	if value := f.FieldDictionary().GetCOSArray(cos.I); value != nil {
		return intList(value)
	}
	return []int{}
}

// SetSelectedOptionsIndex sets the /I indices of the chosen options, and
// removes them where the list is empty.
//
// Java throws IllegalArgumentException for a field that allows only one
// selection, which is unchecked, so the port panics.
func (f *PDChoice) SetSelectedOptionsIndex(values []int) {
	if len(values) == 0 {
		f.FieldDictionary().RemoveItem(cos.I)
		return
	}
	if !f.IsMultiSelect() {
		panic("Setting the indices is not allowed for choice fields not allowing multiple selections.")
	}
	f.FieldDictionary().SetItem(cos.I, cos.ArrayOfIntegers(values))
}

// IsSort reports whether the options are sorted.
func (f *PDChoice) IsSort() bool { return f.FieldDictionary().GetFlag(cos.Ff, flagSort) }

// SetSort sets whether the options are sorted.
func (f *PDChoice) SetSort(sort bool) { f.FieldDictionary().SetFlag(cos.Ff, flagSort, sort) }

// IsMultiSelect reports whether more than one option may be chosen.
func (f *PDChoice) IsMultiSelect() bool {
	return f.FieldDictionary().GetFlag(cos.Ff, flagMultiSelect)
}

// SetMultiSelect sets whether more than one option may be chosen.
func (f *PDChoice) SetMultiSelect(multiSelect bool) {
	f.FieldDictionary().SetFlag(cos.Ff, flagMultiSelect, multiSelect)
}

// IsDoNotSpellCheck reports whether the value is spell checked.
func (f *PDChoice) IsDoNotSpellCheck() bool {
	return f.FieldDictionary().GetFlag(cos.Ff, flagDoNotSpellCheck)
}

// SetDoNotSpellCheck sets whether the value is spell checked.
func (f *PDChoice) SetDoNotSpellCheck(doNotSpellCheck bool) {
	f.FieldDictionary().SetFlag(cos.Ff, flagDoNotSpellCheck, doNotSpellCheck)
}

// IsCommitOnSelChange reports whether a change is committed as soon as it is
// made.
func (f *PDChoice) IsCommitOnSelChange() bool {
	return f.FieldDictionary().GetFlag(cos.Ff, flagCommitOnSelChange)
}

// SetCommitOnSelChange sets whether a change is committed as soon as it is
// made.
func (f *PDChoice) SetCommitOnSelChange(commitOnSelChange bool) {
	f.FieldDictionary().SetFlag(cos.Ff, flagCommitOnSelChange, commitOnSelChange)
}

// IsCombo reports whether the field is a combo box rather than a list box.
func (f *PDChoice) IsCombo() bool { return f.FieldDictionary().GetFlag(cos.Ff, flagCombo) }

// SetCombo sets whether the field is a combo box rather than a list box.
func (f *PDChoice) SetCombo(combo bool) { f.FieldDictionary().SetFlag(cos.Ff, flagCombo, combo) }

// SetValue sets the value of the field from a string.
func (f *PDChoice) SetValue(value string) error {
	f.FieldDictionary().SetString(cos.V, value)
	// remove I key for single valued choice field
	f.SetSelectedOptionsIndex(nil)
	return f.applyChange()
}

// SetDefaultValue sets the /DV of the field from a string.
func (f *PDChoice) SetDefaultValue(value string) {
	f.FieldDictionary().SetString(cos.DV, value)
}

// SetValues sets the values of the field.
//
// Java names this setValue(List), overloading setValue(String). It throws
// IllegalArgumentException for a field that allows one value only, and for a
// value that is not among the options; both are unchecked, so the port panics.
func (f *PDChoice) SetValues(values []string) error {
	if len(values) == 0 {
		f.FieldDictionary().RemoveItem(cos.V)
		f.FieldDictionary().RemoveItem(cos.I)
		return f.applyChange()
	}
	if !f.IsMultiSelect() {
		panic("The list box does not allow multiple selections.")
	}
	options := f.Options()
	for _, value := range values {
		if !slices.Contains(options, value) {
			panic("The values are not contained in the selectable options.")
		}
	}
	f.FieldDictionary().SetItem(cos.V, cos.ArrayOfStrings(values))
	f.updateSelectedOptionsIndex(values, options)
	return f.applyChange()
}

// Value returns the values of the field.
func (f *PDChoice) Value() []string { return f.valueFor(cos.V) }

// DefaultValue returns the default values of the field.
func (f *PDChoice) DefaultValue() []string { return f.valueFor(cos.DV) }

// valueFor reads a value that may be one string or an array of them. Java
// declares it private.
func (f *PDChoice) valueFor(name *cos.Name) []string {
	switch value := f.FieldDictionary().GetDictionaryObject(name).(type) {
	case *cos.StringObj:
		return []string{value.Value()}
	case *cos.Array:
		return stringList(value)
	}
	return []string{}
}

// ValueAsString returns the values of the field as one string, which Java
// writes with Arrays.toString.
func (f *PDChoice) ValueAsString() string {
	values := f.Value()
	out := "["
	for i, value := range values {
		if i > 0 {
			out += ", "
		}
		out += value
	}
	return out + "]"
}

// updateSelectedOptionsIndex records where the given values sit among the
// options. Java declares it private.
func (f *PDChoice) updateSelectedOptionsIndex(values, options []string) {
	indices := make([]int, 0, len(values))
	for _, value := range values {
		indices = append(indices, slices.Index(options, value))
	}
	// Indices have to be "... array of integers, sorted in ascending order ..."
	slices.Sort(indices)
	f.SetSelectedOptionsIndex(indices)
}

// choice answers this field, which ImportFDF finds through the interface every
// field embedding PDChoice carries because of it.
//
// Java writes `this instanceof PDChoice`, which Go cannot ask of an embedded
// struct; see AsVariableText for the same device.
func (f *PDChoice) choice() *PDChoice { return f }
