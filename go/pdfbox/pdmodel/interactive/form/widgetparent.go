package form

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// SetWidgetParent sets the field the given widget belongs to.
//
// Port of PDAnnotationWidget.setParent, which is a function here because it
// names PDTerminalField and interactive/annotation sits below this package.
//
// Java throws IllegalArgumentException where the widget shares its dictionary
// with the field, which is unchecked, so the port panics.
func SetWidgetParent(widget *annotation.PDAnnotationWidget, field PDField) {
	if widget.AnnotationDictionary() == field.FieldDictionary() {
		panic("setParent() is not to be called for a field that shares a dictionary " +
			"with its only widget")
	}
	widget.AnnotationDictionary().SetItem(cos.Parent, field.FieldDictionary())
}
