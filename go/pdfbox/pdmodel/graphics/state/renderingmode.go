package state

// RenderingMode is a text rendering mode.
//
// Port of the enum org.apache.pdfbox.pdmodel.graphics.state.RenderingMode.
type RenderingMode int

const (
	// Fill text.
	Fill RenderingMode = iota

	// Stroke text.
	Stroke

	// FillStroke fills, then strokes text.
	FillStroke

	// Neither fills nor strokes text (invisible).
	Neither

	// FillClip fills text and adds it to the path for clipping.
	FillClip

	// StrokeClip strokes text and adds it to the path for clipping.
	StrokeClip

	// FillStrokeClip fills, then strokes text and adds it to the path for
	// clipping.
	FillStrokeClip

	// NeitherClip adds text to the path for clipping.
	NeitherClip
)

// renderingModeValues is the equivalent of the private VALUES array Java caches
// so that fromInt does not copy values() on every call.
var renderingModeValues = [...]RenderingMode{
	Fill, Stroke, FillStroke, Neither,
	FillClip, StrokeClip, FillStrokeClip, NeitherClip,
}

var renderingModeNames = [...]string{
	Fill:           "FILL",
	Stroke:         "STROKE",
	FillStroke:     "FILL_STROKE",
	Neither:        "NEITHER",
	FillClip:       "FILL_CLIP",
	StrokeClip:     "STROKE_CLIP",
	FillStrokeClip: "FILL_STROKE_CLIP",
	NeitherClip:    "NEITHER_CLIP",
}

// RenderingModeFromInt returns the mode with the given integer value.
//
// An out-of-range value is not mapped to a default: Java indexes the values
// array with it and throws, and every caller checks the range first, so this
// panics rather than inventing a mode the file did not ask for.
func RenderingModeFromInt(value int) RenderingMode {
	return renderingModeValues[value]
}

// IntValue returns the integer value of this mode, as used in a PDF file.
func (r RenderingMode) IntValue() int { return int(r) }

// IsFill reports whether this mode fills text.
func (r RenderingMode) IsFill() bool {
	return r == Fill ||
		r == FillStroke ||
		r == FillClip ||
		r == FillStrokeClip
}

// IsStroke reports whether this mode strokes text.
func (r RenderingMode) IsStroke() bool {
	return r == Stroke ||
		r == FillStroke ||
		r == StrokeClip ||
		r == FillStrokeClip
}

// IsClip reports whether this mode clips text.
func (r RenderingMode) IsClip() bool {
	return r == FillClip ||
		r == StrokeClip ||
		r == FillStrokeClip ||
		r == NeitherClip
}

// String returns the Java enum constant name.
func (r RenderingMode) String() string {
	return renderingModeNames[r]
}
