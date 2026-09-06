package handlers

import (
	"log/slog"
	"math"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/blend"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// quadPointsBounds grows the annotation rectangle to hold every quad point,
// padded by the given amounts, and returns it.
//
// Port of the block the underline, strikeout, squiggly and highlight handlers
// each carry a copy of: "Adjust rectangle even if not empty, see
// PLPDF.com-MarkupAnnotations.pdf".
func quadPointsBounds(rect *common.PDRectangle, pathsArray []float32,
	lowerPadding, upperPadding float32) *common.PDRectangle {
	minX := float32(math.MaxFloat32)
	minY := float32(math.MaxFloat32)
	maxX := float32(smallestPositiveFloat32)
	maxY := float32(smallestPositiveFloat32)
	for i := 0; i < len(pathsArray)/2; i++ {
		x := pathsArray[i*2]
		y := pathsArray[i*2+1]
		minX = minFloat32(minX, x)
		minY = minFloat32(minY, y)
		maxX = maxFloat32(maxX, x)
		maxY = maxFloat32(maxY, y)
	}
	rect.SetLowerLeftX(minFloat32(minX-lowerPadding, rect.LowerLeftX()))
	rect.SetLowerLeftY(minFloat32(minY-lowerPadding, rect.LowerLeftY()))
	rect.SetUpperRightX(maxFloat32(maxX+upperPadding, rect.UpperRightX()))
	rect.SetUpperRightY(maxFloat32(maxY+upperPadding, rect.UpperRightY()))
	return rect
}

// PDUnderlineAppearanceHandler draws an underline annotation.
//
// Port of PDUnderlineAppearanceHandler.
type PDUnderlineAppearanceHandler struct {
	PDAbstractAppearanceHandler
}

// NewPDUnderlineAppearanceHandler builds a handler for the given annotation.
func NewPDUnderlineAppearanceHandler(annot annotation.PDAnnotation) *PDUnderlineAppearanceHandler {
	return NewPDUnderlineAppearanceHandlerInDocument(annot, nil)
}

// NewPDUnderlineAppearanceHandlerInDocument builds one whose streams belong to
// the given document.
func NewPDUnderlineAppearanceHandlerInDocument(annot annotation.PDAnnotation,
	document common.COSDocumentLike) *PDUnderlineAppearanceHandler {
	h := &PDUnderlineAppearanceHandler{}
	h.initAppearanceHandler(h, annot, document)
	return h
}

// GenerateNormalAppearance draws a line under each quad.
func (h *PDUnderlineAppearanceHandler) GenerateNormalAppearance() error {
	annot, isUnderline := h.Annotation().(*annotation.PDAnnotationUnderline)
	if !isUnderline {
		panic("handlers: the annotation of an underline handler is not an underline")
	}
	rect := annot.Rectangle()
	if rect == nil {
		return nil
	}
	pathsArray := annot.QuadPoints()
	if pathsArray == nil {
		return nil
	}
	ab := getAnnotationBorder(annot, annot.BorderStyle())
	markupColor := annot.Color()
	if markupColor == nil || len(markupColor.Components()) == 0 {
		return nil
	}
	if ab.width == 0 {
		// value found in adobe reader
		ab.width = 1.5
	}

	// Adjust rectangle even if not empty, see PLPDF.com-MarkupAnnotations.pdf
	// TODO in a class structure this should be overridable
	// this is similar to polyline but different data type
	// all coordinates (unlike painting) are used because I'm lazy
	annot.SetRectangle(quadPointsBounds(rect, pathsArray, ab.width/2, ab.width/2))

	cs, err := h.NormalAppearanceAsContentStream()
	if err != nil {
		slog.Error("handlers: underline appearance", slog.Any("error", err))
		return nil
	}
	defer cs.Close()

	if err := h.drawUnderline(annot, ab, markupColor, pathsArray, cs); err != nil {
		slog.Error("handlers: underline appearance", slog.Any("error", err))
	}
	return nil
}

// drawUnderline is the body of the try block Java writes.
func (h *PDUnderlineAppearanceHandler) drawUnderline(annot *annotation.PDAnnotationUnderline,
	ab *annotationBorder, markupColor *color.PDColor, pathsArray []float32,
	cs annotation.AppearanceContentStream) error {
	if err := h.SetOpacity(cs, annot.ConstantOpacity()); err != nil {
		return err
	}
	if err := cs.SetStrokingColor(markupColor); err != nil {
		return err
	}
	if ab.dashArray != nil {
		if err := cs.SetLineDashPattern(ab.dashArray, 0); err != nil {
			return err
		}
	}
	if err := cs.SetLineWidth(ab.width); err != nil {
		return err
	}

	// spec is incorrect
	// https://stackoverflow.com/questions/9855814/pdf-spec-vs-acrobat-creation-quadpoints
	for i := 0; i < len(pathsArray)/8; i++ {
		// Adobe doesn't use the lower coordinate for the line, it uses lower + delta / 7.
		// do the math for diagonal annotations with this weird old trick:
		// https://stackoverflow.com/questions/7740507/extend-a-line-segment-a-specific-distance
		len0 := float32(math.Sqrt(
			math.Pow(float64(pathsArray[i*8]-pathsArray[i*8+4]), 2) +
				math.Pow(float64(pathsArray[i*8+1]-pathsArray[i*8+5]), 2)))
		x0 := pathsArray[i*8+4]
		y0 := pathsArray[i*8+5]
		if len0 != 0 {
			// only if both coordinates are not identical to avoid divide by zero
			x0 += (pathsArray[i*8] - pathsArray[i*8+4]) / len0 * len0 / 7
			y0 += (pathsArray[i*8+1] - pathsArray[i*8+5]) / len0 * (len0 / 7)
		}
		len1 := float32(math.Sqrt(
			math.Pow(float64(pathsArray[i*8+2]-pathsArray[i*8+6]), 2) +
				math.Pow(float64(pathsArray[i*8+3]-pathsArray[i*8+7]), 2)))
		x1 := pathsArray[i*8+6]
		y1 := pathsArray[i*8+7]
		if len1 != 0 {
			// only if both coordinates are not identical to avoid divide by zero
			x1 += (pathsArray[i*8+2] - pathsArray[i*8+6]) / len1 * len1 / 7
			y1 += (pathsArray[i*8+3] - pathsArray[i*8+7]) / len1 * len1 / 7
		}
		if err := cs.MoveTo(x0, y0); err != nil {
			return err
		}
		if err := cs.LineTo(x1, y1); err != nil {
			return err
		}
	}
	return cs.Stroke()
}

// GenerateRolloverAppearance does nothing: no rollover appearance generated.
func (h *PDUnderlineAppearanceHandler) GenerateRolloverAppearance() error { return nil }

// GenerateDownAppearance does nothing: no down appearance generated.
func (h *PDUnderlineAppearanceHandler) GenerateDownAppearance() error { return nil }

// PDStrikeoutAppearanceHandler draws a strikeout annotation.
//
// Port of PDStrikeoutAppearanceHandler.
type PDStrikeoutAppearanceHandler struct {
	PDAbstractAppearanceHandler
}

// NewPDStrikeoutAppearanceHandler builds a handler for the given annotation.
func NewPDStrikeoutAppearanceHandler(annot annotation.PDAnnotation) *PDStrikeoutAppearanceHandler {
	return NewPDStrikeoutAppearanceHandlerInDocument(annot, nil)
}

// NewPDStrikeoutAppearanceHandlerInDocument builds one whose streams belong to
// the given document.
func NewPDStrikeoutAppearanceHandlerInDocument(annot annotation.PDAnnotation,
	document common.COSDocumentLike) *PDStrikeoutAppearanceHandler {
	h := &PDStrikeoutAppearanceHandler{}
	h.initAppearanceHandler(h, annot, document)
	return h
}

// GenerateNormalAppearance draws a line through each quad.
func (h *PDStrikeoutAppearanceHandler) GenerateNormalAppearance() error {
	annot, isStrikeout := h.Annotation().(*annotation.PDAnnotationStrikeout)
	if !isStrikeout {
		panic("handlers: the annotation of a strikeout handler is not a strikeout")
	}
	rect := annot.Rectangle()
	if rect == nil {
		return nil
	}
	pathsArray := annot.QuadPoints()
	if pathsArray == nil {
		return nil
	}
	ab := getAnnotationBorder(annot, annot.BorderStyle())
	markupColor := annot.Color()
	if markupColor == nil || len(markupColor.Components()) == 0 {
		return nil
	}
	if ab.width == 0 {
		// value found in adobe reader
		ab.width = 1.5
	}

	// Adjust rectangle even if not empty, see PLPDF.com-MarkupAnnotations.pdf
	// TODO in a class structure this should be overridable
	// this is similar to polyline but different data type
	annot.SetRectangle(quadPointsBounds(rect, pathsArray, ab.width/2, ab.width/2))

	cs, err := h.NormalAppearanceAsContentStream()
	if err != nil {
		slog.Error("handlers: strikeout appearance", slog.Any("error", err))
		return nil
	}
	defer cs.Close()

	if err := h.drawStrikeout(annot, ab, markupColor, pathsArray, cs); err != nil {
		slog.Error("handlers: strikeout appearance", slog.Any("error", err))
	}
	return nil
}

// drawStrikeout is the body of the try block Java writes.
func (h *PDStrikeoutAppearanceHandler) drawStrikeout(annot *annotation.PDAnnotationStrikeout,
	ab *annotationBorder, markupColor *color.PDColor, pathsArray []float32,
	cs annotation.AppearanceContentStream) error {
	if err := h.SetOpacity(cs, annot.ConstantOpacity()); err != nil {
		return err
	}
	if err := cs.SetStrokingColor(markupColor); err != nil {
		return err
	}
	if ab.dashArray != nil {
		if err := cs.SetLineDashPattern(ab.dashArray, 0); err != nil {
			return err
		}
	}
	if err := cs.SetLineWidth(ab.width); err != nil {
		return err
	}

	// spec is incorrect
	// https://stackoverflow.com/questions/9855814/pdf-spec-vs-acrobat-creation-quadpoints
	for i := 0; i < len(pathsArray)/8; i++ {
		// get mid point between bounds, subtract the line width to approximate what Adobe is doing
		// See e.g. CTAN-example-Annotations.pdf and PLPDF.com-MarkupAnnotations.pdf
		// and https://bugs.ghostscript.com/show_bug.cgi?id=693664
		// do the math for diagonal annotations with this weird old trick:
		// https://stackoverflow.com/questions/7740507/extend-a-line-segment-a-specific-distance
		len0 := float32(math.Sqrt(
			math.Pow(float64(pathsArray[i*8]-pathsArray[i*8+4]), 2) +
				math.Pow(float64(pathsArray[i*8+1]-pathsArray[i*8+5]), 2)))
		x0 := pathsArray[i*8+4]
		y0 := pathsArray[i*8+5]
		if len0 != 0 {
			// only if both coordinates are not identical to avoid divide by zero
			x0 += (pathsArray[i*8] - pathsArray[i*8+4]) / len0 * (len0/2 - ab.width)
			y0 += (pathsArray[i*8+1] - pathsArray[i*8+5]) / len0 * (len0/2 - ab.width)
		}
		len1 := float32(math.Sqrt(
			math.Pow(float64(pathsArray[i*8+2]-pathsArray[i*8+6]), 2) +
				math.Pow(float64(pathsArray[i*8+3]-pathsArray[i*8+7]), 2)))
		x1 := pathsArray[i*8+6]
		y1 := pathsArray[i*8+7]
		if len1 != 0 {
			// only if both coordinates are not identical to avoid divide by zero
			x1 += (pathsArray[i*8+2] - pathsArray[i*8+6]) / len1 * (len1/2 - ab.width)
			y1 += (pathsArray[i*8+3] - pathsArray[i*8+7]) / len1 * (len1/2 - ab.width)
		}
		if err := cs.MoveTo(x0, y0); err != nil {
			return err
		}
		if err := cs.LineTo(x1, y1); err != nil {
			return err
		}
	}
	return cs.Stroke()
}

// GenerateRolloverAppearance does nothing: no rollover appearance generated.
func (h *PDStrikeoutAppearanceHandler) GenerateRolloverAppearance() error { return nil }

// GenerateDownAppearance does nothing: no down appearance generated.
func (h *PDStrikeoutAppearanceHandler) GenerateDownAppearance() error { return nil }

// PDHighlightAppearanceHandler draws a highlight annotation.
//
// Port of PDHighlightAppearanceHandler.
type PDHighlightAppearanceHandler struct {
	PDAbstractAppearanceHandler
}

// NewPDHighlightAppearanceHandler builds a handler for the given annotation.
func NewPDHighlightAppearanceHandler(annot annotation.PDAnnotation) *PDHighlightAppearanceHandler {
	return NewPDHighlightAppearanceHandlerInDocument(annot, nil)
}

// NewPDHighlightAppearanceHandlerInDocument builds one whose streams belong to
// the given document.
func NewPDHighlightAppearanceHandlerInDocument(annot annotation.PDAnnotation,
	document common.COSDocumentLike) *PDHighlightAppearanceHandler {
	h := &PDHighlightAppearanceHandler{}
	h.initAppearanceHandler(h, annot, document)
	return h
}

// GenerateNormalAppearance fills each quad, multiplied over the page.
func (h *PDHighlightAppearanceHandler) GenerateNormalAppearance() error {
	annot, isHighlight := h.Annotation().(*annotation.PDAnnotationHighlight)
	if !isHighlight {
		panic("handlers: the annotation of a highlight handler is not a highlight")
	}
	pathsArray := annot.QuadPoints()
	if pathsArray == nil {
		return nil
	}
	markupColor := annot.Color()
	if markupColor == nil || len(markupColor.Components()) == 0 {
		return nil
	}
	rect := annot.Rectangle()
	if rect == nil {
		return nil
	}
	ab := getAnnotationBorder(annot, annot.BorderStyle())

	// Adjust rectangle even if not empty, see PLPDF.com-MarkupAnnotations.pdf
	// TODO in a class structure this should be overridable
	// this is similar to polyline but different data type
	// TODO padding should consider the curves too; needs to know in advance where the curve is
	minX := float32(math.MaxFloat32)
	minY := float32(math.MaxFloat32)
	maxX := float32(smallestPositiveFloat32)
	maxY := float32(smallestPositiveFloat32)
	for i := 0; i < len(pathsArray)/2; i++ {
		x := pathsArray[i*2]
		y := pathsArray[i*2+1]
		minX = minFloat32(minX, x)
		minY = minFloat32(minY, y)
		maxX = maxFloat32(maxX, x)
		maxY = maxFloat32(maxY, y)
	}

	// get the delta used for curves and use it for padding
	maxDelta := float32(0)
	for i := 0; i < len(pathsArray)/8; i++ {
		// one of the two is 0, depending whether the rectangle is
		// horizontal or vertical
		// if it is diagonal then... uh...
		delta := maxFloat32((pathsArray[i+0]-pathsArray[i+4])/4,
			(pathsArray[i+1]-pathsArray[i+5])/4)
		maxDelta = maxFloat32(delta, maxDelta)
	}

	rect.SetLowerLeftX(minFloat32(minX-ab.width/2-maxDelta, rect.LowerLeftX()))
	rect.SetLowerLeftY(minFloat32(minY-ab.width/2-maxDelta, rect.LowerLeftY()))
	rect.SetUpperRightX(maxFloat32(maxX+ab.width+maxDelta, rect.UpperRightX()))
	rect.SetUpperRightY(maxFloat32(maxY+ab.width+maxDelta, rect.UpperRightY()))
	annot.SetRectangle(rect)

	cs, err := h.NormalAppearanceAsContentStream()
	if err != nil {
		slog.Error("handlers: highlight appearance", slog.Any("error", err))
		return nil
	}
	defer cs.Close()

	if err := h.drawHighlight(annot, markupColor, pathsArray, cs); err != nil {
		slog.Error("handlers: highlight appearance", slog.Any("error", err))
	}
	return nil
}

// drawHighlight is the body of the try block Java writes.
func (h *PDHighlightAppearanceHandler) drawHighlight(annot *annotation.PDAnnotationHighlight,
	markupColor *color.PDColor, pathsArray []float32,
	cs annotation.AppearanceContentStream) error {
	r0 := state.NewPDExtendedGraphicsState()
	r1 := state.NewPDExtendedGraphicsState()
	r0.SetAlphaSourceFlag(false)
	opacity := annot.ConstantOpacity()
	r0.SetStrokingAlphaConstant(&opacity)
	r0.SetNonStrokingAlphaConstant(&opacity)
	r1.SetAlphaSourceFlag(false)
	r1.SetBlendMode(blend.Multiply)
	if err := cs.SetGraphicsStateParameters(r0); err != nil {
		return err
	}
	if err := cs.SetGraphicsStateParameters(r1); err != nil {
		return err
	}

	frm1 := form.NewPDFormXObjectOfStream(h.createCOSStream())
	frm2 := form.NewPDFormXObjectOfStream(h.createCOSStream())
	frm1.SetResources(form.NewEmptyResources())

	mwfofrmCS, err := annotation.NewFormContentStream(frm1)
	if err != nil {
		return err
	}
	if err := mwfofrmCS.DrawForm(frm2); err != nil {
		mwfofrmCS.Close()
		return err
	}
	if err := mwfofrmCS.Close(); err != nil {
		return err
	}

	frm1.SetBBox(annot.Rectangle())
	frm1.SetGroup(form.NewPDTransparencyGroupAttributes())
	if err := cs.DrawForm(frm1); err != nil {
		return err
	}
	frm2.SetBBox(annot.Rectangle())

	frm2CS, err := annotation.NewFormContentStream(frm2)
	if err != nil {
		return err
	}
	defer frm2CS.Close()
	return h.drawQuads(markupColor, pathsArray, frm2CS)
}

// drawQuads fills each quad of the highlight, with the curved ends Adobe draws.
func (h *PDHighlightAppearanceHandler) drawQuads(markupColor *color.PDColor,
	pathsArray []float32, frm2CS annotation.FormContentStream) error {
	if err := frm2CS.SetNonStrokingColor(markupColor); err != nil {
		return err
	}
	of := 0
	for of+7 < len(pathsArray) {
		// quadpoints spec sequence is incorrect, correct one is (4,5 0,1 2,3 6,7)
		// https://stackoverflow.com/questions/9855814/pdf-spec-vs-acrobat-creation-quadpoints

		// for "curvy" highlighting, two Bezier control points are used that seem to have a
		// distance of about 1/4 of the height.
		// note that curves won't appear if outside of the rectangle
		delta := float32(0)
		if pathsArray[of+0] == pathsArray[of+4] &&
			pathsArray[of+1] == pathsArray[of+3] &&
			pathsArray[of+2] == pathsArray[of+6] &&
			pathsArray[of+5] == pathsArray[of+7] {
			// horizontal highlight
			delta = (pathsArray[of+1] - pathsArray[of+5]) / 4
		} else if pathsArray[of+1] == pathsArray[of+5] &&
			pathsArray[of+0] == pathsArray[of+2] &&
			pathsArray[of+3] == pathsArray[of+7] &&
			pathsArray[of+4] == pathsArray[of+6] {
			// vertical highlight
			delta = (pathsArray[of+0] - pathsArray[of+4]) / 4
		}

		if err := frm2CS.MoveTo(pathsArray[of+4], pathsArray[of+5]); err != nil {
			return err
		}
		switch {
		case pathsArray[of+0] == pathsArray[of+4]:
			// horizontal highlight
			if err := frm2CS.CurveTo(pathsArray[of+4]-delta, pathsArray[of+5]+delta,
				pathsArray[of+0]-delta, pathsArray[of+1]-delta,
				pathsArray[of+0], pathsArray[of+1]); err != nil {
				return err
			}
		case pathsArray[of+5] == pathsArray[of+1]:
			// vertical highlight
			if err := frm2CS.CurveTo(pathsArray[of+4]+delta, pathsArray[of+5]+delta,
				pathsArray[of+0]-delta, pathsArray[of+1]+delta,
				pathsArray[of+0], pathsArray[of+1]); err != nil {
				return err
			}
		default:
			if err := frm2CS.LineTo(pathsArray[of+0], pathsArray[of+1]); err != nil {
				return err
			}
		}

		if err := frm2CS.LineTo(pathsArray[of+2], pathsArray[of+3]); err != nil {
			return err
		}
		switch {
		case pathsArray[of+2] == pathsArray[of+6]:
			// horizontal highlight
			if err := frm2CS.CurveTo(pathsArray[of+2]+delta, pathsArray[of+3]-delta,
				pathsArray[of+6]+delta, pathsArray[of+7]+delta,
				pathsArray[of+6], pathsArray[of+7]); err != nil {
				return err
			}
		case pathsArray[of+3] == pathsArray[of+7]:
			// vertical highlight
			if err := frm2CS.CurveTo(pathsArray[of+2]-delta, pathsArray[of+3]-delta,
				pathsArray[of+6]+delta, pathsArray[of+7]-delta,
				pathsArray[of+6], pathsArray[of+7]); err != nil {
				return err
			}
		default:
			if err := frm2CS.LineTo(pathsArray[of+6], pathsArray[of+7]); err != nil {
				return err
			}
		}

		if err := frm2CS.Fill(); err != nil {
			return err
		}
		of += 8
	}
	return nil
}

// GenerateRolloverAppearance does nothing: no rollover appearance generated.
func (h *PDHighlightAppearanceHandler) GenerateRolloverAppearance() error { return nil }

// GenerateDownAppearance does nothing: no down appearance generated.
func (h *PDHighlightAppearanceHandler) GenerateDownAppearance() error { return nil }

// PDSquigglyAppearanceHandler draws a squiggly underline annotation.
//
// Port of PDSquigglyAppearanceHandler.
//
// generateNormalAppearance is not ported: it fills the squiggle with a tiling
// pattern, and PDTilingPattern, PDPatternContentStream and the PDPattern colour
// space all belong to the rendering this port has not reached. See
// migration/STATUS.md.
type PDSquigglyAppearanceHandler struct {
	PDAbstractAppearanceHandler
}

// NewPDSquigglyAppearanceHandler builds a handler for the given annotation.
func NewPDSquigglyAppearanceHandler(annot annotation.PDAnnotation) *PDSquigglyAppearanceHandler {
	return NewPDSquigglyAppearanceHandlerInDocument(annot, nil)
}

// NewPDSquigglyAppearanceHandlerInDocument builds one whose streams belong to
// the given document.
func NewPDSquigglyAppearanceHandlerInDocument(annot annotation.PDAnnotation,
	document common.COSDocumentLike) *PDSquigglyAppearanceHandler {
	h := &PDSquigglyAppearanceHandler{}
	h.initAppearanceHandler(h, annot, document)
	return h
}

// GenerateNormalAppearance does nothing: the squiggle is drawn with a tiling
// pattern this port has not reached.
func (h *PDSquigglyAppearanceHandler) GenerateNormalAppearance() error { return nil }

// GenerateRolloverAppearance does nothing: no rollover appearance generated.
func (h *PDSquigglyAppearanceHandler) GenerateRolloverAppearance() error { return nil }

// GenerateDownAppearance does nothing: no down appearance generated.
func (h *PDSquigglyAppearanceHandler) GenerateDownAppearance() error { return nil }
