package interactive

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// AppearanceStyle is the font, size and leading an appearance is written with.
//
// Port of AppearanceStyle.
type AppearanceStyle struct {
	font     font.PDFont
	fontSize float32
	leading  float32
}

// NewAppearanceStyle returns a style of twelve point text with the leading
// Java's field initialisers give it.
func NewAppearanceStyle() *AppearanceStyle {
	return &AppearanceStyle{fontSize: 12.0, leading: 14.4}
}

// Font returns the font of the style.
func (s *AppearanceStyle) Font() font.PDFont { return s.font }

// SetFont sets the font of the style.
func (s *AppearanceStyle) SetFont(f font.PDFont) { s.font = f }

// FontSize returns the size of the style.
func (s *AppearanceStyle) FontSize() float32 { return s.fontSize }

// SetFontSize sets the size of the style, and the leading with it.
func (s *AppearanceStyle) SetFontSize(fontSize float32) {
	s.fontSize = fontSize
	s.leading = fontSize * 1.2
}

// Leading returns the leading of the style.
func (s *AppearanceStyle) Leading() float32 { return s.leading }

// SetLeading sets the leading of the style.
func (s *AppearanceStyle) SetLeading(leading float32) { s.leading = leading }
