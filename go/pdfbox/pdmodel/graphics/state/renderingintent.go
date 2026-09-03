// Package state holds the graphics state of a content stream.
//
// Port of org.apache.pdfbox.pdmodel.graphics.state.
package state

// RenderingIntent is a rendering intent.
//
// Port of the enum org.apache.pdfbox.pdmodel.graphics.state.RenderingIntent.
// Java enum constants are reference-identity singletons; a Go defined type with
// constants gives the same value semantics, and the declaration order is the
// ordinal order Java would report.
type RenderingIntent int

const (
	// AbsoluteColorimetric is Absolute Colorimetric.
	AbsoluteColorimetric RenderingIntent = iota

	// RelativeColorimetric is Relative Colorimetric.
	RelativeColorimetric

	// Saturation is Saturation.
	Saturation

	// Perceptual is Perceptual.
	Perceptual
)

// renderingIntentValues holds the string value of each intent, as used in a PDF
// file, in ordinal order.
var renderingIntentValues = [...]string{
	AbsoluteColorimetric: "AbsoluteColorimetric",
	RelativeColorimetric: "RelativeColorimetric",
	Saturation:           "Saturation",
	Perceptual:           "Perceptual",
}

// renderingIntentNames holds the Java constant names, which are what a Java
// enum prints.
var renderingIntentNames = [...]string{
	AbsoluteColorimetric: "ABSOLUTE_COLORIMETRIC",
	RelativeColorimetric: "RELATIVE_COLORIMETRIC",
	Saturation:           "SATURATION",
	Perceptual:           "PERCEPTUAL",
}

// RenderingIntentFromString returns the intent with the given PDF name.
func RenderingIntentFromString(value string) RenderingIntent {
	for instance, name := range renderingIntentValues {
		if name == value {
			return RenderingIntent(instance)
		}
	}
	// "If a conforming reader does not recognize the specified name,
	// it shall use the RelativeColorimetric intent by default."
	return RelativeColorimetric
}

// StringValue returns the string value, as used in a PDF file.
func (r RenderingIntent) StringValue() string {
	return renderingIntentValues[r]
}

// String returns the Java enum constant name.
func (r RenderingIntent) String() string {
	return renderingIntentNames[r]
}
