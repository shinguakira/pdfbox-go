// Package handlers draws the appearance streams of the annotations that have
// no appearance of their own.
//
// Port of org.apache.pdfbox.pdmodel.interactive.annotation.handlers. Java's
// handlers and the annotations reference each other, which Go forbids: an
// annotation reaches its handler through annotation.DefaultAppearanceHandlers,
// which this package fills from its init, and pdmodel imports this package for
// the side effect, so that a program that has the document model has the
// handlers too.
//
// Java's generateNormalAppearance and its two siblings return void and log the
// IOException they catch; the port keeps that -- it logs and answers nil where
// Java logs -- and returns an error only where Java lets one out.
package handlers

import (
	"math"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// arrowAngle is the angle of an arrow head, in radians.
//
// Port of the package-private PDAbstractAppearanceHandler.ARROW_ANGLE.
var arrowAngle = 30 * math.Pi / 180

// shortStyles are the line endings that shorten the line they end.
//
// Port of the protected SHORT_STYLES.
var shortStyles = map[string]bool{
	annotation.LEOpenArrow:   true,
	annotation.LEClosedArrow: true,
	annotation.LESquare:      true,
	annotation.LECircle:      true,
	annotation.LEDiamond:     true,
}

// interiorColorStyles are the line endings that are filled with the interior
// colour.
//
// Port of the protected INTERIOR_COLOR_STYLES.
var interiorColorStyles = map[string]bool{
	annotation.LEClosedArrow:  true,
	annotation.LECircle:       true,
	annotation.LEDiamond:      true,
	annotation.LERClosedArrow: true,
	annotation.LESquare:       true,
}

// angledStyles are the line endings drawn along the direction of the line.
//
// Port of the protected ANGLED_STYLES.
var angledStyles = map[string]bool{
	annotation.LEClosedArrow:  true,
	annotation.LEOpenArrow:    true,
	annotation.LERClosedArrow: true,
	annotation.LEROpenArrow:   true,
	annotation.LEButt:         true,
	annotation.LESlash:        true,
}

// appearanceHandler is the three methods a concrete handler carries, which the
// base calls through self, since Go embedding does not dispatch.
type appearanceHandler interface {
	annotation.PDAppearanceHandler

	// GenerateNormalAppearance draws the normal appearance.
	GenerateNormalAppearance() error

	// GenerateRolloverAppearance draws the rollover appearance.
	GenerateRolloverAppearance() error

	// GenerateDownAppearance draws the down appearance.
	GenerateDownAppearance() error
}

// PDAbstractAppearanceHandler carries the state and the drawing every handler
// shares.
//
// Port of the abstract PDAbstractAppearanceHandler.
type PDAbstractAppearanceHandler struct {
	self        appearanceHandler
	annot       annotation.PDAnnotation
	defaultFont font.PDFont

	// document is the document the appearance streams belong to, which may be
	// nil. Java declares the field protected.
	document common.COSDocumentLike
}

// initAppearanceHandler is the protected
// PDAbstractAppearanceHandler(PDAnnotation, PDDocument) constructor. A concrete
// handler calls it from its own constructor with itself as self.
func (h *PDAbstractAppearanceHandler) initAppearanceHandler(self appearanceHandler,
	annot annotation.PDAnnotation, document common.COSDocumentLike) {
	h.self = self
	h.annot = annot
	h.document = document
}

// GenerateAppearanceStreams draws the normal, rollover and down appearances.
//
// Port of the default method PDAppearanceHandler.generateAppearanceStreams.
func (h *PDAbstractAppearanceHandler) GenerateAppearanceStreams() error {
	if err := h.self.GenerateNormalAppearance(); err != nil {
		return err
	}
	if err := h.self.GenerateRolloverAppearance(); err != nil {
		return err
	}
	return h.self.GenerateDownAppearance()
}

// DefaultFont returns the font a handler draws text with, which is Helvetica.
// Java declares it protected.
func (h *PDAbstractAppearanceHandler) DefaultFont() (font.PDFont, error) {
	if h.defaultFont == nil {
		helvetica, err := font.NewPDType1FontStandard14(font.Helvetica)
		if err != nil {
			return nil, err
		}
		h.defaultFont = helvetica
	}
	return h.defaultFont, nil
}

// Annotation returns the annotation being drawn. Java declares it
// package-private.
func (h *PDAbstractAppearanceHandler) Annotation() annotation.PDAnnotation {
	return h.annot
}

// Color returns the colour of the annotation. Java declares it package-private.
func (h *PDAbstractAppearanceHandler) Color() *color.PDColor {
	return h.annot.Color()
}

// Rectangle returns the rectangle of the annotation. Java declares it
// package-private.
func (h *PDAbstractAppearanceHandler) Rectangle() *common.PDRectangle {
	return h.annot.Rectangle()
}

// createCOSStream returns a new stream, belonging to the document where there
// is one. Java declares it protected.
func (h *PDAbstractAppearanceHandler) createCOSStream() *cos.Stream {
	if h.document == nil {
		return cos.NewStream(filter.Provider{})
	}
	return h.document.CreateStream()
}

// Appearance returns the appearance dictionary of the annotation, adding an
// empty one where it has none. Java declares it package-private.
func (h *PDAbstractAppearanceHandler) Appearance() *annotation.PDAppearanceDictionary {
	appearanceDictionary := h.annot.Appearance()
	if appearanceDictionary == nil {
		appearanceDictionary = annotation.NewPDAppearanceDictionary()
		h.annot.SetAppearance(appearanceDictionary)
	}
	return appearanceDictionary
}

// NormalAppearanceAsContentStream returns the normal appearance to draw into,
// uncompressed. Java declares it package-private.
func (h *PDAbstractAppearanceHandler) NormalAppearanceAsContentStream() (
	annotation.AppearanceContentStream, error) {
	return h.NormalAppearanceAsContentStreamCompressed(false)
}

// NormalAppearanceAsContentStreamCompressed returns the normal appearance to
// draw into. Java declares it package-private.
func (h *PDAbstractAppearanceHandler) NormalAppearanceAsContentStreamCompressed(
	compress bool) (annotation.AppearanceContentStream, error) {
	return h.appearanceEntryAsContentStream(h.normalAppearance(), compress)
}

// DownAppearance returns the down appearance entry, replacing a sub-dictionary
// with a fresh stream. Java declares it package-private.
func (h *PDAbstractAppearanceHandler) DownAppearance() *annotation.PDAppearanceEntry {
	appearanceDictionary := h.Appearance()
	downAppearanceEntry := appearanceDictionary.DownAppearance()
	if downAppearanceEntry.IsSubDictionary() {
		downAppearanceEntry = annotation.NewPDAppearanceEntryOf(h.createCOSStream())
		appearanceDictionary.SetDownAppearance(downAppearanceEntry)
	}
	return downAppearanceEntry
}

// RolloverAppearance returns the rollover appearance entry, replacing a
// sub-dictionary with a fresh stream. Java declares it package-private.
func (h *PDAbstractAppearanceHandler) RolloverAppearance() *annotation.PDAppearanceEntry {
	appearanceDictionary := h.Appearance()
	rolloverAppearanceEntry := appearanceDictionary.RolloverAppearance()
	if rolloverAppearanceEntry.IsSubDictionary() {
		rolloverAppearanceEntry = annotation.NewPDAppearanceEntryOf(h.createCOSStream())
		appearanceDictionary.SetRolloverAppearance(rolloverAppearanceEntry)
	}
	return rolloverAppearanceEntry
}

// PaddedRectangle returns the given rectangle inset by the given padding on
// every side. Java declares it package-private.
func (h *PDAbstractAppearanceHandler) PaddedRectangle(rectangle *common.PDRectangle,
	padding float32) *common.PDRectangle {
	return common.NewPDRectangleOf(rectangle.LowerLeftX()+padding,
		rectangle.LowerLeftY()+padding,
		rectangle.Width()-2*padding,
		rectangle.Height()-2*padding)
}

// AddRectDifferences grows the given rectangle by the four differences. Java
// declares it package-private.
func (h *PDAbstractAppearanceHandler) AddRectDifferences(rectangle *common.PDRectangle,
	differences []float32) *common.PDRectangle {
	if len(differences) != 4 {
		return rectangle
	}
	return common.NewPDRectangleOf(rectangle.LowerLeftX()-differences[0],
		rectangle.LowerLeftY()-differences[1],
		rectangle.Width()+differences[0]+differences[2],
		rectangle.Height()+differences[1]+differences[3])
}

// ApplyRectDifferences shrinks the given rectangle by the four differences.
// Java declares it package-private.
func (h *PDAbstractAppearanceHandler) ApplyRectDifferences(rectangle *common.PDRectangle,
	differences []float32) *common.PDRectangle {
	if len(differences) != 4 {
		return rectangle
	}
	return common.NewPDRectangleOf(rectangle.LowerLeftX()+differences[0],
		rectangle.LowerLeftY()+differences[1],
		rectangle.Width()-differences[0]-differences[2],
		rectangle.Height()-differences[1]-differences[3])
}

// SetOpacity writes the given opacity into the content stream, where it is not
// already full. Java declares it package-private.
func (h *PDAbstractAppearanceHandler) SetOpacity(contentStream annotation.AppearanceContentStream,
	opacity float32) error {
	if opacity < 1 {
		gs := state.NewPDExtendedGraphicsState()
		gs.SetStrokingAlphaConstant(&opacity)
		gs.SetNonStrokingAlphaConstant(&opacity)
		return contentStream.SetGraphicsStateParameters(gs)
	}
	return nil
}

// DrawStyle draws one line ending. Java declares it package-private.
func (h *PDAbstractAppearanceHandler) DrawStyle(style string,
	cs annotation.AppearanceContentStream, x, y, width float32,
	hasStroke, hasBackground, ending bool) error {
	sign := float32(1)
	if ending {
		sign = -1
	}
	var err error
	switch style {
	case annotation.LEOpenArrow, annotation.LEClosedArrow:
		err = h.DrawArrow(cs, x+sign*width, y, sign*width*9)
	case annotation.LEButt:
		if err = cs.MoveTo(x, y-width*3); err == nil {
			err = cs.LineTo(x, y+width*3)
		}
	case annotation.LEDiamond:
		err = h.DrawDiamond(cs, x, y, width*3)
	case annotation.LESquare:
		err = cs.AddRect(x-width*3, y-width*3, width*6, width*6)
	case annotation.LECircle:
		err = h.DrawCircle(cs, x, y, width*3)
	case annotation.LEROpenArrow, annotation.LERClosedArrow:
		err = h.DrawArrow(cs, x+(-sign)*width, y, (-sign)*width*9)
	case annotation.LESlash:
		width9 := float64(width * 9)
		// the line is 18 x linewidth at an angle of 60 degrees
		if err = cs.MoveTo(x+float32(math.Cos(60*math.Pi/180)*width9),
			y+float32(math.Sin(60*math.Pi/180)*width9)); err == nil {
			err = cs.LineTo(x+float32(math.Cos(240*math.Pi/180)*width9),
				y+float32(math.Sin(240*math.Pi/180)*width9))
		}
	default:
		return nil
	}
	if err != nil {
		return err
	}
	if style == annotation.LERClosedArrow || style == annotation.LEClosedArrow {
		if err := cs.ClosePath(); err != nil {
			return err
		}
	}
	// make sure to only paint a background color (/IC value)
	// for interior color styles, even if an /IC value is set.
	return cs.DrawShape(width, hasStroke, interiorColorStyles[style] && hasBackground)
}

// DrawArrow draws one arrow head. Java declares it package-private.
func (h *PDAbstractAppearanceHandler) DrawArrow(cs annotation.AppearanceContentStream,
	x, y, length float32) error {
	// strategy for arrows: angle 30 degrees, arrow arm length = 9 * line width
	// cos(angle) = x position
	// sin(angle) = y position
	// this comes very close to what Adobe is doing
	armX := x + float32(math.Cos(arrowAngle)*float64(length))
	armYdelta := float32(math.Sin(arrowAngle) * float64(length))
	if err := cs.MoveTo(armX, y+armYdelta); err != nil {
		return err
	}
	if err := cs.LineTo(x, y); err != nil {
		return err
	}
	return cs.LineTo(armX, y-armYdelta)
}

// DrawDiamond draws a diamond of the given radius. Java declares it
// package-private.
func (h *PDAbstractAppearanceHandler) DrawDiamond(cs annotation.AppearanceContentStream,
	x, y, r float32) error {
	if err := cs.MoveTo(x-r, y); err != nil {
		return err
	}
	if err := cs.LineTo(x, y+r); err != nil {
		return err
	}
	if err := cs.LineTo(x+r, y); err != nil {
		return err
	}
	if err := cs.LineTo(x, y-r); err != nil {
		return err
	}
	return cs.ClosePath()
}

// DrawCircle draws a circle of the given radius, anticlockwise. Java declares
// it package-private.
func (h *PDAbstractAppearanceHandler) DrawCircle(cs annotation.AppearanceContentStream,
	x, y, r float32) error {
	// http://stackoverflow.com/a/2007782/535646
	magic := r * 0.551784
	if err := cs.MoveTo(x, y+r); err != nil {
		return err
	}
	if err := cs.CurveTo(x+magic, y+r, x+r, y+magic, x+r, y); err != nil {
		return err
	}
	if err := cs.CurveTo(x+r, y-magic, x+magic, y-r, x, y-r); err != nil {
		return err
	}
	if err := cs.CurveTo(x-magic, y-r, x-r, y-magic, x-r, y); err != nil {
		return err
	}
	if err := cs.CurveTo(x-r, y+magic, x-magic, y+r, x, y+r); err != nil {
		return err
	}
	return cs.ClosePath()
}

// DrawCircle2 draws a circle of the given radius, clockwise. Java declares it
// package-private.
func (h *PDAbstractAppearanceHandler) DrawCircle2(cs annotation.AppearanceContentStream,
	x, y, r float32) error {
	// http://stackoverflow.com/a/2007782/535646
	magic := r * 0.551784
	if err := cs.MoveTo(x, y+r); err != nil {
		return err
	}
	if err := cs.CurveTo(x-magic, y+r, x-r, y+magic, x-r, y); err != nil {
		return err
	}
	if err := cs.CurveTo(x-r, y-magic, x-magic, y-r, x, y-r); err != nil {
		return err
	}
	if err := cs.CurveTo(x+magic, y-r, x+r, y-magic, x+r, y); err != nil {
		return err
	}
	if err := cs.CurveTo(x+r, y+magic, x+magic, y+r, x, y+r); err != nil {
		return err
	}
	return cs.ClosePath()
}

// normalAppearance returns the normal appearance entry, replacing a missing one
// or a sub-dictionary with a fresh stream. Java declares it private.
func (h *PDAbstractAppearanceHandler) normalAppearance() *annotation.PDAppearanceEntry {
	appearanceDictionary := h.Appearance()
	normalAppearanceEntry := appearanceDictionary.NormalAppearance()
	if normalAppearanceEntry == nil || normalAppearanceEntry.IsSubDictionary() {
		normalAppearanceEntry = annotation.NewPDAppearanceEntryOf(h.createCOSStream())
		appearanceDictionary.SetNormalAppearance(normalAppearanceEntry)
	}
	return normalAppearanceEntry
}

// appearanceEntryAsContentStream returns the given entry to draw into, having
// set its bounding box and matrix and given it resources. Java declares it
// private.
func (h *PDAbstractAppearanceHandler) appearanceEntryAsContentStream(
	appearanceEntry *annotation.PDAppearanceEntry, compress bool) (
	annotation.AppearanceContentStream, error) {
	appearanceStream := appearanceEntry.AppearanceStream()
	h.setTransformationMatrix(appearanceStream)

	// ensure there are resources
	if appearanceStream.Resources() == nil {
		appearanceStream.SetResources(form.NewEmptyResources())
	}

	return annotation.NewAppearanceContentStream(appearanceStream, compress)
}

// setTransformationMatrix sets the bounding box and the matrix of the given
// appearance to the annotation's rectangle. Java declares it private.
func (h *PDAbstractAppearanceHandler) setTransformationMatrix(
	appearanceStream *annotation.PDAppearanceStream) {
	bbox := h.Rectangle()
	appearanceStream.SetBBox(bbox)
	if bbox == nil {
		return
	}
	appearanceStream.SetMatrix(util.NewMatrixOf(1, 0, 0, 1,
		-bbox.LowerLeftX(), -bbox.LowerLeftY()))
}

// HandleBorderBox returns the box a square or circle annotation's border is
// drawn in, setting the rectangle differences where they are missing. Java
// declares it package-private.
func (h *PDAbstractAppearanceHandler) HandleBorderBox(
	annot *annotation.PDAnnotationSquareCircle, lineWidth float32) *common.PDRectangle {
	// There are two options. The handling is not part of the PDF specification but
	// implementation specific to Adobe Reader
	// - if /RD is set the border box is the /Rect entry inset by the respective
	//   border difference.
	// - if /RD is not set the border box is defined by the /Rect entry. The /RD entry will
	//   be set to be the line width and the /Rect is enlarged by the /RD amount
	rectDifferences := annot.RectDifferences()
	if len(rectDifferences) == 0 {
		borderBox := h.PaddedRectangle(h.Rectangle(), lineWidth/2)

		// the differences rectangle
		annot.SetRectDifferences(lineWidth / 2)
		annot.SetRectangle(h.AddRectDifferences(h.Rectangle(), annot.RectDifferences()))

		// when the normal appearance stream was generated BBox and Matrix have been set to the
		// values of the original /Rect. As the /Rect was changed that needs to be adjusted too.
		rect := h.Rectangle()
		appearanceStream := annot.NormalAppearanceStream()
		appearanceStream.SetBBox(rect)
		appearanceStream.SetMatrix(util.NewMatrixOf(1, 0, 0, 1,
			-rect.LowerLeftX(), -rect.LowerLeftY()))
		return borderBox
	}
	borderBox := h.ApplyRectDifferences(h.Rectangle(), rectDifferences)
	return h.PaddedRectangle(borderBox, lineWidth/2)
}

// annotationBorder is the width, dash pattern and underline flag of an
// annotation's border.
//
// Port of the package-private class AnnotationBorder.
type annotationBorder struct {
	dashArray []float32
	underline bool
	width     float32
}

// getAnnotationBorder returns the border of the given annotation.
//
// Port of the static AnnotationBorder.getAnnotationBorder. The border style is
// a parameter because the method is not available in the base class.
func getAnnotationBorder(annot annotation.PDAnnotation,
	borderStyle *annotation.PDBorderStyleDictionary) *annotationBorder {
	ab := &annotationBorder{}
	if borderStyle == nil {
		border := annot.Border()
		if border.Size() >= 3 {
			if base, isNumber := border.GetObject(2).(cos.Number); isNumber {
				ab.width = base.FloatValue()
			}
		}
		if border.Size() > 3 {
			if base3, isArray := border.GetObject(3).(*cos.Array); isArray {
				ab.dashArray = base3.ToFloatArray()
			}
		}
	} else {
		ab.width = borderStyle.Width()
		style := borderStyle.Style()
		if style == annotation.BorderStyleDashed {
			ab.dashArray = borderStyle.DashStyle().DashArray()
		}
		if style == annotation.BorderStyleUnderline {
			ab.underline = true
		}
	}
	if ab.dashArray != nil {
		allZero := true
		for _, f := range ab.dashArray {
			if f != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			ab.dashArray = nil
		}
	}
	return ab
}
