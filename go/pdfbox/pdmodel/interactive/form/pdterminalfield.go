package form

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/action"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// terminalField is what a concrete terminal field carries beyond PDField: the
// appearance construction the base calls through self.
type terminalField interface {
	PDField

	// constructAppearances is the abstract method of the same name.
	constructAppearances() error
}

// PDTerminalField is a field that holds a value, rather than one that only
// gathers other fields.
//
// Port of the abstract PDTerminalField.
type PDTerminalField struct {
	PDFieldBase
}

// initTerminalField is the protected PDTerminalField(PDAcroForm) constructor.
func (f *PDTerminalField) initTerminalField(self PDField, acroForm *PDAcroForm) {
	f.initField(self, acroForm)
}

// initTerminalFieldOf is the package-private PDTerminalField(PDAcroForm,
// COSDictionary, PDNonTerminalField) constructor.
func (f *PDTerminalField) initTerminalFieldOf(self PDField, acroForm *PDAcroForm,
	field *cos.Dictionary, parent *PDNonTerminalField) {
	f.initFieldOf(self, acroForm, field, parent)
}

// SetActions sets the /AA additional actions of the field.
func (f *PDTerminalField) SetActions(actions *action.PDFormFieldAdditionalActions) {
	if actions == nil {
		f.FieldDictionary().SetItem(cos.AA, nil)
		return
	}
	f.FieldDictionary().SetItem(cos.AA, actions.COSObject())
}

// FieldFlags returns the /Ff flags of the field, or of its nearest parent that
// has them.
//
// Java casts the entry to COSInteger without a check; the port asserts the same
// way.
func (f *PDTerminalField) FieldFlags() int {
	retval := 0
	if ff := f.FieldDictionary().GetDictionaryObject(cos.Ff); ff != nil {
		retval = ff.(*cos.Integer).IntValue()
	} else if f.Parent() != nil {
		retval = f.Parent().FieldFlags()
	}
	return retval
}

// FieldType returns the /FT of the field, or of its nearest parent that has it.
func (f *PDTerminalField) FieldType() string {
	fieldType := f.FieldDictionary().GetNameAsString(cos.FT, "")
	if fieldType == "" && f.Parent() != nil {
		fieldType = f.Parent().FieldType()
	}
	return fieldType
}

// Widgets returns the widget annotations of the field, which is the field
// itself where it has no kids.
func (f *PDTerminalField) Widgets() []*annotation.PDAnnotationWidget {
	widgets := []*annotation.PDAnnotationWidget{}
	kids := f.FieldDictionary().GetCOSArray(cos.Kids)
	if kids == nil {
		// the field itself is a widget
		widgets = append(widgets, annotation.NewPDAnnotationWidgetOf(f.FieldDictionary()))
		return widgets
	}
	if !kids.IsEmpty() {
		// there are multiple widgets
		for i := 0; i < kids.Size(); i++ {
			if kid, isDictionary := kids.GetObject(i).(*cos.Dictionary); isDictionary {
				widgets = append(widgets, annotation.NewPDAnnotationWidgetOf(kid))
			}
		}
	}
	return widgets
}

// SetWidgets sets the widget annotations of the field, and makes the field
// their parent.
func (f *PDTerminalField) SetWidgets(children []*annotation.PDAnnotationWidget) {
	kidsArray := cos.NewArray()
	for _, widget := range children {
		kidsArray.Add(widget.COSObject())
	}
	f.FieldDictionary().SetItem(cos.Kids, kidsArray)
	for _, widget := range children {
		widget.AnnotationDictionary().SetItem(cos.Parent, f.FieldDictionary())
	}
}

// applyChange rebuilds the appearance of the field after its value changed.
// Java declares it protected and final.
func (f *PDTerminalField) applyChange() error {
	terminal, isTerminal := f.self.(terminalField)
	if !isTerminal {
		return nil
	}
	// if we supported JavaScript we would raise a field changed event here
	return terminal.constructAppearances()
}
