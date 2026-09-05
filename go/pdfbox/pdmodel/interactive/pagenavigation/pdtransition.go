// Package pagenavigation holds the article threads a document defines and the
// transitions it plays between pages.
//
// Port of org.apache.pdfbox.pdmodel.interactive.pagenavigation.
package pagenavigation

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDTransitionStyle is the visual effect played when a page is turned.
//
// Port of the enum PDTransitionStyle. Java writes the constant's own name into
// the file, so the port is a string type whose values are those names.
type PDTransitionStyle string

const (
	// TransitionStyleSplit wipes two lines apart or together.
	TransitionStyleSplit PDTransitionStyle = "Split"
	// TransitionStyleBlinds sweeps several lines at once.
	TransitionStyleBlinds PDTransitionStyle = "Blinds"
	// TransitionStyleBox sweeps a rectangle out from or in to the centre.
	TransitionStyleBox PDTransitionStyle = "Box"
	// TransitionStyleWipe sweeps a single line across.
	TransitionStyleWipe PDTransitionStyle = "Wipe"
	// TransitionStyleDissolve dissolves the old page into the new one.
	TransitionStyleDissolve PDTransitionStyle = "Dissolve"
	// TransitionStyleGlitter dissolves in a sweep.
	TransitionStyleGlitter PDTransitionStyle = "Glitter"
	// TransitionStyleR replaces the page with no effect, and is the default.
	TransitionStyleR PDTransitionStyle = "R"
	// TransitionStyleFly flies the old page off.
	TransitionStyleFly PDTransitionStyle = "Fly"
	// TransitionStylePush pushes the old page off.
	TransitionStylePush PDTransitionStyle = "Push"
	// TransitionStyleCover slides the new page over the old one.
	TransitionStyleCover PDTransitionStyle = "Cover"
	// TransitionStyleUncover slides the old page off the new one.
	TransitionStyleUncover PDTransitionStyle = "Uncover"
	// TransitionStyleFade fades the new page in.
	TransitionStyleFade PDTransitionStyle = "Fade"
)

// PDTransitionDimension is the axis a transition runs along.
//
// Port of the enum PDTransitionDimension.
type PDTransitionDimension string

const (
	// TransitionDimensionH is horizontal.
	TransitionDimensionH PDTransitionDimension = "H"
	// TransitionDimensionV is vertical.
	TransitionDimensionV PDTransitionDimension = "V"
)

// PDTransitionMotion is whether a transition moves inward or outward.
//
// Port of the enum PDTransitionMotion.
type PDTransitionMotion string

const (
	// TransitionMotionI is inward.
	TransitionMotionI PDTransitionMotion = "I"
	// TransitionMotionO is outward.
	TransitionMotionO PDTransitionMotion = "O"
)

// PDTransitionDirection is the direction a transition runs in.
//
// Port of the enum PDTransitionDirection. Every constant but NONE writes its
// angle in degrees; NONE overrides getCOSBase to write /None, which the port
// does with a switch rather than a constant-specific body.
type PDTransitionDirection int

const (
	// TransitionDirectionLeftToRight runs left to right.
	TransitionDirectionLeftToRight PDTransitionDirection = iota
	// TransitionDirectionBottomToTop runs bottom to top.
	TransitionDirectionBottomToTop
	// TransitionDirectionRightToLeft runs right to left.
	TransitionDirectionRightToLeft
	// TransitionDirectionTopToBottom runs top to bottom.
	TransitionDirectionTopToBottom
	// TransitionDirectionTopLeftToBottomRight runs diagonally.
	TransitionDirectionTopLeftToBottomRight
	// TransitionDirectionNone has no direction.
	TransitionDirectionNone
)

// degrees returns the angle Java stores on the constant.
func (d PDTransitionDirection) degrees() int64 {
	switch d {
	case TransitionDirectionBottomToTop:
		return 90
	case TransitionDirectionRightToLeft:
		return 180
	case TransitionDirectionTopToBottom:
		return 270
	case TransitionDirectionTopLeftToBottomRight:
		return 315
	}
	// LEFT_TO_RIGHT and NONE are both declared with 0.
	return 0
}

// COSBase returns what this direction is written as.
func (d PDTransitionDirection) COSBase() cos.Base {
	if d == TransitionDirectionNone {
		return cos.None
	}
	return cos.GetInteger(d.degrees())
}

// PDTransition is the transition played when a page is turned.
//
// Port of PDTransition, which extends PDDictionaryWrapper.
type PDTransition struct {
	common.PDDictionaryWrapper
}

// NewPDTransition creates a transition with the default style, R.
func NewPDTransition() *PDTransition {
	return NewPDTransitionOfStyle(TransitionStyleR)
}

// NewPDTransitionOfStyle creates a transition with the given style.
func NewPDTransitionOfStyle(style PDTransitionStyle) *PDTransition {
	t := &PDTransition{PDDictionaryWrapper: *common.NewPDDictionaryWrapper()}
	t.Dictionary().SetName(cos.Type, cos.Trans.Name())
	t.Dictionary().SetName(cos.S, string(style))
	return t
}

// NewPDTransitionOf creates a transition over the given dictionary.
func NewPDTransitionOf(dictionary *cos.Dictionary) *PDTransition {
	return &PDTransition{PDDictionaryWrapper: *common.NewPDDictionaryWrapperOf(dictionary)}
}

// Style returns the /S entry, which defaults to R.
func (t *PDTransition) Style() string {
	return t.Dictionary().GetNameAsString(cos.S, string(TransitionStyleR))
}

// Dimension returns the /Dm entry, which defaults to H.
func (t *PDTransition) Dimension() string {
	return t.Dictionary().GetNameAsString(cos.Dm, string(TransitionDimensionH))
}

// SetDimension sets the /Dm entry.
func (t *PDTransition) SetDimension(dimension PDTransitionDimension) {
	t.Dictionary().SetName(cos.Dm, string(dimension))
}

// Motion returns the /M entry, which defaults to I.
func (t *PDTransition) Motion() string {
	return t.Dictionary().GetNameAsString(cos.M, string(TransitionMotionI))
}

// SetMotion sets the /M entry.
func (t *PDTransition) SetMotion(motion PDTransitionMotion) {
	t.Dictionary().SetName(cos.M, string(motion))
}

// Direction returns the /Di entry, which is either an angle or /None, and
// which defaults to the integer zero.
func (t *PDTransition) Direction() cos.Base {
	item := t.Dictionary().GetItem(cos.Di)
	if item == nil {
		return cos.GetInteger(0)
	}
	return item
}

// SetDirection sets the /Di entry.
func (t *PDTransition) SetDirection(direction PDTransitionDirection) {
	t.Dictionary().SetItem(cos.Di, direction.COSBase())
}

// Duration returns the /D entry in seconds, which defaults to 1.
func (t *PDTransition) Duration() float32 {
	return t.Dictionary().GetFloat(cos.D, 1)
}

// SetDuration sets the /D entry in seconds.
func (t *PDTransition) SetDuration(duration float32) {
	t.Dictionary().SetItem(cos.D, cos.NewFloat(duration))
}

// FlyScale returns the /SS entry, which defaults to 1.
func (t *PDTransition) FlyScale() float32 {
	return t.Dictionary().GetFloat(cos.SS, 1)
}

// SetFlyScale sets the /SS entry.
func (t *PDTransition) SetFlyScale(scale float32) {
	t.Dictionary().SetItem(cos.SS, cos.NewFloat(scale))
}

// IsFlyAreaOpaque reports the /B entry, which defaults to false.
func (t *PDTransition) IsFlyAreaOpaque() bool {
	return t.Dictionary().GetBoolean(cos.B, false)
}

// SetFlyAreaOpaque sets the /B entry.
func (t *PDTransition) SetFlyAreaOpaque(opaque bool) {
	t.Dictionary().SetItem(cos.B, cos.GetBoolean(opaque))
}
