package state

import "testing"

// Written from org.apache.pdfbox.pdmodel.graphics.state.PDTextState; the Java
// suite has no test for it.

// TestPDTextStateDefaults pins the values a content stream starts with, which
// are not all zero: horizontal scaling is a percentage and knockout is set.
func TestPDTextStateDefaults(t *testing.T) {
	s := NewPDTextState()

	if got := s.CharacterSpacing(); got != 0 {
		t.Errorf("CharacterSpacing = %v, want 0", got)
	}
	if got := s.WordSpacing(); got != 0 {
		t.Errorf("WordSpacing = %v, want 0", got)
	}
	if got := s.HorizontalScaling(); got != 100 {
		t.Errorf("HorizontalScaling = %v, want 100", got)
	}
	if got := s.Leading(); got != 0 {
		t.Errorf("Leading = %v, want 0", got)
	}
	if got := s.FontSize(); got != 0 {
		t.Errorf("FontSize = %v, want 0", got)
	}
	if got := s.RenderingMode(); got != Fill {
		t.Errorf("RenderingMode = %v, want FILL", got)
	}
	if got := s.Rise(); got != 0 {
		t.Errorf("Rise = %v, want 0", got)
	}
	if !s.KnockoutFlag() {
		t.Error("KnockoutFlag = false, want true")
	}
}

func TestPDTextStateSetters(t *testing.T) {
	s := NewPDTextState()
	s.SetCharacterSpacing(1)
	s.SetWordSpacing(2)
	s.SetHorizontalScaling(50)
	s.SetLeading(3)
	s.SetFontSize(12)
	s.SetRenderingMode(StrokeClip)
	s.SetRise(4)
	s.SetKnockoutFlag(false)

	if s.CharacterSpacing() != 1 || s.WordSpacing() != 2 || s.HorizontalScaling() != 50 ||
		s.Leading() != 3 || s.FontSize() != 12 || s.RenderingMode() != StrokeClip ||
		s.Rise() != 4 || s.KnockoutFlag() {
		t.Errorf("a setter did not take: %+v", s)
	}
}

func TestPDTextStateCloneIsIndependent(t *testing.T) {
	s := NewPDTextState()
	s.SetFontSize(12)

	clone := s.Clone()
	clone.SetFontSize(24)

	if got := s.FontSize(); got != 12 {
		t.Errorf("the original changed to %v", got)
	}
	if got := clone.FontSize(); got != 24 {
		t.Errorf("the clone is %v, want 24", got)
	}
}
