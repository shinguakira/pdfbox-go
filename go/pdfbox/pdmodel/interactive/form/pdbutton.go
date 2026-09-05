package form

import (
	"fmt"
	"strconv"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// The bits of the /Ff flags a button carries. Java declares them
// package-private.
const (
	flagRadio          = 1 << 15
	flagPushButton     = 1 << 16
	flagRadiosInUnison = 1 << 25
)

// PDButton is a check box, a radio button or a push button.
//
// Port of the abstract PDButton.
type PDButton struct {
	PDTerminalField
}

// initButton is the package-private PDButton(PDAcroForm) constructor.
func (f *PDButton) initButton(self PDField, acroForm *PDAcroForm) {
	f.initTerminalField(self, acroForm)
	f.FieldDictionary().SetItem(cos.FT, cos.Btn)
}

// initButtonOf is the package-private PDButton(PDAcroForm, COSDictionary,
// PDNonTerminalField) constructor.
func (f *PDButton) initButtonOf(self PDField, acroForm *PDAcroForm, field *cos.Dictionary,
	parent *PDNonTerminalField) {
	f.initTerminalFieldOf(self, acroForm, field, parent)
}

// IsPushButton reports whether the field is a push button.
func (f *PDButton) IsPushButton() bool {
	return f.FieldDictionary().GetFlag(cos.Ff, flagPushButton)
}

// IsRadioButton reports whether the field is a radio button.
func (f *PDButton) IsRadioButton() bool {
	return f.FieldDictionary().GetFlag(cos.Ff, flagRadio)
}

// Value returns the value of the button, which is Off where it has none.
func (f *PDButton) Value() string {
	value, isName := f.inheritableAttribute(cos.V).(*cos.Name)
	if !isName {
		// Off is the default value if there is nothing else set.
		// See PDF Spec.
		return "Off"
	}
	stringValue := value.Name()
	exportValues := f.ExportValues()
	if len(exportValues) != 0 {
		idx, err := strconv.Atoi(stringValue)
		if err != nil {
			return stringValue
		}
		if idx >= 0 && idx < len(exportValues) {
			return exportValues[idx]
		}
	}
	return stringValue
}

// SetValue sets the value of the button.
func (f *PDButton) SetValue(value string) error {
	f.checkValue(value)

	// if there are export values/an Opt entry there is a different
	// approach to setting the value
	if len(f.ExportValues()) != 0 {
		f.updateByOption(value)
	} else {
		f.updateByValue(value)
	}
	return f.applyChange()
}

// SetValueIndex sets the value of the button to the option at the given index.
//
// Java names this setValue(int), overloading setValue(String). It throws
// IllegalArgumentException for an index outside the options, which is
// unchecked, so the port panics.
func (f *PDButton) SetValueIndex(index int) error {
	exportValues := f.ExportValues()
	if len(exportValues) == 0 || index < 0 || index >= len(exportValues) {
		panic(fmt.Sprintf("index '%d' is not a valid index for the field %s, "+
			"valid indices are from 0 to %d", index, f.FullyQualifiedName(),
			len(exportValues)-1))
	}
	f.updateByValue(strconv.Itoa(index))
	return f.applyChange()
}

// DefaultValue returns the /DV of the button.
func (f *PDButton) DefaultValue() string {
	if value, isName := f.inheritableAttribute(cos.DV).(*cos.Name); isName {
		return value.Name()
	}
	return ""
}

// SetDefaultValue sets the /DV of the button.
func (f *PDButton) SetDefaultValue(value string) {
	f.checkValue(value)
	f.FieldDictionary().SetName(cos.DV, value)
}

// ValueAsString returns the value of the button as a string.
func (f *PDButton) ValueAsString() string { return f.Value() }

// ExportValues returns the /Opt export values of the button.
func (f *PDButton) ExportValues() []string {
	switch value := f.inheritableAttribute(cos.Opt).(type) {
	case *cos.StringObj:
		stringValue := value.Value()
		if stringValue == "" {
			return []string{}
		}
		return []string{stringValue}
	case *cos.Array:
		return stringList(value)
	}
	return []string{}
}

// SetExportValues sets the /Opt export values of the button, and removes them
// where the list is empty.
func (f *PDButton) SetExportValues(values []string) {
	if len(values) != 0 {
		f.FieldDictionary().SetItem(cos.Opt, cos.ArrayOfStrings(values))
		return
	}
	f.FieldDictionary().RemoveItem(cos.Opt)
}

// constructAppearances sets the appearance state of each widget from the value.
func (f *PDButton) constructAppearances() error {
	for _, widget := range f.Widgets() {
		appearance := widget.Appearance()
		if appearance == nil {
			continue
		}
		appearanceEntry := appearance.NormalAppearance()
		value := f.FieldDictionary().GetCOSName(cos.V)
		if dict, isDictionary := appearanceEntry.COSObject().(*cos.Dictionary); isDictionary &&
			dict.ContainsKey(value) {
			widget.SetAppearanceStateName(value)
		} else {
			widget.SetAppearanceStateName(cos.Off)
		}
	}
	return nil
}

// OnValues returns the values that turn the button on, in the order they are
// found.
//
// Java answers a LinkedHashSet, which keeps that order and drops the repeats a
// field appearing more than once would give; the port keeps the order and drops
// the repeats the same way.
func (f *PDButton) OnValues() []string {
	// we need a set as the field can appear multiple times
	onValues := []string{}
	seen := map[string]bool{}
	add := func(value string) {
		if !seen[value] {
			seen[value] = true
			onValues = append(onValues, value)
		}
	}

	exportValues := f.ExportValues()
	if len(exportValues) != 0 {
		for _, value := range exportValues {
			add(value)
		}
		return onValues
	}

	for _, widget := range f.Widgets() {
		add(onValueForWidget(widget))
	}
	return onValues
}

// onValue returns the value that turns the widget at the given index on. Java
// declares it private.
func (f *PDButton) onValue(index int) string {
	widgets := f.Widgets()
	if index < len(widgets) {
		return onValueForWidget(widgets[index])
	}
	return ""
}

// onValueForWidget returns the appearance state of the widget that is not Off.
// Java declares it private.
func onValueForWidget(widget *annotation.PDAnnotationWidget) string {
	apDictionary := widget.Appearance()
	if apDictionary == nil {
		return ""
	}
	normalAppearance := apDictionary.NormalAppearance()
	if normalAppearance == nil {
		return ""
	}
	dict, isDictionary := normalAppearance.COSObject().(*cos.Dictionary)
	if !isDictionary {
		return ""
	}
	for _, entry := range dict.KeySet() {
		if entry != cos.Off {
			return entry.Name()
		}
	}
	return ""
}

// checkValue panics unless the value is Off or one of the on values, which is
// the IllegalArgumentException Java throws. Java declares it package-private.
func (f *PDButton) checkValue(value string) {
	onValues := f.OnValues()
	if value == cos.Off.Name() {
		return
	}
	for _, onValue := range onValues {
		if onValue == value {
			return
		}
	}
	panic(fmt.Sprintf("value '%s' is not a valid option for the field %s, "+
		"valid values are: %v and %s", value, f.FullyQualifiedName(), onValues,
		cos.Off.Name()))
}

// updateByValue sets the appearance state of each widget to the given value.
// Java declares it private.
func (f *PDButton) updateByValue(value string) {
	// Find the matching appearance key from the first widget that has it
	var matchingKey *cos.Name

	// update the appearance state (AS) for each widget
	for _, widget := range f.Widgets() {
		appearance := widget.Appearance()
		if appearance == nil {
			continue
		}
		appearanceEntry := appearance.NormalAppearance()
		appearanceDict, isDictionary := appearanceEntry.COSObject().(*cos.Dictionary)
		if !isDictionary {
			widget.SetAppearanceStateName(cos.Off)
			continue
		}

		// Find the matching appearance key by searching through the actual keys
		// and comparing their decoded names. This handles encoding differences:
		// the appearance key might be ISO-8859-1 encoded (e.g. /m#e4nnlich for "männlich")
		// while the value String is UTF-8.
		widgetMatchingKey := findMatchingAppearanceKey(appearanceDict, value)

		// Save the first matching key to use for the V entry
		if widgetMatchingKey != nil && matchingKey == nil {
			matchingKey = widgetMatchingKey
		}

		if widgetMatchingKey != nil {
			// Use the exact COSName from the appearance dictionary to preserve encoding
			widget.SetAppearanceStateName(widgetMatchingKey)
		} else {
			// Fall back to Off if no match found for this widget
			widget.SetAppearanceStateName(cos.Off)
		}
	}

	// Set the V entry once using the first matching key found
	if matchingKey != nil {
		f.FieldDictionary().SetItem(cos.V, matchingKey)
		return
	}
	// Fall back to UTF-8 encoding if no match found in any widget
	f.FieldDictionary().SetName(cos.V, value)
}

// findMatchingAppearanceKey returns the key of the appearance dictionary whose
// name reads as the given value. Java declares it private.
func findMatchingAppearanceKey(appearanceDict *cos.Dictionary, value string) *cos.Name {
	// Search all keys in the appearance dictionary and compare their decoded names
	// COSName.getName() uses UTF-8 decoding with ISO-8859-1 fallback for non-UTF-8 bytes
	for _, key := range appearanceDict.KeySet() {
		if value == key.Name() {
			return key
		}
	}
	return nil
}

// updateByOption sets the value from the /Opt options. Java declares it
// private, and throws IllegalArgumentException where the counts disagree.
func (f *PDButton) updateByOption(value string) {
	widgets := f.Widgets()
	options := f.ExportValues()

	if len(widgets) != len(options) {
		panic("The number of options doesn't match the number of widgets")
	}

	if value == cos.Off.Name() {
		f.updateByValue(value)
		return
	}

	// the value is the index of the matching option
	optionsIndex := -1
	for i, option := range options {
		if option == value {
			optionsIndex = i
			break
		}
	}

	// get the values the options are pointing to as
	// this might not be numerical
	// see PDFBOX-3682
	if optionsIndex != -1 {
		f.updateByValue(f.onValue(optionsIndex))
	}
}
