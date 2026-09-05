package interactive

// TextAlign is how a line of text sits in the width it is given.
//
// Port of the enum TextAlign. Java enum constants are reference-identity
// singletons; a Go defined type with constants gives the same value semantics,
// and the value of each is the /Q quadding it comes from.
type TextAlign int

const (
	// TextAlignLeft is LEFT.
	TextAlignLeft TextAlign = 0

	// TextAlignCenter is CENTER.
	TextAlignCenter TextAlign = 1

	// TextAlignRight is RIGHT.
	TextAlignRight TextAlign = 2

	// TextAlignJustify is JUSTIFY.
	TextAlignJustify TextAlign = 4
)

// textAlignNames holds the Java constant names, which are what a Java enum
// prints.
var textAlignNames = map[TextAlign]string{
	TextAlignLeft:    "LEFT",
	TextAlignCenter:  "CENTER",
	TextAlignRight:   "RIGHT",
	TextAlignJustify: "JUSTIFY",
}

// TextAlignValue returns the alignment of the style. Java declares
// getTextAlign package-private.
func (a TextAlign) TextAlignValue() int { return int(a) }

// String returns the Java enum constant name.
func (a TextAlign) String() string {
	if name, known := textAlignNames[a]; known {
		return name
	}
	return "LEFT"
}

// TextAlignOf returns the alignment with the given value, and left where none
// has it.
//
// Port of the static TextAlign.valueOf(int).
func TextAlignOf(alignment int) TextAlign {
	for _, textAlignment := range []TextAlign{
		TextAlignLeft, TextAlignCenter, TextAlignRight, TextAlignJustify,
	} {
		if int(textAlignment) == alignment {
			return textAlignment
		}
	}
	return TextAlignLeft
}
