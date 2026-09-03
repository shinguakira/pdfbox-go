package state

// PDTextState holds the current state of the text parameters when executing a
// content stream.
//
// Port of org.apache.pdfbox.pdmodel.graphics.state.PDTextState.
//
// The font and its accessors are not here. PDFont is a slice of the port that
// has not been reached, and nothing in a content stream walk can resolve one
// yet; the field arrives with the type. See migration/STATUS.md.
type PDTextState struct {
	characterSpacing  float32
	wordSpacing       float32
	horizontalScaling float32
	leading           float32
	fontSize          float32
	renderingMode     RenderingMode
	rise              float32
	knockout          bool
}

// NewPDTextState returns the text state a content stream starts in.
func NewPDTextState() *PDTextState {
	return &PDTextState{
		horizontalScaling: 100,
		renderingMode:     Fill,
		knockout:          true,
	}
}

// CharacterSpacing returns the character spacing.
func (s *PDTextState) CharacterSpacing() float32 { return s.characterSpacing }

// SetCharacterSpacing sets the character spacing.
func (s *PDTextState) SetCharacterSpacing(value float32) { s.characterSpacing = value }

// WordSpacing returns the word spacing.
func (s *PDTextState) WordSpacing() float32 { return s.wordSpacing }

// SetWordSpacing sets the word spacing.
func (s *PDTextState) SetWordSpacing(value float32) { s.wordSpacing = value }

// HorizontalScaling returns the horizontal scaling. The default is 100. This
// value is the percentage value 0-100 and not 0-1, so for mathematical
// operations you will probably need to divide by 100 first.
func (s *PDTextState) HorizontalScaling() float32 { return s.horizontalScaling }

// SetHorizontalScaling sets the horizontal scaling.
func (s *PDTextState) SetHorizontalScaling(value float32) { s.horizontalScaling = value }

// Leading returns the leading.
func (s *PDTextState) Leading() float32 { return s.leading }

// SetLeading sets the leading.
func (s *PDTextState) SetLeading(value float32) { s.leading = value }

// FontSize returns the font size.
func (s *PDTextState) FontSize() float32 { return s.fontSize }

// SetFontSize sets the font size.
func (s *PDTextState) SetFontSize(value float32) { s.fontSize = value }

// RenderingMode returns the text rendering mode.
func (s *PDTextState) RenderingMode() RenderingMode { return s.renderingMode }

// SetRenderingMode sets the text rendering mode.
func (s *PDTextState) SetRenderingMode(renderingMode RenderingMode) {
	s.renderingMode = renderingMode
}

// Rise returns the text rise.
func (s *PDTextState) Rise() float32 { return s.rise }

// SetRise sets the text rise.
func (s *PDTextState) SetRise(value float32) { s.rise = value }

// KnockoutFlag returns the knockout flag.
func (s *PDTextState) KnockoutFlag() bool { return s.knockout }

// SetKnockoutFlag sets the knockout flag.
func (s *PDTextState) SetKnockoutFlag(value bool) { s.knockout = value }

// Clone returns an independent copy of this text state.
func (s *PDTextState) Clone() *PDTextState {
	clone := *s
	return &clone
}
