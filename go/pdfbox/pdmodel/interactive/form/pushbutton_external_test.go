package form_test

import (
	"fmt"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// TestPushButtonSettersCheckAgainstThePushButton pins the calls PDButton's own
// methods make that a PDPushButton answers differently.
//
// PDButton.setValue(String), setValue(int) and setDefaultValue check the value
// against getOnValues() and getExportValues(), and both are virtual: on a
// PDPushButton they are the push button's, which are always empty, so every
// value but Off is refused. The port's PDButton called its own, which read the
// widget's appearance states and the /Opt entry, so a push button with an On
// state accepted "On", and one with an /Opt accepted an index into it.
//
// The button below has a widget with On and Off normal appearances, and the
// second one an /Opt of (a). What the running PDFBox answered for the same two:
//
//	setValue("On")                  threw: value 'On' is not a valid option for the field push, valid values are: [] and Off
//	setValue("Off")                 returned; V=Off AS=Off
//	setDefaultValue("On")           threw: value 'On' is not a valid option ...
//	setDefaultValue("Off")          returned; DV=Off
//	with /Opt: setValue(0)          threw: index '0' is not a valid index for the field push, valid indices are from 0 to -1
//	with /Opt: setValue("a")        threw: value 'a' is not a valid option ...
//	with /Opt: setValue("Off")      returned; V=Off AS=Off
//
// IllegalArgumentException is unchecked, so the port panics where Java throws.
func TestPushButtonSettersCheckAgainstThePushButton(t *testing.T) {
	newButton := func(withOpt bool) *form.PDPushButton {
		button := form.NewPDPushButton(form.NewPDAcroForm(pdmodel.NewPDDocument()))
		button.SetPartialName("push")
		normal := cos.NewDictionary()
		normal.SetItem(cos.GetPDFName("On"), cos.NewStream(nil))
		normal.SetItem(cos.Off, cos.NewStream(nil))
		appearance := cos.NewDictionary()
		appearance.SetItem(cos.N, normal)
		button.Widgets()[0].AnnotationDictionary().SetItem(cos.AP, appearance)
		if withOpt {
			button.FieldDictionary().SetItem(cos.Opt, cos.ArrayOfStrings([]string{"a"}))
		}
		return button
	}
	panicOf := func(call func()) (message string) {
		defer func() {
			if recovered := recover(); recovered != nil {
				message = fmt.Sprint(recovered)
			}
		}()
		call()
		return ""
	}
	state := func(button *form.PDPushButton) string {
		return fmt.Sprintf("V=%v DV=%v AS=%v",
			button.FieldDictionary().GetDictionaryObject(cos.V),
			button.FieldDictionary().GetDictionaryObject(cos.DV),
			button.Widgets()[0].AnnotationDictionary().GetDictionaryObject(cos.AS))
	}
	const refusedOn = "value 'On' is not a valid option for the field push, valid values are: [] and Off"

	button := newButton(false)
	if got := panicOf(func() { button.SetValue("On") }); got != refusedOn { //nolint:errcheck // it panics
		t.Errorf("SetValue(On) panicked with %q, want %q", got, refusedOn)
	}
	if got := panicOf(func() {
		if err := button.SetValue("Off"); err != nil {
			t.Errorf("SetValue(Off): %v", err)
		}
	}); got != "" {
		t.Errorf("SetValue(Off) panicked: %s", got)
	}
	if got, want := state(button), "V=COSName{Off} DV=<nil> AS=COSName{Off}"; got != want {
		t.Errorf("after SetValue(Off): %s, want %s", got, want)
	}
	if got := panicOf(func() { button.SetDefaultValue("On") }); got != refusedOn {
		t.Errorf("SetDefaultValue(On) panicked with %q, want %q", got, refusedOn)
	}
	if got := panicOf(func() { button.SetDefaultValue("Off") }); got != "" {
		t.Errorf("SetDefaultValue(Off) panicked: %s", got)
	}

	withOpt := newButton(true)
	const refusedIndex = "index '0' is not a valid index for the field push, valid indices are from 0 to -1"
	if got := panicOf(func() { withOpt.SetValueIndex(0) }); got != refusedIndex { //nolint:errcheck // it panics
		t.Errorf("SetValueIndex(0) panicked with %q, want %q", got, refusedIndex)
	}
	const refusedA = "value 'a' is not a valid option for the field push, valid values are: [] and Off"
	if got := panicOf(func() { withOpt.SetValue("a") }); got != refusedA { //nolint:errcheck // it panics
		t.Errorf("SetValue(a) panicked with %q, want %q", got, refusedA)
	}
	if got := panicOf(func() {
		if err := withOpt.SetValue("Off"); err != nil {
			t.Errorf("SetValue(Off) with /Opt: %v", err)
		}
	}); got != "" {
		t.Errorf("SetValue(Off) with /Opt panicked: %s", got)
	}
	if got, want := state(withOpt), "V=COSName{Off} DV=<nil> AS=COSName{Off}"; got != want {
		t.Errorf("after SetValue(Off) with /Opt: %s, want %s", got, want)
	}
}
